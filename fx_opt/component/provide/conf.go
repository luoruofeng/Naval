package provide

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luoruofeng/Naval/conf"
	"go.uber.org/fx"
)

func getPath(configFile string) string {
	//default value
	if configFile == "" {
		absPath, err := filepath.Abs(os.Args[0])
		if err != nil {
			panic(err)
		}
		// 获取执行程序所在的目录
		dir := filepath.Dir(absPath)
		// 拼接文件路径
		configFile = filepath.Join(dir, "./conf/conf.json")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			panic(fmt.Sprintf("文件 %s 不存在\n", configFile))
		}
	}
	return configFile
}

func printInfo(configFile string) {
	configFile = getPath(configFile)
	cbs, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	fmt.Println("配置文件", configFile)
	fmt.Println("读取配置文件内容\n", string(cbs))
}

func NewConfig(lc fx.Lifecycle, configPath string) *conf.Config {
	var instance *conf.Config
	configPath = getPath(configPath)
	cbs, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(cbs, &instance)
	if err != nil {
		panic(err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			printInfo(configPath)
			return nil
		},
	})

	return instance
}
