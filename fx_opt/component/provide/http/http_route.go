package http

import (
	"net/http"

	"go.uber.org/fx"
)

// Route is an http.Handler that knows the mux pattern
// under which it will be registered.
type Route interface {
	http.Handler

	// Pattern reports the path at which this is registered.
	Pattern() string
}

// AsRoute annotates the given constructor to state that
// it provides a route to the "handlers" group.
func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Route)),
		fx.ResultTags(`group:"handlers"`),
	)
}

func AllAsRoute(fs ...any) []any {
	r := make([]any, 0, len(fs))
	for _, f := range fs {
		r = append(r, AsRoute(f))
	}
	return r
}
