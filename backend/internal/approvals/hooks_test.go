package approvals

import (
	"context"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// restart does what features 006 and 007 do when a re-approval change is saved: in one Write, the activity
// id goes back to Created and approvals is told, as a, with changes.
func (f *fixture) restart(t *testing.T, a actor.Actor, id string, changes ...history.Change) {
	t.Helper()
	previous := f.activities.facts[id].Status
	f.write(t, func(ctx context.Context) error {
		if _, err := f.activities.SetStatus(ctx, a, id, StatusCreated); err != nil {
			return err
		}
		return f.service.ApprovalRestarted(ctx, a, id, previous, changes)
	})
}

// A re-approval change on a Submitted activity ends its open round as restarted at the level it waited
// at, with the changed values; the activity's owner and the waiting level's approvers are told, the person
// who changed it is not, and approvals writes no log entry of its own: the saving feature did (FR-022,
// FR-035, FR-036).
func TestApprovalRestarted_Submitted_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	f.approve(t, carlos, "A-1009", "")
	f.clock.Advance(time.Hour)
	restartedAt := f.clock.Now()
	entries := len(f.log(t, "A-1009"))

	f.restart(t, priya, "A-1009", history.Change{Field: "plannedCost", Label: "Planned cost", OldText: "$50,000.00", NewText: "$65,000.00"})

	round := f.rounds(t, "A-1009")[0]
	if round.Outcome != OutcomeRestarted || round.CurrentLevel != 2 || round.EndedAt == nil || !round.EndedAt.Equal(restartedAt) || round.UpdatedBy != priya {
		t.Errorf("round = %+v, want restarted at level 2 now, by Priya Raman", round)
	}
	wantEntry := Action{
		ID: "AH-1003", RoundID: round.ID, RoundNumber: 1, ActivityID: "A-1009", Action: ActionRestarted, Level: 2,
		Approvers: []actor.Actor{alex}, Actor: priya, OnBehalfOf: []actor.Actor{}, At: restartedAt,
		Changes: []ChangeText{{Label: "Planned cost", OldText: "$50,000.00", NewText: "$65,000.00"}},
	}
	if got := f.actions(t, "A-1009")[2]; !reflect.DeepEqual(got, wantEntry) {
		t.Errorf("history entry =\n%+v\nwant\n%+v", got, wantEntry)
	}
	approval, err := f.service.Approval(t.Context(), mei, "A-1009")
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}
	latest := approval.Rounds[0].History[0]
	if latest.Action != "restarted" || latest.LevelLabel != "Level 2" || len(latest.Changes) != 1 ||
		latest.Changes[0].Text != "Planned cost changed from $50,000.00 to $65,000.00" {
		t.Errorf("latest history entry = %+v, want the restart with its change sentence", latest)
	}
	if states := chainStates(approval.Rounds[0]); states != "L1 approved, L2 notDecided" || !approval.Actions.Submit.Available {
		t.Errorf("chain = %s, actions %+v; want L1 approved, L2 notDecided, submit again", states, approval.Actions)
	}
	text := "Priya Raman changed 'Fleet card launch' (A-1009) — it needs approval again"
	for _, p := range []actor.Actor{mei, alex} {
		if got := f.notifications(t, p.ID); got[0].Kind != NotifyRestarted || got[0].Title != "Approval restarted" || got[0].Text != text {
			t.Errorf("%s's latest notification = %+v, want the restart", p.Name, got[0])
		}
	}
	if got := f.notifications(t, carlos.ID); len(got) != 1 {
		t.Errorf("Carlos's notifications = %+v, want only the submission: his level was not waiting", got)
	}
	if got := f.notifications(t, priya.ID); len(got) != 0 {
		t.Errorf("Priya's notifications = %+v, want none: she made the change", got)
	}
	if got := len(f.log(t, "A-1009")); got != entries {
		t.Errorf("%d log entries, want still %d", got, entries)
	}
}

// A re-approval change on an Approved activity keeps its approved round approved and records the restart
// on it without a level, with the change as 007's ready-made sentence; the owner and the people who
// approved the round are told — here the owner made the change, so only the approvers (FR-022, D-161). An
// activity without a round, or one that was not in approval, is left alone.
func TestApprovalRestarted_Approved_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.addActivity(t, "A-1008", "Grand opening Shanghai", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	f.approve(t, carlos, "A-1009", "")
	f.approve(t, sara, "A-1009", "") // on behalf of Alex Morgan
	f.clock.Advance(time.Hour)
	reason := "Line item 'Agency fee' (L-1003) added: 1,200.00 USD"

	f.restart(t, mei, "A-1009", history.Change{Label: reason})
	f.restart(t, mei, "A-1008") // Created, never submitted

	round := f.rounds(t, "A-1009")[0]
	if round.Outcome != OutcomeApproved || round.Version != 3 {
		t.Errorf("round = %s, version %d; want it still approved and unchanged", round.Outcome, round.Version)
	}
	last := f.actions(t, "A-1009")[3]
	if last.Action != ActionRestarted || last.Level != 0 || last.LevelName != "" || len(last.Approvers) != 0 || last.Actor != mei ||
		!reflect.DeepEqual(last.Changes, []ChangeText{{Label: reason}}) {
		t.Errorf("history entry = %+v, want a restart without a level with the reason", last)
	}
	approval, err := f.service.Approval(t.Context(), mei, "A-1009")
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}
	if latest := approval.Rounds[0].History[0]; latest.LevelLabel != "" || latest.Changes[0].Text != reason ||
		chainStates(approval.Rounds[0]) != "L1 approved, L2 approved" {
		t.Errorf("latest history entry = %+v, chain %s", latest, chainStates(approval.Rounds[0]))
	}
	text := "Mei Lin changed 'Fleet card launch' (A-1009) — it needs approval again"
	for _, p := range []actor.Actor{carlos, sara} {
		if got := f.notifications(t, p.ID); len(got) == 0 || got[0].Kind != NotifyRestarted || got[0].Text != text {
			t.Errorf("%s's notifications = %+v, want the restart on top", p.Name, got)
		}
	}
	if got := f.notifications(t, alex.ID); len(got) != 1 || got[0].Kind != NotifyApprovalNeeded {
		t.Errorf("Alex's notifications = %+v: he was asked, but Sara Novak approved on his behalf", got)
	}
	for _, n := range f.notifications(t, mei.ID) {
		if n.Kind == NotifyRestarted {
			t.Errorf("Mei was told about her own change: %+v", n)
		}
	}
	if rounds, actions := f.rounds(t, "A-1008"), f.actions(t, "A-1008"); len(rounds)+len(actions) != 0 {
		t.Errorf("A-1008: rounds %+v, history %+v; want none", rounds, actions)
	}
}

// A restore from the recycle bin brings a deleted Submitted activity back Created: the round that was open
// at the delete ends as cancelled at its waiting level, dated at the delete and by the person who deleted
// it, with a note for the "restored" log entry; nobody is told (FR-023, 008-P3).
func TestActivityRestored_CancelsOpenRound_FR023(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	f.clock.Advance(time.Hour)
	deletedAt := f.clock.Now()
	delete(f.activities.facts, "A-1004") // in the recycle bin: not stored
	f.clock.Advance(24 * time.Hour)
	rc := history.RestoreContext{Actor: daniel, Now: f.clock.Now(), DeletedAt: deletedAt, DeletedBy: mei}
	told := len(f.notifications(t, daniel.ID)) + len(f.notifications(t, priya.ID))

	var notes, again []string
	f.write(t, func(ctx context.Context) (err error) {
		if notes, err = f.service.ActivityRestored(ctx, rc, "A-1004"); err != nil {
			return err
		}
		again, err = f.service.ActivityRestored(ctx, rc, "A-1004") // no round is open any more
		return err
	})

	want := []string{"Approval round 1 was cancelled because the activity was deleted — submit it for approval again."}
	if !reflect.DeepEqual(notes, want) || len(again) != 0 {
		t.Errorf("notes = %q then %q, want %q then none", notes, again, want)
	}
	round := f.rounds(t, "A-1004")[0]
	if round.Outcome != OutcomeCancelled || round.EndedAt == nil || !round.EndedAt.Equal(deletedAt) || round.UpdatedBy != daniel {
		t.Errorf("round = %+v, want cancelled at the delete time, changed by Daniel Okoro", round)
	}
	last := f.actions(t, "A-1004")[1]
	if last.Action != ActionCancelled || last.Level != 1 || last.Actor != mei || !last.At.Equal(deletedAt) {
		t.Errorf("history entry = %+v, want cancelled at level 1 by Mei Lin at the delete time", last)
	}
	if now := len(f.notifications(t, daniel.ID)) + len(f.notifications(t, priya.ID)); now != told {
		t.Errorf("%d notifications, want still %d", now, told)
	}
}
