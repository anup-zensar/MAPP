package memo_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/memo"
)

// A store forgets its own area when it changes data, and what other areas keep stays (028 FR-006).
func TestForget_OnlyItsArea_028_FR006(t *testing.T) {
	t.Parallel()
	ctx := memo.With(t.Context())
	memo.Keep(ctx, "plans:P-1", "kept plan")
	memo.Keep(ctx, "people:U-1", "kept person")

	memo.Forget(ctx, "plans")

	loads := 0
	load := func(context.Context) (string, error) { loads++; return "read again", nil }
	plan, _ := memo.Get(ctx, "plans:P-1", load)
	person, _ := memo.Get(ctx, "people:U-1", load)
	if plan != "read again" || person != "kept person" || loads != 1 {
		t.Errorf("after Forget(plans): plan %q, person %q, %d loads; want read again, kept person, 1", plan, person, loads)
	}
}

// Many reads only the keys not kept yet, each once, and keeps a key the read did not find as none, so it is not
// read again (028 FR-005).
func TestMany_ReadsOnlyWhatIsMissing_028_FR005(t *testing.T) {
	t.Parallel()
	ctx := memo.With(t.Context())
	var asked [][]string
	read := func(_ context.Context, missing []string) (map[string]int, error) {
		asked = append(asked, missing)
		found := map[string]int{}
		for _, key := range missing {
			if key != "gone" {
				found[key] = len(key)
			}
		}
		return found, nil
	}

	first, err := memo.Many(ctx, "test:", []string{"a", "bb", "a", "gone"}, read)
	if err != nil {
		t.Fatalf("Many: %v", err)
	}
	second, err := memo.Many(ctx, "test:", []string{"bb", "gone", "ccc"}, read)
	if err != nil {
		t.Fatalf("Many: %v", err)
	}

	if diff := cmp.Diff([][]string{{"a", "bb", "gone"}, {"ccc"}}, asked); diff != "" {
		t.Errorf("keys read (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(map[string]int{"a": 1, "bb": 2, "gone": 0}, first); diff != "" {
		t.Errorf("first answer (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(map[string]int{"bb": 2, "gone": 0, "ccc": 3}, second); diff != "" {
		t.Errorf("second answer (-want +got):\n%s", diff)
	}
}

// Without a transaction's memo every call reads, so nothing outlives the transaction (028 FR-005).
func TestMany_WithoutWithReadsEveryTime_028_FR005(t *testing.T) {
	t.Parallel()
	reads := 0
	read := func(_ context.Context, missing []string) (map[string]int, error) {
		reads++
		return map[string]int{}, nil
	}
	for range 2 {
		if _, err := memo.Many(t.Context(), "test:", []string{"a"}, read); err != nil {
			t.Fatalf("Many: %v", err)
		}
	}
	if reads != 2 {
		t.Errorf("reads = %d, want 2", reads)
	}
}
