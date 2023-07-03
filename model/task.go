package model

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Item struct {
	IsRunning      bool      `yaml:"is_running" bson:""`                 //是否需要运行
	NeedDelete     bool      `yaml:"need_delete" bson:""`                //是否需要删除
	ComposeContent string    `yaml:"compose_content,omitempty" bson:""`  //docker-compose文件内容
	K8SYamlContent string    `yaml:"k8s_yaml_content,omitempty" bson:""` //k8s yaml文件内容
	ExtTime        time.Time `yaml:"ExtTime,omitempty" bson:""`          //扩展字段，用于记录任务执行时间
	ExtDoneTime    time.Time `yaml:"ExtDoneTime,omitempty" bson:""`      //扩展字段，用于记录任务执行完成时间
	ExtTimes       int       `yaml:"ExtTimes,omitempty" bson:""`         //扩展字段，用于记录任务执行次数
}

type SC int //任务状态码

const (
	Unknown SC = iota
	Pending    //等待执行
	Running    //正在执行
	Stopped    //已停止
)

type RSC int //任务执行结果状态码

const (
	ResultSuccess RSC = iota //执行成功
	ResultFail               //执行失败
)

type Task struct {
	Id               string             `yaml:"id" bson:"Id"`                    //客户传过来的任务id
	Available        bool               `yaml:"available" bson:"Available"`      //是否可用
	WaitSeconds      int                `yaml:"wait_seconds" bson:"WaitSeconds"` //等待执行时间
	Items            []Item             `yaml:"items" bson:"Items"`
	Uuid             string             `yaml:"uuid,omitempty" bson:"Uuid"`                           //系统生成的每个HTTP请求的uuid
	CreatedAt        time.Time          `yaml:"created_at,omitempty" bson:"CreatedAt"`                //创建时间
	IsRunning        bool               `yaml:"is_running,omitempty" bson:"IsRunning"`                //是否正在执行
	UpdateAt         time.Time          `yaml:"update_at,omitempty" bson:"UpdateAt,omitempty"`        //修改时间
	DeleteAt         time.Time          `yaml:"delete_at,omitempty" bson:"DeleteAt,omitempty"`        //修改时间
	Sechedule        string             `yaml:"sechedule,omitempty" bson:"Sechedule"`                 //定时任务表达式,暂时没有开发此功能
	MongoId          primitive.ObjectID `yaml:"mongo_id,omitempty" bson:"_id,omitempty"`              //mongo id
	PlanExecAt       time.Time          `yaml:"plan_exec_at,omitempty" bson:"PlanExecAt,omitempty"`   //计划执行时间
	ExtTime          time.Time          `yaml:"ext_time,omitempty" bson:"ExtTime,omitempty"`          //扩展字段，用于记录任务执行时间
	ExtDoneTime      time.Time          `yaml:"ext_done_time,omitempty" bson:"ExtDoneTime,omitempty"` //扩展字段，用于记录任务执行完成时间
	ExtTimes         int                `yaml:"ext_times,omitempty" bson:"ExtTimes,omitempty"`        //扩展字段，用于记录任务执行次数
	StateCode        SC                 `yaml:"statecode,omitempty" bson:"StateCode"`                 //扩展字段，用于记录任务执行状态码
	ExecResultIds    []string           `yaml:"exec_result_ids,omitempty" bson:"ExecResultIds,omitempty"`
	ExecSuccessfully bool               `yaml:"exec_successfully,omitempty" bson:"ExecSuccessfully,omitempty"` //执行任务是否成功的总体结果
}

type TaskResult struct {
	Id        string             `json:"id" bson:"Id"`                            //任务结果id
	MongoId   primitive.ObjectID `yaml:"mongo_id,omitempty" bson:"_id,omitempty"` //mongo id
	TaskId    string             `json:"task_id" bson:"TaskId"`                   //任务id
	CreatedAt time.Time          `json:"created_at" bson:"CreatedAt"`             //创建时间
	Error     string             `json:"error,omitempty" bson:"Error"`            //错误信息
	Message   string             `json:"message,omitempty" bson:"Message"`        //消息
	StateCode RSC                `json:"statecode,omitempty" bson:"StateCode"`    //执行结果状态码
}

func NewTaskResult(taskId string, itemIndex int, err string, msg string, statecode RSC) TaskResult {
	now := time.Now()

	taskResultId := fmt.Sprintf("%v-%d-%v", taskId, itemIndex, now.Format("2006-01-02-15-04-05"))
	return TaskResult{
		Id:        taskResultId,
		TaskId:    taskId,
		CreatedAt: now,
		StateCode: statecode,
		Message:   msg,
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
