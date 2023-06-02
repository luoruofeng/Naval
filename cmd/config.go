package cmd

import "flag"

//go run  . -cnf="./conf.json"
func GetConfigFilePath() string {
	configPath := flag.String("cnf", "", "Config file path,The default value is in the current folder.")
	flag.Parse()
	return *configPath
}
