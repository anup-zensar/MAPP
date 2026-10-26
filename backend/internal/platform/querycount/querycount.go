// Package querycount counts the SQL statements sent while serving one request, so that the request log shows
// a page whose statements grow with its records (N+1) as soon as it appears.
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

// Of returns what ctx's counter holds, 0 without one.
func Of(ctx context.Context) int64 {
	if counter, ok := ctx.Value(contextKey{}).(*atomic.Int64); ok {
		return counter.Load()
	}
	return 0
}
