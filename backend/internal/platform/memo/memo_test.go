package memo_test

import (
	"context"
	"errors"
	"testing"

	"mapp/internal/platform/memo"
)

func TestGet_OncePerContextOfWith(t *testing.T) {
	t.Parallel()
	loads := 0
	count := func(context.Context) (int, error) {
		loads++
		return loads, nil
	}
	ctx := memo.With(t.Context())
	for range 3 {
		if n, err := memo.Get(ctx, "test:count", count); err != nil || n != 1 {
			t.Errorf("Get() = %d, %v; want 1 (the first load), nil", n, err)
		}
	}
	if n, err := memo.Get(memo.With(t.Context()), "test:count", count); err != nil || n != 2 {
		t.Errorf("Get() in a new context = %d, %v; want 2", n, err)
	}
}

func TestGet_WithoutWithLoadsEveryTime(t *testing.T) {
	t.Parallel()
	loads := 0
	for range 2 {
		_, _ = memo.Get(t.Context(), "test:count", func(context.Context) (int, error) { loads++; return loads, nil })
	}
	if loads != 2 {
		t.Errorf("loads = %d, want 2", loads)
	}
}

func TestGet_KeepsNoErrorAndRefusesAnotherType(t *testing.T) {
	t.Parallel()
	ctx := memo.With(t.Context())
	failing := errors.New("not now")
	if _, err := memo.Get(ctx, "test:value", func(context.Context) (int, error) { return 0, failing }); !errors.Is(err, failing) {
		t.Errorf("Get() error = %v, want the load's error", err)
	}
	if n, err := memo.Get(ctx, "test:value", func(context.Context) (int, error) { return 7, nil }); err != nil || n != 7 {
		t.Errorf("Get() after a failed load = %d, %v; want 7, nil", n, err)
	}
	if _, err := memo.Get(ctx, "test:value", func(context.Context) (string, error) { return "", nil }); err == nil {
		t.Error("Get() as another type: no error, want one")
	}
}
