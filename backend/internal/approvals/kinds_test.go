package approvals

import (
	"testing"

	"northstar/internal/platform/errs"
)

// Undo puts a deleted workflow back exactly; when another workflow has meanwhile taken its name, ignoring
// letter case, Undo is refused with name_taken and the workflow stays in the recycle bin (FR-008).
func TestWorkflowUndo_NameTaken_FR008(t *testing.T) {
	f := newFixture(t)
	created := f.createWorkflow(t, alex, "Temp flow", []string{carlos.ID})
	first, err := f.service.DeleteWorkflow(t.Context(), alex, created.ID)
	if err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
	if _, err := f.history.Undo(t.Context(), alex, first.ID); err != nil {
		t.Fatalf("Undo of a free name: %v", err)
	}
	second, err := f.service.DeleteWorkflow(t.Context(), alex, created.ID)
	if err != nil {
		t.Fatalf("DeleteWorkflow again: %v", err)
	}
	f.createWorkflow(t, priya, "TEMP FLOW", []string{daniel.ID})

	_, refused := f.history.Undo(t.Context(), alex, second.ID)

	assertError(t, refused, errs.NameTaken(
		"Can't undo — there is now another approval workflow called 'TEMP FLOW'. Restore it from the recycle bin instead."))
	if _, found := f.storedWorkflow(t, created.ID); found {
		t.Error("the workflow came back, want it kept in the recycle bin")
	}
}

// Restore from the recycle bin renames a workflow whose name was taken meanwhile, to "<name> (restored)",
// with a new version and a note saying so (FR-008).
func TestWorkflowRestore_RenamesWhenTaken_FR008(t *testing.T) {
	f := newFixture(t)
	created := f.createWorkflow(t, alex, "Temp flow", []string{carlos.ID})
	entry, err := f.service.DeleteWorkflow(t.Context(), alex, created.ID)
	if err != nil {
		t.Fatalf("DeleteWorkflow: %v", err)
	}
	f.createWorkflow(t, priya, "Temp flow", []string{daniel.ID})

	result, err := f.history.Restore(t.Context(), priya, entry.ID)

	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if result.RecordName != "Temp flow (restored)" || result.Details != "Renamed to 'Temp flow (restored)' because the name was taken." {
		t.Errorf("result = %+v, want renamed with the note", result)
	}
	if back, found := f.storedWorkflow(t, created.ID); !found || back.Name != "Temp flow (restored)" || back.Version != 2 || back.UpdatedBy != priya {
		t.Errorf("restored = %+v, want renamed by Priya Raman in version 2", back)
	}
}
