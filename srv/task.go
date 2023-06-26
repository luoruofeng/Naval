package srv

import (
	"context"
	"errors"
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
	deleteTaskChan     chan string              // 任务删除通道
	updateTaskChan     chan m.Task              //任务修改通道
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
	deleteTaskChan := make(chan string)
	logger.Info("初始化task删除通道")
	updateTaskChan := make(chan m.Task)
	logger.Info("初始化task更新通道")
	execTaskChan := make(chan m.Task)
	logger.Info("初始化待执行的任务列表")
	// TODO 初始化mongo任务数据到pendingTasks
	pendingTasks := make([]m.Task, 0)

	result := TaskSrv{
		logger:             logger,
		taskResultChan:     taskResults,
		createTaskChan:     taskCreatedChan,
		execTaskChan:       execTaskChan,
		mongoT:             taskMongoSrv,
		mongoTR:            taskResultMongoSrv,
		ctx:                ctx,
		pendingTasks:       pendingTasks,
		lastExecTimeSecond: 1,
		deleteTaskChan:     deleteTaskChan,
		updateTaskChan:     updateTaskChan,
	}

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

func (ts *TaskSrv) walkPendingTasks(p func(i int, t *m.Task) (bool, error)) error {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	for i, task := range ts.pendingTasks {
		b, err := p(i, &task)
		if err != nil {
			return err
		}
		if b {
			return nil
		}
	}
	return nil
}

func (ts *TaskSrv) addPendingTask(t m.Task) error {
	err := ts.walkPendingTasks(func(i int, task *m.Task) (bool, error) {
		if task.Id == t.Id {
			return true, errors.New("打算新增到pendingtasks中的任务id已经存在")
		}
		return false, nil
	})

	if err != nil {
		return err
	}

	ts.lock.Lock()
	ts.pendingTasks = append(ts.pendingTasks, t)
	ts.lock.Unlock()
	return nil
}

func (ts *TaskSrv) deletePendingTask(id string) error {
	return ts.walkPendingTasks(func(i int, t *m.Task) (bool, error) {
		if t.Id == id {
			ts.pendingTasks = append(ts.pendingTasks[:i], ts.pendingTasks[i+1:]...)
			return true, nil
		}
		return false, nil
	})
}

func (ts *TaskSrv) updatePendingTask(task m.Task) error {
	return ts.walkPendingTasks(func(i int, t *m.Task) (bool, error) {
		if t.Id == task.Id {
			ts.pendingTasks[i] = task
			return true, nil
		}
		return false, nil
	})
}

// Calculates the latest execution time of the pending task
func (ts *TaskSrv) CalcLatestExecTime() {
	log := ts.logger
	now := time.Now()
	lastPlanExecTime := now
	ts.lastExecTimeSecond = 0
	ts.walkPendingTasks(func(i int, task *m.Task) (bool, error) {
		if task.Available {
			if task.PlanExecAt.Before(now) ||
				task.PlanExecAt.Equal(now) { // 计划执行时间小于等于当前时间
				// 从pendingTasks移除任务
				ts.pendingTasks = append(ts.pendingTasks[:i], ts.pendingTasks[i+1:]...)
				// 执行任务
				go ts.ExecTask(*task)
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
		return false, nil
	})
	if ts.lastExecTimeSecond == 0 {
		ts.lastExecTimeSecond = 1
	}
	log.Info("计算-最近的执行时间 ", zap.Int("lastExecTimeSecond", ts.lastExecTimeSecond), zap.Any("lastPlanExecTime", lastPlanExecTime), zap.Any("pending_tasks", ts.pendingTasks))
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

func (ts *TaskSrv) Delete(id string) error {
	ts.logger.Info("删除任务", zap.Any("id", id))
	if t, err := ts.mongoT.FindById(id); err != nil {
		ts.logger.Error("删除任务失败-查询任务失败", zap.Any("id", id), zap.Error(err))
		return err
	} else {
		ts.logger.Info("删除任务-开始删除mongo中的任务", zap.Any("id", id), zap.Any("task", t))
		if t.StateCode == m.Pending ||
			t.StateCode == m.Stopped ||
			t.StateCode == m.Unknown {
			// mongo删除任务
			if r, err := ts.mongoT.Delete(t.MongoId); err != nil {
				ts.logger.Error("删除任务失败-删除mongo中的任务失败", zap.Any("task", t), zap.Error(err))
				return err
			} else {
				ts.logger.Info("删除任务-删除mongo中的任务成功", zap.Any("task", t), zap.Any("delete_result", r))
				// 任务从待执行列表移除
				ts.deletePendingTask(id)
				ts.logger.Info("删除任务-任务从pend_tasks中成功删除", zap.Any("task", t))
				// 任务从执行通道移除后重新计算最近的执行时间
				if t.StateCode == m.Pending {
					ts.deleteTaskChan <- id // 任务放入删除通道
				}
				return nil
			}
		} else if t.StateCode == m.Running { // 运行中不能删除
			ts.logger.Info("任务正在运行中不能删除", zap.Any("task", t))
			return fmt.Errorf("任务正在运行中不能删除 task:%v", t)
		}
		return nil
	}
}

func (ts *TaskSrv) ExecTask(task m.Task) {
	log := ts.logger
	// 更新任务状态为正在执行
	task.StateCode = m.Running
	// 更新任务执行时间
	task.ExtTime = time.Now()
	// 更新任务执行次数
	task.ExtTimes++
	task.IsRunning = true // 设置任务正在运行
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

func (ts *TaskSrv) Update(task m.Task) error {
	log := ts.logger

	log.Info("更新任务到mongoDB", zap.Any("task", task))
	// mongo更新任务前先查询任务确保任务存在并且确定任务状态不为运行中
	if t, err := ts.mongoT.FindById(task.Id); err != nil {
		log.Error("更新任务失败-查询任务失败", zap.Any("task", task), zap.Error(err))
		return err
	} else if !t.Available {
		log.Info("更新任务失败-任务不可用", zap.Any("task", task))
		return errors.New("更新任务失败-任务不可用")
	} else if t.StateCode == m.Running {
		log.Info("更新任务失败-任务正在运行中不能更新", zap.Any("task", task))
		return errors.New("更新任务失败-任务正在运行中不能更新")
	} else {
		task.UpdateAt = time.Now()
		task.StateCode = t.StateCode
		task.IsRunning = t.IsRunning
		task.CreatedAt = t.CreatedAt
		task.Available = t.Available
		task.MongoId = t.MongoId
		task.PlanExecAt = time.Now().Add(time.Duration(task.WaitSeconds) * time.Second)
		if r, err := ts.mongoT.Update(task); err != nil {
			log.Error("更新任务失败", zap.Any("task", task), zap.Error(err))
			return err
		} else {
			log.Info("更新任务成功", zap.Any("task", task), zap.Any("result", r))
		}
		ts.updateTaskChan <- task
		return nil

	}

}

func (ts *TaskSrv) Add(task m.Task) error {
	log := ts.logger
	task.IsRunning = false // 设置任务不在运行
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
		return err
	} else {
		mongoId, ok := r.InsertedID.(primitive.ObjectID)
		if !ok {
			log.Info("保存任务成功 mongo Id转换失败", zap.Any("task", task), zap.Any("mongo_id", r.InsertedID))
			return errors.New("保存任务成功-mongoId转换失败")
		} else {
			task.MongoId = mongoId
			log.Info("保存任务成功", zap.Any("task", task), zap.Any("mongo_id", r.InsertedID))
		}
	}
	ts.createTaskChan <- task
	return nil
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
			close(ts.deleteTaskChan)
			ts.logger.Info("关闭任务删除通道 deleteTaskChan")
			close(ts.taskResultChan)
			ts.logger.Info("关闭任务修改通道 updateTaskChan")
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
		case task_id, ok := <-ts.deleteTaskChan:
			if !ok {
				log.Info("接收到任务删除通知 但是deleteTaskChan通道已经被关闭", zap.Any("task_id", task_id))
				return
			}
			log.Info("开始任务调度-接收到任务删除通知", zap.Any("task_id", task_id))
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
						ts.logger.Info("创建任务-待执行列表添加任务", zap.Any("pending_tasks", ts.pendingTasks))
						err := ts.addPendingTask(t)
						if err != nil {
							ts.logger.Error("创建任务-任务无法添加到pendingtasks中", zap.Error(err))
						} else {
							ts.execTaskChan <- t // 任务放入执行通道
						}
					} else {
						ts.logger.Info("创建任务-任务不可用", zap.Any("task", t))
					}
				} else {
					ts.logger.Info("创建任务-任务状态不是等待执行", zap.Any("task", t))
				}
			}
		case t, ok := <-ts.updateTaskChan:
			if !ok {
				ts.logger.Info("任务更新通道已关闭! updateTaskChan", zap.Any("task", t))
				return
			} else {
				ts.logger.Info("任务更新通道接受到任务 updateTaskChan", zap.Any("task", t))
				if t.StateCode == m.Pending { // 等待执行
					if t.Available { // 可用
						// 任务放入待执行列表
						if t.PlanExecAt.Before(time.Now()) ||
							t.PlanExecAt.Equal(time.Now()) { // 计划执行时间小于等于当前时间
							ts.logger.Info("更新任务立即执行-任务计划执行时间小于等于当前时间 任务进入待执行列表", zap.Any("task", t))
						} else {
							ts.logger.Info("更新任务稍后执行-任务计划执行时间大于当前时间", zap.Any("task", t))
						}
						ts.logger.Info("更新任务添加到待执行列表添加任务", zap.Any("pending_tasks", ts.pendingTasks))
						err := ts.addPendingTask(t)
						if err != nil {
							ts.logger.Error("更新任务-任务再pendingtasks中已经存在", zap.Error(err))
							ts.deletePendingTask(t.Id)
							ts.logger.Info("更新任务-删除pendingtasks中已经存在的任务", zap.Any("id", t.Id))
							ts.addPendingTask(t)
							ts.logger.Info("更新任务-再次将任务放入pendingtasks", zap.Any("id", t.Id))
							ts.execTaskChan <- t // 任务放入执行通道
						} else {
							ts.execTaskChan <- t // 任务放入执行通道
						}
					} else {
						ts.logger.Info("更新任务-任务不可用", zap.Any("task", t))
					}
				} else {
					ts.logger.Info("更新任务-任务状态不是等待执行", zap.Any("task", t))
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
