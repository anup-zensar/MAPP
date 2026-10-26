// Package memo keeps what one transaction read or worked out from the data for the rest of that transaction,
// by area: a store forgets its area as soon as it changes the data, so a save reads its own changes.
package memo

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type table struct {
	mu    sync.Mutex
	areas map[string]map[string]any // area → key → value; a name is "<area>:<key>"
}

type contextKey struct{}

func With(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, &table{areas: map[string]map[string]any{}})
}

// Get works load's value for name out once per context of With, keeping no error; without one, on every call.
func Get[T any](ctx context.Context, name string, load func(ctx context.Context) (T, error)) (T, error) {
	t, ok := ctx.Value(contextKey{}).(*table)
	if !ok {
		return load(ctx)
	}
	if cached, found := t.get(name); found {
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
	t.put(name, value)
	return value, nil
}

// Many returns the value of each key under prefix, keeping each once per context of With. One call of load
// reads the keys not kept yet; a key load leaves out is kept as T's zero value, which callers read as none.
func Many[T any](ctx context.Context, prefix string, keys []string,
	load func(ctx context.Context, missing []string) (map[string]T, error)) (map[string]T, error) {
	t, ok := ctx.Value(contextKey{}).(*table)
	if !ok {
		return load(ctx, keys)
	}
	found, missing, err := kept[T](t, prefix, keys)
	if err != nil || len(missing) == 0 {
		return found, err
	}
	loaded, err := load(ctx, missing)
	if err != nil {
		return nil, err
	}
	for _, key := range missing {
		found[key] = loaded[key]
		t.put(prefix+key, loaded[key])
	}
	return found, nil
}

// Keep keeps value under name for the rest of ctx's transaction, as Get would after loading it.
func Keep(ctx context.Context, name string, value any) {
	if t, ok := ctx.Value(contextKey{}).(*table); ok {
		t.put(name, value)
	}
}

// Forget drops everything kept under the area, the part of a name before its first colon.
func Forget(ctx context.Context, area string) {
	if t, ok := ctx.Value(contextKey{}).(*table); ok {
		t.mu.Lock()
		delete(t.areas, area)
		t.mu.Unlock()
	}
}

// kept returns the values t keeps of keys under prefix, and the keys it keeps none of, each once.
func kept[T any](t *table, prefix string, keys []string) (map[string]T, []string, error) {
	found := make(map[string]T, len(keys))
	var missing []string
	asked := make(map[string]bool, len(keys))
	for _, key := range keys {
		cached, isKept := t.get(prefix + key)
		value, isT := cached.(T)
		switch {
		case isKept && !isT:
			return nil, nil, fmt.Errorf("memo: %s%s holds a %T, not the type asked for", prefix, key, cached)
		case isKept:
			found[key] = value
		case !asked[key]:
			asked[key] = true
			missing = append(missing, key)
		}
	}
	return found, missing, nil
}

func (t *table) get(name string) (any, bool) {
	area, key := split(name)
	t.mu.Lock()
	defer t.mu.Unlock()
	value, found := t.areas[area][key]
	return value, found
}

func (t *table) put(name string, value any) {
	area, key := split(name)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.areas[area] == nil {
		t.areas[area] = map[string]any{}
	}
	t.areas[area][key] = value
}

func split(name string) (area, key string) {
	area, key, _ = strings.Cut(name, ":")
	return area, key
}
