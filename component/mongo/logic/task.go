package logic

import (
	"context"
	"time"

	ms "github.com/luoruofeng/Naval/component/mongo"
	"github.com/luoruofeng/Naval/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
		return nil, err
	} else {
		return r, nil
	}
}

func (s TaskMongoSrv) GetAll() ([]model.Task, error) {
	collection := s.Collection
	filter := bson.M{
		"Available": true,
	}
	cursor, err := collection.Find(context.Background(), filter)
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

func (s TaskMongoSrv) GetPendingTask() ([]model.Task, error) {
	collection := s.Collection
	filter := bson.M{
		"Available": true,
		"StateCode": model.Pending,
	}
	cursor, err := collection.Find(context.Background(), filter)
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

func (s TaskMongoSrv) FindById(id string) (*model.Task, error) {
	findFilter := bson.M{"Id": id, "Available": true}
	r := s.Collection.FindOne(context.Background(), findFilter)
	if r.Err() != nil {
		return nil, r.Err()
	} else {
		var task model.Task
		r.Decode(&task)
		return &task, nil
	}
}

func (s TaskMongoSrv) Delete(id primitive.ObjectID) (*mongo.UpdateResult, error) {
	r, err := s.Collection.UpdateByID(context.Background(), id, bson.M{
		"$set": bson.M{
			"Available": false,
			"DeleteAt":  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s TaskMongoSrv) Update(task model.Task) (*mongo.UpdateResult, error) {
	if bs, err := bson.Marshal(task); err != nil {
		return nil, err
	} else {
		var updateKVs bson.D
		bson.Unmarshal(bs, &updateKVs)
		updateData := bson.M{
			"$set": updateKVs,
		}
		r, err := s.Collection.UpdateByID(context.Background(), task.MongoId, updateData)
		if err != nil {
			return nil, err
		}
		return r, nil
	}
}

func (s TaskMongoSrv) UpdateKVs(mongoId primitive.ObjectID, kvs map[string]interface{}) (*mongo.UpdateResult, error) {
	// 将 map 转换为更新的文档
	update := bson.M{"$set": kvs}

	// 执行更新操作
	result, err := s.Collection.UpdateOne(context.Background(), bson.M{"_id": mongoId}, update)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s TaskMongoSrv) UpdatePushKV(mongoId primitive.ObjectID, key string, value interface{}) (*mongo.UpdateResult, error) {
	updateData := bson.M{
		"$push": bson.D{{Key: key, Value: value}},
	}
	r, err := s.Collection.UpdateByID(context.Background(), mongoId, updateData)
	if err != nil {
		return nil, err
	}
	return r, nil
}
