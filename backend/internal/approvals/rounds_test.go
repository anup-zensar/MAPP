package approvals

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
)

// The confirmation shows the workflow, each level with its approvers, a positive unassigned planned cost as
// a note and the number of the round it starts; asking changes nothing (FR-011, FR-010).
func TestSubmission_ListsLevelsAndNote_FR011(t *testing.T) {
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
			{Number: 1, Label: "Level 1", Approvers: []PersonView{{ID: carlos.ID, Name: "Carlos Mendes", Initials: "CM", Active: true}}},
			{Number: 2, Label: "Level 2", Approvers: []PersonView{{ID: alex.ID, Name: "Alex Morgan", Initials: "AM", Active: true}}},
		},
		UnassignedPlannedCost: &note,
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
func TestSubmit_InactiveWorkflowIsAProblem_FR010(t *testing.T) {
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
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	note := money.USD(mustAmount(t, "50000.00")) // the whole planned cost: no line items
	f.activities.unassigned["A-1001"] = &note

	approval := f.submit(t, priya, "A-1001", "")

	if approval.Status != StatusSubmitted {
		t.Errorf("status = %s, want submitted", approval.Status)
	}
}

// Submitting a ready Created activity, in one go: it becomes Submitted with a new version, round 1 starts at
// level 1 with a copy of the workflow's levels, the submitter, the time and the comment; the round's history,
// the activity log and the level-1 approvers' notifications record it (FR-011, FR-026, FR-035).
func TestSubmit_StartsRound1_FR011(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)

	approval := f.submit(t, mei, "A-1009", "  Ready for your review.  ")

	if facts := f.activities.facts["A-1009"]; facts.Status != StatusSubmitted || facts.Version != 2 || facts.UpdatedBy != mei {
		t.Errorf("activity = %s, version %d, changed by %v; want submitted, version 2, by Mei Lin", facts.Status, facts.Version, facts.UpdatedBy)
	}
	wantRound := Round{
		ID: "AR-1001", ActivityID: "A-1009", Number: 1, WorkflowID: chinaFlow, WorkflowName: "China BC approval workflow",
		Levels: []RoundLevel{
			{Number: 1, Approvers: []actor.Actor{carlos}},
			{Number: 2, Approvers: []actor.Actor{alex}},
		},
		CurrentLevel: 1, Outcome: OutcomeOpen, SubmittedAt: start, SubmittedBy: mei, Comment: "Ready for your review.",
		Meta: record.NewMeta(mei, start),
	}
	if rounds := f.rounds(t, "A-1009"); len(rounds) != 1 || !reflect.DeepEqual(rounds[0], wantRound) {
		t.Errorf("rounds =\n%+v\nwant\n%+v", rounds, wantRound)
	}
	wantAction := Action{
		ID: "AH-1001", RoundID: "AR-1001", RoundNumber: 1, ActivityID: "A-1009", Action: ActionSubmitted, Level: 1,
		Approvers: []actor.Actor{carlos}, Actor: mei, OnBehalfOf: []actor.Actor{}, At: start, Comment: "Ready for your review.",
		Changes: []ChangeText{},
	}
	if actions := f.actions(t, "A-1009"); len(actions) != 1 || !reflect.DeepEqual(actions[0], wantAction) {
		t.Errorf("history =\n%+v\nwant\n%+v", actions, wantAction)
	}
	assertLog(t, f, "A-1009", []string{
		"Mei Lin submitted activity 'Fleet card launch' [created → submitted] " +
			`"Ready for your review." (Round 1 · China BC approval workflow · waiting for level 1 (Carlos Mendes))`,
	})
	wantNotification := Notification{
		ID: "NT-1001", RecipientID: carlos.ID, Kind: NotifyApprovalNeeded, Title: "Approval needed",
		Text: "Mei Lin submitted 'Fleet card launch' (A-1009)", ActivityID: "A-1009", ActivityName: "Fleet card launch",
		Actor: mei, At: start,
	}
	if got := f.notifications(t, carlos.ID); len(got) != 1 || !reflect.DeepEqual(got[0], wantNotification) {
		t.Errorf("Carlos's notifications =\n%+v\nwant\n%+v", got, wantNotification)
	}
	if got := f.notifications(t, mei.ID); len(got) != 0 {
		t.Errorf("Mei's notifications = %+v, want none: nobody is told about their own action", got)
	}

	if approval.Status != StatusSubmitted || approval.ActivityVersion != 2 || len(approval.Rounds) != 1 {
		t.Fatalf("approval = %+v, want submitted, version 2, one round", approval)
	}
	if states := chainStates(approval.Rounds[0]); states != "L1 waiting, L2 notReached" {
		t.Errorf("chain = %s, want L1 waiting, L2 notReached", states)
	}
}

// Only a Created activity can be submitted and only a Declined one re-submitted; the others are refused and
// the confirmation says why (FR-009).
func TestSubmit_OnlyCreatedOrDeclined_FR009(t *testing.T) {
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
	f := newFixture(t)
	facts := f.addActivity(t, "A-1010", "Lunar new year promo", mei.ID, chinaFlow)
	facts.Path[0] = errs.RecordRef{Type: "plan", ID: "P-1002", Name: "FY2026 Mobility China"} // a Draft plan
	f.activities.facts["A-1010"] = facts

	approval := f.submit(t, mei, "A-1010", "")

	if approval.Status != StatusSubmitted {
		t.Errorf("status = %s, want submitted", approval.Status)
	}
}

// Re-submitting a Declined activity starts the next round at level 1 with the workflow's levels as they
// are now, not those of the earlier round (FR-020, FR-006).
func TestResubmit_NewRoundWithCurrentLevels_FR020(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	f.declineDirectly(t, "A-1004", daniel, "Please split the agency fee by country before re-submitting.")
	f.write(t, func(ctx context.Context) error { // the workflow now has Sara Novak beside Daniel Okoro
		w, _, err := f.service.repo.GetWorkflow(ctx, lubricantsFlow)
		if err != nil {
			return err
		}
		w.Levels = []Level{{Name: "Team leaders", ApproverIDs: []string{daniel.ID, sara.ID}}}
		return f.service.repo.PutWorkflow(ctx, w)
	})
	f.clock.Advance(time.Hour)

	submission, err := f.service.Submission(t.Context(), priya, "A-1004")
	if err != nil {
		t.Fatalf("Submission: %v", err)
	}
	approval := f.submit(t, priya, "A-1004", "Split by country.")

	if submission.Kind != "resubmit" || submission.RoundNumber != 2 || !submission.Available {
		t.Errorf("submission: kind %s, round %d, available %v; want resubmit, 2, true", submission.Kind, submission.RoundNumber, submission.Available)
	}
	rounds := f.rounds(t, "A-1004")
	if len(rounds) != 2 || rounds[0].Levels[0].Name != "" || rounds[1].Number != 2 || rounds[1].CurrentLevel != 1 ||
		!reflect.DeepEqual(rounds[1].Levels, []RoundLevel{{Number: 1, Name: "Team leaders", Approvers: []actor.Actor{daniel, sara}}}) {
		t.Errorf("rounds = %+v, want round 1 unchanged and round 2 with the current levels", rounds)
	}
	actions := f.actions(t, "A-1004")
	if last := actions[len(actions)-1]; last.Action != ActionResubmitted || last.RoundNumber != 2 {
		t.Errorf("last history entry = %+v, want resubmitted in round 2", last)
	}
	if approval.Status != StatusSubmitted || len(approval.Rounds) != 2 || approval.Rounds[0].Number != 2 {
		t.Errorf("approval: %s with %d rounds, want submitted with round 2 first", approval.Status, len(approval.Rounds))
	}
	assertLog(t, f, "A-1004", []string{
		`Priya Raman re-submitted activity 'Trade magazine ads' [declined → submitted] "Split by country." ` +
			`(Round 2 · Lubricants EU approval · waiting for level 1 (Daniel Okoro, Sara Novak))`,
		"Priya Raman submitted activity 'Trade magazine ads' [created → submitted] (Round 1 · Lubricants EU approval · waiting for level 1 (Daniel Okoro))",
	})
	if got := f.notifications(t, sara.ID); len(got) != 1 || got[0].Text != "Priya Raman re-submitted 'Trade magazine ads' (A-1004)" {
		t.Errorf("Sara's notifications = %+v, want the re-submission", got)
	}
}

// declineDirectly ends the open round of the activity id as declined by a, as a decline does, without
// the service: the tests of submitting do not depend on deciding.
func (f *fixture) declineDirectly(t *testing.T, id string, by actor.Actor, comment string) {
	t.Helper()
	f.endDirectly(t, id, ActionDeclined, OutcomeDeclined, StatusDeclined, by, comment)
}

// endDirectly ends the open round of the activity id with outcome, by a history entry kind of by at the
// waiting level, and gives the activity status: the ends of a round that features 006, 007 and the
// recall of this feature bring about.
func (f *fixture) endDirectly(t *testing.T, id string, kind ActionKind, outcome Outcome, status string, by actor.Actor, comment string) {
	t.Helper()
	f.write(t, func(ctx context.Context) error {
		rounds, err := f.service.repo.RoundsOf(ctx, id)
		if err != nil {
			return err
		}
		round := rounds[len(rounds)-1]
		now := f.clock.Now()
		round.Outcome, round.EndedAt = outcome, &now
		round.Touch(by, now)
		if err := f.service.repo.PutRound(ctx, round); err != nil {
			return err
		}
		if err := f.service.addAction(ctx, round, kind, round.CurrentLevel, by, []actor.Actor{}, now, comment); err != nil {
			return err
		}
		_, err = f.activities.SetStatus(ctx, by, id, status)
		return err
	})
}

// approve approves the current level of the activity's open round as a, and stops the test when that fails.
func (f *fixture) approve(t *testing.T, a actor.Actor, id, comment string) ActivityApproval {
	t.Helper()
	round := f.openRound(t, id)
	approval, err := f.service.Approve(t.Context(), a, id, DecisionInput{RoundID: round.ID, Level: version(round.CurrentLevel), Comment: comment})
	if err != nil {
		t.Fatalf("Approve(%s): %v", id, err)
	}
	return approval
}

// decline declines the activity at the current level of its open round as a, and stops the test when that
// fails.
func (f *fixture) decline(t *testing.T, a actor.Actor, id, comment string) ActivityApproval {
	t.Helper()
	round := f.openRound(t, id)
	approval, err := f.service.Decline(t.Context(), a, id, DecisionInput{RoundID: round.ID, Level: version(round.CurrentLevel), Comment: comment})
	if err != nil {
		t.Fatalf("Decline(%s): %v", id, err)
	}
	return approval
}

// openRound returns the open round of the activity id, and stops the test when there is none.
func (f *fixture) openRound(t *testing.T, id string) Round {
	t.Helper()
	round, found := openRound(f.rounds(t, id))
	if !found {
		t.Fatalf("%s has no open round", id)
	}
	return round
}

// addWorkflow stores a workflow with the given levels of approvers, active, and returns its id.
func (f *fixture) addWorkflow(t *testing.T, id, name string, levels ...[]string) string {
	t.Helper()
	workflow := Workflow{ID: id, Name: name, Active: true, Levels: []Level{}, Meta: record.NewMeta(actor.System, start)}
	for _, approvers := range levels {
		workflow.Levels = append(workflow.Levels, Level{ApproverIDs: approvers})
	}
	f.write(t, func(ctx context.Context) error { return f.service.repo.PutWorkflow(ctx, workflow) })
	return id
}

// Approving a level other than the last moves the round to the next level and keeps the activity
// Submitted; the next level's approvers are asked and the owner is told (FR-015, FR-035).
func TestApprove_MovesToNextLevel_FR015(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	f.clock.Advance(time.Hour)
	approvedAt := f.clock.Now()

	approval := f.approve(t, carlos, "A-1009", " Looks good. ")

	if facts := f.activities.facts["A-1009"]; facts.Status != StatusSubmitted || facts.Version != 2 {
		t.Errorf("activity = %s, version %d; want still submitted, version 2", facts.Status, facts.Version)
	}
	round := f.openRound(t, "A-1009")
	if round.CurrentLevel != 2 || round.Version != 2 || round.UpdatedBy != carlos || round.EndedAt != nil {
		t.Errorf("round = %+v, want open at level 2, changed by Carlos Mendes", round)
	}
	if states := chainStates(approval.Rounds[0]); states != "L1 approved, L2 waiting" {
		t.Errorf("chain = %s, want L1 approved, L2 waiting", states)
	}
	wantDecision := &Decision{Action: "approved", Actor: carlos, OnBehalfOf: []actor.Actor{}, At: approvedAt, Comment: "Looks good."}
	if got := approval.Rounds[0].Levels[0].Decision; !reflect.DeepEqual(got, wantDecision) {
		t.Errorf("decision = %+v, want %+v", got, wantDecision)
	}
	assertLog(t, f, "A-1009", []string{
		`Carlos Mendes approved level 1 of activity 'Fleet card launch' "Looks good." ` +
			`(Round 1 · level 1 of 2 · now waiting for level 2 (Alex Morgan))`,
		"Mei Lin submitted activity 'Fleet card launch' [created → submitted] (Round 1 · China BC approval workflow · waiting for level 1 (Carlos Mendes))",
	})
	assertNotified(t, f, alex, "approvalNeeded: Approval needed — Carlos Mendes approved level 1 of 'Fleet card launch' (A-1009) — level 2 waits for you")
	assertNotified(t, f, mei, "levelApproved: Level 1 approved — Carlos Mendes approved level 1 of 2 of 'Fleet card launch' (A-1009)")
	assertNotified(t, f, carlos, "approvalNeeded: Approval needed — Mei Lin submitted 'Fleet card launch' (A-1009)")
}

// Approving the last level ends the round as approved and makes the activity Approved; the owner is told
// (FR-015, FR-035).
func TestApprove_LastLevelApproves_FR015(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	f.approve(t, carlos, "A-1009", "")
	round := f.openRound(t, "A-1009")
	f.clock.Advance(time.Hour)
	approvedAt := f.clock.Now()

	approval := f.approve(t, alex, "A-1009", "")

	if facts := f.activities.facts["A-1009"]; facts.Status != StatusApproved || facts.Version != 3 || facts.UpdatedBy != alex {
		t.Errorf("activity = %s, version %d, by %v; want approved, version 3, by Alex Morgan", facts.Status, facts.Version, facts.UpdatedBy)
	}
	ended := f.rounds(t, "A-1009")[0]
	if ended.ID != round.ID || ended.Outcome != OutcomeApproved || ended.CurrentLevel != 2 || ended.EndedAt == nil || !ended.EndedAt.Equal(approvedAt) {
		t.Errorf("round = %+v, want approved at level 2, ended now", ended)
	}
	if states := chainStates(approval.Rounds[0]); states != "L1 approved, L2 approved" {
		t.Errorf("chain = %s, want both levels approved", states)
	}
	if approval.Status != StatusApproved || approval.Actions.Approve.Available || approval.Actions.Recall.Available {
		t.Errorf("approval: %s, actions %+v; want approved without approve or recall", approval.Status, approval.Actions)
	}
	if entries := f.log(t, "A-1009"); logLine(entries[0]) != "Alex Morgan approved activity 'Fleet card launch' [submitted → approved] (Round 1 · level 2 of 2)" {
		t.Errorf("latest log entry = %s", logLine(entries[0]))
	}
	assertNotified(t, f, mei,
		"approved: Activity approved — Alex Morgan approved 'Fleet card launch' (A-1009)",
		"levelApproved: Level 1 approved — Carlos Mendes approved level 1 of 2 of 'Fleet card launch' (A-1009)")
}

// Every level applies to every submission, in order, and any one designated approver of a level approves
// it (FR-004, D-205).
func TestApprove_EveryLevelAppliesAnyApprover_FR004(t *testing.T) {
	f := newFixture(t)
	f.addWorkflow(t, "WF-1003", "Two team leaders, then the champion", []string{daniel.ID, sara.ID}, []string{carlos.ID})
	f.addActivity(t, "A-1001", "Launch film", priya.ID, "WF-1003")
	f.submit(t, priya, "A-1001", "")

	approval := f.approve(t, sara, "A-1001", "")

	if approval.Status != StatusSubmitted || chainStates(approval.Rounds[0]) != "L1 approved, L2 waiting" {
		t.Errorf("approval: %s, chain %s; want still submitted, waiting at level 2", approval.Status, chainStates(approval.Rounds[0]))
	}
	if decision := approval.Rounds[0].Levels[0].Decision; len(decision.OnBehalfOf) != 0 {
		t.Errorf("Sara Novak approved on behalf of %v; she is an approver of the level", decision.OnBehalfOf)
	}
	assertNotified(t, f, carlos, "approvalNeeded: Approval needed — Sara Novak approved level 1 of 'Launch film' (A-1001) — level 2 waits for you")
	assertNotified(t, f, daniel, "approvalNeeded: Approval needed — Priya Raman submitted 'Launch film' (A-1001)")
}

// Declining needs a comment that is not blank; comments hold at most 2,000 characters. A refused
// decision changes nothing (FR-016).
func TestDecline_CommentRequired_FR016(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	round := f.openRound(t, "A-1004")
	level := version(1)

	_, blank := f.service.Decline(t.Context(), daniel, "A-1004", DecisionInput{RoundID: round.ID, Level: level, Comment: "  \n "})
	_, long := f.service.Decline(t.Context(), daniel, "A-1004", DecisionInput{RoundID: round.ID, Level: level, Comment: strings.Repeat("x", 2001)})
	_, longApproval := f.service.Approve(t.Context(), daniel, "A-1004", DecisionInput{RoundID: round.ID, Level: level, Comment: strings.Repeat("x", 2001)})

	assertError(t, blank, errs.Validation(errs.FieldError{Field: "comment", Message: "Add a comment explaining why the activity is declined."}))
	assertError(t, long, errs.Validation(errs.FieldError{Field: "comment", Message: "Use 2,000 characters or fewer."}))
	assertError(t, longApproval, errs.Validation(errs.FieldError{Field: "comment", Message: "Use 2,000 characters or fewer."}))
	if facts, actions := f.activities.facts["A-1004"], f.actions(t, "A-1004"); facts.Status != StatusSubmitted || len(actions) != 1 {
		t.Errorf("activity %s with %d history entries, want still submitted with only the submission", facts.Status, len(actions))
	}
}

// A decline ends the round at any level and makes the activity Declined; the comment heads its Approval
// section and the owner is told with the comment's first 80 characters (FR-016, FR-035).
func TestDecline_EndsRoundAtAnyLevel_FR016(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.addActivity(t, "A-1008", "Grand opening Shanghai", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	f.submit(t, mei, "A-1008", "")
	f.approve(t, carlos, "A-1009", "")
	f.clock.Advance(time.Hour)
	comment := "The planned cost is above this quarter's budget — please reduce it or move part of it to Q3."

	atLevel2 := f.decline(t, alex, "A-1009", comment)
	atLevel1 := f.decline(t, carlos, "A-1008", "Not this year.")

	if chainStates(atLevel2.Rounds[0]) != "L1 approved, L2 declined" || chainStates(atLevel1.Rounds[0]) != "L1 declined, L2 notReached" {
		t.Errorf("chains = %s / %s", chainStates(atLevel2.Rounds[0]), chainStates(atLevel1.Rounds[0]))
	}
	ended := f.rounds(t, "A-1009")[0]
	if ended.Outcome != OutcomeDeclined || ended.CurrentLevel != 2 || ended.EndedAt == nil || f.activities.facts["A-1009"].Status != StatusDeclined {
		t.Errorf("round %+v, activity %s; want declined at level 2", ended, f.activities.facts["A-1009"].Status)
	}
	wantNote := &DeclineNote{RoundNumber: 1, Level: 2, LevelLabel: "Level 2", Actor: alex, OnBehalfOf: []actor.Actor{}, At: f.clock.Now(), Comment: comment}
	if !reflect.DeepEqual(atLevel2.LatestDecline, wantNote) {
		t.Errorf("latest decline = %+v, want %+v", atLevel2.LatestDecline, wantNote)
	}
	if !atLevel2.Actions.Resubmit.Available || atLevel2.Actions.Decline.Available {
		t.Errorf("actions = %+v, want re-submit only", atLevel2.Actions)
	}
	if entries := f.log(t, "A-1009"); logLine(entries[0]) != "Alex Morgan declined activity 'Fleet card launch' [submitted → declined] \""+comment+"\" (Round 1 · level 2 of 2)" {
		t.Errorf("latest log entry = %s", logLine(entries[0]))
	}
	notifications := f.notifications(t, mei.ID)
	want := `Alex Morgan declined 'Fleet card launch' (A-1009): "The planned cost is above this quarter's budget — please reduce it or move part …"`
	if notifications[1].Kind != NotifyDeclined || notifications[1].Title != "Activity declined" || notifications[1].Text != want {
		t.Errorf("Mei's notification = %+v, want the decline quoting the comment's first 80 characters", notifications[1])
	}
}

// Anyone may decide (D-020); a person who is not an approver of the level decides on behalf of its
// approvers, and the confirmation, the history and the log say so (FR-017).
func TestDecide_OnBehalfOf_FR017(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	before, err := f.service.Approval(t.Context(), alex, "A-1004")
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}

	approval := f.approve(t, alex, "A-1004", "Daniel is on leave.")

	danielView := PersonView{ID: daniel.ID, Name: "Daniel Okoro", Initials: "DO", Active: true}
	if before.Viewer.IsCurrentApprover || !reflect.DeepEqual(before.Viewer.OnBehalfOf, []PersonView{danielView}) {
		t.Errorf("viewer = %+v, want Alex deciding on behalf of Daniel Okoro", before.Viewer)
	}
	entry := approval.Rounds[0].History[0]
	if entry.Actor != alex || !reflect.DeepEqual(entry.OnBehalfOf, []actor.Actor{daniel}) || entry.Action != "approved" ||
		entry.LevelLabel != "Level 1" || !reflect.DeepEqual(entry.Approvers, []PersonView{danielView}) {
		t.Errorf("history entry = %+v, want Alex Morgan on behalf of Daniel Okoro at level 1", entry)
	}
	if entries := f.log(t, "A-1004"); logLine(entries[0]) != "Alex Morgan approved activity 'Trade magazine ads' [submitted → approved] "+
		`"Daniel is on leave." (Round 1 · level 1 of 1; on behalf of Daniel Okoro)` {
		t.Errorf("latest log entry = %s", logLine(entries[0]))
	}
}

// The owner may approve a level they are an approver of, and is not told about their own decision; a
// person on two levels decides each level separately; an inactive approver stays on the chain, shown
// inactive, and others decide on their behalf (spec edge cases, D-020).
func TestDecide_OwnerAndTwoLevelsAndInactiveApprover_EdgeCases(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", daniel.ID, lubricantsFlow)
	f.addWorkflow(t, "WF-1003", "Carlos twice", []string{carlos.ID}, []string{carlos.ID})
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, "WF-1003")
	f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	for _, id := range []string{"A-1004", "A-1009", "A-1001"} {
		f.submit(t, priya, id, "")
	}

	t.Run("owner approves", func(t *testing.T) {
		before, err := f.service.Approval(t.Context(), daniel, "A-1004")
		if err != nil {
			t.Fatalf("Approval: %v", err)
		}
		approval := f.approve(t, daniel, "A-1004", "")
		if !before.Viewer.IsOwner || !before.Viewer.IsCurrentApprover || approval.Status != StatusApproved {
			t.Errorf("viewer %+v, status %s; want owner and approver, then approved", before.Viewer, approval.Status)
		}
		for _, n := range f.notifications(t, daniel.ID) {
			if n.Kind == NotifyApproved {
				t.Errorf("Daniel Okoro was told about his own approval: %+v", n)
			}
		}
	})
	t.Run("one person on two levels", func(t *testing.T) {
		first := f.approve(t, carlos, "A-1009", "")
		second := f.approve(t, carlos, "A-1009", "")
		if chainStates(first.Rounds[0]) != "L1 approved, L2 waiting" || second.Status != StatusApproved {
			t.Errorf("after level 1: %s; after level 2: %s", chainStates(first.Rounds[0]), second.Status)
		}
	})
	t.Run("inactive approver", func(t *testing.T) {
		f.write(t, func(ctx context.Context) error {
			p, _, err := f.service.people.Get(ctx, daniel.ID)
			if err != nil {
				return err
			}
			p.Active = false
			return f.service.people.Put(ctx, p)
		})
		before, err := f.service.Approval(t.Context(), priya, "A-1001")
		if err != nil {
			t.Fatalf("Approval: %v", err)
		}
		approval := f.approve(t, priya, "A-1001", "")
		if approvers := before.Rounds[0].Levels[0].Approvers; len(approvers) != 1 || approvers[0].Active {
			t.Errorf("chain approvers = %+v, want Daniel Okoro shown inactive", approvers)
		}
		if decision := approval.Rounds[0].Levels[0].Decision; !reflect.DeepEqual(decision.OnBehalfOf, []actor.Actor{daniel}) {
			t.Errorf("decision = %+v, want Priya Raman on behalf of Daniel Okoro", decision)
		}
	})
}

// Only the first decision on a level counts: a later Approve or Decline of a level that was decided
// changes nothing and says who decided it and when; a level the round has not reached is a stale request
// (FR-018, D-155).
func TestDecide_FirstDecisionCounts_FR018(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	round := f.openRound(t, "A-1009")
	f.clock.Advance(time.Hour)
	decidedAt := f.clock.Now()
	f.approve(t, carlos, "A-1009", "")
	f.clock.Advance(time.Minute)

	_, approveAgain := f.service.Approve(t.Context(), sara, "A-1009", DecisionInput{RoundID: round.ID, Level: version(1)})
	_, declineAgain := f.service.Decline(t.Context(), sara, "A-1009", DecisionInput{RoundID: round.ID, Level: version(1), Comment: "No."})
	_, ahead := f.service.Approve(t.Context(), sara, "A-1009", DecisionInput{RoundID: round.ID, Level: version(3)})

	already := decidedConflict("Level 1 was already approved by Carlos Mendes", "Carlos Mendes", decidedAt)
	assertError(t, approveAgain, already)
	assertError(t, declineAgain, already)
	assertError(t, ahead, errs.Validation(errs.FieldError{Field: "level", Message: "Reload the page and try again."}))
	if actions := f.actions(t, "A-1009"); len(actions) != 2 {
		t.Errorf("history = %+v, want the submission and one approval", actions)
	}
}

// A decision on a round that has ended explains what ended it: a decline, a recall, a restart or the
// activity's deletion; a round of another activity, or none, is refused (FR-018).
func TestDecide_EndedRoundExplains_FR018(t *testing.T) {
	tests := []struct {
		name    string
		end     func(f *fixture, id string)
		message string
	}{
		{"declined", func(f *fixture, id string) { f.decline(t, daniel, id, "Too expensive.") },
			"Level 1 was already declined by Daniel Okoro"},
		{"recalled", func(f *fixture, id string) {
			f.endDirectly(t, id, ActionRecalled, OutcomeRecalled, StatusCreated, priya, "Wrong workflow.")
		}, "The submission was recalled by Priya Raman"},
		{"restarted", func(f *fixture, id string) {
			f.endDirectly(t, id, ActionRestarted, OutcomeRestarted, StatusCreated, mei, "")
		}, "Approval was restarted by Mei Lin"},
		{"cancelled", func(f *fixture, id string) {
			f.endDirectly(t, id, ActionCancelled, OutcomeCancelled, StatusCreated, alex, "")
		}, "The approval round was cancelled because the activity was deleted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
			f.submit(t, priya, "A-1004", "")
			round := f.openRound(t, "A-1004")
			f.clock.Advance(time.Hour)
			endedAt := f.clock.Now()
			tt.end(f, "A-1004")
			f.clock.Advance(time.Minute)

			_, err := f.service.Approve(t.Context(), alex, "A-1004", DecisionInput{RoundID: round.ID, Level: version(1)})

			last := f.actions(t, "A-1004")[1]
			assertError(t, err, decidedConflict(tt.message, last.Actor.Name, endedAt))
		})
	}

	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	other := f.openRound(t, "A-1004")
	_, otherRound := f.service.Approve(t.Context(), daniel, "A-1001", DecisionInput{RoundID: other.ID, Level: version(1)})
	_, noRound := f.service.Approve(t.Context(), daniel, "A-1004", DecisionInput{RoundID: "AR-9999", Level: version(1)})
	_, missing := f.service.Decline(t.Context(), daniel, "A-1004", DecisionInput{})
	assertError(t, otherRound, errs.NotFound("This approval round doesn't belong to this activity. Reload the page and try again."))
	assertError(t, noRound, errs.NotFound("This approval round doesn't belong to this activity. Reload the page and try again."))
	assertError(t, missing, errs.Validation(
		errs.FieldError{Field: "roundId", Message: "Reload the page and try again."},
		errs.FieldError{Field: "level", Message: "Reload the page and try again."},
		errs.FieldError{Field: "comment", Message: "Add a comment explaining why the activity is declined."},
	))
}

// In progress and Complete are displayed statuses of an approved activity: time passing changes them
// without any action, so nobody is ever notified of them (FR-019).
func TestDisplayStatusesNeverNotify_FR019(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	f.approve(t, daniel, "A-1004", "")
	told := len(f.notifications(t, priya.ID)) + len(f.notifications(t, daniel.ID))
	facts := f.activities.facts["A-1004"]
	facts.DisplayStatus = "complete" // the dates have passed
	f.activities.facts["A-1004"] = facts
	f.clock.Advance(200 * 24 * time.Hour)

	approval, err := f.service.Approval(t.Context(), priya, "A-1004")

	if err != nil || approval.Status != StatusApproved || approval.DisplayStatus != "complete" {
		t.Fatalf("approval = %s / %s (%v), want approved, shown complete", approval.Status, approval.DisplayStatus, err)
	}
	if now := len(f.notifications(t, priya.ID)) + len(f.notifications(t, daniel.ID)); now != told {
		t.Errorf("%d notifications, want still %d", now, told)
	}
}

// Every approval action is in the round's history and the activity log with its round, level, designated
// approvers, actor and "on behalf of" (FR-026, SC-004).
func TestApprovalActions_Logged_FR026_SC004(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "First try.")
	f.approve(t, alex, "A-1009", "")                   // level 1, on behalf of Carlos Mendes
	f.decline(t, alex, "A-1009", "Split the cost.")    // level 2, Alex's own level
	f.submit(t, mei, "A-1009", "Split.")               // round 2
	f.approve(t, carlos, "A-1009", "")                 // level 1
	f.approve(t, sara, "A-1009", "Approved for Alex.") // level 2, on behalf of Alex Morgan

	var got []string
	for _, e := range f.actions(t, "A-1009") {
		line := fmt.Sprintf("round %d level %d (%s) %s by %s", e.RoundNumber, e.Level, names(e.Approvers), e.Action, e.Actor.Name)
		if len(e.OnBehalfOf) > 0 {
			line += " on behalf of " + names(e.OnBehalfOf)
		}
		got = append(got, line)
	}
	want := []string{
		"round 1 level 1 (Carlos Mendes) submitted by Mei Lin",
		"round 1 level 1 (Carlos Mendes) approved by Alex Morgan on behalf of Carlos Mendes",
		"round 1 level 2 (Alex Morgan) declined by Alex Morgan",
		"round 2 level 1 (Carlos Mendes) resubmitted by Mei Lin",
		"round 2 level 1 (Carlos Mendes) approved by Carlos Mendes",
		"round 2 level 2 (Alex Morgan) approved by Sara Novak on behalf of Alex Morgan",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("history =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	assertLog(t, f, "A-1009", []string{
		`Sara Novak approved activity 'Fleet card launch' [submitted → approved] "Approved for Alex." (Round 2 · level 2 of 2; on behalf of Alex Morgan)`,
		"Carlos Mendes approved level 1 of activity 'Fleet card launch' (Round 2 · level 1 of 2 · now waiting for level 2 (Alex Morgan))",
		`Mei Lin re-submitted activity 'Fleet card launch' [declined → submitted] "Split." (Round 2 · China BC approval workflow · waiting for level 1 (Carlos Mendes))`,
		`Alex Morgan declined activity 'Fleet card launch' [submitted → declined] "Split the cost." (Round 1 · level 2 of 2)`,
		"Alex Morgan approved level 1 of activity 'Fleet card launch' (Round 1 · level 1 of 2 · now waiting for level 2 (Alex Morgan); on behalf of Carlos Mendes)",
		`Mei Lin submitted activity 'Fleet card launch' [created → submitted] "First try." (Round 1 · China BC approval workflow · waiting for level 1 (Carlos Mendes))`,
	})
}

// recall recalls the submission of the activity id as a, naming its open round, and stops the test when
// that fails.
func (f *fixture) recall(t *testing.T, a actor.Actor, id, reason string) ActivityApproval {
	t.Helper()
	approval, err := f.service.Recall(t.Context(), a, id, RecallInput{RoundID: f.openRound(t, id).ID, Reason: reason})
	if err != nil {
		t.Fatalf("Recall(%s): %v", id, err)
	}
	return approval
}

// Recalling a Submitted activity ends its round as recalled at the level it waited at, with the reason;
// the activity is Created again, can be submitted again and leaves the approvers' Waiting for me; the
// history, the log and the owner and waiting approvers record it. Anyone may recall (D-020). Recalling the
// same round again explains what ended it; the reason holds at most 2,000 characters (FR-021, FR-018).
func TestRecall_FR021(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	f.approve(t, carlos, "A-1009", "")
	round := f.openRound(t, "A-1009")
	f.clock.Advance(time.Hour)
	recalledAt := f.clock.Now()
	_, tooLongReason := f.service.Recall(t.Context(), priya, "A-1009", RecallInput{RoundID: round.ID, Reason: strings.Repeat("x", 2001)})
	_, noRound := f.service.Recall(t.Context(), priya, "A-1009", RecallInput{})

	approval := f.recall(t, priya, "A-1009", "  Wrong workflow.  ")

	assertError(t, tooLongReason, errs.Validation(errs.FieldError{Field: "reason", Message: "Use 2,000 characters or fewer."}))
	assertError(t, noRound, errs.Validation(errs.FieldError{Field: "roundId", Message: "Reload the page and try again."}))
	if facts := f.activities.facts["A-1009"]; facts.Status != StatusCreated || facts.UpdatedBy != priya {
		t.Errorf("activity = %s, changed by %v; want Created by Priya Raman", facts.Status, facts.UpdatedBy)
	}
	ended := f.rounds(t, "A-1009")[0]
	if ended.Outcome != OutcomeRecalled || ended.CurrentLevel != 2 || ended.EndedAt == nil || !ended.EndedAt.Equal(recalledAt) || ended.UpdatedBy != priya {
		t.Errorf("round = %+v, want recalled at level 2 now, by Priya Raman", ended)
	}
	last := f.actions(t, "A-1009")[2]
	if last.Action != ActionRecalled || last.Level != 2 || !reflect.DeepEqual(last.Approvers, []actor.Actor{alex}) || last.Actor != priya ||
		last.Comment != "Wrong workflow." || !last.At.Equal(recalledAt) {
		t.Errorf("history entry = %+v, want recalled at level 2 by Priya Raman with the reason", last)
	}
	if states := chainStates(approval.Rounds[0]); states != "L1 approved, L2 notDecided" || approval.Rounds[0].Outcome != OutcomeRecalled {
		t.Errorf("chain = %s (%s), want L1 approved, L2 notDecided, recalled", states, approval.Rounds[0].Outcome)
	}
	if !approval.Actions.Submit.Available || approval.Actions.Recall.Available {
		t.Errorf("actions = %+v, want submit again, no recall", approval.Actions)
	}
	if entries := f.log(t, "A-1009"); logLine(entries[0]) !=
		`Priya Raman recalled activity 'Fleet card launch' [submitted → created] "Wrong workflow." (Round 1 · was waiting for level 2 (Alex Morgan))` {
		t.Errorf("latest log entry = %s", logLine(entries[0]))
	}
	assertNotified(t, f, alex, "recalled: Submission recalled — Priya Raman recalled 'Fleet card launch' (A-1009)",
		"approvalNeeded: Approval needed — Carlos Mendes approved level 1 of 'Fleet card launch' (A-1009) — level 2 waits for you")
	if got := f.notifications(t, mei.ID); len(got) != 2 || got[0].Kind != NotifyRecalled {
		t.Errorf("Mei's notifications = %+v, want the recall on top", got)
	}
	if waiting := f.inboxIDs(t, alex, TabWaiting); len(waiting) != 0 {
		t.Errorf("Alex's waiting = %q, want none", waiting)
	}

	f.clock.Advance(time.Minute)
	_, again := f.service.Recall(t.Context(), mei, "A-1009", RecallInput{RoundID: round.ID})
	assertError(t, again, decidedConflict("The submission was recalled by Priya Raman", "Priya Raman", recalledAt))
	_, otherRound := f.service.Recall(t.Context(), mei, "A-1009", RecallInput{RoundID: "AR-9999"})
	assertError(t, otherRound, errs.NotFound("This approval round doesn't belong to this activity. Reload the page and try again."))
}

// An Approved activity can't be recalled: the refusal says that a change of planned cost, dates or
// line-item amounts sends it back for approval, and nothing changes (FR-021).
func TestRecall_ApprovedRefused_FR021(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	round := f.openRound(t, "A-1004")
	approval := f.approve(t, daniel, "A-1004", "")
	entries, told := len(f.log(t, "A-1004")), len(f.notifications(t, daniel.ID))

	_, err := f.service.Recall(t.Context(), priya, "A-1004", RecallInput{RoundID: round.ID, Reason: "Changed my mind."})

	reason := "Approved activities can't be recalled. Changing its planned cost, dates or line-item amounts sends it back for approval."
	assertError(t, err, errs.BadRequest(reason))
	if approval.Actions.Recall.Available || approval.Actions.Recall.Reason != reason {
		t.Errorf("recall action = %+v, want unavailable with the reason", approval.Actions.Recall)
	}
	if f.activities.facts["A-1004"].Status != StatusApproved || f.rounds(t, "A-1004")[0].Outcome != OutcomeApproved ||
		len(f.log(t, "A-1004")) != entries || len(f.notifications(t, daniel.ID)) != told {
		t.Error("the refused recall changed something")
	}
}

// assertNotified checks the notifications of person p, newest first, each written as
// "<kind>: <title> — <text>".
func assertNotified(t *testing.T, f *fixture, p actor.Actor, want ...string) {
	t.Helper()
	var got []string
	for _, n := range f.notifications(t, p.ID) {
		got = append(got, fmt.Sprintf("%s: %s — %s", n.Kind, n.Title, n.Text))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s's notifications =\n%s\nwant\n%s", p.Name, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// assertNothingHappened checks that the activity id still has status, no round, no history entry, no log
// entry and that nobody was notified.
func assertNothingHappened(t *testing.T, f *fixture, id, status string) {
	t.Helper()
	if got := f.activities.facts[id].Status; got != status {
		t.Errorf("%s: status = %s, want %s", id, got, status)
	}
	if rounds, actions, entries := f.rounds(t, id), f.actions(t, id), f.log(t, id); len(rounds)+len(actions)+len(entries) > 0 {
		t.Errorf("%s: rounds %+v, history %+v, log %+v; want none", id, rounds, actions, entries)
	}
	for _, p := range []actor.Actor{alex, priya, daniel, mei, carlos, sara} {
		if got := f.notifications(t, p.ID); len(got) > 0 {
			t.Errorf("%s was notified: %+v", p.Name, got)
		}
	}
}

// assertLog checks the activity log of the record id, newest first, each entry written as
// "<actor> <verb> <type> '<name>' [<old> → <new>] "<comment>" (<details>)".
func assertLog(t *testing.T, f *fixture, id string, want []string) {
	t.Helper()
	var got []string
	for _, e := range f.log(t, id) {
		got = append(got, logLine(e))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("log of %s =\n%s\nwant\n%s", id, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// logLine writes one activity-log entry as assertLog compares it.
func logLine(e history.Entry) string {
	line := e.Actor.Name + " " + e.Verb + " " + e.RecordTypeLabel + " '" + e.RecordName + "'"
	for _, c := range e.Changes {
		var oldValue, newValue any
		_ = json.Unmarshal(c.Old, &oldValue)
		_ = json.Unmarshal(c.New, &newValue)
		line += " [" + text(oldValue) + " → " + text(newValue) + "]"
	}
	if e.Comment != "" {
		line += ` "` + e.Comment + `"`
	}
	if e.Details != "" {
		line += " (" + e.Details + ")"
	}
	return line
}

// text writes a decoded JSON value for logLine.
func text(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	encoded, _ := json.Marshal(v)
	return string(encoded)
}

// chainStates writes a round's chain as "L1 approved, L2 waiting".
func chainStates(r RoundView) string {
	var states []string
	for _, l := range r.Levels {
		states = append(states, "L"+string(rune('0'+l.Number))+" "+l.State)
	}
	return strings.Join(states, ", ")
}
