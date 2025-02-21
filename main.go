package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Wordsum  int
	Wordlen  int
	Dbip     string
	Dbport   string
	Dbuser   string
	Dbpasswd string
	Dbtable  string
}

var gconf Config

func initConfig() error {
	bs, err := os.ReadFile("./config.yaml")
	if err != nil {
		println(err)
		return err
	}

	err = yaml.Unmarshal(bs, &gconf)
	if err != nil {
		println(err)
		return err
	}

	println("parse config success!!")
	return nil
}

const (
	OP_Qing = iota
	OP_Zhuo
	OP_All
)

const (
	OP_Ping = iota
	OP_Pian
)

type Langrage struct {
	Ping string `json:"ping"`
	Pian string `json:"pian"`
	Yin  string `json:"yin"`
	Row  string `json:"row"`
	Op   int    `json:"op"`
}

type Record struct {
	Char  string `db:"chars"`
	Wrong int    `db:"wrong"`
	Yin   string `db:"yin"`
}

type Test struct {
	dates   []*Langrage
	qing    []*Langrage
	zhuo    []*Langrage
	datem   map[int]*Langrage
	winsert map[string]*Record
	wupdate map[string]*Record
	dbmap   *gorp.DbMap
}

func NewTest(db *gorp.DbMap) *Test {
	t := Test{
		datem:   make(map[int]*Langrage),
		winsert: make(map[string]*Record),
		wupdate: make(map[string]*Record),
		dbmap:   db,
	}
	return &t
}

func (t *Test) loadJsons() error {
	load := func(path string) error {
		data, err := os.ReadFile(path)
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
		t.dates = append(t.dates, tmp...)
		for i, v := range tmp {
			t.datem[i] = v
			switch v.Op {
			case OP_Qing:
				t.qing = append(t.qing, v)
			case OP_Zhuo:
				t.zhuo = append(t.zhuo, v)
			}
		}
		return nil
	}

	var err error
	if err = load("./qing.json"); err != nil {
		return err
	}
	if err = load("./zhuo.json"); err != nil {
		return err
	}
	return nil
}

func (t *Test) printAll() {
	f := func(pool []*Langrage) {
		for _, v := range pool {
			fmt.Printf("[%s,%s]\t", v.Ping, v.Pian)
			switch v.Row {
			case "-":
				println("")
			case "1":
				fmt.Printf(" \t")
			case "2":
				fmt.Printf(" \t \t")
			case "3":
				fmt.Printf(" \t \t \t")
			}
		}
	}

	f(t.qing)
	f(t.zhuo)
}

func (t *Test) Select() bool {
	var number int

	fmt.Println("[0]:PrintAll")
	fmt.Println("[1]:Exit")
	fmt.Println("[3]:Test")
	fmt.Scanln(&number)
	if number == 0 {
		t.printAll()
		return false
	} else if number == 1 {
		return true
	}

	var pingOrpian int
	fmt.Println("[0]:Ping")
	fmt.Println("[1]:Pian")
	fmt.Scanln(&number)
	if number == 0 {
		pingOrpian = OP_Ping
	} else {
		pingOrpian = OP_Pian
	}

	fmt.Println("[0]:World")
	fmt.Println("[1]:Char")
	fmt.Scanln(&number)
	if number == 0 {
		t.words(pingOrpian)
		return false
	}

	var pool []*Langrage
	fmt.Println("[0]:qing")
	fmt.Println("[1]:zhuo")
	fmt.Println("[2]:all")
	fmt.Scanln(&number)
	if number == 0 {
		pool = t.qing
	} else if number == 1 {
		pool = t.zhuo
	} else {
		pool = t.dates
	}

	t.one(pingOrpian, pool)
	return false
}

func (t *Test) start() {
	for {
		exit := t.Select()
		if exit {
			break
		}
		fmt.Println("############################")
	}
}

func (t *Test) initDb() error {
	var records []Record
	_, err := t.dbmap.Select(&records, "select * from record")
	if err != nil {
		return err
	}
	for i, v := range records {
		t.wupdate[v.Char] = &records[i]
	}
	return nil
}

func (t *Test) toDb() {
	//insert
	if len(t.winsert) > 0 {
		sqlstr := "insert into `record` values"
		vals := []interface{}{}
		for _, v := range t.winsert {
			sqlstr += "(?,?,?),"
			vals = append(vals, v.Char, v.Wrong, v.Yin)
		}
		sqlstr = sqlstr[0 : len(sqlstr)-1] //删除最后一个','
		stmt, err := t.dbmap.Db.Prepare(sqlstr)
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
	for _, v := range t.wupdate {
		sql := "update `record` set `wrong`=?,`yin`=? where `chars`=?"
		_, err := t.dbmap.Db.Exec(sql, v.Wrong, v.Yin, v.Char)
		if err != nil {
			fmt.Printf("update error:%v", err)
			return
		}
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func initDbMap() (*gorp.DbMap, error) {
	connstr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", gconf.Dbuser, gconf.Dbpasswd, gconf.Dbip, gconf.Dbport, gconf.Dbtable)
	// db, err := sql.Open("mysql", "root:chen1992@tcp(127.0.0.1:3306)/japaness")
	db, err := sql.Open("mysql", connstr)
	if err != nil {
		fmt.Printf("open db err:%v", err)
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		fmt.Printf("ping err:%v", err)
		return nil, err
	}
	return &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{}}, nil
}

func (t *Test) addWrong(chars, yin string) {
	info, has := t.wupdate[chars]
	if has {
		info.Wrong++
		info.Yin = yin
	} else {
		v, has1 := t.winsert[chars]
		if has1 {
			v.Wrong++
		} else {
			t.winsert[chars] = &Record{
				Char:  chars,
				Yin:   yin,
				Wrong: 1,
			}
		}
	}
}

func (t *Test) getstring(lan *Langrage, flag int) string {
	if flag == OP_Ping {
		return lan.Ping
	} else {
		return lan.Pian
	}
}

func (t *Test) one(pingOrpian int, pool []*Langrage) {
	r := rand.Perm(len(pool))
	begin := time.Now()
	for _, k := range r {
		lan := pool[k]
		lstr := t.getstring(lan, pingOrpian)
		if lstr == "" {
			continue
		}
		fmt.Printf("[%s]:", lstr)
		var answer string
		fmt.Scan(&answer)
		if answer != lan.Yin {
			fmt.Printf("  [%s]==>%s\n", lstr, lan.Yin)
			t.addWrong(lstr, lan.Yin)
		}
	}
	fmt.Printf("use time:%v\n", time.Since(begin))
}

func (t *Test) createWords() []int {
	slice := rand.Perm(len(t.datem))
	if gconf.Wordlen > len(slice) {
		return nil
	}
	randlen := rand.Intn(gconf.Wordlen) + 1
	return slice[:randlen]
}

func (t *Test) words(pingOrpian int) {
	begin := time.Now()
	for i := 0; i < gconf.Wordsum; i++ {
		words := t.createWords()

		var show string
		for _, k := range words {
			lan := t.datem[k]
			lstr := t.getstring(lan, pingOrpian)
			if lstr == "" {
				continue
			}
			show += lstr
		}
		fmt.Printf("[%s]:", show)

		var input string
		fmt.Scan(&input)
		anwser := strings.Split(input, ",")
		if len(words) == 0 || len(words) != len(anwser) {
			continue
		}
		for i := 0; i < len(words); i++ {
			lan := t.datem[words[i]]
			if lan.Yin != anwser[i] {
				lstr := t.getstring(lan, pingOrpian)
				fmt.Println(lstr, "==>", lan.Yin)
				t.addWrong(lstr, lan.Yin)
			}
		}
	}
	fmt.Printf("use time:%v\n", time.Since(begin))
}

func main() {
	if err := initConfig(); err != nil {
		return
	}
	db, err := initDbMap()
	if err != nil {
		fmt.Printf("initdb err:%v\n", err)
		return
	}
	println("init db success!!")
	t := NewTest(db)
	if err := t.loadJsons(); err != nil {
		fmt.Printf("initdb err:%v\n", err)
		return
	}
	println("load json success!!")
	if err := t.initDb(); err != nil {
		fmt.Printf("initdb err:%v\n", err)
		return
	}
	t.start()
	t.toDb()
}
