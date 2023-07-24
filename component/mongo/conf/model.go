package conf

import (
	_ "embed"
	"encoding/json"
	"os"
	"strings"

	"go.uber.org/zap"
)

//go:embed config.json
var configStr []byte

func GetConfig(log *zap.Logger, configPath string) *Config {

	var c *Config
	if configPath != "" && strings.HasSuffix(configPath, "json") {
		configStr, _ = os.ReadFile(configPath)
		log.Info("mongoDB配置文件路径", zap.String("configPath", configPath), zap.String("configStr", string(configStr)))
	} else {
		log.Info("mongoDB配置文件路径", zap.String("configPath", "默认使用编译前的配置文件"), zap.String("configStr", string(configStr)))
	}
	json.Unmarshal(configStr, &c)
	return c
}

type Config struct {
	Addr                   string `json:"addr"`
	DbName                 string `json:"db_name"`
	Username               string `json:"username"`
	Password               string `json:"password"`
	ConnectTimeout         int    `json:"connect_timeout"`
	SocketTimeout          int    `json:"socket_timeout"`
	ServerSelectionTimeout int    `json:"server_selection_timeout"`
}
