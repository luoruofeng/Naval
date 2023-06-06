package handler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func getRootProjPath(current string) string {
	path := filepath.Join(current, "go.mod")
	_, err := os.Stat(path)
	if err == nil {
		return current
	} else {
		if os.IsNotExist(err) {
			return getRootProjPath(filepath.Dir(current))
		} else {
			panic(err)
		}
	}
}

func TestHandler(t *testing.T) {
	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("获取当前工作目录失败", err)
	} else {
		fmt.Println("current working directory:", cwd)
	}

	// 获取项目根目录
	rootDir := getRootProjPath(cwd)
	data, err := ioutil.ReadFile(filepath.Join(rootDir, "example/task1.yaml"))

	if err != nil {
		fmt.Println("获取项目根目录失败", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:8080/task", bytes.NewBuffer(data))
	if err != nil {
		fmt.Println("创建请求失败", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-yaml")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("发送消息失败", err)
		return
	}
	defer resp.Body.Close()

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败", err)
		return
	}
	fmt.Println(string(respData))
}
