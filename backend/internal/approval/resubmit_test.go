package approval

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/errs"
)

// roundState is what the re-submit tests compare of a round.
type roundState struct {
	Number   int
	Outcome  Outcome
	Workflow string
	Current  int
	States   []string
	History  []string
}

func stateOf(round RoundView) roundState {
	state := roundState{Number: round.Number, Outcome: round.Outcome, Workflow: round.Workflow.ID, Current: round.CurrentLevel}
	for _, l := range round.Levels {
		state.States = append(state.States, l.State)
	}
	for _, h := range round.History {
		state.History = append(state.History, h.Action)
	}
	return state
}

func TestResubmitSubmitted(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-017 FR-020 the open round ends as superseded and a new round waits", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
		f.submit(t, mei, "A-1009", "")
		f.approve(t, carlos, "A-1009", "Fine by me.")
		facts := f.activities.facts["A-1009"]
		facts.WorkflowID = lubricantsFlow
		f.activities.facts["A-1009"] = facts

		approval := f.submit(t, mei, "A-1009", "Moved to the Lubricants workflow.")

		got := []roundState{stateOf(approval.Rounds[0]), stateOf(approval.Rounds[1])}
		want := []roundState{
			{Number: 2, Outcome: OutcomeOpen, Workflow: lubricantsFlow, Current: 1, States: []string{LevelWaiting}, History: []string{"resubmitted"}},
			{Number: 1, Outcome: OutcomeSuperseded, Workflow: chinaFlow, Current: 2, States: []string{LevelApproved, LevelNotDecided},
				History: []string{"superseded", "approved", "submitted"}},
		}
		if diff := cmp.Diff(want, got); diff != "" || approval.Status != StatusSubmitted {
			t.Errorf("status %s, rounds (-want +got):\n%s", approval.Status, diff)
		}
		assertLog(t, f, "A-1009", []string{
			`Mei Lin re-submitted activity 'Fleet card launch' "Moved to the Lubricants workflow." (Round 2 · Lubricants EU approval · replaces round 1 · waiting for level 1 (Daniel Okoro))`,
			`Carlos Mendes approved level 1 of activity 'Fleet card launch' "Fine by me." (Round 1 · level 1 of 2 · now waiting for level 2 (Alex Morgan))`,
			"Mei Lin submitted activity 'Fleet card launch' [created → submitted] (Round 1 · China BC approval workflow · waiting for level 1 (Carlos Mendes))",
		})
		assertNotified(t, f, daniel, "approvalNeeded: Approval needed — Mei Lin re-submitted 'Fleet card launch' (A-1009)")
	})
	t.Run("031 FR-017 a problem changes nothing", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
		f.submit(t, mei, "A-1009", "")
		facts := f.activities.facts["A-1009"]
		facts.WorkflowID = ""
		f.activities.facts["A-1009"] = facts

		_, err := f.service.Submit(t.Context(), mei, "A-1009", SubmitInput{Version: version(facts.Version)})

		assertError(t, err, errs.Validation(errs.FieldError{Field: workflowField, Message: workflowNeededMessage}))
		if rounds := f.rounds(t, "A-1009"); len(rounds) != 1 || rounds[0].Outcome != OutcomeOpen {
			t.Errorf("rounds = %+v, want round 1 still open", rounds)
		}
	})
}

func TestResubmitApprovedAtSubmission(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-020 a new round needing no decision approves the activity at once", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
		f.submit(t, mei, "A-1009", "")
		f.addStandardFlow(t)
		f.setFacts(t, "A-1009", "8000.00", lubricantsUnit)
		facts := f.activities.facts["A-1009"]
		facts.WorkflowID = standardFlow
		f.activities.facts["A-1009"] = facts

		approval := f.submit(t, mei, "A-1009", "")

		got := []Outcome{approval.Rounds[0].Outcome, approval.Rounds[1].Outcome}
		if approval.Status != StatusApproved || !slices.Equal(got, []Outcome{OutcomeApproved, OutcomeSuperseded}) {
			t.Errorf("status %s, outcomes %v; want Approved with round 1 superseded", approval.Status, got)
		}
	})
}

func TestDecisionOnReplacedRound(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-019 a decision on a replaced round is refused, saying who re-submitted", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
		f.submit(t, priya, "A-1004", "")
		first := f.openRound(t, "A-1004")
		f.submit(t, priya, "A-1004", "")

		_, err := f.service.Approve(t.Context(), daniel, "A-1004", DecisionInput{RoundID: first.ID, Level: version(1)})

		assertError(t, err, decidedConflict("The submission was re-submitted by Priya Raman", "Priya Raman", start))
	})
}

func TestSubmissionReplaces(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-018 the confirmation names the round that ends and its level", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
		f.submit(t, mei, "A-1009", "")
		f.approve(t, carlos, "A-1009", "")

		submission, err := f.service.Submission(t.Context(), mei, "A-1009")

		if err != nil {
			t.Fatalf("Submission: %v", err)
		}
		got := []any{submission.Kind, submission.Available, submission.RoundNumber, submission.Replaces}
		want := []any{kindResubmit, true, 2, &ReplacedRound{RoundNumber: 1, Level: 2, LevelLabel: "Level 2"}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("submission (-want +got):\n%s", diff)
		}
	})
}
