package approval

import (
	"testing"
	"time"

	"mapp/internal/platform/errs"
)

// Only the first decision on a level counts: a later Approve or Decline of a level that was decided
// changes nothing and says who decided it and when; a level the round has not reached is a stale request
// (FR-018, D-155).
func TestDecide_FirstDecisionCounts_FR018(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	tests := []struct {
		name    string
		end     func(t *testing.T, f *fixture, id string)
		message string
	}{
		{"declined", func(t *testing.T, f *fixture, id string) { t.Helper(); f.decline(t, daniel, id, "Too expensive.") },
			"Level 1 was already declined by Daniel Okoro"},
		{"recalled", func(t *testing.T, f *fixture, id string) {
			t.Helper()
			f.endDirectly(t, id, roundEnd{kind: ActionRecalled, outcome: OutcomeRecalled, status: StatusCreated, by: priya, comment: "Wrong workflow."})
		}, "The submission was recalled by Priya Raman"},
		{"restarted", func(t *testing.T, f *fixture, id string) {
			t.Helper()
			f.endDirectly(t, id, roundEnd{kind: ActionRestarted, outcome: OutcomeRestarted, status: StatusCreated, by: mei})
		}, "Approval was restarted by Mei Lin"},
		{"cancelled", func(t *testing.T, f *fixture, id string) {
			t.Helper()
			f.endDirectly(t, id, roundEnd{kind: ActionCancelled, outcome: OutcomeCancelled, status: StatusCreated, by: alex})
		}, "The approval round was cancelled because the activity was deleted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
			f.submit(t, priya, "A-1004", "")
			round := f.openRound(t, "A-1004")
			f.clock.Advance(time.Hour)
			endedAt := f.clock.Now()
			tt.end(t, f, "A-1004")
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
