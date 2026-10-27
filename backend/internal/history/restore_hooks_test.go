package history

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/errs"
)

// PrepareRestore learns who restores, when and whether it is an Undo; its changes and notes are logged.
func TestRestore_AppliesPrepareRestore_FR025(t *testing.T) { //nolint:gocyclo // 10: the restore rule written out here, then each thing it changed
	t.Parallel()
	f := newBareFixture(t)
	kinds := f.standardKinds()
	var seen []RestoreContext
	kinds[0].PrepareRestore = func(_ context.Context, rc RestoreContext, s *Snapshot) ([]string, error) {
		seen = append(seen, rc)
		if rc.Undo {
			return nil, nil // an Undo puts the record back exactly
		}
		var r testRecord
		if err := json.Unmarshal(s.Data, &r); err != nil {
			return nil, err
		}
		r.Name += " (restored)"
		r.Touch(rc.Actor, rc.Now)
		data, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		s.Name, s.Data = r.Name, data
		return []string{"Renamed to '" + r.Name + "' because the name was taken."}, nil
	}
	f.register(t, kinds...)
	f.addTree(t)
	deleted := f.moveToBin(t, "parent", "P-1")
	f.clock.Advance(time.Hour)
	restoredAt := start.Add(time.Hour)

	got, err := f.service.Restore(t.Context(), daniel, deleted.ID)

	note := "Renamed to 'Parent one (restored)' because the name was taken."
	want := RestoreResult{RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one (restored)", Details: note}
	if err != nil || got != want {
		t.Fatalf("Restore = %+v, %v; want %+v", got, err, want)
	}
	if want := []RestoreContext{{Actor: daniel, Now: restoredAt, Undo: false, DeletedAt: start, DeletedBy: priya}}; !slices.Equal(seen, want) {
		t.Errorf("PrepareRestore saw %+v, want %+v", seen, want)
	}
	wantParent := newRecord("P-1", "Parent one (restored)", "")
	wantParent.Touch(daniel, restoredAt)
	if stored, _ := f.stored(t, parentsTable, "P-1"); stored != wantParent {
		t.Errorf("stored parent = %+v, want %+v", stored, wantParent)
	}
	if _, found := f.stored(t, grandchildrenTable, "G-3"); !found {
		t.Error("the grandchild G-3 did not come back with its parent")
	}
	assertEntries(t, f.entries(t)[1:], []Entry{{
		ID: "LOG-1002", Seq: 1002, At: restoredAt, Actor: daniel, Action: ActionRestored, Verb: "restored",
		RecordType: "parent", RecordTypeLabel: "parent", RecordID: "P-1", RecordName: "Parent one (restored)",
		Path: []RecordRef{},
		Changes: []Change{{Field: "name", Label: "Name", Old: raw(`"Parent one"`), New: raw(`"Parent one (restored)"`),
			OldText: "Parent one", NewText: "Parent one (restored)"}},
		Details: "Restored from the recycle bin, with 2 children and 3 grandchildren. " + note,
	}})

	undone := f.moveToBin(t, "parent", "P-1")
	f.undo(t, daniel, undone.ID)
	if last := seen[len(seen)-1]; !last.Undo {
		t.Errorf("PrepareRestore during Undo saw %+v, want Undo true", last)
	}
}

// A refusing PrepareRestore brings nothing back, not even the records already put back in the same restore.
func TestUndo_RefusedByPrepareRestoreChangesNothing_FR026(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	kinds := f.standardKinds()
	refusal := errs.NameTaken("Can't undo — there is now another grandchild called 'Grandchild one'. Restore it from the recycle bin instead.")
	kinds[2].PrepareRestore = func(_ context.Context, rc RestoreContext, _ *Snapshot) ([]string, error) {
		if rc.Undo {
			return nil, refusal // grandchildren come back after their parent and child
		}
		return nil, nil
	}
	f.register(t, kinds...)
	f.addTree(t)
	deleted := f.moveToBin(t, "parent", "P-1")
	entriesBefore := len(f.entries(t))

	_, err := f.service.Undo(t.Context(), priya, deleted.ID)

	assertError(t, err, refusal)
	for table, records := range f.allRecords(t) {
		if len(records) != 0 {
			t.Errorf("%s = %+v, want nothing back", table, records)
		}
	}
	if bin := f.listBin(t); len(bin) != 1 {
		t.Errorf("recycle bin = %s, want the entry still in it", asJSON(bin))
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("%d log entries, want still %d", got, entriesBefore)
	}
}

// The restore rules learn who deleted the records and when, in an Undo and a Restore alike.
func TestRestore_ContextHasDeleteTimeAndPerson_FR023(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	kinds := f.standardKinds()
	var prepared, after []RestoreContext
	kinds[0].PrepareRestore = func(_ context.Context, rc RestoreContext, _ *Snapshot) ([]string, error) {
		prepared = append(prepared, rc)
		return nil, nil
	}
	kinds[0].AfterRestore = func(_ context.Context, rc RestoreContext, _ Snapshot) ([]string, error) {
		after = append(after, rc)
		return nil, nil
	}
	f.register(t, kinds...)
	f.addTree(t)

	f.clock.Advance(time.Hour)
	firstDelete := f.clock.Now()
	undone, err := f.service.MoveToBin(t.Context(), mei, "parent", "P-1")
	if err != nil {
		t.Fatalf("MoveToBin: %v", err)
	}
	f.clock.Advance(time.Minute)
	f.undo(t, daniel, undone.ID)
	f.clock.Advance(time.Hour)
	secondDelete := f.clock.Now()
	restored := f.moveToBin(t, "parent", "P-1")
	f.clock.Advance(24 * time.Hour)
	f.restore(t, daniel, restored.ID)

	want := []RestoreContext{
		{Actor: daniel, Now: firstDelete.Add(time.Minute), Undo: true, DeletedAt: firstDelete, DeletedBy: mei},
		{Actor: daniel, Now: secondDelete.Add(24 * time.Hour), Undo: false, DeletedAt: secondDelete, DeletedBy: priya},
	}
	if !slices.Equal(prepared, want) || !slices.Equal(after, want) {
		t.Errorf("PrepareRestore saw %+v\nAfterRestore saw %+v\nwant %+v", prepared, after, want)
	}
}

// AfterRestore runs once, for the deleted record's kind only, after every record of the entry is back.
func TestAfterRestore_RunsOnceForDeletedRecord_FR033(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	kinds := f.standardKinds()
	var calls []string
	for i := range kinds {
		name := kinds[i].Name
		kinds[i].AfterRestore = func(_ context.Context, rc RestoreContext, top Snapshot) ([]string, error) {
			_, grandchildBack, err := f.records(grandchildrenTable).Get("G-1")
			if err != nil {
				return nil, err
			}
			calls = append(calls, fmt.Sprintf("%s hook: %s %s, undo %v, G-1 back %v", name, top.Kind, top.ID, rc.Undo, grandchildBack))
			return nil, nil
		}
	}
	f.register(t, kinds...)
	f.addTree(t)

	f.undo(t, daniel, f.moveToBin(t, "child", "C-1").ID)
	f.restore(t, daniel, f.moveToBin(t, "parent", "P-1").ID)

	want := []string{
		"child hook: child C-1, undo true, G-1 back true",
		"parent hook: parent P-1, undo false, G-1 back true",
	}
	if !slices.Equal(calls, want) {
		t.Errorf("calls =\n%q\nwant\n%q", calls, want)
	}
}

// AfterRestore's notes follow PrepareRestore's, and the hook sees the record as PrepareRestore left it.
func TestAfterRestore_Notes(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	kinds := f.standardKinds()
	kinds[0].PrepareRestore = func(_ context.Context, _ RestoreContext, s *Snapshot) ([]string, error) {
		s.Name = "Parent one (restored)" // as a kind renames a record whose name is taken
		return []string{"Renamed."}, nil
	}
	var seenName string
	kinds[0].AfterRestore = func(_ context.Context, _ RestoreContext, top Snapshot) ([]string, error) {
		seenName = top.Name
		return []string{"Something else changed.", "And one more thing."}, nil
	}
	f.register(t, kinds...)
	f.addTree(t)
	deleted := f.moveToBin(t, "parent", "P-1")

	result, err := f.service.Restore(t.Context(), daniel, deleted.ID)

	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	notes := "Renamed. Something else changed. And one more thing."
	if result.Details != notes || seenName != "Parent one (restored)" {
		t.Errorf("details %q, hook saw %q; want %q and the renamed record", result.Details, seenName, notes)
	}
	entries := f.entries(t)
	if last := entries[len(entries)-1]; last.Details != "Restored from the recycle bin, with 2 children and 3 grandchildren. "+notes {
		t.Errorf("restored entry details = %q", last.Details)
	}
}

// A refusing AfterRestore changes nothing: even its changes to other records in the Write are undone.
func TestAfterRestore_ErrorChangesNothing(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	kinds := f.standardKinds()
	refusal := errors.New("the other feature failed")
	kinds[1].AfterRestore = func(ctx context.Context, rc RestoreContext, _ Snapshot) ([]string, error) {
		other := newRecord("P-2", "Parent two", "")
		other.Touch(rc.Actor, rc.Now) // a change to a record outside the entry, in the same Write
		if err := f.records(parentsTable).Put(ctx, other); err != nil {
			return nil, err
		}
		return nil, refusal
	}
	f.register(t, kinds...)
	f.addTree(t)
	f.put(t, parentsTable, newRecord("P-2", "Parent two", ""))
	deleted := f.moveToBin(t, "child", "C-1")
	entriesBefore := len(f.entries(t))

	_, err := f.service.Undo(t.Context(), daniel, deleted.ID)

	if !errors.Is(err, refusal) {
		t.Errorf("Undo error = %v, want %v", err, refusal)
	}
	if _, found := f.stored(t, childrenTable, "C-1"); found {
		t.Error("the refused Undo brought the child back")
	}
	if other, _ := f.stored(t, parentsTable, "P-2"); other != newRecord("P-2", "Parent two", "") {
		t.Errorf("P-2 = %+v, want it unchanged", other)
	}
	if bin := f.listBin(t); len(bin) != 1 || bin[0].ID != deleted.ID {
		t.Errorf("recycle bin = %s, want the entry still in it", asJSON(bin))
	}
	if got := len(f.entries(t)); got != entriesBefore {
		t.Errorf("%d log entries, want still %d", got, entriesBefore)
	}
}
