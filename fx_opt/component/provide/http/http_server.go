package http

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/luoruofeng/Naval/conf"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewHTTPServer(lc fx.Lifecycle, logger *zap.Logger, c *conf.Config, r *mux.Router, cancel context.CancelFunc) *http.Server {
	server := &http.Server{
		Addr:         c.HttpAddr,
		Handler:      r,
		WriteTimeout: time.Duration(c.HttpWriteOverTime) * time.Second,
		ReadTimeout:  time.Duration(c.HttpReadOverTime) * time.Second,
	}
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ln, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return err
			}
			go func() {
				logger.Info("HTTP server 启动!", zap.String("addr", server.Addr))
				server.Serve(ln)
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping HTTP server!")
			server.Close()
			logger.Info("HTTP server 停止!")
			cancel()
			logger.Info("context 取消!")
			return server.Shutdown(ctx)
		},
	})

	return server
}
