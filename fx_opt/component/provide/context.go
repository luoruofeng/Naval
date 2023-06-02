package provide

import (
	"context"
)

func NewContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.TODO())
	return ctx, cancel
}
