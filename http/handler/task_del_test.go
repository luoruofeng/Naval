package handler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestDelete(t *testing.T) {
	fmt.Println("测试删除任务接口")
	id := "task-2"
	req, err := http.NewRequest(http.MethodDelete, "http://127.0.0.1:8080/task/"+id, strings.NewReader(""))
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

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败", err)
		return
	}
	fmt.Println(string(respData))
}
