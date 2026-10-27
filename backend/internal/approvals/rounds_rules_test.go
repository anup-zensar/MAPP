package approvals

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// standardFlow is the fixture's copy of the seeded "Standard marketing approval" (017 data-model §13).
const standardFlow = "WF-1003"

// addStandardFlow stores "Standard marketing approval": level 1 the owner's line manager, approved
// automatically below $10,000.00 (backup Alex Morgan); level 2 Alex Morgan, only from $100,000.00 (backup
// Carlos Mendes); level 3 Daniel Okoro and Alex Morgan, all must approve, only for Mobility (backup Carlos
// Mendes); remind after 3 days, escalate after 5.
func (f *fixture) addStandardFlow(t *testing.T) {
	t.Helper()
	f.putWorkflow(t, standardFlow, "Standard marketing approval", days(3), days(5),
		Level{Name: "Line manager", ApproverType: SourceManager, AutoApproveWhen: &Condition{PlannedCostBelow: amount(t, "10000.00")},
			BackupApproverID: alex.ID},
		Level{Name: "Marketing administrator", ApproverIDs: []string{alex.ID}, AppliesWhen: &Condition{PlannedCostAtLeast: amount(t, "100000.00")},
			BackupApproverID: carlos.ID},
		Level{Name: "Mobility second check", ApproverIDs: []string{daniel.ID, alex.ID}, Mode: ModeAll,
			AppliesWhen: &Condition{BusinessUnitIDs: []string{mobilityUnit}}, BackupApproverID: carlos.ID},
	)
}

// setFacts changes the planned cost (a USD text) and business unit of the stored activity id.
func (f *fixture) setFacts(t *testing.T, id, cost, unit string) {
	t.Helper()
	facts := f.activities.facts[id]
	usd := money.USD(mustAmount(t, cost))
	facts.PlannedCost = &usd
	label := map[string]string{lubricantsUnit: "Lubricants", mobilityUnit: "Mobility"}[unit]
	facts.BusinessUnits = []Choice{{ID: unit, Label: label, Active: true}}
	f.activities.facts[id] = facts
}

// delegate stores a delegation of person to delegate from the first to the last day (UTC dates).
func (f *fixture) delegate(t *testing.T, id string, person, delegate actor.Actor, from, to date.Date) {
	t.Helper()
	f.write(t, func(ctx context.Context) error {
		return f.service.repo.PutDelegation(ctx, Delegation{ID: id, PersonID: person.ID, DelegateID: delegate.ID, StartDate: from, EndDate: to,
			Meta: record.NewMeta(person, start)})
	})
}

// today is the fixture clock's date.
func (f *fixture) today() date.Date {
	return date.Today(f.clock.Now())
}

// Submitting evaluates the rules once and keeps the outcome in the round with the values used and the
// workflow's timings (017 FR-010); levels not needed get a history entry by MAPP with the rule; the
// round waits at the first level that needs a decision, whose approvers and their current delegates are
// told (FR-014, FR-032); after it, the round moves to the next level that needs a decision.
func TestSubmit_PartlyAutomatic_FR010_FR014(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addStandardFlow(t)
	f.setManager(t, mei, carlos)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, standardFlow)
	f.setFacts(t, "A-1009", "50000.00", mobilityUnit)
	f.delegate(t, "DLG-1001", carlos, sara, f.today(), f.today().AddDays(2))

	f.submit(t, mei, "A-1009", "")

	round := f.openRound(t, "A-1009")
	usd := money.USD(mustAmount(t, "50000.00"))
	wantFacts := &RuleFacts{PlannedCost: &usd, ActivityType: &FactRef{ID: eventType, Name: "Event"},
		BusinessUnits: []FactOption{{ID: mobilityUnit, Label: "Mobility"}}}
	if !reflect.DeepEqual(round.Facts, wantFacts) || *round.ReminderDays != 3 || *round.EscalationDays != 5 || round.CurrentLevel != 1 {
		t.Errorf("round = %+v, want the facts, the timings and level 1 waiting", round)
	}
	levels := round.Levels
	if levels[0].Resolution != ResolutionDecide || !reflect.DeepEqual(levels[0].Approvers, []actor.Actor{carlos}) ||
		levels[0].StartedAt == nil || levels[1].Resolution != ResolutionSkipped || levels[1].StartedAt != nil ||
		levels[2].Resolution != ResolutionDecide || levels[2].Mode != ModeAll || !reflect.DeepEqual(levels[2].Approvers, []actor.Actor{daniel, alex}) ||
		levels[2].Backup == nil || levels[2].Backup.ID != carlos.ID {
		t.Errorf("levels = %+v", levels)
	}
	entries := f.actions(t, "A-1009")
	if len(entries) != 2 || entries[0].Action != ActionSubmitted || entries[0].Level != 1 || entries[1].Action != ActionSkipped ||
		entries[1].Level != 2 || entries[1].Actor != actor.System || entries[1].Rule != "only when planned cost is at least $100,000.00" {
		t.Errorf("history = %+v, want submitted at level 1, then level 2 not needed by MAPP", entries)
	}
	assertLog(t, f, "A-1009", []string{
		"Mei Lin submitted activity 'Fleet card launch' [created → submitted] (Round 1 · Standard marketing approval · " +
			"level 2 not needed (only when planned cost is at least $100,000.00) · waiting for level 1 (Carlos Mendes))",
	})
	assertNotified(t, f, carlos, "approvalNeeded: Approval needed — Mei Lin submitted 'Fleet card launch' (A-1009)")
	assertNotified(t, f, sara, "approvalNeeded: Approval needed — Mei Lin submitted 'Fleet card launch' (A-1009) — you approve as delegate of Carlos Mendes")

	f.clock.Advance(time.Hour)
	approval := f.approve(t, carlos, "A-1009", "")

	round = f.openRound(t, "A-1009")
	if round.CurrentLevel != 3 || round.Levels[2].StartedAt == nil || !round.Levels[2].StartedAt.Equal(f.clock.Now()) {
		t.Errorf("round at level %d, started %v; want level 3 from now (level 2 passed over)", round.CurrentLevel, round.Levels[2].StartedAt)
	}
	if states := chainStates(approval.Rounds[0]); states != "L1 approved, L2 skipped, L3 waiting" {
		t.Errorf("chain = %s", states)
	}
	assertNotified(t, f, daniel, "approvalNeeded: Approval needed — Carlos Mendes approved level 1 of 'Fleet card launch' (A-1009) — level 3 waits for you")
	assertNotified(t, f, mei, "levelApproved: Level 1 approved — Carlos Mendes approved level 1 of 3 of 'Fleet card launch' (A-1009)")
}

// When no level needs a decision the activity is Approved at submission (017 FR-014, 017-P4): the round ends
// approved and approved automatically, the history has MAPP's entries per level, the log has the
// submitter's "submitted" and MAPP's "approved automatically", and the owner is told only when someone
// else submitted.
func TestSubmit_ApprovedAtSubmission_FR014_017P4(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addStandardFlow(t)
	f.setManager(t, priya, daniel)
	f.addActivity(t, "A-1001", "Launch film", priya.ID, standardFlow)
	f.setFacts(t, "A-1001", "8000.00", lubricantsUnit)
	f.addActivity(t, "A-1002", "Social always-on", priya.ID, standardFlow)
	f.setFacts(t, "A-1002", "5000.00", lubricantsUnit)

	approval := f.submit(t, mei, "A-1001", "")
	f.submit(t, priya, "A-1002", "")

	round := f.rounds(t, "A-1001")[0]
	if approval.Status != StatusApproved || round.Outcome != OutcomeApproved || !round.AutoApproved || round.EndedAt == nil ||
		!round.EndedAt.Equal(start) || round.CurrentLevel != 3 {
		t.Errorf("status %s, round %+v; want Approved at submission", approval.Status, round)
	}
	var got []string
	for _, e := range f.actions(t, "A-1001") {
		got = append(got, string(e.Action)+" "+e.Actor.Name+" "+e.Rule)
	}
	want := []string{"submitted Mei Lin ", "autoApproved MAPP planned cost is below $10,000.00",
		"skipped MAPP only when planned cost is at least $100,000.00", "skipped MAPP only when business unit is Mobility"}
	if !slices.Equal(got, want) {
		t.Errorf("history = %q, want %q", got, want)
	}
	assertLog(t, f, "A-1001", []string{
		"MAPP approved automatically activity 'Launch film' [submitted → approved] (Round 1 · level 1 approved automatically " +
			"(planned cost is below $10,000.00); level 2 not needed (only when planned cost is at least $100,000.00); " +
			"level 3 not needed (only when business unit is Mobility))",
		"Mei Lin submitted activity 'Launch film' [created → submitted] (Round 1 · Standard marketing approval)",
	})
	assertNotified(t, f, priya, "autoApproved: Activity approved automatically — 'Launch film' (A-1001) was approved automatically: planned cost is below $10,000.00")
	if n := f.notifications(t, priya.ID); n[0].Actor != actor.System {
		t.Errorf("notification by %v, want MAPP", n[0].Actor)
	}
	if states := chainStates(approval.Rounds[0]); states != "L1 autoApproved, L2 skipped, L3 skipped" {
		t.Errorf("chain = %s", states)
	}

	// A workflow whose only level is never needed: "no level applies".
	f.putWorkflow(t, "WF-1004", "Mobility only", nil, nil, Level{ApproverIDs: []string{daniel.ID}, AppliesWhen: &Condition{BusinessUnitIDs: []string{mobilityUnit}}})
	f.addActivity(t, "A-1003", "Billboard winter wave", priya.ID, "WF-1004")
	f.submit(t, mei, "A-1003", "")
	if log := f.log(t, "A-1003"); logLine(log[0]) != "MAPP approved automatically activity 'Billboard winter wave' [submitted → approved] (Round 1 · no level applies)" {
		t.Errorf("log = %s", logLine(log[0]))
	}
}

// An "all must approve" level completes when every approver has approved (017 FR-015): a partial approval
// keeps the activity Submitted, tells nobody and is logged with the progress; a second approval of the same
// approver is refused; a delegate's approval counts for the person they act for; an approval on behalf
// completes the level; any decline declines the activity.
func TestDecide_AllMustApprove_FR015(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.onBehalf = []string{alex.ID}
	f.putWorkflow(t, "WF-1003", "Two must approve", nil, nil, Level{ApproverIDs: []string{daniel.ID, carlos.ID}, Mode: ModeAll})
	for _, id := range []string{"A-1001", "A-1002", "A-1003", "A-1004"} {
		f.addActivity(t, id, "Activity "+id, priya.ID, "WF-1003")
		f.submit(t, priya, id, "")
	}
	counts := f.counts(t)

	first := f.approve(t, daniel, "A-1001", "")
	level := first.Rounds[0].Levels[0]
	if first.Status != StatusSubmitted || level.State != LevelWaiting || level.ApprovedCount != 1 || level.RequiredCount != 2 || len(level.Decisions) != 1 {
		t.Errorf("after Daniel: %s, level %+v; want submitted, 1 of 2 approved", first.Status, level)
	}
	if got := f.notifiedSince(t, counts); len(got) != 0 {
		t.Errorf("notified %q, want nobody for a partial approval", got)
	}
	if log := f.log(t, "A-1001"); logLine(log[0]) != "Daniel Okoro approved level 1 of activity 'Activity A-1001' "+
		"(Round 1 · level 1 of 1 · 1 of 2 approved · still waiting for Carlos Mendes)" {
		t.Errorf("log = %s", logLine(log[0]))
	}
	if rows := f.inboxIDs(t, daniel, TabWaiting); slices.Contains(rows, "A-1001") {
		t.Errorf("Daniel's Waiting for me = %v, want A-1001 gone", rows)
	}
	_, err := f.service.Approve(t.Context(), daniel, "A-1001", DecisionInput{RoundID: f.openRound(t, "A-1001").ID, Level: version(1)})
	assertError(t, err, errs.Forbidden("You already approved level 1. Waiting for Carlos Mendes."))
	if done := f.approve(t, carlos, "A-1001", ""); done.Status != StatusApproved || done.Rounds[0].Levels[0].ApprovedCount != 2 {
		t.Errorf("after Carlos: %s, want approved with 2 of 2", done.Status)
	}

	f.delegate(t, "DLG-1001", daniel, mei, f.today(), f.today())
	f.approve(t, mei, "A-1002", "")
	if done := f.approve(t, carlos, "A-1002", ""); done.Status != StatusApproved {
		t.Errorf("Mei's approval as delegate of Daniel did not count for him: %s", done.Status)
	}

	if done := f.approve(t, alex, "A-1003", ""); done.Status != StatusApproved {
		t.Errorf("an approval on behalf left the level open: %s", done.Status)
	}

	f.approve(t, daniel, "A-1004", "")
	if declined := f.decline(t, carlos, "A-1004", "No."); declined.Status != StatusDeclined || declined.Rounds[0].Levels[0].State != LevelDeclined {
		t.Errorf("after a decline: %s, want declined at once", declined.Status)
	}
}

// The chain shows the new level states (017 FR-033): Not needed and Approved automatically with their
// reasons, the approver source, the people left out, "<a> of <b> approved", the backup approver, reminder
// and escalation times, and "No approver" with its advice.
func TestApproval_ChainRuleStates_FR033(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addStandardFlow(t)
	f.setManager(t, mei, carlos)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, standardFlow)
	f.setFacts(t, "A-1009", "5000.00", mobilityUnit)
	f.submit(t, mei, "A-1009", "")
	f.clock.Advance(6 * 24 * time.Hour)
	if _, err := f.service.RunEscalations(t.Context(), alex); err != nil {
		t.Fatalf("RunEscalations: %v", err)
	}
	f.putWorkflow(t, "WF-1004", "Own", nil, nil, Level{ApproverIDs: []string{daniel.ID}})
	f.addActivity(t, "A-1004", "Trade magazine ads", daniel.ID, "WF-1004")
	f.submit(t, daniel, "A-1004", "")

	approval, err := f.service.Approval(t.Context(), alex, "A-1009")
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}
	levels := approval.Rounds[0].Levels
	if levels[0].State != LevelAutoApproved || levels[0].RuleText != "planned cost is below $10,000.00" || levels[0].Source != "" ||
		levels[1].State != LevelSkipped || levels[1].RuleText != "only when planned cost is at least $100,000.00" {
		t.Errorf("levels 1–2 = %+v / %+v", levels[0], levels[1])
	}
	third := levels[2]
	escalatedAt := f.clock.Now()
	if third.State != LevelWaiting || third.Mode != ModeAll || third.ApprovedCount != 0 || third.RequiredCount != 2 ||
		third.Backup == nil || third.Backup.Name != "Carlos Mendes" || third.EscalatedTo == nil || third.EscalatedTo.Name != "Carlos Mendes" ||
		third.EscalatedAt == nil || !third.EscalatedAt.Equal(escalatedAt) || third.RemindedAt == nil || third.WaitingSince == nil ||
		!third.WaitingSince.Equal(start) || third.NoApprover {
		t.Errorf("level 3 = %+v, want waiting 0 of 2, escalated to Carlos Mendes", third)
	}
	if f := approval.Rounds[0].Facts; f == nil || f.ActivityType == nil || f.ActivityType.Name != "Event" || f.BusinessUnits[0].Label != "Mobility" {
		t.Errorf("facts = %+v, want the values the rules used", f)
	}

	own, err := f.service.Approval(t.Context(), alex, "A-1004")
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}
	l := own.Rounds[0].Levels[0]
	if !l.NoApprover || l.Advice != "You are the only approver of level 1. An administrator will have to approve on your behalf." ||
		len(l.Excluded) != 1 || l.Excluded[0].Reason != ReasonOwner || l.Excluded[0].Person.Name != "Daniel Okoro" {
		t.Errorf("own level = %+v, want no approver with 014's advice and Daniel left out as owner", l)
	}
}

// mustRun runs reminders and escalations as Alex Morgan and stops the test when that fails.
func (f *fixture) mustRun(t *testing.T) EscalationRun {
	t.Helper()
	run, err := f.service.RunEscalations(t.Context(), alex)
	if err != nil {
		t.Fatalf("RunEscalations: %v", err)
	}
	return run
}
