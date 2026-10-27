package approvals

import (
	"reflect"
	"testing"
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
