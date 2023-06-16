package main

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"

	m "github.com/luoruofeng/Naval/component/mongo"
)

type Person struct {
	Name string `bson:"name" json:"name"`
	Age  int    `bson:"age" json:"age"`
	City string `bson:"city" json:"city"`
}

func NewCrud(lc fx.Lifecycle, mongoSrv m.MongoSrv, logger *zap.Logger) Crud {
	crud := Crud{MongoSrv: mongoSrv, Logger: logger}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			crud.Run()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})
	return crud
}

type Crud struct {
	MongoSrv m.MongoSrv
	Logger   *zap.Logger
}

func (crud Crud) Run() {
	log := crud.Logger
	log.Info("流水线示例")
	// client := crud.MongoSrv.Cli
	db := crud.MongoSrv.Db
	if err := db.CreateCollection(context.Background(), "persons", nil); err != nil {
		log.Error("创建collection：persons失败", zap.Error(err))
	}
	collection := db.Collection("persons")

	// Insert a new person document
	person := Person{"John Doe", 30, "New York"}
	ir, err := collection.InsertOne(context.Background(), person)
	if err != nil {
		log.Error("插入collection：persons失败", zap.Error(err))
	} else {
		log.Info("插入collection：persons成功", zap.Any("插入结果", ir))
	}

	// Find all person documents
	cursor, err := collection.Find(context.Background(), struct{}{})
	if err != nil {
		log.Error("查询collection cursor：persons失败", zap.Error(err))
	}
	defer cursor.Close(context.Background())

	var persons []Person
	if err = cursor.All(context.Background(), &persons); err != nil {
		log.Error("查询collection：persons失败", zap.Error(err))
	}
	log.Info("查询collection：persons成功", zap.Any("查询结果", persons))
}

func main() {
	fx.New(
		fx.Provide(
			zap.NewExample,
			m.NewMongoSrv,
			NewCrud,
		),
		fx.Invoke(func(m.MongoSrv) {}, func(Crud) {}),
	).Run()
}
