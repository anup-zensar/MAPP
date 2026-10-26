package approvals

import (
	"context"
	"reflect"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Who may decide, step by step (017 FR-017, FR-018, FR-020, SC-002, data-model §6): a waiting approver who
// can approve; a current delegate of one; the backup approver once the level is escalated; someone allowed
// to decide on behalf; an approver who already approved is told who is still waiting; anyone else is
// refused with 014's message 5 (and no reason shown). While 014's rule is on, the owner and the round's
// submitter are refused in every capacity, delegates included; with it off they may decide.
func TestAuthority_Table_FR017_FR018_FR020_SC002(t *testing.T) {
	t.Parallel()
	escalated := start.Add(time.Hour)
	level := RoundLevel{Number: 1, Approvers: []actor.Actor{daniel, carlos}, ApproverType: SourcePeople, Mode: ModeAll,
		Resolution: ResolutionDecide, Excluded: []Excluded{}, Backup: &mei}
	round := Round{ID: "AR-1001", ActivityID: "A-1001", Number: 1, Levels: []RoundLevel{level}, CurrentLevel: 1, Outcome: OutcomeOpen,
		SubmittedBy: priya}
	escalatedRound := round
	escalatedRound.Levels = []RoundLevel{level}
	escalatedRound.Levels[0].EscalatedAt = &escalated
	noApprover := round
	noApprover.Levels = []RoundLevel{{Number: 1, Approvers: []actor.Actor{}, Mode: ModeAny, Resolution: ResolutionDecide, NoApprover: true}}
	danielApproved := []Action{{Level: 1, Action: ActionApproved, Actor: daniel, Capacity: CapacityApprover}}
	onlyApprovers := "Only the approvers of level 1 (Daniel Okoro, Carlos Mendes) can approve or decline this activity."

	tests := []struct {
		name        string
		who         actor.Actor
		round       Round
		entries     []Action
		owner       string
		ruleOff     bool
		delegations []Delegation
		capacity    Capacity
		refusal     string
		shown       bool // the refusal is shown as the actions' reason
	}{
		{name: "approver", who: daniel, round: round, capacity: CapacityApprover},
		{name: "approver who cannot approve", who: carlos, round: round, refusal: onlyApprovers},
		{name: "delegate", who: sara, round: round, delegations: []Delegation{{PersonID: daniel.ID, DelegateID: sara.ID}}, capacity: CapacityDelegate},
		{name: "backup before escalation", who: mei, round: round, refusal: onlyApprovers},
		{name: "backup after escalation", who: mei, round: escalatedRound, capacity: CapacityBackup},
		{name: "on behalf", who: alex, round: round, capacity: CapacityOnBehalf},
		{name: "already approved", who: daniel, round: round, entries: danielApproved,
			refusal: "You already approved level 1. Waiting for Carlos Mendes.", shown: true},
		{name: "anyone else", who: sara, round: round, refusal: onlyApprovers},
		{name: "nobody to approve", who: sara, round: noApprover,
			refusal: "Only the approvers of level 1 (nobody) can approve or decline this activity."},
		{name: "owner", who: daniel, round: round, owner: daniel.ID, refusal: "You own this activity, so someone else must approve it.", shown: true},
		{name: "owner, rule off", who: daniel, round: round, owner: daniel.ID, ruleOff: true, capacity: CapacityApprover},
		{name: "submitter", who: priya, round: round, refusal: "You submitted this activity, so someone else must approve it.", shown: true},
		{name: "submitter on behalf, rule off", who: priya, round: round, ruleOff: true, capacity: CapacityOnBehalf},
		{name: "delegate who owns", who: sara, round: round, owner: sara.ID, delegations: []Delegation{{PersonID: daniel.ID, DelegateID: sara.ID}},
			refusal: "You own this activity, so someone else must approve it.", shown: true},
		{name: "owner on behalf", who: alex, round: round, owner: alex.ID, refusal: "You own this activity, so someone else must approve it.", shown: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			f.access.cannotApprove[carlos.ID] = true
			f.access.onBehalf = []string{alex.ID, priya.ID}
			f.access.rule = !tt.ruleOff
			facts := ActivityFacts{ID: "A-1001", OwnerID: tt.owner}
			var au authority
			f.read(t, func(ctx context.Context) error {
				dc, err := f.service.deciderOf(ctx, tt.who, tt.delegations)
				if err != nil {
					return err
				}
				au, err = f.service.authority(ctx, dc, facts, tt.round, tt.entries, true)
				return err
			})
			if tt.refusal == "" {
				if !au.allowed || au.capacity != tt.capacity || au.refusal != nil || au.shown != "" {
					t.Errorf("authority = %+v, want allowed as %s", au, tt.capacity)
				}
				return
			}
			if au.allowed || !reflect.DeepEqual(au.refusal, errs.Forbidden(tt.refusal)) {
				t.Errorf("authority = %+v, want refused: %s", au, tt.refusal)
			}
			if wantShown := map[bool]string{true: tt.refusal, false: ""}[tt.shown]; au.shown != wantShown {
				t.Errorf("shown = %q, want %q", au.shown, wantShown)
			}
		})
	}
}

// A person may see an activity because of its approval (017 FR-021, extends 014 FR-006): named as an
// approver in any round, a current delegate of a waiting approver, or the backup approver of its escalated
// level — not a planned delegate, nor a backup before escalation.
func TestNamedIn_DelegatesAndBackup_FR021(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.putWorkflow(t, "WF-1003", "Backed up", days(1), days(2), Level{ApproverIDs: []string{daniel.ID}, BackupApproverID: alex.ID})
	f.addActivity(t, "A-1001", "Launch film", priya.ID, "WF-1003")
	f.submit(t, priya, "A-1001", "")
	f.delegate(t, "DLG-1001", daniel, carlos, f.today(), f.today())
	f.delegate(t, "DLG-1002", daniel, mei, f.today().AddDays(3), f.today().AddDays(4))

	named := func(p actor.Actor) bool {
		var yes bool
		f.read(t, func(ctx context.Context) (err error) {
			yes, err = f.service.NamedIn(ctx, p.ID, "A-1001")
			return err
		})
		return yes
	}
	if !named(daniel) || !named(carlos) || named(mei) || named(alex) || named(sara) {
		t.Errorf("named: Daniel %v, Carlos %v, Mei %v, Alex %v, Sara %v; want only Daniel and his current delegate",
			named(daniel), named(carlos), named(mei), named(alex), named(sara))
	}
	f.clock.Advance(49 * time.Hour)
	f.mustRun(t)
	if !named(alex) {
		t.Error("the backup approver of the escalated level is not named")
	}
}
