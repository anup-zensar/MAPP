package approvals

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// eventLine writes one event of a run: "<activity> L<n> <action> <days>d to <names>".
func eventLine(e EscalationEvent) string {
	var to []string
	for _, p := range e.To {
		to = append(to, p.Name)
	}
	return e.Activity.ID + " " + e.Action + " " + daysText(e.WaitedDays) + " to [" + strings.Join(to, ", ") + "]"
}

// Running reminders and escalations (017 FR-026 – FR-028, SC-005): a level that has waited the reminder
// days gets one reminder, to its waiting approvers and their delegates; one that has waited the escalation
// days is escalated to its usable backup approver — who, the waiting approvers and the owner are told — and
// no separate reminder is sent; a backup approver who is inactive, cannot approve, owns or submitted the
// activity, or already approves the level cannot take it and the level is reported; a second run sends
// nothing; legacy rounds wait from 008's time; it needs workflows.manage.
func TestRunEscalations_FR026_FR027_FR028_SC005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.putWorkflow(t, "WF-1003", "Remind 3, escalate 5", days(3), days(5), Level{ApproverIDs: []string{daniel.ID}, BackupApproverID: alex.ID})
	f.putWorkflow(t, "WF-1004", "Remind only", days(3), nil, Level{ApproverIDs: []string{daniel.ID}})
	f.putWorkflow(t, "WF-1005", "Owner backup", days(1), days(2), Level{ApproverIDs: []string{daniel.ID}, BackupApproverID: priya.ID})
	f.putWorkflow(t, "WF-1006", "Never", nil, nil, Level{ApproverIDs: []string{daniel.ID}})
	f.addActivity(t, "A-1001", "Reminded", priya.ID, "WF-1004")
	f.addActivity(t, "A-1002", "Escalated", priya.ID, "WF-1003")
	f.addActivity(t, "A-1003", "Owner backup", priya.ID, "WF-1005")
	f.addActivity(t, "A-1004", "Never due", priya.ID, "WF-1006")
	for _, id := range []string{"A-1001", "A-1002", "A-1003", "A-1004"} {
		f.submit(t, priya, id, "")
	}
	f.delegate(t, "DLG-1001", daniel, carlos, f.today(), f.today().AddDays(30))
	f.clock.Advance(3*24*time.Hour + time.Minute)
	counts := f.counts(t)

	first := f.mustRun(t)

	if got := eventLines(first.Events); !slices.Equal(got, []string{
		"A-1001 reminded 3 days to [Daniel Okoro, Carlos Mendes]", "A-1002 reminded 3 days to [Daniel Okoro, Carlos Mendes]",
	}) || first.Reminded != 2 || first.Escalated != 0 {
		t.Errorf("first run = %+v", first)
	}
	if got := eventLines(first.NotEscalated); !slices.Equal(got, []string{"A-1003 notEscalated 3 days to []"}) {
		t.Errorf("not escalated = %q, want A-1003: its backup approver owns it", got)
	}
	assertNotified(t, f, carlos,
		"reminder: Reminder — approval waiting 3 days — 'Escalated' (A-1002) has waited 3 days for you at level 1 — as delegate of Daniel Okoro",
		"reminder: Reminder — approval waiting 3 days — 'Reminded' (A-1001) has waited 3 days for you at level 1 — as delegate of Daniel Okoro")
	if got := f.notifiedSince(t, counts); !slices.Equal(got, []string{"Daniel Okoro: reminder, reminder", "Carlos Mendes: reminder, reminder"}) {
		t.Errorf("notified = %q, want Daniel and Carlos twice", got)
	}
	if log := f.log(t, "A-1001"); logLine(log[0]) != "MAPP sent a reminder about activity 'Reminded' "+
		"(Round 1 · level 1 of 1 · waited 3 days · to Daniel Okoro, Carlos Mendes · run by Alex Morgan)" {
		t.Errorf("log = %s", logLine(log[0]))
	}

	again := f.mustRun(t)
	if again.Reminded != 0 || again.Escalated != 0 || len(again.Events) != 0 {
		t.Errorf("second run = %+v, want nothing sent", again)
	}

	f.clock.Advance(2 * 24 * time.Hour)
	counts = f.counts(t)
	third := f.mustRun(t)
	if got := eventLines(third.Events); !slices.Equal(got, []string{"A-1002 escalated 5 days to [Alex Morgan]"}) || third.Escalated != 1 {
		t.Errorf("third run = %+v", third)
	}
	if got := f.notifiedSince(t, counts); !slices.Equal(got, []string{"Alex Morgan: escalated", "Priya Raman: escalated", "Daniel Okoro: escalated"}) {
		t.Errorf("notified = %q", got)
	}
	assertNotifiedFirst(t, f, alex, "escalated: Approval escalated to you — 'Escalated' (A-1002) waited 5 days at level 1. You can now approve it as backup approver.")
	assertNotifiedFirst(t, f, priya, "escalated: Approval escalated — 'Escalated' (A-1002) waited 5 days at level 1 and was escalated to Alex Morgan.")
	round := f.openRound(t, "A-1002")
	if l := round.Levels[0]; l.EscalatedAt == nil || l.RemindedAt == nil || !l.EscalatedAt.Equal(f.clock.Now()) {
		t.Errorf("level = %+v, want escalated now", l)
	}
	entries := f.actions(t, "A-1002")
	if last := entries[len(entries)-1]; last.Action != ActionEscalated || last.Actor != actor.System || last.WaitedDays != 5 ||
		last.EscalatedTo == nil || *last.EscalatedTo != alex {
		t.Errorf("last entry = %+v, want escalated to Alex Morgan by MAPP", last)
	}
	if f.mustRun(t).Escalated != 0 {
		t.Error("a level was escalated twice")
	}

	// Escalation due on the first run replaces the reminder; the backup approver already approves the level.
	g := newFixture(t)
	g.putWorkflow(t, "WF-1003", "Fast", days(1), days(2), Level{ApproverIDs: []string{daniel.ID, alex.ID}, BackupApproverID: alex.ID})
	g.putWorkflow(t, "WF-1004", "Fast backup", days(1), days(2), Level{ApproverIDs: []string{daniel.ID}, BackupApproverID: carlos.ID})
	g.addActivity(t, "A-1001", "Approver backup", priya.ID, "WF-1003")
	g.addActivity(t, "A-1002", "Escalate at once", priya.ID, "WF-1004")
	g.submit(t, priya, "A-1001", "")
	g.submit(t, priya, "A-1002", "")
	g.clock.Advance(3 * 24 * time.Hour)
	run := g.mustRun(t)
	if got, want := eventLines(run.Events), []string{"A-1002 escalated 3 days to [Carlos Mendes]"}; !slices.Equal(got, want) || run.Reminded != 0 {
		t.Errorf("run = %q, want %q and no reminder", got, want)
	}
	if got := eventLines(run.NotEscalated); !slices.Equal(got, []string{"A-1001 notEscalated 3 days to []"}) {
		t.Errorf("not escalated = %q", got)
	}

	// A round stored before 017 waits from its submission.
	h := newFixture(t)
	h.addActivity(t, "A-1001", "Legacy", priya.ID, lubricantsFlow)
	h.submit(t, priya, "A-1001", "")
	r := h.openRound(t, "A-1001")
	h.write(t, func(ctx context.Context) error {
		r.Levels[0].StartedAt, r.Facts = nil, nil
		r.ReminderDays = days(1)
		return h.service.repo.PutRound(ctx, r)
	})
	h.clock.Advance(25 * time.Hour)
	if run := h.mustRun(t); run.Reminded != 1 || run.Events[0].WaitedDays != 1 {
		t.Errorf("legacy run = %+v, want one reminder after 1 day", run)
	}

	f.access.managers = []string{alex.ID}
	_, err := f.service.RunEscalations(t.Context(), priya)
	assertError(t, err, errs.Forbidden("Changing approval workflows isn't part of your role (Planner). Ask an administrator if you need it."))
}

// eventLines writes events with eventLine.
func eventLines(events []EscalationEvent) []string {
	lines := []string{}
	for _, e := range events {
		lines = append(lines, eventLine(e))
	}
	return lines
}

// assertNotifiedFirst checks the newest notification of p, written as "<kind>: <title> — <text>".
func assertNotifiedFirst(t *testing.T, f *fixture, p actor.Actor, want string) {
	t.Helper()
	list := f.notifications(t, p.ID)
	if len(list) == 0 {
		t.Fatalf("%s has no notification", p.Name)
	}
	if got := string(list[0].Kind) + ": " + list[0].Title + " — " + list[0].Text; got != want || list[0].Actor != actor.System {
		t.Errorf("%s's newest notification = %q by %v, want %q by MAPP", p.Name, got, list[0].Actor, want)
	}
}

// Monitoring is computed over the rows of All open (017 FR-029, FR-030, SC-006): waiting days in whole days
// from when the level was reached; overdue at the escalation days, or the reminder days without escalation;
// the five ageing buckets add up to All open; waiting by level per workflow and level; the overdue rows
// oldest first; the month's average time to decision without approvals at submission, and their count.
func TestMonitoring_FR029_FR030_SC006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.managers = []string{alex.ID}
	f.putWorkflow(t, "WF-1003", "Escalate after 2", days(1), days(2), Level{ApproverIDs: []string{daniel.ID}, BackupApproverID: alex.ID})
	f.putWorkflow(t, "WF-1004", "Remind after 4", days(4), nil, Level{ApproverIDs: []string{daniel.ID}})
	f.putWorkflow(t, "WF-1005", "Automatic", nil, nil, Level{ApproverIDs: []string{daniel.ID}, AutoApproveWhen: &Condition{PlannedCostBelow: amount(t, "100000.00")}})
	f.addActivity(t, "A-1001", "Old", priya.ID, "WF-1003")
	f.addActivity(t, "A-1002", "Decided", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1003", "Automatic", priya.ID, "WF-1005")
	f.submit(t, priya, "A-1001", "")
	f.submit(t, priya, "A-1002", "")
	f.submit(t, priya, "A-1003", "")
	f.clock.Advance(36 * time.Hour)
	f.approve(t, daniel, "A-1002", "") // decided after 1.5 days
	f.addActivity(t, "A-1004", "Reminder", priya.ID, "WF-1004")
	f.submit(t, priya, "A-1004", "")
	f.clock.Advance(11 * 24 * time.Hour)
	f.addActivity(t, "A-1005", "New", priya.ID, "WF-1004")
	f.submit(t, priya, "A-1005", "")
	f.mustRun(t)

	m, err := f.service.Monitoring(t.Context(), alex, nil, nil)
	if err != nil {
		t.Fatalf("Monitoring: %v", err)
	}
	average := "1.5"
	wantCards := MonitoringCards{Overdue: 2, Escalated: 1, AverageDecisionDays: &average, DecidedThisMonth: 1, AutoApprovedThisMonth: 1}
	if !reflect.DeepEqual(m.Cards, wantCards) || !m.CanRunEscalations {
		t.Errorf("cards = %+v (%v), want %+v", m.Cards, *m.Cards.AverageDecisionDays, wantCards)
	}
	var buckets []int
	total := 0
	for _, b := range m.Ageing {
		buckets = append(buckets, b.Count)
		total += b.Count
	}
	if !slices.Equal(buckets, []int{1, 0, 0, 0, 2}) || total != len(f.inbox(t, alex, TabOpen)) || m.Ageing[4].Label != "More than 10 days" {
		t.Errorf("ageing = %+v, want under 1 day once and more than 10 days twice, adding up to All open", m.Ageing)
	}
	if len(m.ByLevel) != 2 || m.ByLevel[0].Workflow.Name != "Escalate after 2" || m.ByLevel[0].Waiting != 1 || m.ByLevel[0].Overdue != 1 ||
		m.ByLevel[1].Workflow.Name != "Remind after 4" || m.ByLevel[1].Waiting != 2 || m.ByLevel[1].Overdue != 1 ||
		!m.ByLevel[1].OldestWaitingSince.Equal(start.Add(36*time.Hour)) {
		t.Errorf("by level = %+v", m.ByLevel)
	}
	if len(m.Overdue) != 2 || m.Overdue[0].Activity.ID != "A-1001" || m.Overdue[1].Activity.ID != "A-1004" ||
		*m.Overdue[0].WaitingDays != 12 || !m.Overdue[0].Overdue || m.Overdue[0].EscalatedTo == nil {
		t.Errorf("overdue = %+v, want A-1001 (escalated) then A-1004", m.Overdue)
	}
	if other, _ := f.service.Monitoring(t.Context(), priya, nil, nil); other.CanRunEscalations {
		t.Error("Priya may run escalations, want only people who can change workflows")
	}
	if _, err := f.service.Monitoring(t.Context(), alex, &start, nil); err == nil {
		t.Error("one month bound was accepted")
	}
}

// Every approval event of the feature is in the approval history and the activity log with the person or
// MAPP, the capacity and the rule or reason (017 FR-034, FR-035, SC-004): automatic approval and levels
// not needed at submission, a partial approval, decisions as delegate, backup and on behalf, reminders and
// escalations, and delegations created, changed and cancelled.
func TestApprovalRuleEvents_Logged_FR034_FR035_SC004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.onBehalf = []string{alex.ID}
	f.addStandardFlow(t)
	f.setManager(t, mei, carlos)
	f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, standardFlow)
	f.setFacts(t, "A-1009", "5000.00", mobilityUnit)
	f.submit(t, mei, "A-1009", "")
	f.clock.Advance(3*24*time.Hour + time.Hour)
	f.mustRun(t)
	f.clock.Advance(2 * 24 * time.Hour)
	f.mustRun(t)
	view := f.createDelegation(t, daniel, DelegationInput{DelegateID: sara.ID, StartDate: f.day(0), EndDate: f.day(1)})
	note := "Away"
	if _, err := f.service.UpdateDelegation(t.Context(), daniel, view.ID, DelegationUpdate{Note: &note, Version: version(1)}); err != nil {
		t.Fatalf("UpdateDelegation: %v", err)
	}
	f.approve(t, sara, "A-1009", "")
	f.approve(t, carlos, "A-1009", "")
	if _, err := f.service.CancelDelegation(t.Context(), daniel, view.ID, DelegationCancel{Version: version(2)}); err != nil {
		t.Fatalf("CancelDelegation: %v", err)
	}

	assertLog(t, f, "A-1009", []string{
		"Carlos Mendes approved activity 'Fleet card launch' [submitted → approved] (Round 1 · level 3 of 3; as backup approver)",
		"Sara Novak approved level 3 of activity 'Fleet card launch' (Round 1 · level 3 of 3 · 1 of 2 approved · still waiting for Alex Morgan; as delegate of Daniel Okoro)",
		"MAPP escalated activity 'Fleet card launch' (Round 1 · level 3 of 3 · waited 5 days · escalated to Carlos Mendes · run by Alex Morgan)",
		"MAPP sent a reminder about activity 'Fleet card launch' (Round 1 · level 3 of 3 · waited 3 days · to Daniel Okoro, Alex Morgan · run by Alex Morgan)",
		"Mei Lin submitted activity 'Fleet card launch' [created → submitted] (Round 1 · Standard marketing approval · " +
			"level 1 approved automatically (planned cost is below $10,000.00); level 2 not needed (only when planned cost is at least $100,000.00) · " +
			"waiting for level 3 (Daniel Okoro, Alex Morgan))",
	})
	var lines []string
	for _, e := range f.log(t, view.ID) {
		lines = append(lines, e.Actor.Name+" "+e.Verb+" "+e.RecordTypeLabel+" '"+e.RecordName+"'")
	}
	if want := []string{
		"Daniel Okoro cancelled approval delegation 'Daniel Okoro → Sara Novak'",
		"Daniel Okoro changed approval delegation 'Daniel Okoro → Sara Novak'",
		"Daniel Okoro created approval delegation 'Daniel Okoro → Sara Novak'",
	}; !slices.Equal(lines, want) {
		t.Errorf("delegation log = %q, want %q", lines, want)
	}
	var kinds []string
	for _, e := range f.actions(t, "A-1009") {
		kinds = append(kinds, string(e.Action)+"/"+e.Actor.Name+"/"+string(e.Capacity))
	}
	if want := []string{"submitted/Mei Lin/", "autoApproved/MAPP/", "skipped/MAPP/", "reminded/MAPP/", "escalated/MAPP/",
		"approved/Sara Novak/delegate", "approved/Carlos Mendes/backup"}; !slices.Equal(kinds, want) {
		t.Errorf("history = %q, want %q", kinds, want)
	}
}
