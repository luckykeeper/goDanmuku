// 从json文件读取设置
package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Config struct {
	// 弹幕姬服务端口
	ServerPort string `json:"ServerPort"`
	//Pgsql账号
	Username string `json:"username"`
	//Pgsql密码
	Password string `json:"password"`
	//Pgsql地址
	Address string `json:"address"`
	//Pgsql端口
	Port int `json:"port"`
	// 游戏ID ，请填写游戏名称
	GameName string `json:"gameName"`
	// Scene 数量
	SceneNumber int `json:"sceneNumber"`
	// 允许的 UA
	AllowUA string `json:"allowUA"`
}

//配置文件路径
var ConfigPath string = "config/config.json"

// pgsql配置信息缓存
var ConfigData *Config

func Init() {
	//初始化配置对象
	ConfigData = new(Config)

	//读取配置文件
	file, err := os.Open(ConfigPath)
	if err != nil {
		fmt.Println("config path:", err)
		os.Exit(1)
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("config file:", err)
		os.Exit(1)
	}
	//使用json转换至config对象中
	err = json.Unmarshal(bytes, ConfigData)
	if err != nil {
		fmt.Println("json unmarshal:", err)
		os.Exit(1)
	}
}
