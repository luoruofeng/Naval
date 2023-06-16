package mongo

import (
	"context"
	"fmt"
	"time"

	c "github.com/luoruofeng/Naval/component/mongo/conf"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MongoSrv struct {
	Cli *mongo.Client
	Log *zap.Logger
	Db  *mongo.Database
}

func NewMongoSrv(lc fx.Lifecycle, log *zap.Logger) MongoSrv {
	conf := c.GetConfig()
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client()
	opts = opts.ApplyURI(conf.Addr)
	opts = opts.SetServerAPIOptions(serverAPI)
	opts = opts.SetConnectTimeout(time.Duration(conf.ConnectTimeout) * time.Second)
	opts = opts.SetServerSelectionTimeout(time.Duration(conf.ServerSelectionTimeout) * time.Second)
	opts = opts.SetSocketTimeout(time.Duration(conf.SocketTimeout) * time.Second)
	credential := options.Credential{
		Username: conf.Username,
		Password: conf.Password,
	}
	opts = opts.SetAuth(credential)
	client, err := mongo.Connect(context.TODO(), opts)
	db := client.Database(conf.DbName, nil)
	if err != nil {
		log.Error("mongo connect error", zap.Error(err))
		panic(err)
	}
	result := MongoSrv{Cli: client, Log: log, Db: db}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("mongo client启动")
			err = client.Ping(context.Background(), nil)
			if err != nil {
				log.Error("mongo client ping error", zap.Error(err))
				panic(err)
			}
			fmt.Println("已经成功连接MongoDB!")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err = client.Disconnect(context.TODO()); err != nil {
				log.Error("mongo client关闭失败", zap.Error(err))
				panic(err)
			}
			log.Info("mongo client关闭")
			return nil
		},
	})

	return result
}
