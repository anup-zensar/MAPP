package history

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
)

// A delete moves the record and everything under it into one recycle-bin entry with who deleted it and
// when, removes them (children first) and writes one log entry (FR-024). The entry counts what was under
// the record by kind, in the order the kinds were registered, and lists only kinds it holds.
func TestMoveToBin_WithChildren_FR024(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	attachments := f.testKind("attachment", "attachments", "parent", "attachments") // none under P-1
	f.register(t, append(f.standardKinds(), attachments)...)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	f.put(t, childrenTable, newRecord("C-3", "Child of parent two", "P-2"))
	f.clock.Advance(time.Hour)

	got, err := f.service.MoveToBin(t.Context(), priya, "parent", "P-1")
	if err != nil {
		t.Fatalf("MoveToBin: %v", err)
	}

	want := BinEntryView{
		ID: "BIN-1001", RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one",
		Path: []RecordRef{}, Parent: nil, DeletedAt: start.Add(time.Hour), DeletedBy: priya,
		Contents: []ContentCount{
			{RecordType: "child", Label: "children", Count: 2},
			{RecordType: "grandchild", Label: "grandchildren", Count: 3},
		},
	}
	if view := got.View(); !reflect.DeepEqual(view, want) {
		t.Errorf("MoveToBin =\n%s\nwant\n%s", asJSON(view), asJSON(want))
	}
	entry := f.binEntry(t, "BIN-1001")
	if !reflect.DeepEqual(entry, got) {
		t.Errorf("stored entry =\n%s\nwant it exactly as returned:\n%s", asJSON(entry), asJSON(got))
	}

	// The entry keeps every record as it was stored: the deleted one first, parents before children.
	wantSnapshots := []Snapshot{
		snapshotOf("parent", newRecord("P-1", "Parent one", "")),
		snapshotOf("child", newRecord("C-1", "Child one", "P-1")),
		snapshotOf("grandchild", newRecord("G-1", "Grandchild one", "C-1")),
		snapshotOf("grandchild", newRecord("G-2", "Grandchild two", "C-1")),
		snapshotOf("child", newRecord("C-2", "Child two", "P-1")),
		snapshotOf("grandchild", newRecord("G-3", "Grandchild three", "C-2")),
	}
	if !reflect.DeepEqual(entry.Snapshots, wantSnapshots) {
		t.Errorf("snapshots =\n%s\nwant\n%s", asJSON(entry.Snapshots), asJSON(wantSnapshots))
	}
	if entry.Resolution != nil {
		t.Errorf("resolution = %+v, want none while the entry is in the bin", entry.Resolution)
	}

	// The records are removed, children first; the other parent keeps its child.
	if want := []string{"G-3", "C-2", "G-2", "G-1", "C-1", "P-1"}; !slices.Equal(f.removed, want) {
		t.Errorf("removed %v, want %v", f.removed, want)
	}
	wantLeft := map[string][]testRecord{
		parentsTable:       {newRecord("P-2", "Parent two", "")},
		childrenTable:      {newRecord("C-3", "Child of parent two", "P-2")},
		grandchildrenTable: {},
	}
	if left := f.allRecords(t); !reflect.DeepEqual(left, wantLeft) {
		t.Errorf("stored records = %+v, want %+v", left, wantLeft)
	}

	// One log entry for the whole delete.
	assertEntries(t, f.entries(t), []Entry{{
		ID: "LOG-1001", Seq: 1001, At: start.Add(time.Hour), Actor: priya, Action: ActionDeleted, Verb: "deleted",
		RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one",
		Path: []RecordRef{}, Changes: []Change{}, Details: "with 2 children and 3 grandchildren",
	}})
}

// One record of a kind under the deleted one is counted in the singular: "with 1 grandchild". A record with
// a parent keeps where it was: its path and its parent (FR-024).
func TestMoveToBin_OneNoteSaysSingular_FR024(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)

	got := f.moveToBin(t, "child", "C-2") // C-2 holds one grandchild, G-3

	parentOneRef := RecordRef{Type: "parent", ID: "P-1", Name: "Parent one"}
	want := BinEntryView{
		ID: "BIN-1001", RecordType: "child", RecordTypeLabel: "child", RecordID: "C-2", RecordName: "Child two",
		Path: []RecordRef{parentOneRef}, Parent: &parentOneRef, DeletedAt: start, DeletedBy: priya,
		Contents: []ContentCount{{RecordType: "grandchild", Label: "grandchild", Count: 1}},
	}
	if view := got.View(); !reflect.DeepEqual(view, want) {
		t.Errorf("MoveToBin =\n%s\nwant\n%s", asJSON(view), asJSON(want))
	}
	entries := f.entries(t)
	if len(entries) != 1 || entries[0].Details != "with 1 grandchild" || !reflect.DeepEqual(entries[0].Path, []RecordRef{parentOneRef}) {
		t.Errorf("log =\n%s\nwant one entry with the details \"with 1 grandchild\" and the path [P-1]", asJSON(entries))
	}
}

// A delete, its recycle-bin entry and its log entry are saved together with the caller's change, or not at
// all: when the caller's Write fails, every record is still there and nothing was binned or logged (FR-024).
func TestMoveToBin_JoinsTheCallersWrite_FR024(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	before := f.allRecords(t)
	failed := errors.New("the caller's change failed")

	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		if _, err := f.service.MoveToBin(ctx, priya, "parent", "P-1"); err != nil {
			return err
		}
		return failed
	})

	if !errors.Is(err, failed) {
		t.Errorf("Write = %v, want the caller's error", err)
	}
	if after := f.allRecords(t); !reflect.DeepEqual(after, before) {
		t.Errorf("records = %+v, want them all still there: %+v", after, before)
	}
	if bin := f.listBin(t); len(bin) != 0 {
		t.Errorf("recycle bin = %s, want it empty", asJSON(bin))
	}
	if entries := f.entries(t); len(entries) != 0 {
		t.Errorf("log = %s, want it empty", asJSON(entries))
	}
}
