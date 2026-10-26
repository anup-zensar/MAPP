// Package querycount counts the SQL statements sent while serving one request, for the request log.
package querycount

import (
	"context"
	"sync/atomic"
)

type contextKey struct{}

// With returns ctx with a counter at zero, or ctx itself when it counts already: a request has one count.
func With(ctx context.Context) context.Context {
	if _, ok := ctx.Value(contextKey{}).(*atomic.Int64); ok {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, new(atomic.Int64))
}

func Add(ctx context.Context) {
	if counter, ok := ctx.Value(contextKey{}).(*atomic.Int64); ok {
		counter.Add(1)
	}
}

func Of(ctx context.Context) int64 {
	if counter, ok := ctx.Value(contextKey{}).(*atomic.Int64); ok {
		return counter.Load()
	}
	return 0
}
