package srv

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
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
	Id        string    `yaml:"id"`         //任务id
	CreatedAt time.Time `yaml:"created_at"` //创建时间
	Available bool      `yaml:"available"`  //是否可用
	Items     []Item    `yaml:"items"`      //任务项
}

type TaskResult struct {
	Id        string    `json:"id"`         //任务id
	CreatedAt time.Time `json:"created_at"` //创建时间
	Error     string    `json:"error"`      //错误信息
	Message   string    `json:"message"`    //消息
	Statecode int       `json:"statecode"`  //状态码
}

func NewTaskResult(id string, err string, msg string, statecode int) TaskResult {
	return TaskResult{
		Id:        id,
		CreatedAt: time.Now(),
		Message:   msg,
		Statecode: statecode,
		Error:     err,
	}
}

type TaskSrv struct {
	logger      *zap.Logger
	taskResults chan TaskResult
	tasks       chan Task
	ctx         context.Context
}

func NewTask(lc fx.Lifecycle, logger *zap.Logger, ctx context.Context) TaskSrv {
	taskResults := make(chan TaskResult, 100)
	tasks := make(chan Task, 100)
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logger.Info("启动task服务")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("销毁task服务")
			return nil
		},
	})

	return TaskSrv{logger: logger, taskResults: taskResults, tasks: tasks, ctx: ctx}
}

func (t TaskSrv) Unmarshal(c []byte) (*Task, error) {
	var task Task
	err := yaml.Unmarshal(c, &task)
	if err != nil {
		t.logger.Error(fmt.Sprintf("Could not parse YAML: %v", err), zap.Any("input", c))
		return nil, err
	}
	return &task, nil
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

func (ts TaskSrv) Handle(task Task) {
	ts.tasks <- task
}

func (ts TaskSrv) InitWorkerpools() {
	ts.logger.Info("初始化workerpools")
	ticker := time.NewTicker(time.Duration(time.Hour * 3))
	for {
		select {
		case <-ts.ctx.Done():
			close(ts.tasks)
			ts.logger.Info("关闭任务通道")
			close(ts.taskResults)
			ts.logger.Info("关闭任务结果通道")
			return
		case _, ok := <-ts.tasks:
			if !ok {
				return
			}
			go func() {}()
		case _, ok := <-ts.taskResults:
			if !ok {
				return
			}
			go func() {}()
		case <-ticker.C:
			ts.logger.Info("The channel is being monitored")
		}
	}
}
