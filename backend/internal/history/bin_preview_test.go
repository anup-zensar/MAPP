package history

import (
	"reflect"
	"slices"
	"testing"
)

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
// make them, once: by the rules' notes, not again as a change (011 FR-018, FR-019; 038 FR-049, L-23): here an
// approved child and grandchild come back as created.
func TestPreviewRestore_StatusChangesOnce_038_FR049_L23(t *testing.T) {
	t.Parallel()
	f := newStatusFixture(t)
	deleted := f.moveToBin(t, "child", "C-1")

	preview := f.preview(t, deleted.ID)

	if len(preview.Changes) != 0 {
		t.Errorf("changes = %s, want none: the notes state them", asJSON(preview.Changes))
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

// When the name is taken meanwhile, the preview says once that the record comes back under a new name
// (011 FR-022; 038 FR-049, L-23).
func TestPreviewRestore_NameClashOnce_038_FR049_L23(t *testing.T) {
	t.Parallel()
	f := newStatusFixture(t)
	deleted := f.moveToBin(t, "child", "C-1")
	f.put(t, childrenTable, newRecord("C-9", "Child one", "P-1")) // takes the name

	preview := f.preview(t, deleted.ID)

	if slices.ContainsFunc(preview.Changes, func(c RecordChange) bool { return c.Field == "name" }) {
		t.Errorf("changes = %s, want the rename only in the notes", asJSON(preview.Changes))
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
