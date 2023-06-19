package model

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Item struct {
	IsRunning      bool      `yaml:"is_running"`                 //是否需要运行
	NeedDelete     bool      `yaml:"need_delete"`                //是否需要删除
	ComposeContent string    `yaml:"compose_content,omitempty"`  //docker-compose文件内容
	K8SYamlContent string    `yaml:"k8s_yaml_content,omitempty"` //k8s yaml文件内容
	ExtTime        time.Time `yaml:"ExtTime,omitempty"`          //扩展字段，用于记录任务执行时间
	ExtDoneTime    time.Time `yaml:"ExtDoneTime,omitempty"`      //扩展字段，用于记录任务执行完成时间
	ExtTimes       int       `yaml:"ExtTimes,omitempty"`         //扩展字段，用于记录任务执行次数
}

type SC int //任务状态码

const (
	Unknown SC = iota
	Pending    //等待执行
	Running    //正在执行
	Stopped    //已停止
)

type Task struct {
	Uuid        string             `yaml:"uuid,omitempty"`                          //系统生成的每个HTTP请求的uuid
	Id          string             `yaml:"id"`                                      //客户传过来的任务id
	MongoId     primitive.ObjectID `yaml:"mongo_id,omitempty" bson:"_id,omitempty"` //mongo id
	CreatedAt   time.Time          `yaml:"created_at"`                              //创建时间
	Available   bool               `yaml:"available"`                               //是否可用
	Items       []Item             `yaml:"items"`
	Sechedule   string             `yaml:"sechedule,omitempty"`    //定时任务表达式
	WaitSeconds int                `yaml:"wait_seconds,omitempty"` //等待执行时间               //任务项
	PlanExecAt  time.Time          `yaml:"plan_exec_at,omitempty"` //计划执行时间
	ExtTime     time.Time          `yaml:"ExtTime,omitempty"`      //扩展字段，用于记录任务执行时间
	ExtDoneTime time.Time          `yaml:"ExtDoneTime,omitempty"`  //扩展字段，用于记录任务执行完成时间
	ExtTimes    int                `yaml:"ExtTimes,omitempty"`     //扩展字段，用于记录任务执行次数
	StateCode   SC                 `yaml:"statecode,omitempty"`    //扩展字段，用于记录任务执行状态码
}

type TaskResult struct {
	Uuid        string    `json:"uuid,omitempty"`        //系统生成的每个HTTP请求的uuid
	Id          string    `json:"id"`                    //任务id
	CreatedAt   time.Time `json:"created_at"`            //创建时间
	Error       string    `json:"error"`                 //错误信息
	Message     string    `json:"message"`               //消息
	Statecode   int       `json:"statecode"`             //状态码
	ExtTime     time.Time `yaml:"ExtTime,omitempty"`     //扩展字段，用于记录任务执行时间
	ExtDoneTime time.Time `yaml:"ExtDoneTime,omitempty"` //扩展字段，用于记录任务执行完成时间
	ExtTimes    int       `yaml:"ExtTimes,omitempty"`    //扩展字段，用于记录任务执行次数

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
