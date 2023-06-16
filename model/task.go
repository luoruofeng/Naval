package model

import (
	"fmt"
	"time"
)

type Item struct {
	IsRunning      bool   `yaml:"is_running"`                 //是否需要运行
	NeedDelete     bool   `yaml:"need_delete"`                //是否需要删除
	ComposeContent string `yaml:"compose_content,omitempty"`  //docker-compose文件内容
	K8SYamlContent string `yaml:"k8s_yaml_content,omitempty"` //k8s yaml文件内容
	Sechedule      string `yaml:"sechedule,omitempty"`        //定时任务表达式
	WaitSeconds    int    `yaml:"wait_seconds,omitempty"`     //等待执行时间
}

type Task struct {
	Uuid      string    `yaml:"uuid,omitempty"` //系统生成的每个HTTP请求的uuid
	Id        string    `yaml:"id"`             //任务id
	CreatedAt time.Time `yaml:"created_at"`     //创建时间
	Available bool      `yaml:"available"`      //是否可用
	Items     []Item    `yaml:"items"`          //任务项
}

type TaskResult struct {
	Uuid      string    `json:"uuid,omitempty"` //系统生成的每个HTTP请求的uuid
	Id        string    `json:"id"`             //任务id
	CreatedAt time.Time `json:"created_at"`     //创建时间
	Error     string    `json:"error"`          //错误信息
	Message   string    `json:"message"`        //消息
	Statecode int       `json:"statecode"`      //状态码
}

func NewTaskResult(uuid string, id string, err string, msg string, statecode int) TaskResult {
	return TaskResult{
		Uuid:      uuid,
		Id:        id,
		CreatedAt: time.Now(),
		Message:   msg,
		Statecode: statecode,
		Error:     err,
	}
}

func (task *Task) Verify() error {
	if task.Id == "" {
		return fmt.Errorf("任务id不能为空")
	}
	if len(task.Items) == 0 {
		return fmt.Errorf("任务项不能为空")
	}
	if !task.Available {
		return fmt.Errorf("任务不可用")
	}
	for _, item := range task.Items {
		if item.IsRunning {
			if item.ComposeContent == "" && item.K8SYamlContent == "" {
				return fmt.Errorf("任务项ComposeContent和K8SYamlContent不能同时为空")
			}
		}
	}
	return nil
}
