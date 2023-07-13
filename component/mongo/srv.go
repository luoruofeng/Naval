package mongo

import (
	"context"
	"os"
	"strconv"
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

func NewMongoSrv(lc fx.Lifecycle, log *zap.Logger, configPathMap map[string]string) MongoSrv {
	conf := c.GetConfig(log, configPathMap["mongo-cnf"])
	opts := options.Client()
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts = opts.SetServerAPIOptions(serverAPI)

	if v := os.Getenv("MONGO_ADDR"); v != "" {
		conf.Addr = v
		log.Info("环境变量MONGO_ADDR已设置", zap.String("MONGO_ADDR", conf.Addr))
	}
	opts = opts.ApplyURI(conf.Addr)

	if v := os.Getenv("MONGO_CONNECT_TIMEOUT"); v != "" {
		if r, err := strconv.Atoi(v); err == nil {
			log.Info("环境变量MONGO_CONNECT_TIMEOUT已设置", zap.Int("MONGO_CONNECT_TIMEOUT", r))
			conf.ConnectTimeout = r
		}
	}
	opts = opts.SetConnectTimeout(time.Duration(conf.ConnectTimeout) * time.Second)

	if v := os.Getenv("MONGO_SERVER_SELECTION_TIMEOUT"); v != "" {
		if r, err := strconv.Atoi(v); err == nil {
			log.Info("环境变量MONGO_SERVER_SELECTION_TIMEOUT已设置", zap.Int("MONGO_SERVER_SELECTION_TIMEOUT", r))
			conf.ServerSelectionTimeout = r
		}
	}
	opts = opts.SetServerSelectionTimeout(time.Duration(conf.ServerSelectionTimeout) * time.Second)

	if v := os.Getenv("MONGO_SOCKET_TIMEOUT"); v != "" {
		if r, err := strconv.Atoi(v); err == nil {
			log.Info("环境变量MONGO_SOCKET_TIMEOUT已设置", zap.Int("MONGO_SOCKET_TIMEOUT", r))
			conf.SocketTimeout = r
		}
	}
	opts = opts.SetSocketTimeout(time.Duration(conf.SocketTimeout) * time.Second)

	if v := os.Getenv("MONGO_USER_NAME"); v != "" {
		conf.Username = v
		log.Info("环境变量MONGO_USER_NAME已设置", zap.String("MONGO_USER_NAME", conf.Username))
	}
	if v := os.Getenv("MONGO_PASSWORD"); v != "" {
		conf.Password = v
		log.Info("环境变量MONGO_PASSWORD已设置", zap.String("MONGO_PASSWORD", conf.Password))
	}

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
			log.Info("已经成功连接MongoDB!")
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
