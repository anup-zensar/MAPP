package approvals

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
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

// The submit confirmation says what will happen to every level, and lists the problems of approver sources
// with every other problem; asking changes nothing (017 FR-012, FR-013, FR-036).
func TestSubmission_OutcomesAndProblems_FR012_FR013_FR036(t *testing.T) {
	f := newFixture(t)
	f.addStandardFlow(t)
	f.addActivity(t, "A-1001", "Launch film", priya.ID, standardFlow)
	f.setFacts(t, "A-1001", "120000.00", lubricantsUnit)
	f.activities.problems["A-1001"] = []SubmitProblem{
		{Field: "startDate", Group: GroupActivity, Message: "Start date — needed to submit"},
		{Field: "lineItems.L-1001.glCode", Group: GroupLineItem, LineItemID: "L-1001", Message: "GL code — needed to submit"},
	}

	blocked, err := f.service.Submission(t.Context(), priya, "A-1001")
	if err != nil {
		t.Fatalf("Submission: %v", err)
	}
	wantProblems := []SubmitProblem{
		{Field: "startDate", Group: GroupActivity, Message: "Start date — needed to submit"},
		{Field: workflowField, Group: GroupActivity, Message: "Level 1: Priya Raman has no line manager. Ask an administrator to set one in Settings › People."},
		{Field: "lineItems.L-1001.glCode", Group: GroupLineItem, LineItemID: "L-1001", Message: "GL code — needed to submit"},
	}
	if blocked.Ready || !reflect.DeepEqual(blocked.Problems, wantProblems) {
		t.Errorf("problems = %+v, want the line-manager problem among the others", blocked.Problems)
	}

	f.activities.problems["A-1001"] = nil
	f.setManager(t, priya, daniel)
	ready, err := f.service.Submission(t.Context(), priya, "A-1001")
	if err != nil {
		t.Fatalf("Submission: %v", err)
	}
	danielView, alexView := PersonView{ID: daniel.ID, Name: "Daniel Okoro", Initials: "DO", Active: true}, PersonView{ID: alex.ID, Name: "Alex Morgan", Initials: "AM", Active: true}
	wantOutcomes := []SubmissionLevel{
		{Number: 1, Label: "Line manager", Outcome: ResolutionDecide, Approvers: []PersonView{danielView}, Source: "line manager of Priya Raman",
			Mode: ModeAny, Excluded: []ExcludedView{}},
		{Number: 2, Label: "Marketing administrator", Outcome: ResolutionDecide, Approvers: []PersonView{alexView}, Mode: ModeAny, Excluded: []ExcludedView{}},
		{Number: 3, Label: "Mobility second check", Outcome: ResolutionSkipped, Approvers: []PersonView{}, Mode: ModeAll,
			RuleText: "only when business unit is Mobility", Excluded: []ExcludedView{}},
	}
	if !ready.Ready || ready.ApprovedAutomatically || !reflect.DeepEqual(ready.Outcomes, wantOutcomes) {
		t.Errorf("outcomes =\n%+v\nwant\n%+v", ready.Outcomes, wantOutcomes)
	}

	f.setFacts(t, "A-1001", "8000.00", lubricantsUnit)
	automatic, err := f.service.Submission(t.Context(), priya, "A-1001")
	if err != nil {
		t.Fatalf("Submission: %v", err)
	}
	if !automatic.ApprovedAutomatically || automatic.Outcomes[0].Outcome != ResolutionAutoApproved ||
		automatic.Outcomes[0].RuleText != "planned cost is below $10,000.00" || automatic.Outcomes[1].Outcome != ResolutionSkipped {
		t.Errorf("outcomes = %+v, want level 1 approved automatically and the activity approved", automatic.Outcomes)
	}
	if rounds := f.rounds(t, "A-1001"); len(rounds) != 0 {
		t.Errorf("rounds = %+v, want none: asking changes nothing", rounds)
	}
}

// Submitting evaluates the rules once and keeps the outcome in the round with the values used and the
// workflow's timings (017 FR-010); levels not needed get a history entry by NorthStar with the rule; the
// round waits at the first level that needs a decision, whose approvers and their current delegates are
// told (FR-014, FR-032); after it, the round moves to the next level that needs a decision.
func TestSubmit_PartlyAutomatic_FR010_FR014(t *testing.T) {
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
		t.Errorf("history = %+v, want submitted at level 1, then level 2 not needed by NorthStar", entries)
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
// approved and approved automatically, the history has NorthStar's entries per level, the log has the
// submitter's "submitted" and NorthStar's "approved automatically", and the owner is told only when someone
// else submitted.
func TestSubmit_ApprovedAtSubmission_FR014_017P4(t *testing.T) {
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
	want := []string{"submitted Mei Lin ", "autoApproved NorthStar planned cost is below $10,000.00",
		"skipped NorthStar only when planned cost is at least $100,000.00", "skipped NorthStar only when business unit is Mobility"}
	if !slices.Equal(got, want) {
		t.Errorf("history = %q, want %q", got, want)
	}
	assertLog(t, f, "A-1001", []string{
		"NorthStar approved automatically activity 'Launch film' [submitted → approved] (Round 1 · level 1 approved automatically " +
			"(planned cost is below $10,000.00); level 2 not needed (only when planned cost is at least $100,000.00); " +
			"level 3 not needed (only when business unit is Mobility))",
		"Mei Lin submitted activity 'Launch film' [created → submitted] (Round 1 · Standard marketing approval)",
	})
	assertNotified(t, f, priya, "autoApproved: Activity approved automatically — 'Launch film' (A-1001) was approved automatically: planned cost is below $10,000.00")
	if n := f.notifications(t, priya.ID); n[0].Actor != actor.System {
		t.Errorf("notification by %v, want NorthStar", n[0].Actor)
	}
	if states := chainStates(approval.Rounds[0]); states != "L1 autoApproved, L2 skipped, L3 skipped" {
		t.Errorf("chain = %s", states)
	}

	// A workflow whose only level is never needed: "no level applies".
	f.putWorkflow(t, "WF-1004", "Mobility only", nil, nil, Level{ApproverIDs: []string{daniel.ID}, AppliesWhen: &Condition{BusinessUnitIDs: []string{mobilityUnit}}})
	f.addActivity(t, "A-1003", "Billboard winter wave", priya.ID, "WF-1004")
	f.submit(t, mei, "A-1003", "")
	if log := f.log(t, "A-1003"); logLine(log[0]) != "NorthStar approved automatically activity 'Billboard winter wave' [submitted → approved] (Round 1 · no level applies)" {
		t.Errorf("log = %s", logLine(log[0]))
	}
}

// An "all must approve" level completes when every approver has approved (017 FR-015): a partial approval
// keeps the activity Submitted, tells nobody and is logged with the progress; a second approval of the same
// approver is refused; a delegate's approval counts for the person they act for; an approval on behalf
// completes the level; any decline declines the activity.
func TestDecide_AllMustApprove_FR015(t *testing.T) {
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

// Every decision records its capacity (017 FR-019): approver; delegate with the approver acted for; backup
// approver after escalation; on behalf with the approvers still waiting — the "on behalf of" list only for
// that capacity. The log details end with the capacity.
func TestDecide_RecordsCapacity_FR019(t *testing.T) {
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

// The viewer's relation and the approve and decline actions follow who may decide (017 FR-017): available
// to an approver; unavailable with the reason to the owner or the submitter who could otherwise decide;
// hidden (no reason) for anyone else; on behalf for someone allowed, with the approvers still waiting.
func TestApproval_ViewerAuthority_FR017(t *testing.T) {
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

// The chain shows the new level states (017 FR-033): Not needed and Approved automatically with their
// reasons, the approver source, the people left out, "<a> of <b> approved", the backup approver, reminder
// and escalation times, and "No approver" with its advice.
func TestApproval_ChainRuleStates_FR033(t *testing.T) {
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

// The approval history shows the new entries and fields (017 FR-034): Not needed and Approved
// automatically by NorthStar with their rule, Reminder sent and Escalated with the days waited and the
// backup approver, each decision's capacity.
func TestApproval_HistoryRuleEntries_FR034(t *testing.T) {
	f := newFixture(t)
	f.addStandardFlow(t)
	f.setManager(t, mei, carlos)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, standardFlow)
	f.setFacts(t, "A-1009", "5000.00", mobilityUnit)
	f.submit(t, mei, "A-1009", "")
	f.clock.Advance(4 * 24 * time.Hour)
	f.mustRun(t)
	f.clock.Advance(24 * time.Hour)
	f.mustRun(t)
	f.approve(t, carlos, "A-1009", "")

	approval, err := f.service.Approval(t.Context(), mei, "A-1009")
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}
	var got []string
	for _, e := range approval.Rounds[0].History {
		line := e.Action + " L" + string(rune('0'+e.Level)) + " " + e.Actor.Name
		switch {
		case e.Rule != "":
			line += " rule " + e.Rule
		case e.Action == "reminded" || e.Action == "escalated":
			line += " waited " + daysText(e.WaitedDays)
			if e.EscalatedTo != nil {
				line += " to " + e.EscalatedTo.Name
			}
		case e.Capacity != "":
			line += " as " + string(e.Capacity)
		}
		got = append(got, line)
	}
	want := []string{
		"approved L3 Carlos Mendes as backup",
		"escalated L3 NorthStar waited 5 days to Carlos Mendes",
		"reminded L3 NorthStar waited 4 days",
		"skipped L2 NorthStar rule only when planned cost is at least $100,000.00",
		"autoApproved L1 NorthStar rule planned cost is below $10,000.00",
		"submitted L3 Mei Lin",
	}
	if !slices.Equal(got, want) {
		t.Errorf("history =\n%q\nwant\n%q", got, want)
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
