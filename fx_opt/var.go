package fx_opt

import (
	m "github.com/luoruofeng/Naval/component/mongo"
	ml "github.com/luoruofeng/Naval/component/mongo/logic"
	kube "github.com/luoruofeng/Naval/kube"
	"github.com/luoruofeng/Naval/srv"
)

// 添加其他需要在fx中构建的实例的方法
var ConstructorFuncs = []interface{}{
	srv.NewTaskSrv,
	m.NewMongoSrv,
	ml.NewTaskMongoSrv,
	ml.NewTaskResultMongoSrv,
	kube.NewTaskKubeSrv,
}

// 在ConstructorFuncs添加了方法后，如果需要在方法的参数中传递fx.Lifecycle，已实现fx.Hook。需要在下方添加fx的invoke方法。
var InvokeFuncs = []interface{}{
	func(ts *srv.TaskSrv) {
	},
	func(m m.MongoSrv) {},
	func(m ml.TaskMongoSrv) {},
	func(m ml.TaskResultMongoSrv) {},
	func(kube *kube.TaskKubeSrv) {},
}
