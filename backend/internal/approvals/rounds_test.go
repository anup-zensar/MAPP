package approvals

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

// Submitting a ready Created activity, in one go: it becomes Submitted with a new version, round 1 starts at
// level 1 with a copy of the workflow's levels, the submitter, the time and the comment; the round's history,
// the activity log and the level-1 approvers' notifications record it (FR-011, FR-026, FR-035).
func TestSubmit_StartsRound1_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)

	approval := f.submit(t, mei, "A-1009", "  Ready for your review.  ")

	if facts := f.activities.facts["A-1009"]; facts.Status != StatusSubmitted || facts.Version != 2 || facts.UpdatedBy != mei {
		t.Errorf("activity = %s, version %d, changed by %v; want submitted, version 2, by Mei Lin", facts.Status, facts.Version, facts.UpdatedBy)
	}
	wantRound := Round{
		ID: "AR-1001", ActivityID: "A-1009", Number: 1, WorkflowID: chinaFlow, WorkflowName: "China BC approval workflow",
		Levels:       []RoundLevel{decideLevel(1, carlos), decideLevel(2, alex)},
		CurrentLevel: 1, Outcome: OutcomeOpen, SubmittedAt: start, SubmittedBy: mei, Comment: "Ready for your review.",
		Facts: fixtureFacts(t), Meta: record.NewMeta(mei, start),
	}
	wantRound.Levels[0].StartedAt = &start // 017 FR-029: level 1 waits from the submission
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

// Re-submitting a Declined activity starts the next round at level 1 with the workflow's levels as they
// are now, not those of the earlier round (FR-020, FR-006).
func TestResubmit_NewRoundWithCurrentLevels_FR020(t *testing.T) {
	t.Parallel()
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
	level := decideLevel(1, daniel, sara)
	level.Name, level.StartedAt = "Team leaders", &rounds[1].SubmittedAt
	if len(rounds) != 2 || rounds[0].Levels[0].Name != "" || rounds[1].Number != 2 || rounds[1].CurrentLevel != 1 ||
		!reflect.DeepEqual(rounds[1].Levels, []RoundLevel{level}) {
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

// The owner of an activity is left out of its levels and may not decide it while 014's access rule is on
// (017 FR-011, FR-018); a person on two levels decides each level separately; an inactive approver stays
// on the chain, shown inactive, and someone allowed to decide on behalf decides for them (spec edge cases,
// 017 FR-017).
func TestDecide_OwnerAndTwoLevelsAndInactiveApprover_017FR018_EdgeCases(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", daniel.ID, lubricantsFlow)
	f.addWorkflow(t, "WF-1003", "Carlos twice", []string{carlos.ID}, []string{carlos.ID})
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, "WF-1003")
	f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	for _, id := range []string{"A-1004", "A-1009", "A-1001"} {
		f.submit(t, priya, id, "")
	}

	t.Run("owner is left out and refused", func(t *testing.T) {
		before, err := f.service.Approval(t.Context(), daniel, "A-1004")
		if err != nil {
			t.Fatalf("Approval: %v", err)
		}
		_, err = f.service.Approve(t.Context(), daniel, "A-1004", DecisionInput{RoundID: f.openRound(t, "A-1004").ID, Level: version(1)})
		level := before.Rounds[0].Levels[0]
		if !before.Viewer.IsOwner || before.Viewer.IsCurrentApprover || before.Viewer.CanDecide || !level.NoApprover ||
			len(level.Excluded) != 1 || level.Excluded[0].Reason != ReasonOwner {
			t.Errorf("viewer %+v, level %+v; want the owner left out, the level with no approver", before.Viewer, level)
		}
		if before.Actions.Approve != (ActionAvailability{Reason: "You own this activity, so someone else must approve it."}) {
			t.Errorf("approve = %+v, want unavailable with the owner message", before.Actions.Approve)
		}
		assertError(t, err, errs.Forbidden("You own this activity, so someone else must approve it."))
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
			p, _, err := people.NewJSONStore().Get(ctx, daniel.ID)
			if err != nil {
				return err
			}
			p.Active = false
			return people.NewJSONStore().Put(ctx, p)
		})
		before, err := f.service.Approval(t.Context(), priya, "A-1001")
		if err != nil {
			t.Fatalf("Approval: %v", err)
		}
		approval := f.approve(t, alex, "A-1001", "")
		if approvers := before.Rounds[0].Levels[0].Approvers; len(approvers) != 1 || approvers[0].Active || !before.Rounds[0].Levels[0].NoApprover {
			t.Errorf("chain approvers = %+v, want Daniel Okoro shown inactive and no approver", approvers)
		}
		if decision := approval.Rounds[0].Levels[0].Decision; !reflect.DeepEqual(decision.OnBehalfOf, []actor.Actor{daniel}) ||
			decision.Capacity != CapacityOnBehalf {
			t.Errorf("decision = %+v, want Alex Morgan on behalf of Daniel Okoro", decision)
		}
	})
}

// Recalling a Submitted activity ends its round as recalled at the level it waited at, with the reason;
// the activity is Created again, can be submitted again and leaves the approvers' Waiting for me; the
// history, the log and the owner and waiting approvers record it. Anyone may recall (D-020). Recalling the
// same round again explains what ended it; the reason holds at most 2,000 characters (FR-021, FR-018).
func TestRecall_FR021(t *testing.T) {
	t.Parallel()
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

// chainStates writes a round's chain as "L1 approved, L2 waiting".
func chainStates(r RoundView) string {
	var states []string
	for _, l := range r.Levels {
		states = append(states, "L"+string(rune('0'+l.Number))+" "+l.State)
	}
	return strings.Join(states, ", ")
}
