package memo_test

import (
	"context"
	"testing"

	"mapp/internal/platform/memo"
)

type row struct{ id, name string }

// A Settings table is read once per transaction and found by id after that (028 FR-005).
func TestGetTable_ReadOnceFindByID_028_FR005(t *testing.T) {
	t.Parallel()
	ctx := memo.With(t.Context())
	reads := 0
	read := func(context.Context) ([]row, error) {
		reads++
		return []row{{"AT-1", "Event"}, {"AT-2", "Digital media"}}, nil
	}
	idOf := func(r row) string { return r.id }

	for range 2 {
		table, err := memo.GetTable(ctx, "settings:types", read, idOf)
		if err != nil {
			t.Fatalf("GetTable: %v", err)
		}
		if found, ok := table.Find("AT-2"); !ok || found.name != "Digital media" {
			t.Errorf("Find(AT-2) = %v, %v; want Digital media, true", found, ok)
		}
		if _, ok := table.Find("AT-9"); ok {
			t.Error("Find(AT-9) found a row, want none")
		}
	}
	if reads != 1 {
		t.Errorf("reads = %d, want 1", reads)
	}
}
