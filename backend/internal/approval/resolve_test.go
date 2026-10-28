package approval

import (
	"context"
	"reflect"
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/record"
)

func (f *fixture) putWorkflow(t *testing.T, id, name string, reminder, escalation *int, levels ...Level) string {
	t.Helper()
	for i := range levels {
		if levels[i].ApproverIDs == nil {
			levels[i].ApproverIDs = []string{}
		}
		if levels[i].Mode == "" {
			levels[i].Mode = ModeAny
		}
		if levels[i].ApproverType == "" {
			levels[i].ApproverType = SourcePeople
		}
	}
	workflow := Workflow{ID: id, Name: name, Active: true, Levels: levels, ReminderDays: reminder, EscalationDays: escalation,
		Meta: record.NewMeta(actor.System, start)}
	f.write(t, func(ctx context.Context) error { return f.service.repo.PutWorkflow(ctx, workflow) })
	return id
}

func named(persons ...actor.Actor) Level {
	ids := []string{}
	for _, p := range persons {
		ids = append(ids, p.ID)
	}
	return Level{ApproverType: SourcePeople, ApproverIDs: ids, Mode: ModeAny}
}

func (f *fixture) resolveLevels(t *testing.T, facts ActivityFacts, submitter actor.Actor, levels ...Level) resolution {
	t.Helper()
	var res resolution
	f.read(t, func(ctx context.Context) error {
		d, err := f.service.directory(ctx)
		if err != nil {
			return err
		}
		sub := activitySubject(facts)
		res, err = f.service.resolve(ctx, evaluate(levels, ruleFacts(sub), fixtureNames), resolving{sub: sub, submitter: submitter, d: d})
		return err
	})
	return res
}

func TestResolve_Sources_FR001_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	facts := f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	f.setManager(t, priya, daniel)
	f.deactivatePerson(t, sara)
	f.access.cannotApprove[carlos.ID] = true
	f.access.holders[approverRole.ID] = []string{alex.ID, daniel.ID}
	withBackup := func(l Level, backup actor.Actor) Level { l.BackupApproverID = backup.ID; return l }

	res := f.resolveLevels(t, facts, mei,
		named(daniel, sara, carlos),
		Level{ApproverType: SourceManager},
		Level{ApproverType: SourceRole, RoleID: approverRole.ID},
		named(priya, mei, daniel),
		withBackup(named(priya), alex),
		withBackup(named(daniel), alex),
	)

	if len(res.problems) != 0 {
		t.Fatalf("problems = %+v, want none", res.problems)
	}
	type want struct {
		approvers  []actor.Actor
		source     string
		excluded   []Excluded
		backupUsed bool
		backup     *actor.Actor
	}
	wants := []want{
		{approvers: []actor.Actor{daniel}, excluded: []Excluded{{sara, ReasonInactive}, {carlos, ReasonCannotApprove}}},
		{approvers: []actor.Actor{daniel}, source: "line manager of Priya Raman"},
		{approvers: []actor.Actor{alex, daniel}, source: "role Approver"},
		{approvers: []actor.Actor{daniel}, excluded: []Excluded{{priya, ReasonOwner}, {mei, ReasonSubmitter}}},
		{approvers: []actor.Actor{alex}, excluded: []Excluded{{priya, ReasonOwner}}, backupUsed: true},
		{approvers: []actor.Actor{daniel}, backup: &alex},
	}
	for i, w := range wants {
		l := res.levels[i]
		if w.excluded == nil {
			w.excluded = []Excluded{}
		}
		if !reflect.DeepEqual(l.Approvers, w.approvers) || l.Source != w.source || !reflect.DeepEqual(l.Excluded, w.excluded) ||
			l.BackupUsed != w.backupUsed || !reflect.DeepEqual(l.Backup, w.backup) || l.NoApprover || l.Resolution != ResolutionDecide {
			t.Errorf("level %d = %+v\nwant %+v", i+1, l, w)
		}
	}
	if res.levels[2].RoleID != approverRole.ID || res.levels[2].RoleName != "Approver" {
		t.Errorf("role level = %+v, want the role copied", res.levels[2])
	}

	f.access.rule = false
	off := f.resolveLevels(t, facts, mei, named(priya, mei, daniel))
	if got := off.levels[0].Approvers; !reflect.DeepEqual(got, []actor.Actor{priya, mei, daniel}) || len(off.levels[0].Excluded) != 0 {
		t.Errorf("with the rule off = %+v, want everyone kept", off.levels[0])
	}
}

func TestResolve_NoApprover_017P8(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	own := f.addActivity(t, "A-1004", "Trade magazine ads", daniel.ID, lubricantsFlow)
	priyas := f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	f.deactivatePerson(t, sara)

	tests := []struct {
		name      string
		facts     ActivityFacts
		submitter actor.Actor
		level     Level
		advice    string
	}{
		{"the submitter owns it and is the only approver", own, daniel, named(daniel),
			"You are the only approver of level 1. An administrator will have to approve on your behalf."},
		{"the owner is the only approver, someone else submits", priyas, mei, named(priya),
			"Nobody else can approve level 1. Someone allowed to decide on behalf, such as an administrator, will have to approve it."},
		{"the only approver is inactive", priyas, priya, named(sara),
			"Nobody else can approve level 1. Someone allowed to decide on behalf, such as an administrator, will have to approve it."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := f.resolveLevels(t, tt.facts, tt.submitter, tt.level)
			l := res.levels[0]
			if len(res.problems) != 0 || !l.NoApprover || len(l.Approvers) != 0 || l.Backup != nil {
				t.Errorf("level = %+v, problems %+v; want no approver and no problem", l, res.problems)
			}
			if got := noApproverAdvice(l, tt.submitter.ID); got != tt.advice {
				t.Errorf("advice = %q, want %q", got, tt.advice)
			}
		})
	}

	res := f.resolveLevels(t, own, priya, Level{ApproverType: SourcePeople, ApproverIDs: []string{sara.ID}, Mode: ModeAny, BackupApproverID: daniel.ID})
	if l := res.levels[0]; !l.NoApprover || l.BackupUsed {
		t.Errorf("level = %+v, want no approver: the backup approver owns the activity", l)
	}
}

func TestResolve_Problems_FR012(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	facts := f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)

	res := f.resolveLevels(t, facts, priya,
		Level{ApproverType: SourceManager},
		Level{ApproverType: SourceRole, RoleID: "RL-9999"},
		Level{ApproverType: SourceRole, RoleID: approverRole.ID},
		named(daniel),
	)

	want := []SubmitProblem{
		{Field: workflowField, Group: GroupActivity, Message: "Level 1: Priya Raman has no line manager. Ask an administrator to set one in Settings › People."},
		{Field: workflowField, Group: GroupActivity, Message: "Level 2: the role of this level no longer exists. Ask an administrator to change the approval workflow."},
		{Field: workflowField, Group: GroupActivity, Message: "Level 3: nobody holds the role 'Approver' for this plan."},
	}
	if !reflect.DeepEqual(res.problems, want) {
		t.Errorf("problems =\n%+v\nwant\n%+v", res.problems, want)
	}
	for i, l := range res.levels[:3] {
		if len(l.Approvers) != 0 || l.NoApprover {
			t.Errorf("level %d = %+v, want nobody resolved", i+1, l)
		}
	}
	if got := res.levels[3].Approvers; !reflect.DeepEqual(got, []actor.Actor{daniel}) {
		t.Errorf("level 4 approvers = %+v, want Daniel Okoro", got)
	}
}
