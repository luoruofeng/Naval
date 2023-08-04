package middleware

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// 定义结构体，结构体中定义我们所需要的实例(该实例已经在fx中注册)，例如我们这里需要*zap.Logger。
type LogMiddleware struct {
	logger *zap.Logger
}

// 定义上面结构体的New函数，参数为我们所需要的实例(该实例已经在fx中注册)，例如我们这里需要*zap.Logger。
func NewLogMiddleware(logger *zap.Logger) *LogMiddleware {
	return &LogMiddleware{logger: logger}
}

// 定义拦截器，该拦截器会在每次HTTP请求中被调用，*next.ServeHTTP(w, r)*为handler方法本身。可以在该方法上下去编写拦截器逻辑。
func (l *LogMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		uuid := uuid.New().String()
		r.Header.Set("X-Request-Id", uuid)
		if r.Method != http.MethodOptions {
			l.logger.Info("客户端发起的HTTP请求开始",
				zap.String("uuid", uuid),
				zap.String("method", r.Method),
				zap.String("url", r.RequestURI),
				zap.String("proto", r.Proto),
				zap.String("remoteAddr", r.RemoteAddr),
				zap.String("userAgent", r.UserAgent()),
				zap.String("referer", r.Referer()),
				zap.Any("header", r.Header),
				zap.Time("startTime", startTime),
			)
		}
		recorder := httptest.NewRecorder()
		next.ServeHTTP(recorder, r)
		for k, v := range recorder.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(recorder.Code)
		_, err := w.Write(recorder.Body.Bytes())
		if err != nil {
			// 处理错误
			l.logger.Error("写入响应失败", zap.Error(err))
		}
		if r.Method != http.MethodOptions {
			l.logger.Info("客户端发起的HTTP请求结束",
				zap.String("uuid", uuid),
				zap.Int("status", recorder.Code),
				zap.Time("endTime", time.Now()),
				zap.Int64("totalTime", time.Since(startTime).Milliseconds()),
			)
		}
	})
}
