package approval

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

// levelInputs returns one unnamed level per list of approvers, in order.
func levelInputs(approvers ...[]string) []LevelInput {
	levels := []LevelInput{}
	for _, ids := range approvers {
		levels = append(levels, LevelInput{ApproverIDs: ids})
	}
	return levels
}

// levelInputsPtr is levelInputs for an update.
func levelInputsPtr(approvers ...[]string) *[]LevelInput {
	levels := levelInputs(approvers...)
	return &levels
}

// textPtr returns a pointer to s, for the fields of an update.
func textPtr(s string) *string {
	return &s
}

// createWorkflow creates a workflow as a and stops the test when that fails.
func (f *fixture) createWorkflow(t *testing.T, a actor.Actor, name string, approvers ...[]string) WorkflowView {
	t.Helper()
	view, err := f.service.CreateWorkflow(t.Context(), a, WorkflowInput{Name: name, Levels: levelInputs(approvers...)})
	if err != nil {
		t.Fatalf("CreateWorkflow(%s): %v", name, err)
	}
	return view
}

// storedWorkflow returns the stored workflow id, and false when there is none.
func (f *fixture) storedWorkflow(t *testing.T, id string) (Workflow, bool) {
	t.Helper()
	var w Workflow
	var found bool
	f.read(t, func(ctx context.Context) (err error) {
		w, found, err = f.service.repo.GetWorkflow(ctx, id)
		return err
	})
	return w, found
}

// deactivatePerson makes the person p inactive, as feature 003 does.
func (f *fixture) deactivatePerson(t *testing.T, p actor.Actor) {
	t.Helper()
	f.write(t, func(ctx context.Context) error {
		person, _, err := people.NewJSONStore().Get(ctx, p.ID)
		if err != nil {
			return err
		}
		person.Active = false
		return people.NewJSONStore().Put(ctx, person)
	})
}

// workflowViews returns ListWorkflows as a, and stops the test when it fails.
func (f *fixture) workflowViews(t *testing.T, a actor.Actor) []WorkflowView {
	t.Helper()
	views, err := f.service.ListWorkflows(t.Context(), a)
	if err != nil {
		t.Fatalf("ListWorkflows: %v", err)
	}
	return views
}

// A new workflow is active, with an id WF-…, its texts without surrounding spaces and its levels in the
// order given; an unnamed level is labelled "Level <n>". It is logged as created (FR-002, FR-026).
func TestCreateWorkflow_SavesOrderedLevels_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{
		Name: "  Mobility India approval ", Description: " For India. ",
		Levels: []LevelInput{{Name: " Regional ", ApproverIDs: []string{carlos.ID, mei.ID}}, {ApproverIDs: []string{alex.ID}}},
	})

	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	meta := record.NewMeta(alex, start)
	want := WorkflowView{
		ID: "WF-1003", Name: "Mobility India approval", Description: "For India.", Active: true,
		Levels: []LevelView{
			{Number: 1, Name: "Regional", Label: "Regional", Approvers: []PersonView{
				{ID: carlos.ID, Name: "Carlos Mendes", Initials: "CM", Active: true}, {ID: mei.ID, Name: "Mei Lin", Initials: "ML", Active: true},
			}, ApproverType: SourcePeople, Mode: ModeAny, Summary: "Regional · Carlos Mendes, Mei Lin"},
			{Number: 2, Label: "Level 2", Approvers: []PersonView{{ID: alex.ID, Name: "Alex Morgan", Initials: "AM", Active: true}},
				ApproverType: SourcePeople, Mode: ModeAny, Summary: "Level 2 · Alex Morgan"},
		},
		LevelsText: "Regional · Carlos Mendes, Mei Lin → Level 2 · Alex Morgan", Warnings: []string{}, Meta: meta,
	}
	if !reflect.DeepEqual(view, want) {
		t.Errorf("view =\n%+v\nwant\n%+v", view, want)
	}
	stored, _ := f.storedWorkflow(t, "WF-1003")
	wantStored := Workflow{
		ID: "WF-1003", Name: "Mobility India approval", Description: "For India.", Active: true,
		Levels: []Level{
			{Name: "Regional", ApproverType: SourcePeople, ApproverIDs: []string{carlos.ID, mei.ID}, Mode: ModeAny},
			{ApproverType: SourcePeople, ApproverIDs: []string{alex.ID}, Mode: ModeAny},
		}, Meta: meta,
	}
	if !reflect.DeepEqual(stored, wantStored) {
		t.Errorf("stored =\n%+v\nwant\n%+v", stored, wantStored)
	}
	assertLog(t, f, "WF-1003", []string{"Alex Morgan created approval workflow 'Mobility India approval'"})
}

// The name is required, at most 120 characters, and unique among all workflows, inactive ones too,
// ignoring letter case and surrounding spaces (FR-002).
func TestCreateWorkflow_NameRules_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	china, _ := f.storedWorkflow(t, chinaFlow)
	china.Active = false
	f.write(t, func(ctx context.Context) error { return f.service.repo.PutWorkflow(ctx, china) })
	levels := levelInputs([]string{carlos.ID})

	for _, c := range []struct{ name, message string }{
		{"   ", "Enter a name."},
		{strings.Repeat("n", 121), "Use 120 characters or fewer."},
		{" china bc APPROVAL workflow ", "There is already an approval workflow called 'China BC approval workflow'. Choose another name."},
	} {
		_, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{Name: c.name, Levels: levels})
		assertError(t, err, errs.Validation(errs.FieldError{Field: "name", Message: c.message}))
	}
	if view, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{Name: strings.Repeat("n", 120), Levels: levels}); err != nil {
		t.Errorf("120 characters: %v, want saved", err)
	} else if view.ID != "WF-1003" {
		t.Errorf("id = %s, want WF-1003", view.ID)
	}
}

// Every problem of a workflow is reported at once, each on the field that fixes it: no name, a too-long
// description, a level without approvers, a too-long level name, a person listed twice, an inactive person
// added and a person who does not exist; a workflow without levels is refused too. Nothing is saved (FR-003).
func TestCreateWorkflow_AllProblemsAtOnce_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.deactivatePerson(t, sara)

	_, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{
		Name: "", Description: strings.Repeat("d", 2001),
		Levels: []LevelInput{
			{ApproverIDs: []string{}},
			{Name: strings.Repeat("l", 61), ApproverIDs: []string{daniel.ID, "U-9999", daniel.ID, sara.ID}},
		},
	})
	_, noLevel := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{Name: "No levels"})

	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "name", Message: "Enter a name."},
		errs.FieldError{Field: "description", Message: "Use 2,000 characters or fewer."},
		errs.FieldError{Field: "levels.1.approverIds", Message: "Level 1: choose at least one approver."},
		errs.FieldError{Field: "levels.2.name", Message: "Level 2: use 60 characters or fewer."},
		errs.FieldError{Field: "levels.2.approverIds", Message: "Level 2: Daniel Okoro is listed twice."},
		errs.FieldError{Field: "levels.2.approverIds", Message: "Level 2: Sara Novak is inactive — choose an active person."},
		errs.FieldError{Field: "levels.2.approverIds", Message: "Level 2: choose approvers from the list."},
	))
	assertError(t, noLevel, errs.Validation(errs.FieldError{Field: "levels", Message: "Add at least one level."}))
	if views := f.workflowViews(t, alex); len(views) != 2 {
		t.Errorf("%d workflows, want the 2 of the fixture: nothing saved", len(views))
	}
}

// Set inactive and Set active change the status from the version the person saw and are logged as
// deactivated and reactivated; the same status again changes nothing; an open round of an inactive
// workflow goes on (FR-007, FR-026).
func TestSetWorkflowActive_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	f.clock.Advance(time.Hour)

	inactive, err := f.service.SetWorkflowActive(t.Context(), alex, chinaFlow, false, version(1))
	if err != nil || inactive.Active || inactive.Version != 2 || inactive.WaitingCount != 1 {
		t.Fatalf("set inactive = %+v (%v), want inactive v2 with 1 waiting", inactive, err)
	}
	again, err := f.service.SetWorkflowActive(t.Context(), alex, chinaFlow, false, version(2))
	if err != nil || again.Version != 2 {
		t.Errorf("set inactive again = v%d (%v), want unchanged v2", again.Version, err)
	}
	f.approve(t, carlos, "A-1009", "") // the open round goes on
	_, stale := f.service.SetWorkflowActive(t.Context(), priya, chinaFlow, true, version(1))
	_, missing := f.service.SetWorkflowActive(t.Context(), priya, chinaFlow, true, nil)
	_, gone := f.service.SetWorkflowActive(t.Context(), priya, "WF-9999", true, version(1))
	f.clock.Advance(time.Hour)
	active, err := f.service.SetWorkflowActive(t.Context(), priya, chinaFlow, true, version(2))

	if err != nil || !active.Active || active.Version != 3 {
		t.Errorf("set active = %+v (%v), want active v3", active, err)
	}
	if round := f.openRound(t, "A-1009"); round.CurrentLevel != 2 {
		t.Errorf("round at level %d, want 2", round.CurrentLevel)
	}
	assertError(t, stale, errs.VersionConflict("approval workflow", "Alex Morgan", start.Add(time.Hour)))
	assertError(t, missing, errs.Validation(errs.FieldError{Field: "version", Message: "Reload the page and try again."}))
	assertError(t, gone, errs.NotFound("This approval workflow no longer exists — someone may have deleted it."))
	assertLog(t, f, chinaFlow, []string{
		"Priya Raman reactivated approval workflow 'China BC approval workflow' [false → true]",
		"Alex Morgan deactivated approval workflow 'China BC approval workflow' [true → false]",
	})
	entry := f.log(t, chinaFlow)[1]
	if c := entry.Changes[0]; c.Label != "Status" || c.OldText != "Active" || c.NewText != "Inactive" {
		t.Errorf("change = %+v, want Status Active → Inactive", c)
	}
}

// A workflow no activity uses moves to the recycle bin and can be undone (FR-008).
func TestDeleteWorkflow_Unused_FR008(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	created := f.createWorkflow(t, alex, "Temp flow", []string{carlos.ID})

	entry, err := f.service.DeleteWorkflow(t.Context(), priya, created.ID)
	_, gone := f.service.DeleteWorkflow(t.Context(), priya, created.ID)

	if err != nil || entry.RecordID != created.ID || entry.RecordTypeLabel != "approval workflow" || entry.RecordName != "Temp flow" {
		t.Fatalf("entry = %+v (%v), want the bin entry of %s", entry, err, created.ID)
	}
	if _, found := f.storedWorkflow(t, created.ID); found {
		t.Error("the workflow is still stored")
	}
	assertError(t, gone, errs.NotFound("This approval workflow no longer exists — someone may have deleted it."))
	if log := f.log(t, created.ID); len(log) != 2 || log[0].Action != history.ActionDeleted || log[0].Actor != priya {
		t.Errorf("log = %+v, want created, then deleted by Priya", log)
	}
	if _, err := f.history.Undo(t.Context(), priya, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if back, found := f.storedWorkflow(t, created.ID); !found || back.Name != "Temp flow" || back.Version != 1 {
		t.Errorf("after undo = %+v, want the workflow back exactly", back)
	}
}

// A workflow that a stored activity uses — in its Approval workflow field or through a round — cannot be
// deleted: the refusal says how many use it and suggests setting it inactive (FR-008).
func TestDeleteWorkflow_InUseRefused_FR008(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Launch film", priya.ID, chinaFlow)
	_, one := f.service.DeleteWorkflow(t.Context(), alex, chinaFlow)
	f.addActivity(t, "A-1002", "Fleet card launch", mei.ID, chinaFlow)
	_, two := f.service.DeleteWorkflow(t.Context(), alex, chinaFlow)
	f.addActivity(t, "A-1003", "Fleet roadshow", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1003", "")
	f.decline(t, daniel, "A-1003", "No.")
	moved := f.activities.facts["A-1003"]
	moved.WorkflowID = ""
	f.activities.facts["A-1003"] = moved
	_, throughRound := f.service.DeleteWorkflow(t.Context(), alex, lubricantsFlow)

	assertError(t, one, errs.InUse("Used by 1 activity — set it inactive instead."))
	assertError(t, two, errs.InUse("Used by 2 activities — set it inactive instead."))
	assertError(t, throughRound, errs.InUse("Used by 1 activity — set it inactive instead."))
	if _, found := f.storedWorkflow(t, chinaFlow); !found {
		t.Error("the used workflow was deleted")
	}
}
