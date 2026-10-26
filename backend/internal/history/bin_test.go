package history

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/store"
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

// The preview names where the record returns: its parent's place now, not the place it had when it was
// deleted (spec edge case: the parent moved meanwhile).
func TestPreviewRestore_DestinationIsCurrentPlace_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	deleted := f.moveToBin(t, "grandchild", "G-1")
	f.put(t, childrenTable, newRecord("C-1", "Child one", "P-2")) // moved to P-2

	preview := f.preview(t, deleted.ID)

	want := []RecordRef{{Type: "parent", ID: "P-2", Name: "Parent two"}, {Type: "child", ID: "C-1", Name: "Child one"}}
	if !reflect.DeepEqual(preview.Destination, want) {
		t.Errorf("destination = %+v, want %+v", preview.Destination, want)
	}
	if top := f.preview(t, f.moveToBin(t, "parent", "P-2").ID); len(top.Destination) != 0 || top.Destination == nil {
		t.Errorf("destination of a top-level record = %+v, want an empty list", top.Destination)
	}
}

// The preview states every status change of the records that come back, as their kinds' restore rules
// make them, with the rules' notes (011 FR-018, FR-019): here an approved child and grandchild come back
// as created.
func TestPreviewRestore_StatusChanges_FR018_FR019(t *testing.T) {
	t.Parallel()
	f := newStatusFixture(t)
	deleted := f.moveToBin(t, "child", "C-1")

	preview := f.preview(t, deleted.ID)

	want := []RecordChange{
		{RecordType: "child", RecordTypeLabel: "child", RecordID: "C-1", RecordName: "Child one", Restored: true,
			Field: "status", Label: "Status", OldText: "approved", NewText: "created"},
		{RecordType: "grandchild", RecordTypeLabel: "grandchild", RecordID: "G-1", RecordName: "Grandchild one", Restored: true,
			Field: "status", Label: "Status", OldText: "approved", NewText: "created"},
		{RecordType: "grandchild", RecordTypeLabel: "grandchild", RecordID: "G-2", RecordName: "Grandchild two", Restored: true,
			Field: "status", Label: "Status", OldText: "approved", NewText: "created"},
	}
	if !reflect.DeepEqual(preview.Changes, want) {
		t.Errorf("changes =\n%s\nwant\n%s", asJSON(preview.Changes), asJSON(want))
	}
	wantNotes := []string{"'Child one' comes back as created.", "'Grandchild one' comes back as created.",
		"'Grandchild two' comes back as created."}
	if !slices.Equal(preview.Notes, wantNotes) {
		t.Errorf("notes = %q, want %q", preview.Notes, wantNotes)
	}
	if preview.Entry.ID != deleted.ID || preview.Entry.RestoreState != Restorable || len(preview.ParentEntries) != 0 {
		t.Errorf("entry = %s, parent entries = %s; want the restorable entry and none", asJSON(preview.Entry), asJSON(preview.ParentEntries))
	}
}

// When the name is taken meanwhile, the preview says the record comes back under a new name (011 FR-022).
func TestPreviewRestore_NameClash_FR022(t *testing.T) {
	t.Parallel()
	f := newStatusFixture(t)
	deleted := f.moveToBin(t, "child", "C-1")
	f.put(t, childrenTable, newRecord("C-9", "Child one", "P-1")) // takes the name

	preview := f.preview(t, deleted.ID)

	name := RecordChange{RecordType: "child", RecordTypeLabel: "child", RecordID: "C-1", RecordName: "Child one", Restored: true,
		Field: "name", Label: "Name", OldText: "Child one", NewText: "Child one (restored)"}
	if !slices.Contains(preview.Changes, name) {
		t.Errorf("changes = %s, want %s", asJSON(preview.Changes), asJSON(name))
	}
	if note := "Renamed to 'Child one (restored)' because the name was taken."; !slices.Contains(preview.Notes, note) {
		t.Errorf("notes = %q, want %q", preview.Notes, note)
	}
}

// A restore that changes a stored record, as a hook that sends the parent back to created, shows that
// change too, marked as not restored (011 FR-020): the preview reads what the hook logs.
func TestPreviewRestore_OutsideRecordChange_FR020(t *testing.T) {
	t.Parallel()
	f := newStatusFixture(t)
	deleted := f.moveToBin(t, "grandchild", "G-1") // its parent C-1 is approved and stays stored

	preview := f.preview(t, deleted.ID)

	outside := RecordChange{RecordType: "child", RecordTypeLabel: "child", RecordID: "C-1", RecordName: "Child one", Restored: false,
		Field: "status", Label: "Status", OldText: "approved", NewText: "created"}
	if !slices.Contains(preview.Changes, outside) {
		t.Errorf("changes = %s, want %s", asJSON(preview.Changes), asJSON(outside))
	}
	if c1, _ := f.stored(t, childrenTable, "C-1"); c1.Status != "approved" {
		t.Errorf("C-1 status = %q after the preview, want still approved", c1.Status)
	}
}

// The preview changes nothing (011 FR-018): the records, the bin, the log and the ID counters stay as
// they were, so the real restore that follows gets the next numbers.
func TestPreviewRestore_ChangesNothing_FR018(t *testing.T) {
	t.Parallel()
	f := newStatusFixture(t)
	f.moveToBin(t, "grandchild", "G-1")
	deleted := f.moveToBin(t, "child", "C-1")
	records, log, entry := f.allRecords(t), f.entries(t), f.binEntry(t, deleted.ID)

	f.preview(t, deleted.ID)

	if got := f.allRecords(t); !reflect.DeepEqual(got, records) {
		t.Errorf("records = %s, want unchanged %s", asJSON(got), asJSON(records))
	}
	if got := f.entries(t); !reflect.DeepEqual(got, log) {
		t.Errorf("log = %s, want unchanged", asJSON(got))
	}
	if got := f.binEntry(t, deleted.ID); !reflect.DeepEqual(got, entry) {
		t.Errorf("bin entry = %s, want unchanged %s", asJSON(got), asJSON(entry))
	}
	f.restore(t, priya, deleted.ID)
	if last, want := f.lastEntry(t), log[len(log)-1].Seq+1; last.Seq != want {
		t.Errorf("the restore's entry is %s, want seq %d: the preview used no number", last.ID, want)
	}
}

// A restored record comes back with the same IDs and values, everything inside it too (011 FR-021).
func TestRestore_SameIDsAndValues_FR021(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	before := f.allRecords(t)
	deleted := f.moveToBin(t, "child", "C-1")
	f.clock.Advance(30 * 24 * time.Hour)

	f.restore(t, daniel, deleted.ID)

	if got := f.allRecords(t); !reflect.DeepEqual(got, before) {
		t.Errorf("records = %s, want exactly %s", asJSON(got), asJSON(before))
	}
}

// Restore is never blocked by another rule (011 FR-025): long after the Undo window, by someone else
// than the person who deleted it.
func TestRestore_NotBlockedByOtherRules_FR025(t *testing.T) {
	t.Parallel()
	f := newStatusFixture(t)
	deleted := f.moveToBin(t, "child", "C-1")
	f.clock.Advance(400 * 24 * time.Hour)
	if _, err := f.service.Undo(t.Context(), daniel, deleted.ID); err == nil {
		t.Fatal("Undo after 400 days succeeded, want it refused")
	}

	got, err := f.service.Restore(t.Context(), daniel, deleted.ID)

	if err != nil || got.RecordID != "C-1" {
		t.Fatalf("Restore = %+v, %v; want C-1 back", got, err)
	}
	if bin := f.listBinItems(t); len(bin) != 0 {
		t.Errorf("bin = %s, want empty", asJSON(bin))
	}
}

// The restore is logged with who restored it, the status change of its record and the rules' notes (011
// FR-019, FR-026).
func TestRestore_LogsActorAndStatusChanges_FR019_FR026(t *testing.T) {
	t.Parallel()
	f := newStatusFixture(t)
	deleted := f.moveToBin(t, "child", "C-1")
	f.clock.Advance(time.Hour)

	f.restore(t, daniel, deleted.ID)

	last := f.lastEntry(t)
	want := Entry{ID: last.ID, Seq: last.Seq, At: start.Add(time.Hour), Actor: daniel, Action: ActionRestored, Verb: "restored",
		RecordType: "child", RecordTypeLabel: "child", RecordID: "C-1", RecordName: "Child one",
		Path: []RecordRef{{Type: "parent", ID: "P-1", Name: "Parent one"}},
		Changes: []Change{{Field: "status", Label: "Status", Old: raw(`"approved"`), New: raw(`"created"`),
			OldText: "approved", NewText: "created"}},
		Details: "Restored from the recycle bin, with 2 grandchildren. 'Child one' comes back as created. " +
			"'Grandchild one' comes back as created. 'Grandchild two' comes back as created.",
	}
	assertEntries(t, []Entry{last}, []Entry{want})
}

// Undo still puts everything back exactly, statuses included (011 FR-027): the restore rules apply only to
// a restore from the bin.
func TestUndo_StillExact_FR027(t *testing.T) {
	t.Parallel()
	f := newStatusFixture(t)
	before := f.allRecords(t)
	deleted := f.moveToBin(t, "child", "C-1")

	f.undo(t, priya, deleted.ID)

	if got := f.allRecords(t); !reflect.DeepEqual(got, before) {
		t.Errorf("records = %s, want exactly %s", asJSON(got), asJSON(before))
	}
}

// Nothing leaves the recycle bin on its own (011 FR-031): an entry is still there after 400 days.
func TestBin_NothingLeavesAutomatically_FR031(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	deleted := f.moveToBin(t, "child", "C-1")

	f.clock.Advance(400 * 24 * time.Hour)

	if items := f.listBinItems(t); len(items) != 1 || items[0].ID != deleted.ID || items[0].RestoreState != Restorable {
		t.Errorf("bin = %s, want %s, restorable", asJSON(items), deleted.ID)
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

// Delete permanently removes the records for good (011 FR-030): the entry leaves the bin with who did it
// and when and keeps no records, the records stay out of storage, and one "permanently deleted" entry names
// the record, its place when it was deleted and what was inside it. The entries about the records stay,
// now saying they are gone for good.
func TestPurge_RemovesForGoodAndLogs_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.logAs(t, priya, childOne(ActionChanged))
	deleted := f.moveToBin(t, "child", "C-1")
	f.put(t, parentsTable, newRecord("P-1", "Parent renamed", "")) // the path stays the one of the delete
	f.clock.Advance(time.Hour)

	got, err := f.service.Purge(t.Context(), daniel, deleted.ID)

	if want := (PurgeResult{RecordType: "child", RecordTypeLabel: "child", RecordID: "C-1", RecordName: "Child one"}); err != nil || got != want {
		t.Fatalf("Purge = %+v, %v; want %+v", got, err, want)
	}
	entry := f.binEntry(t, deleted.ID)
	if want := (Resolution{How: "deletedPermanently", By: daniel, At: start.Add(time.Hour)}); entry.Resolution == nil || *entry.Resolution != want || len(entry.Snapshots) != 0 {
		t.Errorf("entry = %s, want resolved %+v without records", asJSON(entry), want)
	}
	for _, id := range []string{"C-1", "G-1", "G-2"} {
		_, child := f.stored(t, childrenTable, id)
		_, grandchild := f.stored(t, grandchildrenTable, id)
		if child || grandchild {
			t.Errorf("%s is stored again", id)
		}
	}
	last := f.lastEntry(t)
	want := Entry{ID: last.ID, Seq: last.Seq, At: start.Add(time.Hour), Actor: daniel, Action: ActionDeletedPermanently,
		Verb: "permanently deleted", RecordType: "child", RecordTypeLabel: "child", RecordID: "C-1", RecordName: "Child one",
		Path: []RecordRef{{Type: "parent", ID: "P-1", Name: "Parent one"}}, Changes: []Change{}, Details: "with 2 grandchildren"}
	assertEntries(t, []Entry{last}, []Entry{want})
	page := f.query(t, LogQuery{RecordID: "C-1", Limit: 50})
	if page.Total != 3 {
		t.Errorf("entries about C-1 = %d, want 3: changed, deleted, permanently deleted", page.Total)
	}
	for _, item := range page.Items {
		if item.RecordNow.State != RecordDeletedPermanently {
			t.Errorf("%s: record state %q, want deletedPermanently", item.ID, item.RecordNow.State)
		}
	}
	if next := f.moveToBin(t, "child", "C-2"); next.ID == deleted.ID {
		t.Errorf("the next delete got the id %s again", next.ID)
	}
}

// Once an entry is deleted permanently, an entry waiting for it stays in the bin but can no longer be
// restored, naming the record that is gone (011 FR-028, D-181).
func TestPurge_InsideEntriesBecomeUnrestorable_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	inside := f.moveToBin(t, "grandchild", "G-1")
	outer := f.moveToBin(t, "child", "C-1")

	if _, err := f.service.Purge(t.Context(), priya, outer.ID); err != nil {
		t.Fatalf("Purge: %v", err)
	}

	items := f.listBinItems(t)
	want := &LabelledRef{Type: "child", TypeLabel: "child", ID: "C-1", Name: "Child one"}
	if len(items) != 1 || items[0].ID != inside.ID || items[0].RestoreState != ParentDeletedPermanently || !reflect.DeepEqual(items[0].MissingParent, want) {
		t.Errorf("bin = %s, want %s blocked at C-1", asJSON(items), inside.ID)
	}
}

// Empty recycle bin deletes every listed entry permanently in one step and writes one entry that lists
// them, with what was inside each (011 FR-029, FR-030).
func TestEmptyBin_OneEntryWithItems_FR029_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	first := f.moveToBin(t, "grandchild", "G-1")
	second := f.moveToBin(t, "child", "C-2") // with G-3
	f.clock.Advance(time.Hour)

	got, err := f.service.EmptyBin(t.Context(), daniel, []string{second.ID, first.ID})

	if err != nil || got.Count != 2 {
		t.Fatalf("EmptyBin = %+v, %v; want 2", got, err)
	}
	if items := f.listBinItems(t); len(items) != 0 {
		t.Errorf("bin = %s, want empty", asJSON(items))
	}
	for _, e := range []BinEntry{first, second} {
		if resolution := f.binEntry(t, e.ID).Resolution; resolution == nil || resolution.How != "deletedPermanently" || resolution.By != daniel {
			t.Errorf("%s resolution = %+v, want deleted permanently by Daniel", e.ID, resolution)
		}
	}
	last := f.lastEntry(t)
	want := Entry{ID: last.ID, Seq: last.Seq, At: start.Add(time.Hour), Actor: daniel, Action: ActionEmptiedRecycleBin,
		Verb: "emptied the recycle bin", RecordType: "recycleBin", RecordTypeLabel: "recycle bin", RecordName: "Recycle bin",
		Path: []RecordRef{}, Changes: []Change{}, Details: "2 items", Items: []LogItem{
			{RecordType: "child", RecordTypeLabel: "child", RecordID: "C-2", RecordName: "Child two", Details: "with 1 grandchild"},
			{RecordType: "grandchild", RecordTypeLabel: "grandchild", RecordID: "G-1", RecordName: "Grandchild one", Details: ""},
		}}
	assertEntries(t, []Entry{last}, []Entry{want})
	if log := f.entries(t); log[len(log)-2].Action != ActionDeleted {
		t.Errorf("entry before the last = %s, want the delete of C-2: one entry for the whole Empty", log[len(log)-2].Action)
	}
}

// Empty recycle bin leaves alone what someone else restored or deleted meanwhile and what the person did
// not see; an id listed twice counts once (011 FR-029, FR-032, D-180).
func TestEmptyBin_SkipsResolvedAndUnlisted_FR032(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	restored := f.moveToBin(t, "grandchild", "G-1")
	listed := f.moveToBin(t, "grandchild", "G-2")
	unlisted := f.moveToBin(t, "grandchild", "G-3")
	f.restore(t, daniel, restored.ID)

	got, err := f.service.EmptyBin(t.Context(), priya, []string{restored.ID, listed.ID, listed.ID, "BIN-9999"})

	if err != nil || got.Count != 1 {
		t.Fatalf("EmptyBin = %+v, %v; want 1", got, err)
	}
	if items := f.listBinItems(t); len(items) != 1 || items[0].ID != unlisted.ID {
		t.Errorf("bin = %s, want only %s", asJSON(items), unlisted.ID)
	}
	if resolution := f.binEntry(t, restored.ID).Resolution; resolution == nil || resolution.How != "restored" {
		t.Errorf("%s resolution = %+v, want still restored", restored.ID, resolution)
	}
	if last := f.lastEntry(t); last.Details != "1 item" || len(last.Items) != 1 || last.Items[0].RecordID != "G-2" {
		t.Errorf("entry = %s, want 1 item: G-2", asJSON(last))
	}
}

// When nothing listed is left in the bin, Empty deletes nothing and writes no entry; it needs at least one
// id (011 FR-029, FR-032).
func TestEmptyBin_NothingLeft_NoEntry(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	deleted := f.moveToBin(t, "grandchild", "G-1")
	if _, err := f.service.Purge(t.Context(), daniel, deleted.ID); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	logBefore := len(f.entries(t))

	got, err := f.service.EmptyBin(t.Context(), priya, []string{deleted.ID})
	_, none := f.service.EmptyBin(t.Context(), priya, nil)

	if err != nil || got.Count != 0 {
		t.Errorf("EmptyBin = %+v, %v; want 0", got, err)
	}
	if after := len(f.entries(t)); after != logBefore {
		t.Errorf("%d log entries, want still %d", after, logBefore)
	}
	assertError(t, none, errs.Validation(errs.FieldError{Field: "entryIds", Message: "Reload the page and try again."}))
}

// An action on an entry someone else already restored or deleted permanently is refused with how, by whom
// and when (011 FR-032): restore and preview after a purge, purge after a restore.
func TestActions_AlreadyResolved_FR032(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	purged := f.moveToBin(t, "grandchild", "G-1")
	restored := f.moveToBin(t, "grandchild", "G-2")
	f.clock.Advance(time.Hour)
	if _, err := f.service.Purge(t.Context(), daniel, purged.ID); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	f.restore(t, mei, restored.ID)
	at := start.Add(time.Hour)

	_, restoreErr := f.service.Restore(t.Context(), priya, purged.ID)
	_, withParentsErr := f.service.RestoreWithParents(t.Context(), priya, purged.ID)
	_, previewErr := f.service.PreviewRestore(t.Context(), priya, purged.ID)
	_, purgeErr := f.service.Purge(t.Context(), priya, restored.ID)
	_, unknownErr := f.service.Purge(t.Context(), priya, "BIN-9999")

	byDaniel := errs.AlreadyResolved("Grandchild one", Resolution{How: "deletedPermanently", By: daniel, At: at})
	assertError(t, restoreErr, byDaniel)
	assertError(t, withParentsErr, byDaniel)
	assertError(t, previewErr, byDaniel)
	assertError(t, purgeErr, errs.AlreadyResolved("Grandchild two", Resolution{How: "restored", By: mei, At: at}))
	assertError(t, unknownErr, errs.NotFound("This item is not in the recycle bin."))
}

// Entries are never changed or removed, also not by Delete permanently or Empty recycle bin (011 FR-010,
// FR-030): they only get new entries after them.
func TestLog_EntriesNeverChangeOrDisappear_FR010_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.logAs(t, priya, childOne(ActionChanged))
	first := f.moveToBin(t, "child", "C-1")
	second := f.moveToBin(t, "grandchild", "G-3")
	before := f.entries(t)

	if _, err := f.service.Purge(t.Context(), daniel, first.ID); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if _, err := f.service.EmptyBin(t.Context(), daniel, []string{second.ID}); err != nil {
		t.Fatalf("EmptyBin: %v", err)
	}

	after := f.entries(t)
	if len(after) != len(before)+2 {
		t.Fatalf("%d entries, want %d: the old ones and two new", len(after), len(before)+2)
	}
	assertEntries(t, after[:len(before)], before)
}

// newStatusFixture returns a fixture with the tree of addTree in which C-1, G-1 and G-2 are approved, and
// restore rules like those of the planning kinds (004–008): a child or grandchild restored from the bin
// comes back as created (with a note), a child whose name is taken gets " (restored)", and a grandchild
// restored under a stored, approved child sends that child back to created and logs it, as 007's
// AfterRestore does. An Undo keeps everything as it was.
func newStatusFixture(t *testing.T) *fixture {
	t.Helper()
	f := newBareFixture(t)
	kinds := f.standardKinds()
	backToCreated := func(ctx context.Context, rc RestoreContext, s *Snapshot) ([]string, error) {
		if rc.Undo {
			return nil, nil
		}
		var r testRecord
		if err := json.Unmarshal(s.Data, &r); err != nil {
			return nil, err
		}
		var notes []string
		if s.Kind == "child" {
			taken, err := nameTaken(ctx, r)
			if err != nil {
				return nil, err
			}
			if taken {
				r.Name += " (restored)"
				notes = append(notes, "Renamed to '"+r.Name+"' because the name was taken.")
			}
		}
		if r.Status == "approved" {
			r.Status = "created"
			notes = append(notes, "'"+s.Name+"' comes back as created.")
		}
		r.Touch(rc.Actor, rc.Now)
		data, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		s.Name, s.Data = r.Name, data
		if s.Kind == "grandchild" {
			return notes, f.sendParentBack(ctx, rc, r.ParentID)
		}
		return notes, nil
	}
	kinds[1].PrepareRestore = backToCreated
	kinds[2].PrepareRestore = backToCreated
	f.register(t, kinds...)
	f.addTree(t)
	approved := func(r testRecord) testRecord { r.Status = "approved"; return r }
	f.put(t, childrenTable, approved(newRecord("C-1", "Child one", "P-1")))
	f.put(t, grandchildrenTable, approved(newRecord("G-1", "Grandchild one", "C-1")), approved(newRecord("G-2", "Grandchild two", "C-1")))
	return f
}

// nameTaken reports whether another stored child of the same parent has the name of r.
func nameTaken(ctx context.Context, r testRecord) (bool, error) {
	children, err := store.List[testRecord](ctx, childrenTable)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(children, func(c testRecord) bool {
		return c.ID != r.ID && c.ParentID == r.ParentID && c.Name == r.Name
	}), nil
}

// sendParentBack sends the stored child parentID back to created when it is approved, and logs that
// through history, as 007's AfterRestore does for an activity whose line item comes back.
func (f *fixture) sendParentBack(ctx context.Context, rc RestoreContext, parentID string) error {
	parent, found, err := store.Get[testRecord](ctx, childrenTable, parentID)
	if err != nil || !found || parent.Status != "approved" {
		return err
	}
	parent.Status = "created"
	parent.Touch(rc.Actor, rc.Now)
	if err := store.Put(ctx, childrenTable, parent.ID, parent); err != nil {
		return err
	}
	_, err = f.service.Log(ctx, rc.Actor, LogInput{Kind: "child", RecordID: parent.ID, RecordName: parent.Name,
		Action: ActionStatusChanged, Verb: "sent back to created",
		Changes: []Change{{Field: "status", Old: raw(`"approved"`), New: raw(`"created"`)}}})
	return err
}

// listBinItems returns the rows of the recycle bin page and stops the test when that fails.
func (f *fixture) listBinItems(t *testing.T) []BinListItem {
	t.Helper()
	items, err := f.service.ListBinItems(t.Context(), priya)
	if err != nil {
		t.Fatalf("ListBinItems: %v", err)
	}
	return items
}

// preview returns the restore preview of entryID as Priya and stops the test when that fails.
func (f *fixture) preview(t *testing.T, entryID string) RestorePreview {
	t.Helper()
	preview, err := f.service.PreviewRestore(t.Context(), priya, entryID)
	if err != nil {
		t.Fatalf("PreviewRestore(%s): %v", entryID, err)
	}
	return preview
}
