package srv

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	m "github.com/luoruofeng/Naval/model"
)

type TaskSrv struct {
	logger      *zap.Logger
	taskResults chan m.TaskResult
	tasks       chan m.Task
	ctx         context.Context
}

func NewTask(lc fx.Lifecycle, logger *zap.Logger, ctx context.Context) TaskSrv {
	taskResults := make(chan m.TaskResult, 100)
	tasks := make(chan m.Task, 100)
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

func (t TaskSrv) Unmarshal(c []byte) (*m.Task, error) {
	var task m.Task
	err := yaml.Unmarshal(c, &task)
	if err != nil {
		t.logger.Error(fmt.Sprintf("Could not parse YAML: %v", err), zap.Any("input", c))
		return nil, err
	}
	return &task, nil
}

func (ts TaskSrv) Handle(task m.Task) {
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
		case t, ok := <-ts.tasks:
			if !ok {
				ts.logger.Info("任务通道已关闭!", zap.Any("task", t))
				return
			}
			ts.logger.Info("接收到任务", zap.Any("task", t))
			go func() {}()
		case r, ok := <-ts.taskResults:
			if !ok {
				ts.logger.Info("任务结果通道已关闭!", zap.Any("task_result", r))
				return
			}
			ts.logger.Info("接收到任务结果", zap.Any("task_result", r))
			go func() {}()
		case <-ticker.C:
			ts.logger.Info("The channel is being monitored")
		}
	}
}
