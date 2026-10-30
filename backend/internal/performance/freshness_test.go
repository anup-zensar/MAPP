package performance_test

import (
	"context"
	"testing"

	"mapp/internal/performance"
)

// A source that has never loaded is the normal case for every source US1 creates: its freshness is NULL
// in SQL, and reading it must not fail (FR-005).
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
