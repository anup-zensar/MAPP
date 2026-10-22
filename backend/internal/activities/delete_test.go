package activities_test

import (
	"reflect"
	"slices"
	"testing"

	"mapp/internal/activities"
	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// User story 5 — delete an activity (with Undo).

// Deleting moves the activity to the recycle bin with one "deleted" entry; feature 008 is told first; it
// is gone from its plan (FR-031).
func TestDelete_MovesToBin_FR031(t *testing.T) {
	f := newFixture(t)

	entry, err := f.service.Delete(t.Context(), priya, "A-1004")

	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	wantParent := &history.RecordRef{Type: "initiative", ID: printAndOutdoor, Name: "Print & outdoor"}
	if entry.RecordType != "activity" || entry.RecordID != "A-1004" || entry.RecordName != "Trade magazine ads" ||
		!reflect.DeepEqual(entry.Parent, wantParent) || entry.DeletedBy != priya {
		t.Errorf("bin entry = %+v", entry)
	}
	if _, found := f.stored(t, "A-1004"); found {
		t.Error("the deleted activity is still stored")
	}
	if !slices.Equal(f.hooks.deleting, []string{"A-1004"}) {
		t.Errorf("approvals told about %q, want A-1004", f.hooks.deleting)
	}
	if entries := f.entries(t); len(entries) != 1 || entries[0].Action != history.ActionDeleted {
		t.Errorf("entries = %+v, want one deleted entry", entries)
	}
	rows, _ := f.service.ListForPlan(t.Context(), priya, "P-1001")
	if slices.ContainsFunc(rows, func(s activities.Summary) bool { return s.ID == "A-1004" }) {
		t.Error("the plan still lists the deleted activity")
	}
	_, err = f.service.Delete(t.Context(), priya, "A-1004")
	assertError(t, err, errs.NotFound("This activity no longer exists — someone may have deleted it."))
}

// Undo right after the delete brings the activity back exactly: same ID, values, status and version
// (FR-031, R14).
func TestUndo_RestoresExactly_FR031(t *testing.T) {
	f := newFixture(t)
	before, _ := f.stored(t, "A-1002") // Approved
	entry, err := f.service.Delete(t.Context(), priya, "A-1002")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := f.history.Undo(t.Context(), priya, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}

	if after, found := f.stored(t, "A-1002"); !found || !reflect.DeepEqual(after, before) {
		t.Errorf("after Undo = %+v\nwant %+v", after, before)
	}
	if len(f.hooks.restored) != 0 {
		t.Errorf("an Undo told approvals about a restore: %q", f.hooks.restored)
	}
}

// Undo is refused when another activity of the initiative has meanwhile taken the name; the activity stays
// in the recycle bin (FR-031).
func TestUndo_NameTakenRefused_FR031(t *testing.T) {
	f := newFixture(t)
	entry, err := f.service.Delete(t.Context(), priya, "A-1001")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	f.create(t, digitalAndSocial, activities.Input{Name: "LAUNCH FILM", ActivityTypeID: eventType})

	_, err = f.history.Undo(t.Context(), priya, entry.ID)

	assertError(t, err, errs.NameTaken("Can't undo — there is now another activity called 'LAUNCH FILM'. Restore it from the recycle bin instead."))
	if _, found := f.stored(t, "A-1001"); found {
		t.Error("the refused Undo brought the activity back")
	}
}

// A restore from the recycle bin brings the activity back as Created (R14), renamed when its name was
// taken, and tells feature 008 (FR-031, D-196).
func TestRestore_ResetsToCreatedAndRenames_FR031(t *testing.T) {
	f := newFixture(t)
	entry, err := f.service.Delete(t.Context(), priya, "A-1002")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	f.create(t, digitalAndSocial, activities.Input{Name: "Social always-on", ActivityTypeID: eventType})

	result, err := f.history.Restore(t.Context(), daniel, entry.ID)

	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	restored, _ := f.stored(t, "A-1002")
	if restored.Name != "Social always-on (restored)" || restored.Status != activities.StatusCreated || restored.Version != 2 ||
		restored.UpdatedBy != daniel {
		t.Errorf("restored = %s, %s, version %d by %v", restored.Name, restored.Status, restored.Version, restored.UpdatedBy)
	}
	wantDetails := "Renamed to 'Social always-on (restored)' because the name was taken. " +
		"Status reset to Created because it was restored from the recycle bin."
	if result.Details != wantDetails || !slices.Equal(f.hooks.restored, []string{"A-1002"}) {
		t.Errorf("details %q, approvals told %q", result.Details, f.hooks.restored)
	}
}
