package performance_test

import (
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/performance"
	"mapp/internal/performance/performancetest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
)

var (
	alex = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	noon = time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)
)

type fixture struct {
	store   *performancetest.Store
	service *performance.Service
	history *history.Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := fake.NewDB()
	now := func() time.Time { return noon }
	generator := ids.NewGenerator(fake.NewCounters(db))
	f := &fixture{store: performancetest.NewStore(db)}
	f.history = history.NewService(db, historytest.NewStore(db), generator, now)
	f.service = performance.NewService(
		history.Tools{Tx: db, IDs: generator, Log: f.history, Now: now}, f.store)
	for _, k := range f.service.Kinds() {
		if err := f.history.Register(k); err != nil {
			t.Fatalf("Register: %v", err)
		}
	}
	return f
}

func (f *fixture) createSource(t *testing.T, name string) performance.Source {
	t.Helper()
	src, err := f.service.CreateSource(t.Context(), alex, performance.SourceInput{
		Name: name, OwnerID: alex.ID, Category: performance.CategoryMarketingAutomation,
		DeliveryRoute: performance.RouteUpload, Cadence: performance.CadenceMonthly,
	})
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}
	return src
}

func (f *fixture) log(t *testing.T) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), alex, history.LogFilter{Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return entries
}
