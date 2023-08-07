package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestExec(t *testing.T) {
	fmt.Println("测试执行任务接口")
	id := "task-2"
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:8080/task/exec/"+id, strings.NewReader(""))
	if err != nil {
		fmt.Println("创建请求失败", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("发送消息失败", err)
		return
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败", err)
		return
	}
	fmt.Println(string(respData))
}
