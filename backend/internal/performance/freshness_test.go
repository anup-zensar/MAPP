package performance_test

import (
	"context"
	"testing"
	"time"

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

func TestFreshnessAt_DerivedFromTheLastLoadAndTheCadence_FR005(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		status performance.Status
		cadenc performance.Cadence
		loaded time.Time
		want   performance.Freshness
	}{
		{name: "FR-005 a deactivated source is Inactive whatever it holds", status: performance.StatusInactive,
			cadenc: performance.CadenceDaily, loaded: now, want: performance.FreshnessInactive},
		{name: "FR-005 a source that has never loaded is Stale", status: performance.StatusActive,
			cadenc: performance.CadenceMonthly, want: performance.FreshnessStale},
		{name: "FR-005 loaded today on a daily cadence is Current", status: performance.StatusActive,
			cadenc: performance.CadenceDaily, loaded: now.Add(-time.Hour), want: performance.FreshnessCurrent},
		{name: "FR-005 three days silent on a daily cadence is Stale", status: performance.StatusActive,
			cadenc: performance.CadenceDaily, loaded: now.Add(-72 * time.Hour), want: performance.FreshnessStale},
		{name: "FR-005 a month silent on a monthly cadence is still Current", status: performance.StatusActive,
			cadenc: performance.CadenceMonthly, loaded: now.Add(-30 * 24 * time.Hour),
			want: performance.FreshnessCurrent},
		{name: "FR-005 an on-demand source is never late", status: performance.StatusActive,
			cadenc: performance.CadenceOnDemand, loaded: now.Add(-365 * 24 * time.Hour),
			want: performance.FreshnessCurrent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			view := performance.SourceView{
				Source:       performance.Source{Status: tt.status, Cadence: tt.cadenc},
				LastLoadedAt: tt.loaded,
			}

			if got := view.FreshnessAt(now); got != tt.want {
				t.Errorf("FreshnessAt() = %s, want %s", got, tt.want)
			}
		})
	}
}
