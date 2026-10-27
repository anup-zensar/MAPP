package approval

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// An approver who became inactive may stay in their level, with a warning, but may not be added to a level
// (FR-003, 008-P8).
func TestUpdateWorkflow_InactiveApproverStaysButCannotBeAdded_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.deactivatePerson(t, alex)

	kept, err := f.service.UpdateWorkflow(t.Context(), priya, chinaFlow, WorkflowUpdate{
		Name: textPtr("China BC approvals"), Levels: levelInputsPtr([]string{carlos.ID}, []string{alex.ID}), Version: version(1),
	})
	_, added := f.service.UpdateWorkflow(t.Context(), priya, chinaFlow, WorkflowUpdate{
		Levels: levelInputsPtr([]string{carlos.ID, alex.ID}, []string{alex.ID}), Version: version(2),
	})
	_, created := f.service.CreateWorkflow(t.Context(), priya, WorkflowInput{Name: "New", Levels: levelInputs([]string{alex.ID})})

	if err != nil {
		t.Fatalf("UpdateWorkflow keeping Alex: %v", err)
	}
	if kept.LevelsText != "Level 1 · Carlos Mendes → Level 2 · Alex Morgan (inactive)" ||
		!reflect.DeepEqual(kept.Warnings, []string{"Level 2 has no active approver"}) {
		t.Errorf("levels %q, warnings %q; want Alex Morgan kept as inactive with a warning", kept.LevelsText, kept.Warnings)
	}
	inactive := errs.Validation(errs.FieldError{Field: "levels.1.approverIds", Message: "Level 1: Alex Morgan is inactive — choose an active person."})
	assertError(t, added, inactive)
	assertError(t, created, inactive)
}

// The version the person started from is required and must still be the stored one; otherwise the update
// is refused saying who changed the workflow and when (FR-003, D-030).
func TestUpdateWorkflow_VersionConflict_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.clock.Advance(time.Hour)
	changedAt := f.clock.Now()
	first, err := f.service.UpdateWorkflow(t.Context(), priya, chinaFlow, WorkflowUpdate{Name: textPtr("China approvals"), Version: version(1)})
	if err != nil || first.Version != 2 {
		t.Fatalf("first update: version %d (%v), want 2", first.Version, err)
	}
	f.clock.Advance(time.Minute)

	_, stale := f.service.UpdateWorkflow(t.Context(), mei, chinaFlow, WorkflowUpdate{Name: textPtr("China"), Version: version(1)})
	_, missing := f.service.UpdateWorkflow(t.Context(), mei, chinaFlow, WorkflowUpdate{Name: textPtr("")})
	_, gone := f.service.UpdateWorkflow(t.Context(), mei, "WF-9999", WorkflowUpdate{Version: version(1)})

	assertError(t, stale, errs.VersionConflict("approval workflow", "Priya Raman", changedAt))
	assertError(t, missing, errs.Validation(
		errs.FieldError{Field: "version", Message: "Reload the page and try again."},
		errs.FieldError{Field: "name", Message: "Enter a name."},
	))
	assertError(t, gone, errs.NotFound("This approval workflow no longer exists — someone may have deleted it."))
	if stored, _ := f.storedWorkflow(t, chinaFlow); stored.Name != "China approvals" || stored.Version != 2 {
		t.Errorf("stored = %s v%d, want the first update only", stored.Name, stored.Version)
	}
}

// A save that changes nothing — the same values, or only surrounding spaces — makes no new version and no
// log entry (FR-026).
func TestUpdateWorkflow_NoChangeNoLog_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view, err := f.service.UpdateWorkflow(t.Context(), priya, chinaFlow, WorkflowUpdate{
		Name: textPtr(" China BC approval workflow "), Description: textPtr(""),
		Levels: levelInputsPtr([]string{carlos.ID}, []string{alex.ID}), Version: version(1),
	})

	if err != nil || view.Version != 1 || view.UpdatedBy != actor.System {
		t.Errorf("view v%d by %v (%v), want unchanged", view.Version, view.UpdatedBy, err)
	}
	assertLog(t, f, chinaFlow, nil)
}

// An edit applies to later submissions only: the open round keeps the levels it was submitted with and
// goes on with them; the next submission copies the new levels (FR-006).
func TestUpdateWorkflow_OpenRoundsKeepLevels_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.addActivity(t, "A-1010", "Lunar new year promo", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	before := f.openRound(t, "A-1009")

	view, err := f.service.UpdateWorkflow(t.Context(), priya, chinaFlow, WorkflowUpdate{
		Levels: levelInputsPtr([]string{sara.ID}), Version: version(1),
	})

	if err != nil || view.WaitingCount != 1 || view.LevelsText != "Level 1 · Sara Novak" {
		t.Fatalf("view = %+v (%v), want the new level and 1 waiting", view, err)
	}
	if after := f.openRound(t, "A-1009"); !reflect.DeepEqual(after, before) {
		t.Errorf("open round =\n%+v\nwant unchanged\n%+v", after, before)
	}
	f.approve(t, carlos, "A-1009", "")
	if round := f.openRound(t, "A-1009"); round.CurrentLevel != 2 || names(round.Levels[1].Approvers) != "Alex Morgan" {
		t.Errorf("round at level %d, want level 2 of the old levels, waiting for Alex Morgan", round.CurrentLevel)
	}
	f.submit(t, mei, "A-1010", "")
	if round := f.openRound(t, "A-1010"); len(round.Levels) != 1 || names(round.Levels[0].Approvers) != "Sara Novak" {
		t.Errorf("new round levels = %+v, want the new level of Sara Novak", round.Levels)
	}
}

// Every change of a workflow is logged once, with readable labels and texts: the levels as their levels
// text (FR-026).
func TestWorkflowChanges_Logged_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	created := f.createWorkflow(t, alex, "Temp flow", []string{carlos.ID})
	_, err := f.service.UpdateWorkflow(t.Context(), priya, created.ID, WorkflowUpdate{
		Name: textPtr("Temp approvals"), Description: textPtr("Two levels."),
		Levels: levelInputsPtr([]string{carlos.ID}, []string{alex.ID}), Version: version(1),
	})
	if err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}
	if _, err := f.service.SetWorkflowActive(t.Context(), priya, created.ID, false, version(2)); err != nil {
		t.Fatalf("SetWorkflowActive: %v", err)
	}

	log := f.log(t, created.ID)

	var got []string
	for _, e := range log {
		line := e.Actor.Name + " " + e.Verb + " " + e.RecordTypeLabel + " '" + e.RecordName + "'"
		for _, c := range e.Changes {
			line += " | " + c.Label + ": " + c.OldText + " → " + c.NewText
		}
		got = append(got, line)
	}
	want := []string{
		"Priya Raman deactivated approval workflow 'Temp approvals' | Status: Active → Inactive",
		"Priya Raman changed approval workflow 'Temp approvals' | Description: Not filled in → Two levels." +
			" | Levels: Level 1 · Carlos Mendes → Level 1 · Carlos Mendes → Level 2 · Alex Morgan | Name: Temp flow → Temp approvals",
		"Alex Morgan created approval workflow 'Temp flow'",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("log =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
