package ids_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
)

// Many IDs taken at once are the IDs one call each would give, and the next ID follows them.
func TestNextIDs_SameAsOneByOne_028_FR004(t *testing.T) {
	db := fake.NewDB()
	generator := ids.NewGenerator(fake.NewCounters(db))

	var first string
	var batch, none []string
	var after string
	err := db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		if first, err = generator.Next(ctx, "A"); err != nil {
			return err
		}
		if batch, err = generator.NextIDs(ctx, "A", 3); err != nil {
			return err
		}
		if none, err = generator.NextIDs(ctx, "A", 0); err != nil {
			return err
		}
		after, err = generator.Next(ctx, "A")
		return err
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if diff := cmp.Diff([]string{"A-1002", "A-1003", "A-1004"}, batch); diff != "" {
		t.Errorf("NextIDs(3) after %s (-want +got):\n%s", first, diff)
	}
	if len(none) != 0 || after != "A-1005" {
		t.Errorf("NextIDs(0) = %v, then Next = %q; want none, then A-1005", none, after)
	}
}
