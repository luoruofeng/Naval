package http

import (
	"net/http"

	"go.uber.org/fx"
)

type Middleware interface {
	Middleware(next http.Handler) http.Handler
}

func AsMiddleware(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Middleware)),
		fx.ResultTags(`group:"middlewares"`),
	)
}

func AllAsMiddleware(ms ...any) []any {
	r := make([]any, 0, len(ms))
	for _, f := range ms {
		r = append(r, AsMiddleware(f))
	}
	return r
}
