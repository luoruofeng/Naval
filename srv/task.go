package srv

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/luoruofeng/Naval/conf"
	"github.com/luoruofeng/Naval/kube"
	"github.com/luoruofeng/Naval/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	mongo "github.com/luoruofeng/Naval/component/mongo/logic"
	m "github.com/luoruofeng/Naval/model"
)

type TaskSrv struct {
	logger                *zap.Logger              // 日志
	taskResultChan        chan m.TaskResult        // 任务结果通道
	createTaskChan        chan m.Task              // 任务创建通道
	createConvertTaskChan chan m.Task              // 转换任务创建通道
	execTaskChan          chan m.Task              // 任务执行通道 用于将任务放入pendingTasks后的通知
	deleteTaskChan        chan string              // 任务删除通道
	updateTaskChan        chan m.Task              //任务修改通道
	ctx                   context.Context          // 任务调度上下文
	pendingTasks          []*m.Task                //待执行的任务slice
	mongoT                mongo.TaskMongoSrv       // mongo任务服务
	mongoTR               mongo.TaskResultMongoSrv // mongo任务结果服务
	lastExecTimeSecond    int                      // 等待多少秒后开始执行任务
	lock                  sync.Mutex               //用于pendingTasks的锁
	kubeTaskSrv           *kube.TaskKubeSrv        //k8s服务
	cnf                   *conf.Config             //项目主配置
}

func getTaskIds(tasks []*model.Task) []string {
	ids := make([]string, 0)
	for _, t := range tasks {
		ids = append(ids, t.Id)
	}
	return ids
}

func NewTaskSrv(lc fx.Lifecycle, kubeTaskSrv *kube.TaskKubeSrv, logger *zap.Logger, ctx context.Context, taskMongoSrv mongo.TaskMongoSrv, taskResultMongoSrv mongo.TaskResultMongoSrv, cnf *conf.Config) *TaskSrv {
	logger.Info("初始化task服务")
	logger.Info("初始化task结果通道")
	taskResults := make(chan m.TaskResult)
	logger.Info("初始化task创建通道")
	taskCreatedChan := make(chan m.Task)
	logger.Info("初始化task转换创建通道")
	taskConvertCreatedChan := make(chan m.Task)
	logger.Info("初始化task执行通道")
	deleteTaskChan := make(chan string)
	logger.Info("初始化task删除通道")
	updateTaskChan := make(chan m.Task)
	logger.Info("初始化task更新通道")
	execTaskChan := make(chan m.Task)
	logger.Info("初始化待执行的任务列表")
	pendingTasks := make([]*m.Task, 0)
	// 初始化mongo任务数据到pendingTasks
	pts, err := taskMongoSrv.GetPendingTask()
	if err != nil {
		logger.Error("初始化待执行的任务列表失败", zap.Error(err))
		return nil
	} else {
		pendingTasks = append(pendingTasks, pts...)
		logger.Info(
			"初始化待执行的任务列表成功",
			zap.Any("pending_task_length", len(pendingTasks)),
			zap.Any("pending_task_ids", getTaskIds(pendingTasks)),
		)
	}

	result := TaskSrv{
		logger:                logger,
		taskResultChan:        taskResults,
		createTaskChan:        taskCreatedChan,
		createConvertTaskChan: taskConvertCreatedChan,
		execTaskChan:          execTaskChan,
		mongoT:                taskMongoSrv,
		mongoTR:               taskResultMongoSrv,
		ctx:                   ctx,
		pendingTasks:          pendingTasks,
		lastExecTimeSecond:    1,
		deleteTaskChan:        deleteTaskChan,
		updateTaskChan:        updateTaskChan,
		kubeTaskSrv:           kubeTaskSrv,
		cnf:                   cnf,
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

// Calculates the latest execution time of the pending task
func (ts *TaskSrv) CalcLatestExecTime() {
	log := ts.logger
	now := time.Now()
	lastPlanExecTime := now
	ts.lastExecTimeSecond = 0
	ts.WalkPendingTasks(func(i int, task *m.Task) (bool, error) {
		if task.Available {
			if task.PlanExecAt.Before(now) ||
				task.PlanExecAt.Equal(now) { // 计划执行时间小于等于当前时间
				// 标记要从pendingTasks移除任务为nil
				ts.pendingTasks[i] = nil
				// 执行任务
				go ts.ExecPenddingTask(*task)
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
			log.Info("任务不可用", zap.String("task.id", task.Id))
			// 从pendingTasks移除任务
			ts.pendingTasks = append(ts.pendingTasks[:i], ts.pendingTasks[i+1:]...)
		}
		return false, nil
	})
	// 删除pendingTasks中的nil

	ts.DeleteNil()
	if ts.lastExecTimeSecond == 0 {
		ts.lastExecTimeSecond = 1
	}
	log.Info("计算-最近的执行时间 ", zap.Int("lastExecTimeSecond", ts.lastExecTimeSecond), zap.Any("lastPlanExecTime", lastPlanExecTime), zap.Any("pending_tasks number", len(ts.pendingTasks)))
}

func (t *TaskSrv) GetAllTask() ([]m.Task, error) {
	return t.mongoT.GetAll()
}

func (t *TaskSrv) Unmarshal(c []byte) (*m.Task, error) {
	var task m.Task
	err := yaml.Unmarshal(c, &task)
	if err != nil {
		t.logger.Error(fmt.Sprintf("Could not parse YAML: %v", err), zap.Any("input", string(c)))
		return nil, err
	}
	return &task, nil
}

func (ts *TaskSrv) Execete(id string) error {
	ts.logger.Info("执行任务", zap.Any("id", id))
	//从mongo中查询任务
	if t, err := ts.mongoT.FindById(id); err != nil {
		ts.logger.Error("执行任务-失败-查询任务失败", zap.Any("id", id), zap.Error(err))
		return err
	} else {
		stateCode := t.StateCode
		if stateCode == m.Running {
			ts.logger.Info("执行任务-失败-任务正在运行中", zap.Any("id", id), zap.Any("task.Id", t.Id))
			return fmt.Errorf("执行任务-失败-任务正在运行中 task:%v", t)
		}

		ts.logger.Info("执行任务-开始执行mongo中的任务", zap.Any("id", id), zap.Any("task.Id", t.Id))
		if stateCode == m.Pending ||
			stateCode == m.Unknown ||
			t.StateCode == 0 {

			t.WaitSeconds = 0
			ts.mongoT.UpdateKVs(t.MongoId, map[string]interface{}{"WaitSeconds": 0})
			// 执行任务
			err := ts.Update(*t)
			if err != nil {
				ts.logger.Error("执行任务失败-更新任务失败", zap.Any("task.Id", t.Id), zap.Error(err))
				return err
			}
		} else {
			ts.logger.Info("执行任务-失败-任务状态不可用", zap.Any("id", id), zap.Any("task.Id", t.Id))
			return fmt.Errorf("执行任务-失败-任务状态不可用 task:%v", t)
		}
	}
	return nil
}

func (ts *TaskSrv) DeleteResourceByStructs(allSuccessfulRes []struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}) []error {
	var errs []error = make([]error, 0)
	for _, obj := range allSuccessfulRes {
		if err := ts.kubeTaskSrv.Delete(obj.Kind, obj.Name); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// 删除k8s中的resources
func (ts *TaskSrv) DeleteResource(t *model.Task) []error {
	kinds, names, err := kube.GetK8sYamlKindAndName(t)
	if err != nil {
		ts.logger.Error("删除任务后删除k8s资源-获取k8s yaml kind和name失败", zap.Any("task.Id", t.Id), zap.Error(err))
	}
	var errs []error = make([]error, 0)
	for i, kind := range kinds {
		if err := ts.kubeTaskSrv.Delete(kind, names[i]); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// 删除任务中的items
func (ts *TaskSrv) Delete(id string) error {
	ts.logger.Info("删除任务", zap.Any("id", id))
	if t, err := ts.mongoT.FindById(id); err != nil {
		ts.logger.Error("删除任务失败-查询任务失败", zap.Any("id", id), zap.Error(err))
		return err
	} else {
		ts.logger.Info("删除任务-开始删除mongo中的任务", zap.Any("id", id), zap.Any("task.Id", t.Id))
		if t.StateCode != m.Running {
			// mongo删除任务
			if r, err := ts.mongoT.Delete(t.MongoId); err != nil {
				ts.logger.Error("删除任务失败-删除mongo中的任务失败", zap.Any("task.Id", t.Id), zap.Error(err))
				return err
			} else {
				ts.logger.Info("删除任务-删除mongo中的任务成功", zap.Any("task.Id", t.Id), zap.Any("delete_result", r))
				// 任务从待执行列表移除
				ts.DeletePendingTask(id)
				ts.logger.Info("删除任务-任务从pend_tasks中成功删除", zap.Any("task.Id", t.Id))
				// 任务从执行通道移除后重新计算最近的执行时间
				if t.StateCode == m.Pending {
					ts.deleteTaskChan <- id // 任务放入删除通道
				}
				// 如果任务的计划执行时间小于当前时间，删除k8s中的resources
				if time.Now().After(t.PlanExecAt) {
					// 删除k8s中的resources
					go func() {
						ts.DeleteResource(t)
					}()
				}
				return nil
			}
		} else if t.StateCode == m.Running { // 运行中不能删除
			ts.logger.Info("任务正在运行中不能删除", zap.Any("task.Id", t.Id))
			return fmt.Errorf("任务正在运行中不能删除 task:%v", t)
		}
		return nil
	}
}

func (ts *TaskSrv) ExecPenddingTask(task m.Task) {
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
		log.Error("任务执行-执行任务前更新任务状态失败", zap.Error(err))
		return
	} else {
		log.Info("任务执行-执行任务前更新任务状态成功", zap.Any("update_result", r))
	}
	log.Info("任务执行-开始", zap.String("task.id", task.Id))
	// 执行任务
	var execSuccessfully bool = true //任务是否成功的总体结果

	// 记录所有执行成功的k8s资源
	var allSuccessfulRes []struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	}
	// 任务执行项
	for i, item := range task.Items {
		var tr m.TaskResult
		if item.K8SYamlContent != "" {
			// 创建k8s资源
			successfulRes, err := ts.kubeTaskSrv.Create(item.K8SYamlContent)
			log.Info("任务执行项-创建k8s资源-完成", zap.String("task.id", task.Id), zap.String("所有执行成功的k8s资源", fmt.Sprintf("%v", successfulRes)), zap.String("err", fmt.Sprintf("%v", err)))
			allSuccessfulRes = append(allSuccessfulRes, successfulRes...)
			if err != nil {
				log.Error("任务执行项-创建k8s资源-失败", zap.String("item.K8SYamlContent", item.K8SYamlContent), zap.Error(err))
				tr = m.NewTaskResult(task.Id, i, err.Error(), "", m.ResultFail)
				execSuccessfully = false
			} else {
				log.Info("任务执行项-创建k8s资源-成功", zap.String("task.id", task.Id))
				tr = m.NewTaskResult(task.Id, i, "", "任务执行项-成功", m.ResultSuccess)
			}

			// 更新mongoDB任务执行项的结果
			insertR, err := ts.mongoTR.Save(tr)
			if err != nil || insertR.InsertedID == nil {
				log.Error("任务执行项-保存任务结果失败 Save", zap.Any("task result", tr), zap.Error(err))
				continue
			} else {
				log.Info("任务执行项-保存任务结果成功 Save", zap.Any("task result", tr), zap.Any("mongo id", insertR.InsertedID))
			}
			// 更新mongoDB任务执行项的结果
			go func(mongoId primitive.ObjectID, trid string) {
				if updateResult, err := ts.mongoT.UpdatePushKV(mongoId, "ExecResultIds", trid); err != nil || updateResult.ModifiedCount < 1 {
					log.Error("任务执行项-更新任务结果集-失败 UpdatePushKV", zap.Any("mongo id", mongoId), zap.String("task result id", trid), zap.Any("updateResult", updateResult), zap.Error(err))
				} else {
					log.Info("任务执行项-更新任务结果集-成功 UpdatePushKV", zap.Any("mongo id", mongoId), zap.String("task result id", trid), zap.Any("updateResult", updateResult))
				}
			}(task.MongoId, tr.Id)
		}
	}

	// 更新任务执行结果
	task.ExecSuccessfully = execSuccessfully
	// 更新任务执行时间
	task.ExtDoneTime = time.Now()
	// 设置任务没有运行
	task.IsRunning = false
	// 更新任务状态
	if execSuccessfully {
		task.StateCode = m.Executed
	} else {
		task.StateCode = m.ExecuteFailed
		// 回滚操作：删除k8s中的resources
		go func() {
			// 删除k8s中的执行成功的resources
			log.Info("任务执行-执行失败-回滚开始-删除k8s中的执行成功的资源", zap.String("task.id", task.Id), zap.Any("所有执行成功的k8s资源", fmt.Sprintf("%v", allSuccessfulRes)))
			ts.DeleteResourceByStructs(allSuccessfulRes)
		}()
	}

	// 更新mongo任务
	if r, err := ts.mongoT.Update(task); err != nil {
		log.Error("任务执行-结束-更新任务状态失败", zap.Error(err))
		return
	} else {
		log.Info("任务执行-结束-更新任务状态成功", zap.Any("update_result", r))
	}
}

func (ts *TaskSrv) Update(task m.Task) error {
	log := ts.logger

	log.Info("更新任务-mongoDB根据id查询task。", zap.String("task.id", task.Id))
	// mongo更新任务前先查询任务确保任务存在并且确定任务状态不为运行中
	if t, err := ts.mongoT.FindById(task.Id); err != nil {
		log.Error("更新任务-失败。查询任务失败。", zap.Any("task-id", task.Id), zap.Error(err))
		return err
	} else if !t.Available {
		log.Info("更新任务-失败。任务不可用。", zap.Any("task.Id", t.Id))
		return errors.New("更新任务失败-任务不可用")
	} else if t.StateCode == m.Running {
		log.Info("更新任务-失败。任务正在运行中不能更新。", zap.Any("task.Id", t.Id))
		return errors.New("更新任务失败-任务正在运行中不能更新")
	} else if t.StateCode == m.Executed {
		log.Info("更新任务-失败。任务已经执行完毕。", zap.Any("task.Id", t.Id))
		return errors.New("更新任务失败-任务已经执行完毕")
	} else {
		t.UpdateAt = time.Now()
		t.PlanExecAt = time.Now().Add(time.Duration(t.WaitSeconds) * time.Second)
		t.StateCode = m.Pending
		if r, err := ts.mongoT.Update(*t); err != nil {
			log.Error("更新任务-失败", zap.Any("task.Id", t.Id), zap.Error(err))
			return err
		} else {
			log.Info("更新任务-成功", zap.Any("task.Id", t.Id), zap.Any("result", r))
		}

		go func() {
			ts.updateTaskChan <- *t
		}()
		return nil
	}

}

func (ts *TaskSrv) Add(task m.Task) error {
	log := ts.logger
	if task.Type == m.Create {
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
	} else if task.Type == m.Convert {
		task.IsRunning = true // 设置任务在运行
		// 设置可用状态
		task.Available = true
		// 设置任务转化创建时间
		task.ConvertTime = time.Now()
		// 设置任务创建时间
		task.CreatedAt = time.Now()
		// 设置任务执行状态码
		task.StateCode = m.Running
		// 任务执行次数
		task.ExtTimes = 0
		// 设置任务转换次数
		task.ConvertTimes = 1
	}
	// mongo保存任务
	log.Info("创建任务-保存mongoDB", zap.String("task.id", task.Id))
	if r, err := ts.mongoT.Save(task); err != nil {
		log.Error("创建任务-保存到mongoDB失败", zap.Error(err))
		return err
	} else {
		mongoId, ok := r.InsertedID.(primitive.ObjectID)
		if !ok {
			log.Info("创建任务-mongo Id转换失败", zap.Any("mongo_id", r.InsertedID))
			return errors.New("创建任务-mongoId转换失败")
		} else {
			task.MongoId = mongoId
			log.Info("创建任务-成功", zap.Any("mongo_id", r.InsertedID))
		}
	}

	if task.Type == m.Create {
		// 任务放入创建通道
		ts.createTaskChan <- task
	} else if task.Type == m.Convert {
		ts.StartConvert(task)
	}
	return nil
}

func (ts *TaskSrv) StartConvert(task m.Task) error {
	if ts.cnf.AsyncConvert {
		// 异步转换
		ts.createConvertTaskChan <- task
	} else {
		// 同步转换
		return ts.TaskConvert(task)
	}
	return nil
}

func (ts *TaskSrv) UpdateConvert(t m.Task) error {
	//从mongo中查询任务
	if task, err := ts.mongoT.FindById(t.Id); err != nil {
		ts.logger.Error("更新任务-根据Id没有查询到任务", zap.Any("task.Id", t.Id), zap.Error(err))
		return err
	} else if task.StateCode == m.Executed {
		ts.logger.Error("更新任务-失败-任务已经执行完毕", zap.Any("task.Id", t.Id), zap.Error(errors.New("任务已经执行完毕不能再次更改")))
		return fmt.Errorf("更新任务-失败-任务已经执行完毕 task:%v", t)
	} else {
		t.MongoId = task.MongoId
		t.Items = nil
		t.ConvertTimes = task.ConvertTimes + 1
		t.ConvertError = ""
		_, err := ts.mongoT.Update(t)
		if err != nil {
			ts.logger.Error("更新任务-更新任务失败", zap.Any("task.Id", t.Id), zap.Error(err))
			return err
		} else {
			return ts.StartConvert(t)
		}
	}
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
				log.Info("接收到任务添加通知 但是execTaskChan通道已经被关闭", zap.Any("task.Id", t.Id))
				return
			}
			log.Info("开始任务调度-接收到任务添加通知", zap.Any("task.Id", t.Id))
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

func (ts *TaskSrv) TaskConvert(t m.Task) error {
	ts.logger.Info("转换任务-开始", zap.Any("task.Id", t.Id))
	err := ts.Convert(&t)
	if err != nil {
		ts.logger.Error("转换任务-失败", zap.Any("task.Id", t.Id), zap.Error(err))

		// 更新任务状态为停止
		ts.mongoT.UpdateKVs(t.MongoId, map[string]interface{}{"ConvertError": err.Error(), "ConvertSuccessfully": false, "StateCode": model.Wrong, "IsRunning": false})
		// 删除任务中的items
		ts.mongoT.UnsetFieldByID(t.MongoId, "Items")
		return err
	} else {
		ts.logger.Info("转换任务-成功", zap.Any("task.Id", t.Id))
	}
	return nil
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
			close(ts.createConvertTaskChan)
			ts.logger.Info("关闭转换任务创建通道 createConvertTaskChan")
			close(ts.updateTaskChan)
			ts.logger.Info("关闭任务更新通道 updateTaskChan")
			return
		case t, ok := <-ts.createConvertTaskChan:
			if !ok {
				ts.logger.Info("转换任务-任务创建通道已关闭! createConvertTaskChan", zap.Any("task.Id", t.Id))
				return
			} else {
				ts.TaskConvert(t)
			}
		case t, ok := <-ts.createTaskChan:
			if !ok {
				ts.logger.Info("任务创建通道已关闭! createTaskChan", zap.Any("task.Id", t.Id))
				return
			} else {
				ts.logger.Info("任务创建通道接受到任务 createTaskChan", zap.Any("task.Id", t.Id))
				if t.StateCode == m.Pending { // 等待执行
					if t.Available { // 可用
						// 任务放入待执行列表
						if t.PlanExecAt.Before(time.Now()) ||
							t.PlanExecAt.Equal(time.Now()) { // 计划执行时间小于等于当前时间
							ts.logger.Info("任务立即执行-任务计划执行时间小于等于当前时间 任务进入待执行列表", zap.Any("task.Id", t.Id))
						} else {
							ts.logger.Info("任务稍后执行-任务计划执行时间大于当前时间", zap.Any("task.Id", t.Id))
						}
						ts.logger.Info("创建任务-待执行列表添加任务", zap.Any("pending_task number", len(ts.pendingTasks)))
						err := ts.AddPendingTask(t)
						if err != nil {
							ts.logger.Error("创建任务-任务无法添加到pendingtasks中", zap.Error(err))
						} else {
							ts.execTaskChan <- t // 任务放入执行通道
						}
					} else {
						ts.logger.Info("创建任务-任务不可用", zap.Any("task.Id", t.Id))
					}
				} else {
					ts.logger.Info("创建任务-任务状态不是等待执行", zap.Any("task.Id", t.Id))
				}
			}
		case t, ok := <-ts.updateTaskChan:
			//TODO没有进来
			if !ok {
				ts.logger.Info("任务更新通道-已关闭! updateTaskChan", zap.Any("task.Id", t.Id))
				return
			} else {
				ts.logger.Info("任务更新通道接受到任务 updateTaskChan", zap.Any("task.Id", t.Id))
				if t.StateCode == m.Pending { // 等待执行
					if t.Available { // 可用
						// 任务放入待执行列表
						if t.PlanExecAt.Before(time.Now()) ||
							t.PlanExecAt.Equal(time.Now()) { // 计划执行时间小于等于当前时间
							ts.logger.Info("更新任务-立即执行-任务计划执行时间小于等于当前时间 任务进入待执行列表", zap.Any("task.Id", t.Id))
						} else {
							ts.logger.Info("更新任务-稍后执行-任务计划执行时间大于当前时间", zap.Any("task.Id", t.Id))
						}
						ts.logger.Info("更新任务-添加到待执行列表添加任务")
						err := ts.AddPendingTask(t)
						ts.logger.Info("更新任务-添加到待执行列表添加任务完成", zap.Any("pending_tasks", ts.pendingTasks))
						if err != nil {
							ts.logger.Error("更新任务-任务再pendingtasks中已经存在", zap.Error(err))
							ts.DeletePendingTask(t.Id)
							ts.logger.Info("更新任务-删除pendingtasks中已经存在的任务", zap.Any("id", t.Id))
							ts.AddPendingTask(t)
							ts.logger.Info("更新任务-再次将任务放入pendingtasks", zap.Any("id", t.Id))
						}
						ts.execTaskChan <- t // 任务放入执行通道
					} else {
						ts.logger.Info("更新任务-任务不可用", zap.Any("task.Id", t.Id))
					}
				} else {
					ts.logger.Info("更新任务-任务状态不是等待执行", zap.Any("task.Id", t.Id))
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
