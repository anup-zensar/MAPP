package ids_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/ids"
)

func next(t *testing.T, db *database.DB, generator *ids.Generator, prefix string) string {
	t.Helper()
	var id string
	err := db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		id, err = generator.Next(ctx, prefix)
		return err
	})
	if err != nil {
		t.Fatalf("Next(%q) error: %v", prefix, err)
	}
	return id
}

func TestPostgresCountersStartAt1001PerPrefixFR027(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	generator := ids.NewGenerator(ids.NewPostgresCounters(db))

	got := []string{
		next(t, db, generator, ids.PrefixPlan),
		next(t, db, generator, ids.PrefixPlan),
		next(t, db, generator, ids.PrefixCampaign),
	}

	if diff := cmp.Diff([]string{"P-1001", "P-1002", "C-1001"}, got); diff != "" {
		t.Errorf("Next() mismatch (-want +got):\n%s", diff)
	}
}

func TestPostgresCountersNeverRepeatANumberWhenSavesRaceFR027(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	generator := ids.NewGenerator(ids.NewPostgresCounters(db))
	const writers, perWriter = 2, 15

	var saves sync.WaitGroup
	numbers := make([][]int, writers)
	failures := make([]error, writers)
	for w := range writers {
		saves.Go(func() {
			for range perWriter {
				var number int
				err := db.Write(t.Context(), func(ctx context.Context) error {
					var err error
					number, err = generator.NextNumber(ctx, ids.PrefixLogEntry)
					return err
				})
				numbers[w] = append(numbers[w], number)
				failures[w] = errors.Join(failures[w], err)
			}
		})
	}
	saves.Wait()

	if err := errors.Join(failures...); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	got := slices.Sorted(slices.Values(slices.Concat(numbers...)))
	want := make([]int, 0, writers*perWriter)
	for n := 1001; n < 1001+writers*perWriter; n++ {
		want = append(want, n)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("numbers given out mismatch (-want +got):\n%s", diff)
	}
}
