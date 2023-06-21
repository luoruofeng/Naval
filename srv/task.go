package srv

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	mongo "github.com/luoruofeng/Naval/component/mongo/logic"
	m "github.com/luoruofeng/Naval/model"
)

type TaskSrv struct {
	logger             *zap.Logger              // 日志
	taskResultChan     chan m.TaskResult        // 任务结果通道
	createTaskChan     chan m.Task              // 任务创建通道
	execTaskChan       chan m.Task              // 任务执行通道 用于将任务放入pendingTasks后的通知
	ctx                context.Context          // 任务调度上下文
	pendingTasks       []m.Task                 //待执行的任务slice
	mongoT             mongo.TaskMongoSrv       // mongo任务服务
	mongoTR            mongo.TaskResultMongoSrv // mongo任务结果服务
	lastExecTimeSecond int                      // 等待多少秒后开始执行任务
	lock               sync.Mutex               //用于pendingTasks的锁
}

func NewTaskSrv(lc fx.Lifecycle, logger *zap.Logger, ctx context.Context, taskMongoSrv mongo.TaskMongoSrv, taskResultMongoSrv mongo.TaskResultMongoSrv) *TaskSrv {
	logger.Info("初始化task服务")
	logger.Info("初始化task结果通道")
	taskResults := make(chan m.TaskResult)
	logger.Info("初始化task创建通道")
	taskCreatedChan := make(chan m.Task)
	logger.Info("初始化task执行通道")
	execTaskChan := make(chan m.Task)
	logger.Info("初始化待执行的任务列表")
	// TODO 初始化mongo任务数据到pendingTasks
	pendingTasks := make([]m.Task, 0)

	result := TaskSrv{logger: logger, taskResultChan: taskResults, createTaskChan: taskCreatedChan, execTaskChan: execTaskChan, mongoT: taskMongoSrv, mongoTR: taskResultMongoSrv, ctx: ctx, pendingTasks: pendingTasks, lastExecTimeSecond: 1}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logger.Info("启动taskSrv服务", zap.Any("pointer", fmt.Sprintf("%p %p\n", &result, &result.pendingTasks)))
			go result.InitExecTaskScheduler()
			go result.InitEventScheduler()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("销毁taskSrv服务")
			return nil
		},
	})

	return &result
}

// Calculates the latest execution time of the pending task
func (ts *TaskSrv) CalcLatestExecTime() {
	log := ts.logger
	ts.lock.Lock()
	ts.lastExecTimeSecond = 0
	now := time.Now()
	lastPlanExecTime := now
	for i, task := range ts.pendingTasks {
		if task.Available {
			if task.PlanExecAt.Before(now) ||
				task.PlanExecAt.Equal(now) { // 计划执行时间小于等于当前时间
				// 从pendingTasks移除任务
				ts.pendingTasks = append(ts.pendingTasks[:i], ts.pendingTasks[i+1:]...)
				// 执行任务
				go ts.ExecTask(task)
			} else {
				if lastPlanExecTime.Equal(now) {
					// 第一次计算
					lastPlanExecTime = task.PlanExecAt
					ts.lastExecTimeSecond = int(task.PlanExecAt.Sub(now).Seconds())
				} else {
					if task.PlanExecAt.Before(lastPlanExecTime) {
						// 更新最近的执行时间
						lastPlanExecTime = task.PlanExecAt
						ts.lastExecTimeSecond = int(task.PlanExecAt.Sub(now).Seconds())
					}
				}
			}
		} else {
			log.Info("任务不可用", zap.Any("task", task))
			// 从pendingTasks移除任务
			ts.pendingTasks = append(ts.pendingTasks[:i], ts.pendingTasks[i+1:]...)
		}
	}
	if ts.lastExecTimeSecond == 0 {
		ts.lastExecTimeSecond = 1
	}
	log.Info("计算-最近的执行时间 ", zap.Int("lastExecTimeSecond", ts.lastExecTimeSecond), zap.Any("lastPlanExecTime", lastPlanExecTime), zap.Any("pending_tasks", ts.pendingTasks))
	ts.lock.Unlock()
}

func (t *TaskSrv) Unmarshal(c []byte) (*m.Task, error) {
	var task m.Task
	err := yaml.Unmarshal(c, &task)
	if err != nil {
		t.logger.Error(fmt.Sprintf("Could not parse YAML: %v", err), zap.Any("input", c))
		return nil, err
	}
	return &task, nil
}

func (ts *TaskSrv) ExecTask(task m.Task) {
	log := ts.logger
	// 更新任务状态为正在执行
	task.StateCode = m.Running
	// 更新任务执行时间
	task.ExtTime = time.Now()
	// 更新任务执行次数
	task.ExtTimes++
	// 更新mongo任务
	if r, err := ts.mongoT.Update(task); err != nil {
		log.Error("执行任务前更新任务失败", zap.Any("task", task), zap.Error(err))
		return
	} else {
		log.Info("执行任务前更新任务成功", zap.Any("task", task), zap.Any("update_result", r))
	}
	log.Info("任务开始执行", zap.Any("task", task))
	// TODO: 执行任务
}

// Handle 处理Http请求来的任务
func (ts *TaskSrv) Handle(task m.Task) {
	log := ts.logger
	// 设置任务执行时间
	task.PlanExecAt = time.Now().Add(time.Duration(task.WaitSeconds) * time.Second)
	// 设置可用状态
	task.Available = true
	// 任务执行次数
	task.ExtTimes = 0
	// 设置任务执行状态码
	task.StateCode = m.Pending
	// 设置任务创建时间
	task.CreatedAt = time.Now()
	// mongo保存任务
	log.Info("保存任务到mongoDB", zap.Any("task", task))
	if r, err := ts.mongoT.Save(task); err != nil {
		log.Error("保存任务失败", zap.Any("task", task), zap.Error(err))
		return
	} else {
		mongoId, ok := r.InsertedID.(primitive.ObjectID)
		if !ok {
			log.Info("保存任务成功 mongo Id转换失败", zap.Any("task", task), zap.Any("mongo_id", r.InsertedID))
		} else {
			task.MongoId = mongoId
			log.Info("保存任务成功", zap.Any("task", task), zap.Any("mongo_id", r.InsertedID))
		}
	}
	ts.createTaskChan <- task
}

func (ts *TaskSrv) InitExecTaskScheduler() {
	log := ts.logger
	log.Info("初始化执行任务调度器 ExecTaskScheduler")
	defer log.Info("退出 InitExecTaskScheduler")
	timer := time.NewTimer(time.Duration(ts.lastExecTimeSecond) * time.Second)
	for {
		select {
		case <-ts.ctx.Done():
			log.Info("关闭执行任务调度器 ExecTaskScheduler")
			close(ts.execTaskChan)
			ts.logger.Info("关闭任务执行通道 execTaskChan")
			return
		case <-timer.C:
			log.Info("开始任务调度")
			ts.CalcLatestExecTime()
			timer.Reset(time.Duration(ts.lastExecTimeSecond) * time.Second)
		case t, ok := <-ts.execTaskChan:
			if !ok {
				log.Info("接收到任务添加通知 但是execTaskChan通道已经被关闭", zap.Any("task", t))
				return
			}
			log.Info("开始任务调度-接收到任务添加通知", zap.Any("task", t))
			ts.CalcLatestExecTime()
			timer.Reset(time.Duration(ts.lastExecTimeSecond) * time.Second)
		}
	}
}

func (ts *TaskSrv) InitEventScheduler() {
	ts.logger.Info("初始化事件调度器 InitEventScheduler")
	defer ts.logger.Info("退出 InitEventScheduler")
	// 监听任务通道
	for {
		select {
		case <-ts.ctx.Done():
			close(ts.createTaskChan)
			ts.logger.Info("关闭任务创建通道 createTaskChan")
			close(ts.taskResultChan)
			ts.logger.Info("关闭任务结果通道 taskResults")
			return
		case t, ok := <-ts.createTaskChan:
			if !ok {
				ts.logger.Info("任务创建通道已关闭! createTaskChan", zap.Any("task", t))
				return
			} else {
				ts.logger.Info("任务创建通道接受到任务 createTaskChan", zap.Any("task", t))
				if t.StateCode == m.Pending { // 等待执行
					if t.Available { // 可用
						// 任务放入待执行列表
						if t.PlanExecAt.Before(time.Now()) ||
							t.PlanExecAt.Equal(time.Now()) { // 计划执行时间小于等于当前时间
							ts.logger.Info("任务立即执行-任务计划执行时间小于等于当前时间 任务进入待执行列表", zap.Any("task", t))
						} else {
							ts.logger.Info("任务稍后执行-任务计划执行时间大于当前时间", zap.Any("task", t))
						}
						ts.lock.Lock()
						ts.pendingTasks = append(ts.pendingTasks, t)
						ts.logger.Info("待执行列表添加任务", zap.Any("pending_tasks", ts.pendingTasks))
						ts.lock.Unlock()
						ts.execTaskChan <- t // 任务放入执行通道
					} else {
						ts.logger.Info("任务不可用", zap.Any("task", t))
					}
				} else {
					ts.logger.Info("任务状态不是等待执行", zap.Any("task", t))
				}
			}
		case r, ok := <-ts.taskResultChan:
			if !ok {
				ts.logger.Info("任务结果通道已关闭!", zap.Any("task_result", r))
				return
			}
			ts.logger.Info("接收到任务结果", zap.Any("task_result", r))
			go func() {}()
		}
	}
}
