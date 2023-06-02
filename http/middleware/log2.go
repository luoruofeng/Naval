package middleware

import (
	"net/http"

	"go.uber.org/zap"
)

// 定义结构体，结构体中定义我们所需要的实例(该实例已经在fx中注册)，例如我们这里需要*zap.Logger。
type LogMiddleware2 struct {
	logger *zap.Logger
}

// 定义上面结构体的New函数，参数为我们所需要的实例(该实例已经在fx中注册)，例如我们这里需要*zap.Logger。
func NewLogMiddleware2(logger *zap.Logger) *LogMiddleware2 {
	return &LogMiddleware2{logger: logger}
}

// 定义拦截器，该拦截器会在每次HTTP请求中被调用，*next.ServeHTTP(w, r)*为handler方法本身。可以在该方法上下去编写拦截器逻辑。
func (l *LogMiddleware2) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// l.logger.Info("正在执行log2拦截器")
		next.ServeHTTP(w, r)
		// l.logger.Info("log2拦截器即将执行完成")

	})
}
