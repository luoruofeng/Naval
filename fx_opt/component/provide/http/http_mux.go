package http

import (
	"context"

	"github.com/gorilla/mux"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewHTTPRouter(middlewares []Middleware, routes []Route, lc fx.Lifecycle, logger *zap.Logger) *mux.Router {
	logger.Info("Executing NewMux.")
	r := mux.NewRouter()

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			//Set route
			logger.Info("Starting bind handler!")
			for _, route := range routes {
				r.Handle(route.Pattern(), route)
			}

			//Set middleware
			logger.Info("Starting bind middleware!")
			for _, m := range middlewares {
				r.Use(m.Middleware)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})

	return r
}
