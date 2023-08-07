package logic

import (
	"context"

	m "github.com/luoruofeng/Naval/component/mongo"
	"github.com/luoruofeng/Naval/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TaskResultMongoSrv struct {
	Collection     *mongo.Collection
	TaskCollection *mongo.Collection
	MongoSrv       m.MongoSrv
	Logger         *zap.Logger
}

func NewTaskResultMongoSrv(lc fx.Lifecycle, mongoSrv m.MongoSrv, logger *zap.Logger) TaskResultMongoSrv {
	result := TaskResultMongoSrv{MongoSrv: mongoSrv, Logger: logger, Collection: mongoSrv.Db.Collection("task_results"), TaskCollection: mongoSrv.Db.Collection("tasks")}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			db := mongoSrv.Db
			logger.Info("启动mongo-taskresult持久化服务")
			// 检查 TaskResults 是否存在
			collection := result.Collection
			count, err := collection.EstimatedDocumentCount(context.Background())
			if err != nil {
				logger.Error("查询collection：TaskResults失败", zap.Error(err))
				panic(err)
			}

			if count == 0 {
				// 如果 TaskResults 不存在则创建
				err := db.CreateCollection(context.Background(), "task_results")
				if err != nil {
					logger.Error("创建collection：task_results失败", zap.Error(err))
				} else {
					logger.Info("task_results collection 创建成功!")
				}
			} else {
				logger.Info("task_results collection 已经存在!")

			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("销毁mongo-taskresult持久化服务")
			return nil
		},
	})
	return result
}

// Save 保存TaskResult
func (s TaskResultMongoSrv) Save(TaskResult model.TaskResult) (*mongo.InsertOneResult, error) {
	r, err := s.Collection.InsertOne(context.Background(), TaskResult)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// FindById 根据taskId查询TaskResult
func (s TaskResultMongoSrv) FindById(taskId string) ([]*model.TaskResult, error) {
	result := make([]*model.TaskResult, 0)

	findFilter := bson.M{"TaskId": taskId}
	c, err := s.Collection.Find(context.Background(), findFilter)
	if err != nil {
		return nil, err
	}
	err = c.All(context.Background(), &result)
	if err != nil {
		return nil, err
	}

	availableResult := make([]*model.TaskResult, 0)
	//查询TaskCollection中的Task的available字段
	for _, r := range result {
		var task model.Task
		err := s.TaskCollection.FindOne(context.Background(), bson.M{"Id": taskId}).Decode(&task)
		if err != nil {
			return nil, err
		}
		if task.Available {
			availableResult = append(availableResult, r)
		}
	}

	return availableResult, nil
}
