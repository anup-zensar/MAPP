package store

import (
	"context"
	"fmt"

	"mapp/internal/platform/memo"
)

// Memo returns what load returns, worked out once per transaction for key: a later call with the same key
// in the same transaction (or one that joined it) answers the first result without calling load again.
// Every change in a Write (Put, Delete, Append) forgets all memos, so a memo never outlives the data it
// was worked out from; an error is not kept. Every caller in the transaction gets the same value, so
// callers must treat it as read-only. Keys start with their package, such as "settings:form:plan".
// Outside a transaction of this store, such as one on PostgreSQL, memo.Get keeps it.
func Memo[T any](ctx context.Context, key string, load func(ctx context.Context) (T, error)) (T, error) {
	t, ok := ctx.Value(txKey{}).(*tx)
	if !ok {
		return memo.Get(ctx, key, load)
	}
	return memoIn(ctx, t, key, load)
}

// memoIn is Memo in the transaction t, for any comparable key.
func memoIn[T any](ctx context.Context, t *tx, key any, load func(ctx context.Context) (T, error)) (T, error) {
	var zero T
	t.memoMu.Lock()
	cached, found := t.memo[key]
	t.memoMu.Unlock()
	if found {
		if value, ok := cached.(T); ok {
			return value, nil
		}
		return zero, fmt.Errorf("store: memo %v holds a %T, not the type asked for", key, cached)
	}
	value, err := load(ctx) // outside the lock: load may use other memos
	if err != nil {
		return zero, err
	}
	t.memoMu.Lock()
	if t.memo == nil {
		t.memo = map[any]any{}
	}
	t.memo[key] = value
	t.memoMu.Unlock()
	return value, nil
}

// forgetMemos drops every value Memo kept: the data they were worked out from is about to change.
func (t *tx) forgetMemos() {
	t.memoMu.Lock()
	t.memo = nil
	t.memoMu.Unlock()
}
