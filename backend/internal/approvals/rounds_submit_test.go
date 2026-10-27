package approvals

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// The submit confirmation says what will happen to every level, and lists the problems of approver sources
// with every other problem; asking changes nothing (017 FR-012, FR-013, FR-036).
func TestSubmission_OutcomesAndProblems_FR012_FR013_FR036(t *testing.T) { //nolint:gocyclo // 14: three asks of one confirmation, each checked in full
	t.Parallel()
	f := newFixture(t)
	f.addStandardFlow(t)
	f.addActivity(t, "A-1001", "Launch film", priya.ID, standardFlow)
	f.setFacts(t, "A-1001", "120000.00", lubricantsUnit)
	f.activities.problems["A-1001"] = []SubmitProblem{
		{Field: "startDate", Group: GroupActivity, Message: "Start date — needed to submit"},
		{Field: "lineItems.L-1001.glCode", Group: GroupLineItem, LineItemID: "L-1001", Message: "GL code — needed to submit"},
	}

	blocked, err := f.service.Submission(t.Context(), priya, "A-1001")
	if err != nil {
		t.Fatalf("Submission: %v", err)
	}
	wantProblems := []SubmitProblem{
		{Field: "startDate", Group: GroupActivity, Message: "Start date — needed to submit"},
		{Field: workflowField, Group: GroupActivity, Message: "Level 1: Priya Raman has no line manager. Ask an administrator to set one in Settings › People."},
		{Field: "lineItems.L-1001.glCode", Group: GroupLineItem, LineItemID: "L-1001", Message: "GL code — needed to submit"},
	}
	if blocked.Ready || !reflect.DeepEqual(blocked.Problems, wantProblems) {
		t.Errorf("problems = %+v, want the line-manager problem among the others", blocked.Problems)
	}

	f.activities.problems["A-1001"] = nil
	f.setManager(t, priya, daniel)
	ready, err := f.service.Submission(t.Context(), priya, "A-1001")
	if err != nil {
		t.Fatalf("Submission: %v", err)
	}
	danielView, alexView := PersonView{ID: daniel.ID, Name: "Daniel Okoro", Initials: "DO", Active: true}, PersonView{ID: alex.ID, Name: "Alex Morgan", Initials: "AM", Active: true}
	wantOutcomes := []SubmissionLevel{
		{Number: 1, Label: "Line manager", Outcome: ResolutionDecide, Approvers: []PersonView{danielView}, Source: "line manager of Priya Raman",
			Mode: ModeAny, Excluded: []ExcludedView{}},
		{Number: 2, Label: "Marketing administrator", Outcome: ResolutionDecide, Approvers: []PersonView{alexView}, Mode: ModeAny, Excluded: []ExcludedView{}},
		{Number: 3, Label: "Mobility second check", Outcome: ResolutionSkipped, Approvers: []PersonView{}, Mode: ModeAll,
			RuleText: "only when business unit is Mobility", Excluded: []ExcludedView{}},
	}
	if !ready.Ready || ready.ApprovedAutomatically || !reflect.DeepEqual(ready.Outcomes, wantOutcomes) {
		t.Errorf("outcomes =\n%+v\nwant\n%+v", ready.Outcomes, wantOutcomes)
	}

	f.setFacts(t, "A-1001", "8000.00", lubricantsUnit)
	automatic, err := f.service.Submission(t.Context(), priya, "A-1001")
	if err != nil {
		t.Fatalf("Submission: %v", err)
	}
	if !automatic.ApprovedAutomatically || automatic.Outcomes[0].Outcome != ResolutionAutoApproved ||
		automatic.Outcomes[0].RuleText != "planned cost is below $10,000.00" || automatic.Outcomes[1].Outcome != ResolutionSkipped {
		t.Errorf("outcomes = %+v, want level 1 approved automatically and the activity approved", automatic.Outcomes)
	}
	if rounds := f.rounds(t, "A-1001"); len(rounds) != 0 {
		t.Errorf("rounds = %+v, want none: asking changes nothing", rounds)
	}
}

// The confirmation shows the workflow, each level with its approvers, a positive unassigned planned cost as
// a note and the number of the round it starts; asking changes nothing (FR-011, FR-010).
func TestSubmission_ListsLevelsAndNote_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	note := money.USD(mustAmount(t, "1200.00"))
	f.activities.unassigned["A-1009"] = &note

	submission, err := f.service.Submission(t.Context(), mei, "A-1009")

	if err != nil {
		t.Fatalf("Submission: %v", err)
	}
	want := Submission{
		ActivityID: "A-1009", ActivityName: "Fleet card launch", ActivityVersion: 1, Kind: "submit", Available: true,
		RoundNumber: 1, Ready: true, Problems: []SubmitProblem{},
		Workflow: &WorkflowRef{ID: chinaFlow, Name: "China BC approval workflow", Active: true},
		Levels: []LevelView{
			{Number: 1, Label: "Level 1", Approvers: []PersonView{{ID: carlos.ID, Name: "Carlos Mendes", Initials: "CM", Active: true}}, ApproverType: SourcePeople, Mode: ModeAny,
				Summary: "Level 1 · Carlos Mendes"},
			{Number: 2, Label: "Level 2", Approvers: []PersonView{{ID: alex.ID, Name: "Alex Morgan", Initials: "AM", Active: true}}, ApproverType: SourcePeople, Mode: ModeAny,
				Summary: "Level 2 · Alex Morgan"},
		},
		UnassignedPlannedCost: &note,
		Outcomes: []SubmissionLevel{ // 017 FR-013: both levels need a decision
			{Number: 1, Label: "Level 1", Outcome: ResolutionDecide, Approvers: []PersonView{{ID: carlos.ID, Name: "Carlos Mendes", Initials: "CM", Active: true}}, Mode: ModeAny, Excluded: []ExcludedView{}},
			{Number: 2, Label: "Level 2", Outcome: ResolutionDecide, Approvers: []PersonView{{ID: alex.ID, Name: "Alex Morgan", Initials: "AM", Active: true}}, Mode: ModeAny, Excluded: []ExcludedView{}},
		},
	}
	if !reflect.DeepEqual(submission, want) {
		t.Errorf("submission =\n%+v\nwant\n%+v", submission, want)
	}
	if rounds := f.rounds(t, "A-1009"); len(rounds) != 0 {
		t.Errorf("rounds = %+v, want none: asking changes nothing", rounds)
	}
}

// With any problem, Submit changes nothing and lists every problem at once, each with the field that fixes
// it: empty needed fields, an unfinished % split, a line item's missing field, a negative unassigned planned
// cost (from the check of 006 and 007) and the missing approval workflow (FR-010).
func TestSubmit_ProblemsChangeNothing_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1008", "Grand opening Shanghai", mei.ID, "")
	f.activities.problems["A-1008"] = []SubmitProblem{
		{Field: "ownerId", Group: GroupActivity, Message: "Owner — needed to submit"},
		{Field: "attributes.SA-1003", Group: GroupActivity, Message: "Country / OU split totals 90.00 % — it must total 100.00 %"},
		{Field: "lineItems", Group: GroupLineItem, Message: "Line items exceed the activity's planned cost by $1,200.00"},
		{Field: "lineItems.L-1003.glCode", Group: GroupLineItem, LineItemID: "L-1003",
			Message: "Line item 'Agency fee' (L-1003): GL code — needed to submit"},
	}

	submission, err := f.service.Submission(t.Context(), mei, "A-1008")
	if err != nil {
		t.Fatalf("Submission: %v", err)
	}
	_, submitErr := f.service.Submit(t.Context(), mei, "A-1008", SubmitInput{Version: version(1)})

	wantProblems := []SubmitProblem{
		{Field: "ownerId", Group: GroupActivity, Message: "Owner — needed to submit"},
		{Field: "attributes.SA-1003", Group: GroupActivity, Message: "Country / OU split totals 90.00 % — it must total 100.00 %"},
		{Field: "approvalWorkflowId", Group: GroupActivity, Message: "Approval workflow — needed to submit"},
		{Field: "lineItems", Group: GroupLineItem, Message: "Line items exceed the activity's planned cost by $1,200.00"},
		{Field: "lineItems.L-1003.glCode", Group: GroupLineItem, LineItemID: "L-1003",
			Message: "Line item 'Agency fee' (L-1003): GL code — needed to submit"},
	}
	if !reflect.DeepEqual(submission.Problems, wantProblems) || submission.Ready {
		t.Errorf("problems (ready %v) =\n%+v\nwant\n%+v", submission.Ready, submission.Problems, wantProblems)
	}
	var fields []errs.FieldError
	for _, p := range wantProblems {
		fields = append(fields, errs.FieldError{Field: p.Field, Message: p.Message})
	}
	assertError(t, submitErr, errs.Validation(fields...))
	assertNothingHappened(t, f, "A-1008", StatusCreated)
}

// An inactive approval workflow, or one that no longer exists, is a problem of the check; a problem the
// check of 006 already lists for the field is not listed twice (FR-010 e, FR-007).
func TestSubmit_InactiveWorkflowIsAProblem_FR010(t *testing.T) { //nolint:tparallel // the subtests share one fixture
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1002", "Social always-on", priya.ID, "WF-9999")
	f.addActivity(t, "A-1003", "Billboard winter wave", priya.ID, lubricantsFlow)
	f.activities.problems["A-1003"] = []SubmitProblem{
		{Field: "approvalWorkflowId", Group: GroupActivity, Message: "Approval workflow 'Lubricants EU approval' is no longer active — choose an active one"},
	}
	f.write(t, func(ctx context.Context) error {
		w, _, err := f.service.repo.GetWorkflow(ctx, lubricantsFlow)
		if err != nil {
			return err
		}
		w.Active = false
		return f.service.repo.PutWorkflow(ctx, w)
	})
	tests := []struct {
		activity, message string
	}{
		{"A-1001", "Approval workflow 'Lubricants EU approval' is inactive — choose an active one."},
		{"A-1002", "Approval workflow — needed to submit"},
		{"A-1003", "Approval workflow 'Lubricants EU approval' is no longer active — choose an active one"},
	}

	for _, tt := range tests {
		t.Run(tt.activity, func(t *testing.T) {
			_, err := f.service.Submit(t.Context(), priya, tt.activity, SubmitInput{Version: version(1)})

			assertError(t, err, errs.Validation(errs.FieldError{Field: "approvalWorkflowId", Message: tt.message}))
			assertNothingHappened(t, f, tt.activity, StatusCreated)
		})
	}
}

// A positive unassigned planned cost, or no line items at all, does not block submitting; the
// confirmation shows the amount as a note (spec edge case, domain model §4.3).
func TestSubmit_PositiveUnassignedDoesNotBlock_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	note := money.USD(mustAmount(t, "50000.00")) // the whole planned cost: no line items
	f.activities.unassigned["A-1001"] = &note

	approval := f.submit(t, priya, "A-1001", "")

	if approval.Status != StatusSubmitted {
		t.Errorf("status = %s, want submitted", approval.Status)
	}
}

// Only a Created activity can be submitted and only a Declined one re-submitted; the others are refused and
// the confirmation says why (FR-009).
func TestSubmit_OnlyCreatedOrDeclined_FR009(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for i, status := range []string{StatusSubmitted, StatusApproved} {
		id := []string{"A-1004", "A-1002"}[i]
		facts := f.addActivity(t, id, "Activity "+status, priya.ID, lubricantsFlow)
		facts.Status = status
		f.activities.facts[id] = facts

		submission, err := f.service.Submission(t.Context(), priya, id)
		if err != nil {
			t.Fatalf("Submission: %v", err)
		}
		_, submitErr := f.service.Submit(t.Context(), priya, id, SubmitInput{Version: version(1)})

		if submission.Available || submission.Reason != "Only Created or Declined activities can be submitted for approval." {
			t.Errorf("%s: available %v, reason %q", status, submission.Available, submission.Reason)
		}
		assertError(t, submitErr, errs.BadRequest("Only Created or Declined activities can be submitted for approval."))
		if rounds := f.rounds(t, id); len(rounds) != 0 {
			t.Errorf("%s: rounds = %+v, want none", status, rounds)
		}
	}
}

// Submit needs the activity version the person saw: without it the request is refused, and an out-of-date
// one is refused with version_conflict naming who changed the activity and when (D-030); a comment holds
// at most 2,000 characters.
func TestSubmit_StaleVersionRefused_D030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	changedAt := start.Add(-time.Hour)
	facts := f.activities.facts["A-1001"]
	facts.Touch(daniel, changedAt)
	f.activities.facts["A-1001"] = facts

	_, missing := f.service.Submit(t.Context(), priya, "A-1001", SubmitInput{Comment: strings.Repeat("é", 2001)})
	_, stale := f.service.Submit(t.Context(), priya, "A-1001", SubmitInput{Version: version(1)})
	_, gone := f.service.Submit(t.Context(), priya, "A-9999", SubmitInput{Version: version(1)})

	assertError(t, missing, errs.Validation(
		errs.FieldError{Field: "version", Message: "Reload the page and try again."},
		errs.FieldError{Field: "comment", Message: "Use 2,000 characters or fewer."},
	))
	assertError(t, stale, errs.VersionConflict("activity", "Daniel Okoro", changedAt))
	assertError(t, gone, errs.NotFound("This activity no longer exists — someone may have deleted it."))
	assertNothingHappened(t, f, "A-1001", StatusCreated)
}

// The plan's status is never checked: an activity of a Draft plan is submitted like any other (FR-012,
// D-206).
func TestSubmit_DraftPlanAllowed_FR012(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	facts := f.addActivity(t, "A-1010", "Lunar new year promo", mei.ID, chinaFlow)
	facts.Path[0] = errs.RecordRef{Type: "plan", ID: "P-1002", Name: "FY2026 Mobility China"} // a Draft plan
	f.activities.facts["A-1010"] = facts

	approval := f.submit(t, mei, "A-1010", "")

	if approval.Status != StatusSubmitted {
		t.Errorf("status = %s, want submitted", approval.Status)
	}
}
