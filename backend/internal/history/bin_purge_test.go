package history

import (
	"reflect"
	"testing"
	"time"

	"mapp/internal/platform/errs"
)

// Delete permanently removes the records for good (011 FR-030): the entry leaves the bin with who did it
// and when and keeps no records, the records stay out of storage, and one "permanently deleted" entry names
// the record, its place when it was deleted and what was inside it. The entries about the records stay,
// now saying they are gone for good.
func TestPurge_RemovesForGoodAndLogs_FR030(t *testing.T) { //nolint:gocyclo // 13: one purge, then each place the records must be gone from
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
func TestEmptyBin_SkipsResolvedAndUnlisted_FR032(t *testing.T) { //nolint:gocyclo // 10: one Empty, then each entry it had to leave alone
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
