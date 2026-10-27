package history

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/errs"
)

// Each entry of the bin says whether it can come back (011 FR-017, data model §5): its parent is stored or
// it has none (restorable); its parent is in another entry (parent in the recycle bin, with that entry to
// restore first); or its parent is gone for good (parent deleted permanently, naming it). Newest first.
func TestListBin_RestoreStates_FR017(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	topLevel := f.moveToBin(t, "parent", "P-2")
	underStored := f.moveToBin(t, "grandchild", "G-3") // C-2 is stored
	waiting := f.moveToBin(t, "grandchild", "G-1")
	parentEntry := f.moveToBin(t, "child", "C-1") // holds G-2
	f.put(t, childrenTable, newRecord("C-5", "Child five", "P-1"))
	f.put(t, grandchildrenTable, newRecord("G-5", "Grandchild five", "C-5"))
	orphan := f.moveToBin(t, "grandchild", "G-5")
	f.removeForGood(t, childrenTable, "C-5") // gone for good: not stored and in no entry

	items := f.listBinItems(t)

	type state struct {
		ID             string
		RestoreState   string
		MissingParent  *LabelledRef
		ParentEntryIDs []string
	}
	var got []state
	for _, item := range items {
		got = append(got, state{item.ID, item.RestoreState, item.MissingParent, item.ParentEntryIDs})
	}
	want := []state{
		{orphan.ID, ParentDeletedPermanently, &LabelledRef{Type: "child", TypeLabel: "child", ID: "C-5", Name: "Child five"}, []string{}},
		{parentEntry.ID, Restorable, nil, []string{}},
		{waiting.ID, ParentInRecycleBin, &LabelledRef{Type: "child", TypeLabel: "child", ID: "C-1", Name: "Child one"}, []string{parentEntry.ID}},
		{underStored.ID, Restorable, nil, []string{}},
		{topLevel.ID, Restorable, nil, []string{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("restore states =\n%s\nwant\n%s", asJSON(got), asJSON(want))
	}
}

// A row carries the records its entry holds, the record first and then everything inside it, parents
// first (011 FR-015, FR-016), the plural of its kind for the type pills, and the other entries that can't
// be restored once it is deleted permanently (FR-028).
func TestListBin_RecordsAndInside_FR015_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	inside := f.moveToBin(t, "grandchild", "G-1")
	outer := f.moveToBin(t, "child", "C-1") // holds G-2

	items := f.listBinItems(t)

	if len(items) != 2 || items[0].ID != outer.ID || items[1].ID != inside.ID {
		t.Fatalf("bin = %s, want %s then %s", asJSON(items), outer.ID, inside.ID)
	}
	wantOuter := BinListItem{
		BinEntryView:     outer.View(),
		RecordTypePlural: "children",
		RestoreState:     Restorable,
		ParentEntryIDs:   []string{},
		InsideEntryIDs:   []string{inside.ID},
		Records: []BinRecord{
			{RecordType: "child", RecordTypeLabel: "child", RecordID: "C-1", RecordName: "Child one", ParentID: "P-1"},
			{RecordType: "grandchild", RecordTypeLabel: "grandchild", RecordID: "G-2", RecordName: "Grandchild two", ParentID: "C-1"},
		},
	}
	if !reflect.DeepEqual(items[0], wantOuter) {
		t.Errorf("row of C-1 =\n%s\nwant\n%s", asJSON(items[0]), asJSON(wantOuter))
	}
	wantInside := []BinRecord{{RecordType: "grandchild", RecordTypeLabel: "grandchild", RecordID: "G-1", RecordName: "Grandchild one", ParentID: "C-1"}}
	if !reflect.DeepEqual(items[1].Records, wantInside) || len(items[1].InsideEntryIDs) != 0 || items[1].RecordTypePlural != "grandchildren" {
		t.Errorf("row of G-1 = %s, want its one record, no entries inside and the plural grandchildren", asJSON(items[1]))
	}
}

// An entry holding a kind this API does not know is not listed: it could not be restored here (D-186).
func TestListBin_UnregisteredKindHidden(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	other := f.withKind(t, "attachment")
	f.put(t, "others", newRecord("X-1", "Brief", "P-1"))
	if _, err := other.MoveToBin(t.Context(), priya, "attachment", "X-1"); err != nil {
		t.Fatalf("MoveToBin: %v", err)
	}
	known := f.moveToBin(t, "grandchild", "G-1")

	items := f.listBinItems(t)

	if len(items) != 1 || items[0].ID != known.ID {
		t.Errorf("bin = %s, want only %s", asJSON(items), known.ID)
	}
	if all, err := other.ListBinItems(t.Context(), priya); err != nil || len(all) != 2 {
		t.Errorf("a service that knows the kind lists %d entries (%v), want 2", len(all), err)
	}
}

// With its parent and grandparent in the bin, an entry lists both entries to restore first, the top-most
// first, and each of them lists the entries waiting for it (011 FR-023, FR-028).
func TestListBin_ChainParentsFirst_FR023(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	grandchild, child, parent := f.binChain(t)

	items := f.listBinItems(t)

	type chain struct {
		ID, RestoreState, MissingParent string
		ParentEntryIDs, InsideEntryIDs  []string
	}
	var got []chain
	for _, item := range items {
		missing := ""
		if item.MissingParent != nil {
			missing = item.MissingParent.ID
		}
		got = append(got, chain{item.ID, item.RestoreState, missing, item.ParentEntryIDs, item.InsideEntryIDs})
	}
	want := []chain{
		{parent.ID, Restorable, "", []string{}, []string{child.ID, grandchild.ID}},
		{child.ID, ParentInRecycleBin, "P-1", []string{parent.ID}, []string{grandchild.ID}},
		{grandchild.ID, ParentInRecycleBin, "C-1", []string{parent.ID, child.ID}, []string{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bin =\n%s\nwant\n%s", asJSON(got), asJSON(want))
	}
}

// Restore without its parents refuses an entry whose parent is in the bin, naming the parent to restore
// first (011 FR-023); nothing changes.
func TestRestore_ParentInBin_RestoreParentFirst_FR023(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	grandchild, _, _ := f.binChain(t)
	records, log := f.allRecords(t), f.entries(t)

	_, err := f.service.Restore(t.Context(), priya, grandchild.ID)

	assertError(t, err, errs.RestoreParentFirst(RecordRef{Type: "child", ID: "C-1", Name: "Child one"}))
	if !reflect.DeepEqual(f.allRecords(t), records) || len(f.entries(t)) != len(log) {
		t.Error("the refused restore changed records or the log")
	}
}

// Restore all 3 brings back the grandparent's entry, then the parent's, then the chosen one, in one step,
// each with its own "restored" log entry by the person who restored (011 FR-023, FR-026).
func TestRestoreWithParents_ParentsFirstOneStep_FR023(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	before := f.allRecords(t)
	grandchild, child, parent := f.binChain(t)
	logBefore := len(f.entries(t))
	f.clock.Advance(time.Hour)

	got, err := f.service.RestoreWithParents(t.Context(), daniel, grandchild.ID)

	want := RestoreResult{RecordType: "grandchild", RecordTypeLabel: "grandchild", RecordID: "G-1", RecordName: "Grandchild one"}
	if err != nil || got != want {
		t.Fatalf("RestoreWithParents = %+v, %v; want %+v", got, err, want)
	}
	if records := f.allRecords(t); !reflect.DeepEqual(records, before) {
		t.Errorf("records = %s, want everything back: %s", asJSON(records), asJSON(before))
	}
	for _, e := range []BinEntry{parent, child, grandchild} {
		resolution := f.binEntry(t, e.ID).Resolution
		if want := (Resolution{How: "restored", By: daniel, At: start.Add(time.Hour)}); resolution == nil || *resolution != want {
			t.Errorf("%s resolution = %+v, want %+v", e.ID, resolution, want)
		}
	}
	var restored []string
	for _, e := range f.entries(t)[logBefore:] {
		restored = append(restored, string(e.Action)+" "+e.RecordID+" by "+e.Actor.Name)
	}
	wantLog := []string{"restored P-1 by Daniel Okoro", "restored C-1 by Daniel Okoro", "restored G-1 by Daniel Okoro"}
	if !slices.Equal(restored, wantLog) {
		t.Errorf("log = %q, want %q", restored, wantLog)
	}
	if again, err := f.service.RestoreWithParents(t.Context(), daniel, f.moveToBin(t, "grandchild", "G-3").ID); err != nil || again.RecordID != "G-3" {
		t.Errorf("with nothing missing: RestoreWithParents = %+v, %v; want G-3 back as Restore does", again, err)
	}
}

// When a record above it was deleted permanently, the entry can't be restored, with or without its
// parents: the refusal names that record, and nothing changes (011 FR-024).
func TestRestore_ParentPurged_Refused_FR024(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	grandchild, child, parent := f.binChain(t)
	if _, err := f.service.Purge(t.Context(), priya, parent.ID); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	records, log := f.allRecords(t), f.entries(t)
	refusal := errs.ParentDeletedPermanently("grandchild", RecordRef{Type: "parent", ID: "P-1", Name: "Parent one"}, "parent")

	_, withParents := f.service.RestoreWithParents(t.Context(), priya, grandchild.ID)
	_, alone := f.service.Restore(t.Context(), priya, grandchild.ID)

	assertError(t, withParents, refusal)
	assertError(t, alone, refusal)
	if !reflect.DeepEqual(f.allRecords(t), records) || len(f.entries(t)) != len(log) {
		t.Error("the refused restore changed records or the log")
	}
	for _, e := range []BinEntry{child, grandchild} {
		if entry := f.binEntry(t, e.ID); entry.Resolution != nil {
			t.Errorf("%s left the bin: %+v", e.ID, entry.Resolution)
		}
	}
	items := f.listBinItems(t)
	if len(items) != 2 || items[1].RestoreState != ParentDeletedPermanently || items[1].MissingParent.ID != "P-1" {
		t.Errorf("bin = %s, want G-1's entry blocked at P-1", asJSON(items))
	}
	if preview := f.preview(t, grandchild.ID); len(preview.Changes)+len(preview.ParentEntries)+len(preview.Destination)+len(preview.Notes) != 0 {
		t.Errorf("preview = %s, want empty lists: there is nothing to try", asJSON(preview))
	}
}

// binChain deletes the grandchild G-1, then its parent C-1 (with G-2), then the grandparent P-1 (with C-2
// and G-3), and returns the three entries.
func (f *fixture) binChain(t *testing.T) (grandchild, child, parent BinEntry) {
	t.Helper()
	grandchild = f.moveToBin(t, "grandchild", "G-1")
	child = f.moveToBin(t, "child", "C-1")
	parent = f.moveToBin(t, "parent", "P-1")
	return grandchild, child, parent
}
