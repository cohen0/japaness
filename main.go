package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	_ "github.com/go-sql-driver/mysql"
)

type Langrage struct {
	Ping string `json:"ping"`
	Pian string `json:"pian"`
	Yin  string `json:"yin"`
	Row  string `json:"row"`
}

type Record struct {
	Char  string `db:"chars"`
	Wrong int    `db:"wrong"`
	Yin   string `db:"yin"`
}

var gcount = 1
var pool = map[int]*Langrage{}
var poolslice []Langrage
var dbmap *gorp.DbMap
var insert = map[string]*Record{}
var update = map[string]*Record{}
var wordsnum = 15
var wordslen = 4

func init() {
	rand.Seed(time.Now().UnixNano())
}

func initDbMap() error {
	db, err := sql.Open("mysql", "root:chen1992@tcp(127.0.0.1:3306)/japaness")
	if err != nil {
		fmt.Printf("open db err:%v", err)
		return err
	}
	err = db.Ping()
	if err != nil {
		fmt.Printf("ping err:%v", err)
		return err
	}
	dbmap = &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{}}
	return nil
}

func LoadJson() error {
	data, err := ioutil.ReadFile("./pool.json")
	if err != nil {
		fmt.Printf("read file err:%v", err)
		return err
	}
	var tmp []*Langrage
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		fmt.Printf("unmarshal err:%v", err)
		return err
	}
	for i, v := range tmp {
		pool[i] = v
		poolslice = append(poolslice, *v)
	}
	return nil
}

func LoadDb() error {
	var records []Record
	_, err := dbmap.Select(&records, "select * from record")
	if err != nil {
		return err
	}
	for i, v := range records {
		update[v.Char] = &records[i]
	}
	return nil
}

func ToDb() {
	//insert
	if len(insert) > 0 {
		sqlstr := fmt.Sprint("insert into `record` values")
		vals := []interface{}{}
		for _, v := range insert {
			sqlstr += "(?,?,?),"
			vals = append(vals, v.Char, v.Wrong, v.Yin)
		}
		sqlstr = sqlstr[0 : len(sqlstr)-1] //删除最后一个','
		stmt, err := dbmap.Db.Prepare(sqlstr)
		if err != nil || stmt == nil {
			fmt.Printf("Prepare err:%v, sql:%s", err, sqlstr)
			return
		}
		_, err = stmt.Exec(vals...)
		if err != nil {
			fmt.Printf("insert error:%v", err)
		}
	}
	//update
	for _, v := range update {
		sql := "update `record` set `wrong`=?,`yin`=? where `chars`=?"
		_, err := dbmap.Db.Exec(sql, v.Wrong, v.Yin, v.Char)
		if err != nil {
			fmt.Printf("update error:%v", err)
			return
		}
	}
}

func addWrong(chars, yin string) {
	info, has := update[chars]
	if has {
		info.Wrong++
		info.Yin = yin
	} else {
		v, has1 := insert[chars]
		if has1 {
			v.Wrong++
		} else {
			insert[chars] = &Record{
				Char:  chars,
				Yin:   yin,
				Wrong: 1,
			}
		}
	}
}

func TestOne() {
	slice := rand.Perm(len(pool))
	wrong := make([]int, 0)
	for _, k := range slice {
		lan := pool[k]
		count := gcount
		fmt.Printf("[%s]:", lan.Ping)
		for {
			var answer string
			fmt.Scan(&answer)
			if answer == lan.Yin {
				break
			} else {
				count--
				if count == 0 {
					break
				}
				fmt.Printf("[%s]:", lan.Ping)
			}
		}
		if count == 0 {
			fmt.Printf("  [%s]==>%s\n", lan.Ping, lan.Yin)
			wrong = append(wrong, k)
			addWrong(lan.Ping, lan.Yin)
		}
	}
}

func createWords() []int {
	slice := rand.Perm(len(pool))
	if wordslen > len(slice) {
		return nil
	}
	randlen := rand.Intn(wordslen) + 1
	return slice[:randlen]
}

func TestWords() {
	for i := 0; i < wordsnum; i++ {
		words := createWords()

		var show string
		for _, k := range words {
			lan := pool[k]
			show += lan.Ping
		}
		fmt.Printf("[%s]:", show)

		var input string
		fmt.Scan(&input)
		anwser := strings.Split(input, ",")
		if len(words) == 0 || len(words) != len(anwser) {
			continue
		}
		for i := 0; i < len(words); i++ {
			lan := pool[words[i]]
			if lan.Yin != anwser[i] {
				fmt.Println(lan.Ping, "==>", lan.Yin)
				addWrong(lan.Ping, lan.Yin)
			}
		}
	}
}

func PrintAll() {
	for _, v := range poolslice {
		fmt.Printf("%s\t", v.Ping)
		if v.Row == "-" {
			println("")
		}
	}
}

func main() {
	if err := initDbMap(); err != nil {
		return
	}
	if err := LoadJson(); err != nil {
		return
	}
	if err := LoadDb(); err != nil {
		return
	}
	for {
		fmt.Println("[0]:PrintAll")
		fmt.Println("[1]:Test Single")
		fmt.Println("[2]:Test Words")
		fmt.Printf("Enter:")
		var number int
		fmt.Scan(&number)
		switch number {
		case 0:
			PrintAll()
		case 1:
			TestOne()
		case 2:
			TestWords()
		default:
			ToDb()
			return
		}
		fmt.Println("############################")
	}
}
