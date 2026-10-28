package approval

import (
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

func (f *fixture) recall(t *testing.T, a actor.Actor, id, reason string) ActivityApproval {
	t.Helper()
	approval, err := f.service.Recall(t.Context(), a, id, RecallInput{RoundID: f.openRound(t, id).ID, Reason: reason})
	if err != nil {
		t.Fatalf("Recall(%s): %v", id, err)
	}
	return approval
}

// An Approved activity can't be recalled: the refusal says that a change of its planned cost or dates sends
// it back for approval — line-item changes no longer do (034 FR-005) — and nothing changes (FR-021).
func TestRecall_ApprovedRefused_FR021(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	round := f.openRound(t, "A-1004")
	approval := f.approve(t, daniel, "A-1004", "")
	entries, told := len(f.log(t, "A-1004")), len(f.notifications(t, daniel.ID))

	_, err := f.service.Recall(t.Context(), priya, "A-1004", RecallInput{RoundID: round.ID, Reason: "Changed my mind."})

	reason := "Approved activities can't be recalled. Changing its planned cost or dates sends it back for approval."
	assertError(t, err, errs.BadRequest(reason))
	if approval.Actions.Recall.Available || approval.Actions.Recall.Reason != reason {
		t.Errorf("recall action = %+v, want unavailable with the reason", approval.Actions.Recall)
	}
	if f.activities.facts["A-1004"].Status != StatusApproved || f.rounds(t, "A-1004")[0].Outcome != OutcomeApproved ||
		len(f.log(t, "A-1004")) != entries || len(f.notifications(t, daniel.ID)) != told {
		t.Error("the refused recall changed something")
	}
}
