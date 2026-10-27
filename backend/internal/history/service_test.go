package history

import (
	"context"
	"reflect"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
)

// Undo puts back exactly what was deleted: the same IDs, data and versions (FR-026, domain model R14). The
// entry leaves the bin with who undid it and when, keeps no snapshots, and one "restored" entry is logged.
func TestUndo_RestoresExactly_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	changed := newRecord("C-1", "Child one", "P-1")
	changed.Touch(daniel, start) // version 2: Undo must bring the version back too
	f.put(t, childrenTable, changed)
	before := f.allRecords(t)
	deleted := f.moveToBin(t, "parent", "P-1")
	f.clock.Advance(time.Minute)

	got, err := f.service.Undo(t.Context(), daniel, deleted.ID)

	want := RestoreResult{RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one"}
	if err != nil || got != want {
		t.Fatalf("Undo = %+v, %v; want %+v", got, err, want)
	}
	if after := f.allRecords(t); !reflect.DeepEqual(after, before) {
		t.Errorf("records after Undo =\n%+v\nwant exactly\n%+v", after, before)
	}
	entry := f.binEntry(t, deleted.ID)
	wantResolution := Resolution{How: "undone", By: daniel, At: start.Add(time.Minute)}
	if entry.Resolution == nil || *entry.Resolution != wantResolution {
		t.Errorf("resolution = %+v, want %+v", entry.Resolution, wantResolution)
	}
	if entry.Snapshots == nil || len(entry.Snapshots) != 0 {
		t.Errorf("snapshots = %d, want an empty list once the entry is resolved", len(entry.Snapshots))
	}
	if bin := f.listBin(t); len(bin) != 0 {
		t.Errorf("recycle bin = %s, want it empty", asJSON(bin))
	}
	assertEntries(t, f.entries(t)[1:], []Entry{{
		ID: "LOG-1002", Seq: 1002, At: start.Add(time.Minute), Actor: daniel, Action: ActionRestored, Verb: "restored",
		RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one",
		Path: []RecordRef{}, Changes: []Change{}, Details: "Undo, with 2 children and 3 grandchildren",
	}})
}

// The server accepts Undo for 2 minutes after the delete (FR-026, D-037): at exactly 2 minutes it works; a
// millisecond later it is refused with undo_expired and nothing changes. Restore has no time limit.
func TestUndo_ExpiresAfterTwoMinutes_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	onTime := f.moveToBin(t, "parent", "P-1")
	late := f.moveToBin(t, "parent", "P-2")
	f.clock.Advance(2 * time.Minute)

	if _, err := f.service.Undo(t.Context(), priya, onTime.ID); err != nil {
		t.Errorf("Undo exactly 2 minutes after the delete: %v, want it to work", err)
	}
	f.clock.Advance(time.Millisecond)
	entriesBefore := len(f.entries(t))

	_, err := f.service.Undo(t.Context(), priya, late.ID)

	assertError(t, err, errs.UndoExpired())
	if _, found := f.stored(t, parentsTable, "P-2"); found {
		t.Error("P-2 came back although Undo was refused")
	}
	if bin := f.listBin(t); len(bin) != 1 || bin[0].ID != late.ID {
		t.Errorf("recycle bin = %s, want only %s", asJSON(bin), late.ID)
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("%d log entries, want still %d", got, entriesBefore)
	}
	if _, err := f.service.Restore(t.Context(), priya, late.ID); err != nil {
		t.Errorf("Restore after the Undo time: %v, want it to work", err)
	}
}

// A record whose parent is no longer stored cannot come back on its own: Restore and Undo are refused,
// naming the parent to restore first, and nothing changes. Once the parent is back, the record follows
// (FR-025).
func TestRestore_ParentMissing_FR025(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	child := f.moveToBin(t, "child", "C-1")
	parent := f.moveToBin(t, "parent", "P-1")
	entriesBefore := len(f.entries(t))
	attempts := []struct {
		name string
		call func(ctx context.Context, a actor.Actor, entryID string) (RestoreResult, error)
	}{
		{"Restore", f.service.Restore},
		{"Undo", f.service.Undo},
	}

	for _, attempt := range attempts {
		t.Run(attempt.name, func(t *testing.T) {
			_, err := attempt.call(t.Context(), priya, child.ID)

			assertError(t, err, errs.RestoreParentFirst(RecordRef{Type: "parent", ID: "P-1", Name: "Parent one"}))
		})
	}
	if _, found := f.stored(t, childrenTable, "C-1"); found {
		t.Error("C-1 came back without its parent")
	}
	if entry := f.binEntry(t, child.ID); entry.Resolution != nil || len(entry.Snapshots) != 3 {
		t.Errorf("entry = %s, want it unchanged in the bin with its 3 records", asJSON(entry))
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("%d log entries, want still %d", got, entriesBefore)
	}

	f.restore(t, priya, parent.ID)
	f.restore(t, priya, child.ID)
	if _, found := f.stored(t, childrenTable, "C-1"); !found {
		t.Error("C-1 did not come back after its parent")
	}
}

// An entry that has already left the bin cannot be undone or restored again: the refusal says how, by whom
// and when, so the person understands what happened (FR-025, 011 FR-032). Nothing changes.
func TestUndo_AlreadyResolved_FR025(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	undone := f.moveToBin(t, "parent", "P-1")
	restored := f.moveToBin(t, "parent", "P-2")
	f.clock.Advance(time.Minute)
	f.undo(t, daniel, undone.ID)
	f.restore(t, mei, restored.ID)
	entriesBefore := len(f.entries(t))
	tests := []struct {
		name    string
		entryID string
		want    *errs.Error
	}{
		{"undone", undone.ID, errs.AlreadyResolved("Parent one", Resolution{How: "undone", By: daniel, At: start.Add(time.Minute)})},
		{"restored", restored.ID, errs.AlreadyResolved("Parent two", Resolution{How: "restored", By: mei, At: start.Add(time.Minute)})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, undoErr := f.service.Undo(t.Context(), priya, tt.entryID)
			_, restoreErr := f.service.Restore(t.Context(), priya, tt.entryID)

			assertError(t, undoErr, tt.want)
			assertError(t, restoreErr, tt.want)
		})
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("%d log entries, want still %d", got, entriesBefore)
	}
}

// An entry can come back only where all its kinds are registered: in an environment without the developer
// sample records, their entries answer not_found. So does an id the bin never had (FR-025).
func TestUndo_KindNotRegistered_FR025(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	deleted := f.moveToBin(t, "parent", "P-1")
	noKinds := NewService(f.db, NewJSONStore(), ids.NewGenerator(ids.NewJSONCounters()), f.clock.Now)
	parentsOnly := NewService(f.db, NewJSONStore(), ids.NewGenerator(ids.NewJSONCounters()), f.clock.Now)
	if err := parentsOnly.Register(f.standardKinds()[0]); err != nil {
		t.Fatalf("Register: %v", err)
	}
	services := []struct {
		name    string
		service *Service
	}{
		{"no kinds", noKinds},
		{"the parent kind but not its children's", parentsOnly},
	}

	for _, s := range services {
		t.Run(s.name, func(t *testing.T) {
			_, undoErr := s.service.Undo(t.Context(), priya, deleted.ID)
			_, restoreErr := s.service.Restore(t.Context(), priya, deleted.ID)

			assertError(t, undoErr, errs.NotFound("This item can't be restored here."))
			assertError(t, restoreErr, errs.NotFound("This item can't be restored here."))
		})
	}

	_, undoErr := f.service.Undo(t.Context(), priya, "BIN-9999")
	_, restoreErr := f.service.Restore(t.Context(), priya, "BIN-9999")
	assertError(t, undoErr, errs.NotFound("This item is not in the recycle bin."))
	assertError(t, restoreErr, errs.NotFound("This item is not in the recycle bin."))
	if bin := f.listBin(t); len(bin) != 1 {
		t.Errorf("recycle bin = %s, want the entry still in it", asJSON(bin))
	}
}

// The recycle bin lists what is still in it, newest first; entries that were undone or restored are left
// out (data model §6).
func TestListBin_UnresolvedNewestFirst(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""), newRecord("P-3", "Parent three", ""))
	if bin := f.listBin(t); bin == nil || len(bin) != 0 {
		t.Errorf("empty bin: ListBin = %#v, want an empty list", bin)
	}
	first := f.moveToBin(t, "parent", "P-1")
	f.clock.Advance(time.Minute)
	undone := f.moveToBin(t, "parent", "P-2")
	f.clock.Advance(time.Minute)
	last := f.moveToBin(t, "parent", "P-3")
	f.undo(t, priya, undone.ID)

	got := f.listBin(t)

	if want := []BinEntry{last, first}; !reflect.DeepEqual(got, want) {
		t.Errorf("ListBin =\n%s\nwant\n%s", asJSON(got), asJSON(want))
	}
}
