package apitest

import (
	"context"
	"sync"
	"testing"
	"time"

	"mapp/internal/platform/fake"
	"mapp/internal/stores/storestest"
	"mapp/seed"
)

// demoData is loaded once per test binary: seed.Load takes a fifth of a second, a copy a few milliseconds.
type demoData struct {
	once    sync.Once
	db      *fake.DB
	failure error
}

var withHistory, withoutHistory demoData //nolint:gochecknoglobals // a read-only fixture of the test binary, as in _test.go files

// DemoDB returns a test's own copy of the demo data, loaded at the start of New's clock.
func DemoDB(tb testing.TB, noHistory bool) *fake.DB {
	tb.Helper()
	data, options := &withHistory, []seed.Option(nil)
	if noHistory {
		data, options = &withoutHistory, []seed.Option{seed.WithoutHistory()}
	}
	data.once.Do(func() { data.db, data.failure = loadDemoData(options) })
	if data.failure != nil {
		tb.Fatalf("apitest: loading the demo data: %v", data.failure)
	}
	return data.db.Copy()
}

func loadDemoData(options []seed.Option) (*fake.DB, error) {
	db := fake.NewDB()
	at := func() time.Time { return startTime }
	return db, seed.Load(context.Background(), storestest.New(db), at, options...)
}
