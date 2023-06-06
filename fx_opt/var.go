package fx_opt

import (
	"context"

	"github.com/luoruofeng/Naval/srv"
	"go.uber.org/zap"
)

// 添加其他需要在fx中构建的实例的方法
var ConstructorFuncs = []interface{}{
	srv.NewTask,
	srv.NewRegistry,
}

// 在ConstructorFuncs添加了方法后，如果需要在方法的参数中传递fx.Lifecycle，已实现fx.Hook。需要在下方添加fx的invoke方法。
var InvokeFuncs = []interface{}{
	func(ts srv.TaskSrv, cancel context.CancelFunc) {
		go ts.InitWorkerpools()
	},
	func(rs srv.RegistryServer, logger *zap.Logger) {
		logger.Info("registry服务开启端口监听")
		err := rs.Registry.ListenAndServe()
		if err != nil {
			logger.Error("registry listen and serve error", zap.Error(err))
			panic(err)
		}
	},
}
