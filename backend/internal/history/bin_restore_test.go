package history

import (
	"reflect"
	"testing"
	"time"
)

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
