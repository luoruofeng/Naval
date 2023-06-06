package fx_opt

import (
	"context"
	"log"
	"net/http"
	"time"

	fxp "github.com/luoruofeng/Naval/fx_opt/component/provide"
	fxhttp "github.com/luoruofeng/Naval/fx_opt/component/provide/http"
	"github.com/luoruofeng/Naval/http/handler"
	"github.com/luoruofeng/Naval/http/middleware"

	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func NewFxSrv(configPath string) FxSrv {
	return FxSrv{
		configFilePath: configPath,
	}
}

func AddOtherProvide(constructors ...interface{}) fx.Option {
	return fx.Provide(constructors...)
}

type FxSrv struct {
	app            *fx.App
	configFilePath string
}

func (f *FxSrv) Start() {
	err := f.app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	<-f.app.Done()
}

func (f *FxSrv) Shutddown() {
	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := f.app.Stop(stopCtx); err != nil {
		log.Fatal(err)
	}
}

func (f *FxSrv) Setup() {

	//handlers Provide
	handlerProv := fx.Provide(
		fxhttp.AllAsRoute(handler.NewTaskHandler)...,
	)

	//middlewares Provide
	middlewareProv := fx.Provide(
		fxhttp.AllAsMiddleware(middleware.NewLogMiddleware)...,
	)

	//config Provide
	cnfProv := fx.Provide(
		fx.Annotate(
			func() string {
				return f.configFilePath
			},
			fx.ResultTags(`name:"configPath"`),
		),
		fx.Annotate(
			fxp.NewConfig,
			fx.ParamTags(``, `name:"configPath"`),
		),
	)

	//Http server Provide
	httpMuxAnno := fx.Annotate(
		fxhttp.NewHTTPRouter,
		fx.ParamTags(`group:"middlewares"`, `group:"handlers"`),
	)

	httpSrvProv := fx.Provide(
		httpMuxAnno,
		fxhttp.NewHTTPServer,
	)

	//logger Provide
	loggerProv := fx.Provide(fxp.NewLogger)

	//context Provide
	contextProv := fx.Provide(fxp.NewContext)

	app := fx.New(
		// The same Zap logger for Fx's own logs as well.
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),

		//Provides
		handlerProv,
		middlewareProv,
		loggerProv,
		contextProv,
		cnfProv,
		httpSrvProv,
		// 添加其他provide
		AddOtherProvide(ConstructorFuncs...),

		//Invoke
		fx.Invoke(
			func(*mux.Router) {},
			func(*http.Server) {},
		),
		fx.Invoke( // 添加其他invoke
			InvokeFuncs...,
		),
	)
	f.app = app
}
