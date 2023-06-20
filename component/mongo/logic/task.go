package logic

import (
	"context"

	ms "github.com/luoruofeng/Naval/component/mongo"
	"github.com/luoruofeng/Naval/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TaskMongoSrv struct {
	Collection *mongo.Collection
	MongoSrv   ms.MongoSrv
	Logger     *zap.Logger
}

func NewTaskMongoSrv(lc fx.Lifecycle, mongoSrv ms.MongoSrv, logger *zap.Logger) TaskMongoSrv {
	result := TaskMongoSrv{MongoSrv: mongoSrv, Logger: logger, Collection: mongoSrv.Db.Collection("tasks")}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			db := mongoSrv.Db
			logger.Info("启动mongo-task持久化服务")
			// 检查 tasks 是否存在
			collection := result.Collection
			count, err := collection.EstimatedDocumentCount(context.Background())
			if err != nil {
				logger.Error("查询collection：tasks失败", zap.Error(err))
				panic(err)
			}

			if count == 0 {
				// 如果 tasks 不存在则创建
				err := db.CreateCollection(context.Background(), "tasks")
				if err != nil {
					logger.Info("创建collection：tasks失败", zap.Error(err))
				} else {
					logger.Info("tasks collection 创建成功!")
				}
			} else {
				logger.Info("tasks collection 已经存在!")
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("销毁mongo-task持久化服务")
			return nil
		},
	})
	return result
}

func (s TaskMongoSrv) Save(task model.Task) (*mongo.InsertOneResult, error) {
	r, err := s.Collection.InsertOne(context.Background(), task)
	if err != nil {
		s.Logger.Error("插入collection：tasks失败", zap.Error(err))
		return nil, err
	} else {
		s.Logger.Info("插入collection：tasks成功", zap.Any("task", task), zap.Any("插入结果", r))
	}
	return r, nil
}

func (s TaskMongoSrv) List() ([]model.Task, error) {
	collection := s.Collection
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		panic(err)
	}
	defer cursor.Close(context.Background())
	tasks := make([]model.Task, 0)
	for cursor.Next(context.Background()) {
		var task model.Task
		err = cursor.Decode(&task)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s TaskMongoSrv) Update(task model.Task) (*mongo.UpdateResult, error) {
	if m, err := bson.Marshal(task); err != nil {
		s.Logger.Error("更新collection：tasks失败-结构体转bson.M失败", zap.Error(err))
		return nil, err
	} else {
		var updateFields bson.D
		bson.Unmarshal(m, &updateFields)
		update := bson.M{
			"$set": updateFields,
		}
		id := bson.M{"_id": bson.M{"$eq": task.MongoId}}
		r, err := s.Collection.UpdateByID(context.Background(), id, update)
		if err != nil {
			s.Logger.Error("更新collection：tasks失败", zap.Error(err), zap.Any("data", m))
			return nil, err
		} else {
			s.Logger.Info("更新collection：tasks成功", zap.Any("task", task), zap.Any("更新结果", r))
		}
		return r, nil
	}
}
