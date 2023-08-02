package middleware

import (
	"net/http"

	"go.uber.org/zap"
)

type CORSMiddleware struct {
	logger *zap.Logger
}

func NewCORSMiddleware(logger *zap.Logger) *CORSMiddleware {
	return &CORSMiddleware{logger: logger}
}

func (cors *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		// Check if the request is an OPTIONS request
		if r.Method == http.MethodOptions {
			// Respond with a 200 status code for OPTIONS request
			w.WriteHeader(http.StatusOK)
			return
		}
		// Continue with the next handler in the chain
		next.ServeHTTP(w, r)
	})
}
