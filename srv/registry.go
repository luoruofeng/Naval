package srv

import (
	"context"
	"log"
	"time"

	"github.com/docker/distribution/configuration"
	"github.com/docker/distribution/registry"
	"github.com/luoruofeng/Naval/conf"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type RegistryServer struct {
	logger   *zap.Logger
	conf     *conf.Config
	Registry *registry.Registry
}

func NewRegistry(lc fx.Lifecycle, ctx context.Context, logger *zap.Logger, c *conf.Config) RegistryServer {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logger.Info("启动registry服务")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("销毁registry服务")
			return nil
		},
	})

	// 创建 Registry 实例
	storage := configuration.Storage{
		"filesystem": configuration.Parameters{
			"rootdirectory": c.RegistryDataDir,
		},
		"cache": configuration.Parameters{
			"confiblobdescriptor": "inmemory",
		},
	}
	health := configuration.Health{
		StorageDriver: struct {
			Enabled   bool          `yaml:"enabled,omitempty"`
			Interval  time.Duration `yaml:"interval,omitempty"`
			Threshold int           `yaml:"threshold,omitempty"`
		}{
			Enabled:   true,
			Interval:  time.Duration(10 * time.Second),
			Threshold: 3,
		},
	}
	auth := configuration.Auth{
		"htpasswd": configuration.Parameters{
			"realm": "Registry Realm",
			"path":  c.RegistryHtpasswdPath,
		},
	}
	configuration := &configuration.Configuration{
		Storage: storage,
		Health:  health,
		Auth:    auth,
	}
	if c.RegistryLogLevel == "info" {
		configuration.Log.Level = "info"
	} else if c.RegistryLogLevel == "debug" {
		configuration.Log.Level = "debug"
	} else {
		configuration.Log.Level = "error"
	}
	configuration.Log.Fields = map[string]interface{}{
		"service": "registry",
	}
	configuration.HTTP.Addr = c.RegistryAddr
	if c.RegistryEnableTls {
		configuration.HTTP.TLS.Certificate = c.RegistryTlsCert
		configuration.HTTP.TLS.Key = c.RegistryTlsKey
	}

	r, err := registry.NewRegistry(ctx, configuration)
	if err != nil {
		log.Fatal(err)
	}
	return RegistryServer{logger: logger, conf: c, Registry: r}
}
