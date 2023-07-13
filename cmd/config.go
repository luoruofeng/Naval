package cmd

import "flag"

//go run  . -cnf="./conf.json"
func GetConfigFilePath() map[string]string {
	r := make(map[string]string, 0)
	configPath := flag.String("cnf", "", "Config file path,The default value is in the current folder.")
	mongoConfigPath := flag.String("mongo-cnf", "", "mongoDB配置文件路径json格式")
	flag.Parse()

	r["cnf"] = *configPath
	r["mongo-cnf"] = *mongoConfigPath

	return r
}
