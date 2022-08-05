package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"goDanmuku/config"

	_ "github.com/lib/pq"
)

// 数据库初始化
func initializeDanmuDb(db *sql.DB) {
	// 查询 config.ConfigData.GameName 数据库是否存在
	checkInitialize, PgsqlConnErr := db.Query(fmt.Sprintf((`SELECT u.datname FROM pg_catalog.pg_database u where u.datname='%s';`),
		config.ConfigData.GameName))
	checkErr(PgsqlConnErr)

	// 查询结果，存在 true ，不存在 false
	checkInitializeResult := checkInitialize.Next()

	// 不存在时创建 config.ConfigData.GameName 数据库，并按照 Scene 数量建表
	if !checkInitializeResult {
		fmt.Printf("首次启动，正在建立弹幕数据库 %s\n", config.ConfigData.GameName)
		_, pgsqlConnErr := db.Exec(fmt.Sprintf((`CREATE DATABASE %s;`), config.ConfigData.GameName))
		checkErr(pgsqlConnErr)
		fmt.Printf("弹幕数据库 %s 建立成功！\n", config.ConfigData.GameName)
		fmt.Printf("切换到弹幕数据库 %s 建立成功！\n", config.ConfigData.GameName)
		danmuDb, pgsqlConnErr := sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			config.ConfigData.Address, config.ConfigData.Port, config.ConfigData.Username, config.ConfigData.Password, config.ConfigData.GameName))
		checkErr(pgsqlConnErr)
		fmt.Printf("准备按 Scene 数量建表，将建表的数量为：%d\n", config.ConfigData.SceneNumber)
		// 按 Scene 数量建表
		for i := 1; i < config.ConfigData.SceneNumber+1; i++ {
			fmt.Printf("将建立表 Scene%d\n", i)
			_, pgsqlConnErr := danmuDb.Exec(fmt.Sprintf((`CREATE TABLE scene%d(
			textScript TEXT,
			textDanmu  TEXT
		)WITH (OIDS=FALSE);`), i))

			checkErr(pgsqlConnErr)
			_, pgsqlConnErr = danmuDb.Exec(fmt.Sprintf((`INSERT INTO scene%d(textScript,textDanmu) VALUES('luckykeeper','luckykeeper');`), i))
			checkErr(pgsqlConnErr)
		}
		fmt.Println("建表完成！")
	}
}

// 数据库连接与初始化（PgSql）
func connectPgsqldb() {
	// 数据库初始连接
	db, pgsqlConnErr := sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable",
		config.ConfigData.Address, config.ConfigData.Port, config.ConfigData.Username, config.ConfigData.Password))
	checkErr(pgsqlConnErr)
	// 数据库初始化
	initializeDanmuDb(db)
	// 连接指定到数据库
	db, pgsqlConnErr = sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.ConfigData.Address, config.ConfigData.Port, config.ConfigData.Username, config.ConfigData.Password, config.ConfigData.GameName))
	checkErr(pgsqlConnErr)
	fmt.Printf("弹幕数据库 %s 已存在并连接成功！\n", config.ConfigData.GameName)
	defer fmt.Println("数据库初始化完成并已关闭连接！")
	defer db.Close()
}

// 检查数据库连接
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

// 获取弹幕（分scene）
func getDanmu(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// 获取并检查 UA 是否合法
		r.ParseForm()
		ua := r.Header["User-Agent"]
		sceneId := r.Form["sceneId"][0]
		if !checkUA(ua[0]) {
			fmt.Fprintf(w, "Invalid Devices!")
		} else {
			// 开启数据库连接
			db, pgsqlConnErr := sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
				config.ConfigData.Address, config.ConfigData.Port, config.ConfigData.Username, config.ConfigData.Password, config.ConfigData.GameName))
			checkErr(pgsqlConnErr)
			queryDanmuByScene(sceneId, db, w)
			defer fmt.Printf("和数据库的连接已关闭")
			defer db.Close()
		}
	} else {
		fmt.Fprintf(w, "Route To getDanmu!")
	}
}

// 搜索及首次、手动下载弹幕
func searchDanmu(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// 获取并检查 UA 是否合法
		r.ParseForm()
		ua := r.Header["User-Agent"]
		if !checkUA(ua[0]) {
			fmt.Fprintf(w, "Invalid Devices!")
		} else {
			// 开启数据库连接
			db, pgsqlConnErr := sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
				config.ConfigData.Address, config.ConfigData.Port, config.ConfigData.Username, config.ConfigData.Password, config.ConfigData.GameName))
			checkErr(pgsqlConnErr)
			queryAllDanmu(db, w)
			defer fmt.Printf("和数据库的连接已关闭")
			defer db.Close()
		}
	} else {
		fmt.Fprintf(w, "Route To searchDanmu!")
	}
}

// 弹幕
type Danmu struct {
	TextScript string `json:"textScript"`
	TextDanmu  string `json:"textDanmu"`
}

// 查询返回结构
type QueryDanmu struct {
	SceneId int `json:"sceneId"`
	Danmu
}

var (
	DamnuInForResult []byte
	PartSceneResult  []byte
)

// 查询全部弹幕 结构体实现
func queryAllDanmu(db *sql.DB, w http.ResponseWriter) {
	fmt.Fprintf(w, "{\"data\":[")

	for i := 1; i < config.ConfigData.SceneNumber+1; i++ {

		QueryAllDanmuResult := new(QueryDanmu)
		QueryAllDanmuResult.SceneId = i

		allData, pgsqlConnErr := db.Query(fmt.Sprintf((`SELECT textScript,textDanmu FROM scene%d;`), i))
		checkErr(pgsqlConnErr)

		for allData.Next() {
			var (
				textScriptOneLine string
				textDanmuOneLine  string
				outputToClient    string
			)
			pgsqlConnErr = allData.Scan(&textScriptOneLine, &textDanmuOneLine)
			checkErr(pgsqlConnErr)
			QueryAllDanmuResult.TextScript = textScriptOneLine
			QueryAllDanmuResult.TextDanmu = textDanmuOneLine
			DamnuInForResult, _ = json.Marshal(QueryAllDanmuResult)
			fmt.Println(string(DamnuInForResult))
			outputToClient = string(DamnuInForResult)[1 : len(string(DamnuInForResult))-1]
			if i < config.ConfigData.SceneNumber && outputToClient != "" { //循环输出结果到客户端，注意结尾 json 构造
				fmt.Fprintf(w, "{%s},", outputToClient)
			} else {
				fmt.Fprintf(w, "{%s}", outputToClient)
			}
		}

	}
	fmt.Fprintf(w, "],\"status\": \"Success!\"}")
}

// 按照 SceneId 查找弹幕
func queryDanmuByScene(sceneId string, db *sql.DB, w http.ResponseWriter) {

	fmt.Fprintf(w, "{\"data\":[")
	QueryAllDanmuResult := new(QueryDanmu)
	allData, pgsqlConnErr := db.Query(fmt.Sprintf((`SELECT textScript,textDanmu FROM scene%s;`), sceneId))
	checkErr(pgsqlConnErr)
	var count int // count 计数决定 json 结尾构成
	err := db.QueryRow(fmt.Sprintf((`SELECT count(*) textScript FROM scene%s;`), sceneId)).Scan(&count)
	checkErr(err)
	i := 1
	for allData.Next() {
		var (
			textScriptOneLine string
			textDanmuOneLine  string
			outputToClient    string
		)
		pgsqlConnErr = allData.Scan(&textScriptOneLine, &textDanmuOneLine)
		checkErr(pgsqlConnErr)
		QueryAllDanmuResult.TextScript = textScriptOneLine
		QueryAllDanmuResult.TextDanmu = textDanmuOneLine
		DamnuInForResult, _ = json.Marshal(QueryAllDanmuResult)
		fmt.Println(string(DamnuInForResult))
		outputToClient = string(DamnuInForResult)[1 : len(string(DamnuInForResult))-1]
		if i < count && outputToClient != "" {
			fmt.Fprintf(w, "{%s},", outputToClient)
		} else {
			fmt.Fprintf(w, "{%s}", outputToClient) // json 最后一个不加逗号
		}
		i++
	}

	fmt.Fprintf(w, "],\"status\": \"Success!\"}")
}

// 发射弹幕
func fireDanmu(w http.ResponseWriter, r *http.Request) {
	// 发射弹幕使用 POST 请求
	if r.Method == "POST" {
		// 获取并检查 UA 是否合法
		r.ParseForm()
		ua := r.Header["User-Agent"]
		if !checkUA(ua[0]) {
			fmt.Fprintf(w, "{\"status\":\"Invalid Devices!\"}")
		} else if r.Form["textDanmu"][0] == "" { //禁止传空参数（弹幕）
			fmt.Fprintf(w, "{\"status\":\"Invalid Params!\"}")
		} else {
			// 检查文本是否含分隔符“¦”（日文编码，%C2%A6），如果有就替换成|（中英文编码，%7C，常用）
			checkFlag := strings.Contains(r.Form["textDanmu"][0], "¦")
			fmt.Println("checkFlag:", checkFlag)
			var textDanmuSafe string
			if checkFlag {
				textDanmuSafe = changeFlag(r.Form["textDanmu"][0])
			} else {
				textDanmuSafe = r.Form["textDanmu"][0]
			}

			fmt.Println(checkFlag)
			// 以下为调试信息
			fmt.Println("path:", r.URL.Path)
			fmt.Println("UA:", ua[0])
			fmt.Println("弹幕:", textDanmuSafe)
			fmt.Println("文本:", r.Form["textScript"][0])
			fmt.Println("SceneId:", r.Form["sceneId"][0])
			// SceneId 转 int
			sceneIdInt, _ := strconv.Atoi(r.Form["sceneId"][0])

			// 开启数据库连接
			db, pgsqlConnErr := sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
				config.ConfigData.Address, config.ConfigData.Port, config.ConfigData.Username, config.ConfigData.Password, config.ConfigData.GameName))
			if pgsqlConnErr == nil {
				fmt.Printf("数据库 %s 连接成功\n", config.ConfigData.GameName)
			}
			checkErr(pgsqlConnErr)

			// 根据数据存在情况确定合适的数据库写入策略
			if queryIfTextScriptExists(sceneIdInt, r.Form["textScript"][0], db) {
				fmt.Printf("数据存在\n")
				insertTextDanmuWhenTextScriptExists(sceneIdInt, r.Form["textScript"][0], textDanmuSafe, db)
				fmt.Fprintf(w, "{\"status\":\"Success!\"}")
			} else {
				fmt.Printf("数据不存在，将插入新数据\n")
				insertTextDanmuWhenNoneExists(sceneIdInt, r.Form["textScript"][0], textDanmuSafe, db)
				fmt.Fprintf(w, "{\"status\":\"Success!\"}")
			}
			defer db.Close()
			defer fmt.Printf("已关闭到 Scene%s 数据库的连接\n", r.Form["sceneId"][0])
		}
	} else { // 对于其它请求的响应
		fmt.Fprintf(w, "Route To fireDanmu!")
	}
}

// 查询数据是否存在方法
func queryIfTextScriptExists(sceneIdInt int, query string, db *sql.DB) (result bool) {
	querySql := fmt.Sprintf("SELECT textscript FROM scene%d WHERE textscript='%s';", sceneIdInt, query)
	fmt.Println("Query语句:", querySql)
	queryResult, queryErr := db.Query(querySql)
	checkErr(queryErr)
	if queryResult.Next() {
		result = true
	} else {
		result = false
	}
	return
}

// 数据存在时插入数据方法
func insertTextDanmuWhenTextScriptExists(sceneIdInt int, textScript string, textDanmu string, db *sql.DB) {
	// 查询已存在数据，并在后面追加新数据
	getCurrentDanmu := fmt.Sprintf(("SELECT textDanmu FROM scene%d WHERE textScript='%s';"),
		sceneIdInt, textScript)
	fmt.Println("get语句:", getCurrentDanmu)
	currentDanmu, queryErr := db.Query(getCurrentDanmu)
	checkErr(queryErr)
	currentDanmuText := queryCurrentDanmu(currentDanmu)
	fmt.Println("OutcurrentDanmuText:", currentDanmuText)
	// pgsql 不适合存储 Json 格式，使用¦作为文本分隔符，后续处理
	newDanmuText := currentDanmuText + "¦" + textDanmu
	writeSql := fmt.Sprintf(("UPDATE scene%d SET textDanmu = '%s' WHERE textScript='%s';"),
		sceneIdInt, newDanmuText, textScript)
	fmt.Println("Write语句:", writeSql)
	_, writeErr := db.Exec(writeSql)
	checkErr(writeErr)
}

// 查询数据方法
func queryCurrentDanmu(query *sql.Rows) (result string) {
	var CurrentDanmuText string
	for query.Next() {
		query.Scan(&CurrentDanmuText)
		fmt.Println("currentDanmu:", CurrentDanmuText)
	}
	return CurrentDanmuText
}

// 数据不存在时插入数据方法
func insertTextDanmuWhenNoneExists(sceneIdInt int, textScript string, textDanmu string, db *sql.DB) {
	// 调试：输出 SQL 语句
	// 拼接 SQL 语句
	writeSql := fmt.Sprintf(("INSERT INTO scene%d(textScript,textDanmu) VALUES('%s','%s');"),
		sceneIdInt, textScript, textDanmu)
	// sceneIdInt, r.Form["textScript"][0], r.Form["textDanmu"][0])
	fmt.Println("Write语句:", writeSql)
	_, writeErr := db.Exec(writeSql)
	checkErr(writeErr)
}

// 检查用户 UA 是否符合要求，防止用户使用浏览器胡乱提交弹幕
func checkUA(ua string) (result bool) {
	if ua == config.ConfigData.AllowUA {
		result = true
	} else {
		result = false
	}
	return result
}

// 分隔符转换
func changeFlag(str string) (safe string) {
	safe = strings.ReplaceAll(str, "¦", "|")
	return safe
}

// 首页及未定义页面
func helloindex(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Path)
	p := "./static/index.html"
	http.ServeFile(w, r, p)
}

func main() {
	config.Init()
	fmt.Printf(`____________________________
goDanmuku is going to serve at:%s
Powered By Luckykeeper(https://luckykeeper.site | mailto:luckykeeper@luckykeeper.site)
goDanmuku，专为 Ren'Py 打造的弹幕姬服务。代码免费开源，许可开源项目和免费游戏在MIT协议下使用
开源地址：https://github.com/luckykeeper/goDanmuku)
____________________________`+"\n",
		config.ConfigData.ServerPort)
	connectPgsqldb()
	http.HandleFunc("/", helloindex)                                  // 首页及未定义页面
	http.HandleFunc("/getdanmu", getDanmu)                            // 获取弹幕（分scene）
	http.HandleFunc("/searchdanmu", searchDanmu)                      // 搜索及首次、手动下载弹幕
	http.HandleFunc("/firedanmu", fireDanmu)                          // 发射弹幕
	err := http.ListenAndServe(":"+config.ConfigData.ServerPort, nil) // 监听端口(tcp)，在 config.json 设置
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// ————————————————————————————————————————————————————————————————————————————————————————————————————————————————
// // 查询全部弹幕 map 实现 【已废弃】
// func queryAllDanmu(db *sql.DB) {
// 	result := map[string]map[string]string{}
// 	for i := 1; i < config.ConfigData.SceneNumber+1; i++ {
// 		queryDanmuBySceneId := map[string]string{}

// 		allData, pgsqlConnErr := db.Query(fmt.Sprintf((`SELECT textScript,textDanmu FROM scene%d;`), i))
// 		checkErr(pgsqlConnErr)
// 		for allData.Next() {
// 			var textScriptOneLine string
// 			var textDanmuOneLine string
// 			pgsqlConnErr = allData.Scan(&textScriptOneLine, &textDanmuOneLine)
// 			checkErr(pgsqlConnErr)
// 			queryDanmuBySceneId[textScriptOneLine] = textDanmuOneLine
// 		}
// 		result[strconv.Itoa(i)] = queryDanmuBySceneId
// 	}
// 	fmt.Println(result)
// }
