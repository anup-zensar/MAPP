package performance_test

import (
	"context"
	"testing"

	"mapp/internal/performance"
	"mapp/internal/platform/actor"
)

func TestListSources_SourceThatHasNeverLoaded_FR005(t *testing.T) {
	store, db := pgPerformanceDB(t)
	if err := db.Write(t.Context(), func(ctx context.Context) error {
		return store.PutSource(ctx, performance.Source{ID: "PS-9", Name: "Fresh", OwnerID: "U-2001",
			Category: performance.CategoryCRM, Status: performance.StatusActive,
			DeliveryRoute: performance.RouteUpload, Cadence: performance.CadenceMonthly, Meta: pgMeta})
	}); err != nil {
		t.Fatalf("PutSource: %v", err)
	}
	got, err := store.ListSources(t.Context(), "PS-9")
	if err != nil {
		t.Fatalf("ListSources() on a source with no completed import FAILED: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListSources() returned %d sources, want 1", len(got))
	}
	if !got[0].LastLoadedAt.IsZero() || !got[0].DataThrough.IsZero() {
		t.Errorf("a source that never loaded has LastLoadedAt=%v DataThrough=%v, want both zero",
			got[0].LastLoadedAt, got[0].DataThrough)
	}
}

// seed/seed.go is not this lane's to change, so taking a code must create the row (FR-042).
func TestTakeNextCode_OnAFreshDatabase_FR042(t *testing.T) {
	store, db := pgPerformanceDB(t)

	var first, second int64
	if err := db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		if first, err = store.TakeNextCode(ctx, actor.System, pgMeta.CreatedAt); err != nil {
			return err
		}
		second, err = store.TakeNextCode(ctx, actor.System, pgMeta.CreatedAt)
		return err
	}); err != nil {
		t.Fatalf("TakeNextCode on a fresh database: %v", err)
	}

	if first != 9000001 {
		t.Errorf("the first code is %d, want 9000001 so it cannot collide with Hive9's", first)
	}
	if second != first+1 {
		t.Errorf("the second code is %d, want %d: a number must never be reused", second, first+1)
	}
}
