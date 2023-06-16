package conf

import (
	_ "embed"
	"encoding/json"
)

//go:embed config.json
var configStr []byte

func GetConfig() *Config {
	var c *Config
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
