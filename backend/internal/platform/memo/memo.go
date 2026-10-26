// Package memo keeps what a read-only transaction works out from the data, such as Settings' forms, for the
// rest of that transaction: its snapshot cannot change, so neither can what was worked out from it.
package memo

import (
	"context"
	"fmt"
	"sync"
)

type table struct {
	mu     sync.Mutex
	values map[string]any
}

type contextKey struct{}

func With(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, &table{values: map[string]any{}})
}

// Get works load's value for name out once per context of With, keeping no error; without one, on every call.
func Get[T any](ctx context.Context, name string, load func(ctx context.Context) (T, error)) (T, error) {
	t, ok := ctx.Value(contextKey{}).(*table)
	if !ok {
		return load(ctx)
	}
	t.mu.Lock()
	cached, found := t.values[name]
	t.mu.Unlock()
	if found {
		value, isT := cached.(T)
		if !isT {
			return value, fmt.Errorf("memo: %s holds a %T, not the type asked for", name, cached)
		}
		return value, nil
	}
	value, err := load(ctx)
	if err != nil {
		return value, err
	}
	t.mu.Lock()
	t.values[name] = value
	t.mu.Unlock()
	return value, nil
}
