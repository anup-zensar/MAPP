package seed_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
	"mapp/internal/stores/storestest"
	"mapp/seed"
)

// The seed command runs at the computer's time, not on a whole minute: seconds must not push an entry out.
func TestSeed_History_SeedTimeWithSeconds_011(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 20, 7, 41, 38, 512_000_000, time.UTC)
	db := fake.NewDB()
	if err := seed.Load(t.Context(), storestest.New(db), func() time.Time { return at }); err != nil {
		t.Fatalf("Load: %v", err)
	}
	entries, _ := demoLog(t, db)
	for i, e := range entries {
		if e.At.Before(at.Add(-seed.HistoryDays*day)) || e.At.After(at) {
			t.Errorf("%s is not within the 14 days before %s", entryText(e), at.Format(time.RFC3339Nano))
		}
		if i > 0 && e.At.Before(entries[i-1].At) {
			t.Errorf("%s comes after %s, which is later", entryText(e), entryText(entries[i-1]))
		}
	}
}

// Without its history the log and the bin start empty and the deleted initiative takes no IDs.
func TestLoad_WithoutHistory_011(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	if err := seed.Load(t.Context(), storestest.New(db), fixedClock, seed.WithoutHistory()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	entries, bin := demoLog(t, db)
	if len(entries) != 0 || len(bin) != 0 {
		t.Errorf("%d log entries and %d recycle-bin entries, want none", len(entries), len(bin))
	}
	generator := ids.NewGenerator(fake.NewCounters(db))
	err := db.Write(t.Context(), func(ctx context.Context) error {
		for prefix, want := range map[string]string{ids.PrefixInitiative: "I-1011", ids.PrefixActivity: "A-1013",
			ids.PrefixLineItem: "L-1011", ids.PrefixLogEntry: "LOG-1001", ids.PrefixBinEntry: "BIN-1001"} {
			if next, err := generator.Next(ctx, prefix); err != nil || next != want {
				t.Errorf("next %s ID = %s (%v), want %s", prefix, next, err, want)
			}
		}
		return errors.New("roll back") // leave the counters as the seed left them
	})
	if err == nil || err.Error() != "roll back" {
		t.Fatalf("Write: %v", err)
	}
}
