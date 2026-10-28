package approval

import (
	"context"
	"reflect"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

func TestAuthority_Table_FR017_FR018_FR020_SC002(t *testing.T) {
	t.Parallel()
	escalated := start.Add(time.Hour)
	level := RoundLevel{Number: 1, Approvers: []actor.Actor{daniel, carlos}, ApproverType: SourcePeople, Mode: ModeAll,
		Resolution: ResolutionDecide, Excluded: []Excluded{}, Backup: &mei}
	round := Round{ID: "AR-1001", SubjectKind: KindActivity, SubjectID: "A-1001", Number: 1, Levels: []RoundLevel{level}, CurrentLevel: 1, Outcome: OutcomeOpen,
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
		shown       bool
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
				au, err = f.service.authority(ctx, dc, decisionPoint{sub: activitySubject(facts), round: tt.round, entries: tt.entries, onBehalf: true})
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

func TestDecide_RecordsCapacity_FR019(t *testing.T) { //nolint:gocyclo // 10: one table, each entry and log line checked in turn
	t.Parallel()
	f := newFixture(t)
	f.access.onBehalf = []string{alex.ID}
	f.putWorkflow(t, "WF-1003", "Backed up", days(1), days(2), Level{ApproverIDs: []string{daniel.ID}, BackupApproverID: carlos.ID})
	for _, id := range []string{"A-1001", "A-1002", "A-1003", "A-1004"} {
		f.addActivity(t, id, "Activity "+id, priya.ID, "WF-1003")
		f.submit(t, priya, id, "")
	}
	f.delegate(t, "DLG-1001", daniel, mei, f.today(), f.today().AddDays(5))
	f.clock.Advance(49 * time.Hour)
	if _, err := f.service.RunEscalations(t.Context(), alex); err != nil {
		t.Fatalf("RunEscalations: %v", err)
	}

	f.approve(t, daniel, "A-1001", "")
	f.approve(t, mei, "A-1002", "")
	f.approve(t, carlos, "A-1003", "")
	f.approve(t, alex, "A-1004", "")

	tests := []struct {
		id         string
		capacity   Capacity
		delegateOf *actor.Actor
		onBehalfOf []actor.Actor
		details    string
	}{
		{"A-1001", CapacityApprover, nil, []actor.Actor{}, "Round 1 · level 1 of 1"},
		{"A-1002", CapacityDelegate, &daniel, []actor.Actor{}, "Round 1 · level 1 of 1; as delegate of Daniel Okoro"},
		{"A-1003", CapacityBackup, nil, []actor.Actor{}, "Round 1 · level 1 of 1; as backup approver"},
		{"A-1004", CapacityOnBehalf, nil, []actor.Actor{daniel}, "Round 1 · level 1 of 1; on behalf of Daniel Okoro"},
	}
	for _, tt := range tests {
		entries := f.actions(t, tt.id)
		e := entries[len(entries)-1]
		if e.Action != ActionApproved || e.Capacity != tt.capacity || !reflect.DeepEqual(e.AsDelegateOf, tt.delegateOf) ||
			!reflect.DeepEqual(e.OnBehalfOf, tt.onBehalfOf) {
			t.Errorf("%s: entry = %+v, want capacity %s", tt.id, e, tt.capacity)
		}
		if log := f.log(t, tt.id); log[0].Details != tt.details || f.activities.facts[tt.id].Status != StatusApproved {
			t.Errorf("%s: log details %q, want %q", tt.id, log[0].Details, tt.details)
		}
	}
}

func TestApproval_ViewerAuthority_FR017(t *testing.T) { //nolint:gocyclo,tparallel // 13: one table of viewers; the subtests share one fixture
	t.Parallel()
	f := newFixture(t)
	f.access.onBehalf = []string{alex.ID, priya.ID, mei.ID}
	f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	f.submit(t, mei, "A-1001", "")

	daniels := ActionAvailability{Available: true}
	tests := []struct {
		who       actor.Actor
		action    ActionAvailability
		capacity  Capacity
		submitter bool
	}{
		{daniel, daniels, CapacityApprover, false},
		{priya, ActionAvailability{Reason: "You own this activity, so someone else must approve it."}, "", false},
		{mei, ActionAvailability{Reason: "You submitted this activity, so someone else must approve it."}, "", true},
		{sara, ActionAvailability{}, "", false},
		{alex, daniels, CapacityOnBehalf, false},
	}
	for _, tt := range tests {
		t.Run(tt.who.Name, func(t *testing.T) {
			approval, err := f.service.Approval(t.Context(), tt.who, "A-1001")
			if err != nil {
				t.Fatalf("Approval: %v", err)
			}
			v := approval.Viewer
			if approval.Actions.Approve != tt.action || approval.Actions.Decline != tt.action || v.IsSubmitter != tt.submitter ||
				v.CanDecide != tt.action.Available || v.CannotDecideReason != tt.action.Reason {
				t.Errorf("actions %+v, viewer %+v; want %+v", approval.Actions, v, tt.action)
			}
			if (v.Capacity == nil) != (tt.capacity == "") || (v.Capacity != nil && *v.Capacity != tt.capacity) {
				t.Errorf("capacity = %v, want %q", v.Capacity, tt.capacity)
			}
			wantBehalf := []PersonView{}
			if tt.capacity == CapacityOnBehalf {
				wantBehalf = []PersonView{{ID: daniel.ID, Name: "Daniel Okoro", Initials: "DO", Active: true}}
			}
			if !reflect.DeepEqual(v.OnBehalfOf, wantBehalf) {
				t.Errorf("on behalf of = %+v, want %+v", v.OnBehalfOf, wantBehalf)
			}
		})
	}
}
