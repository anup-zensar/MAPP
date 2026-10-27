package history

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"
)

// SnapshotsInBin gives the records of one kind that are in the recycle bin — deleted records and the
// records under them — oldest entry first, leaving out entries that were undone or restored, so that
// feature 002 can count them as still in use (FR-003, data model §10). It joins the caller's transaction.
func TestSnapshotsInBin_KindAndContents_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	f.put(t, childrenTable, newRecord("C-3", "Child of parent two", "P-2"))
	f.moveToBin(t, "child", "C-2") // with G-3
	f.clock.Advance(time.Minute)
	f.moveToBin(t, "parent", "P-2") // with C-3
	f.clock.Advance(time.Minute)
	undone := f.moveToBin(t, "child", "C-1") // with G-1 and G-2, then undone
	f.undo(t, priya, undone.ID)

	inBin := func(kind string) []Snapshot {
		t.Helper()
		snapshots, err := f.service.SnapshotsInBin(t.Context(), kind)
		if err != nil {
			t.Fatalf("SnapshotsInBin(%s): %v", kind, err)
		}
		return snapshots
	}

	want := map[string][]Snapshot{
		"parent":     {snapshotOf("parent", newRecord("P-2", "Parent two", ""))},
		"child":      {snapshotOf("child", newRecord("C-2", "Child two", "P-1")), snapshotOf("child", newRecord("C-3", "Child of parent two", "P-2"))},
		"grandchild": {snapshotOf("grandchild", newRecord("G-3", "Grandchild three", "C-2"))},
		"unknown":    {},
	}
	for kind, wantSnapshots := range want {
		if got := inBin(kind); !reflect.DeepEqual(got, wantSnapshots) {
			t.Errorf("SnapshotsInBin(%s) =\n%s\nwant\n%s", kind, asJSON(got), asJSON(wantSnapshots))
		}
	}

	// Inside a Write, a delete made earlier in the same Write is already in the bin.
	var joined []Snapshot
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		if _, err := f.service.MoveToBin(ctx, priya, "parent", "P-1"); err != nil {
			return err
		}
		var err error
		joined, err = f.service.SnapshotsInBin(ctx, "parent")
		return err
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	ids := []string{}
	for _, s := range joined {
		ids = append(ids, s.ID)
	}
	if !slices.Equal(ids, []string{"P-2", "P-1"}) {
		t.Errorf("SnapshotsInBin(parent) inside the Write = %v, want [P-2 P-1]", ids)
	}
}
