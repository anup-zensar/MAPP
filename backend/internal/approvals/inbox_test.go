package approvals

import (
	"reflect"
	"testing"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// inboxIDs returns the activity ids of a's rows in tab, in order, and stops the test when Inbox fails.
func (f *fixture) inboxIDs(t *testing.T, a actor.Actor, tab string) []string {
	t.Helper()
	ids := []string{}
	for _, row := range f.inbox(t, a, tab) {
		ids = append(ids, row.Activity.ID)
	}
	return ids
}

// inbox returns a's rows in tab, and stops the test when Inbox fails.
func (f *fixture) inbox(t *testing.T, a actor.Actor, tab string) []Row {
	t.Helper()
	rows, err := f.service.Inbox(t.Context(), a, tab)
	if err != nil {
		t.Fatalf("Inbox(%s, %s): %v", a.Name, tab, err)
	}
	return rows
}

// Waiting for me lists the Submitted activities whose current level lists the person, oldest submission
// first; a level already approved, a Created activity and another level's approvers do not count (FR-029).
func TestInbox_Waiting_FR029(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1002", "Fleet card launch", mei.ID, chinaFlow)
	f.addActivity(t, "A-1003", "Fleet roadshow", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1004", "Workshop kits", priya.ID, lubricantsFlow) // stays Created
	f.submit(t, priya, "A-1003", "")
	f.clock.Advance(time.Hour)
	f.submit(t, mei, "A-1002", "")
	f.approve(t, carlos, "A-1002", "") // now waits for Alex at level 2
	f.clock.Advance(time.Hour)
	f.submit(t, priya, "A-1001", "")

	for _, c := range []struct {
		person actor.Actor
		want   []string
	}{
		{daniel, []string{"A-1003", "A-1001"}},
		{alex, []string{"A-1002"}},
		{carlos, []string{}},
		{priya, []string{}},
	} {
		if got := f.inboxIDs(t, c.person, TabWaiting); !reflect.DeepEqual(got, c.want) {
			t.Errorf("waiting for %s = %q, want %q", c.person.Name, got, c.want)
		}
	}
	_, err := f.service.Inbox(t.Context(), daniel, "mine")
	assertError(t, err, errs.Validation(errs.FieldError{Field: "tab", Message: "Choose waiting, open or decided."}))
}

// All open lists every Submitted activity, at any level, oldest submission first, the same for everyone;
// Created, Approved and Declined activities are not open (FR-029).
func TestInbox_Open_FR029(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1002", "Fleet roadshow", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1003", "Workshop kits", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1004", "Retail displays", priya.ID, lubricantsFlow) // stays Created
	f.addActivity(t, "A-1005", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, priya, "A-1005", "")
	f.clock.Advance(time.Minute)
	f.submit(t, priya, "A-1002", "")
	f.approve(t, daniel, "A-1002", "") // Approved
	f.submit(t, priya, "A-1003", "")
	f.decline(t, daniel, "A-1003", "Not this year.") // Declined
	f.clock.Advance(time.Minute)
	f.submit(t, priya, "A-1001", "")

	for _, person := range []actor.Actor{alex, priya, sara} {
		if got, want := f.inboxIDs(t, person, TabOpen), []string{"A-1005", "A-1001"}; !reflect.DeepEqual(got, want) {
			t.Errorf("all open for %s = %q, want %q", person.Name, got, want)
		}
	}
}

// Decided by me lists the activities on which the person approved or declined a level in any round, latest
// decision first, with that decision, the activity's latest round and its status now; a decision made on
// the person's behalf is not theirs (FR-029, FR-030).
func TestInbox_Decided_FR029_FR030(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1002", "Fleet roadshow", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1003", "Workshop kits", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1001", "")
	f.submit(t, priya, "A-1002", "")
	f.submit(t, priya, "A-1003", "")
	f.approve(t, daniel, "A-1001", "")
	approvedAt := f.clock.Now()
	f.clock.Advance(time.Hour)
	f.decline(t, daniel, "A-1002", "Split it by country.")
	declinedAt := f.clock.Now()
	f.clock.Advance(time.Hour)
	f.submit(t, priya, "A-1002", "Split.") // round 2 waits for Daniel again
	f.approve(t, alex, "A-1003", "")       // on behalf of Daniel: Alex's decision

	rows := f.inbox(t, daniel, TabDecided)

	if len(rows) != 2 {
		t.Fatalf("decided by Daniel = %d rows, want 2", len(rows))
	}
	declined, approved := rows[0], rows[1]
	if declined.Activity.ID != "A-1002" || declined.Activity.Status != StatusSubmitted || declined.Round.Number != 2 ||
		declined.Round.Outcome != OutcomeOpen || declined.WaitingSince == nil ||
		!reflect.DeepEqual(declined.MyDecision, &MyDecision{Action: "declined", At: declinedAt, RoundNumber: 1, Level: 1}) {
		t.Errorf("first row = %+v / %+v, want A-1002 re-submitted in round 2, declined by Daniel in round 1", declined, declined.MyDecision)
	}
	if approved.Activity.ID != "A-1001" || approved.Activity.Status != StatusApproved || approved.Round.Outcome != OutcomeApproved ||
		approved.WaitingSince != nil || approved.StartDatePassed ||
		!reflect.DeepEqual(approved.MyDecision, &MyDecision{Action: "approved", At: approvedAt, RoundNumber: 1, Level: 1}) {
		t.Errorf("second row = %+v / %+v, want A-1001 approved by Daniel", approved, approved.MyDecision)
	}
	if got := f.inboxIDs(t, alex, TabDecided); !reflect.DeepEqual(got, []string{"A-1003"}) {
		t.Errorf("decided by Alex = %q, want his decision on behalf of Daniel", got)
	}
	if got := f.inboxIDs(t, priya, TabDecided); len(got) != 0 {
		t.Errorf("decided by Priya = %q, want none: submitting is not deciding", got)
	}
	for _, row := range f.inbox(t, daniel, TabWaiting) {
		if row.MyDecision != nil {
			t.Errorf("waiting row %s has myDecision %+v, want null outside the decided tab", row.Activity.ID, row.MyDecision)
		}
	}
}

// An activity in the recycle bin is in no tab and no count, although its open round stays stored for an
// exact Undo (FR-023, 008-P3).
func TestInbox_DeletedActivityAbsent_FR023(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1002", "Fleet roadshow", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1001", "")
	f.submit(t, priya, "A-1002", "")
	f.decline(t, daniel, "A-1002", "No.")

	delete(f.activities.facts, "A-1001") // moved to the recycle bin by 006
	delete(f.activities.facts, "A-1002")

	for _, tab := range []string{TabWaiting, TabOpen, TabDecided} {
		if got := f.inboxIDs(t, daniel, tab); len(got) != 0 {
			t.Errorf("%s for Daniel = %q, want nothing of deleted activities", tab, got)
		}
	}
	counts, err := f.service.Summary(t.Context(), daniel, nil, nil)
	if err != nil || counts != (Counts{}) {
		t.Errorf("summary = %+v (%v), want all zero", counts, err)
	}
	if rounds := f.rounds(t, "A-1001"); len(rounds) != 1 || rounds[0].Outcome != OutcomeOpen {
		t.Errorf("rounds of A-1001 = %+v, want the open round kept", rounds)
	}
}

// A row shows the activity, its path, owner, planned cost and workflow, "Level n of m" with the level's
// approvers, when it was submitted and since when it waits at its level; the start date passing while it
// waits is flagged (FR-030, edge case).
func TestInbox_RowFacts_FR030_EdgeCase(t *testing.T) {
	f := newFixture(t)
	facts := f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.addActivity(t, "A-1010", "Lunar new year promo", "", chinaFlow)
	submittedAt := f.clock.Now()
	f.submit(t, mei, "A-1009", "")
	f.clock.Advance(3 * time.Hour)
	f.approve(t, carlos, "A-1009", "")
	approvedAt := f.clock.Now()
	f.submit(t, priya, "A-1010", "")
	f.clock.Advance(time.Hour)
	round := f.openRound(t, "A-1009")

	rows := f.inbox(t, alex, TabWaiting)

	start := date.MustParse("2026-04-01")
	cost := money.USD(mustAmount(t, "50000.00"))
	want := Row{
		Activity: RowActivity{ID: "A-1009", Name: "Fleet card launch", Status: StatusSubmitted, DisplayStatus: StatusSubmitted},
		Path:     facts.Path, Owner: &PersonView{ID: mei.ID, Name: "Mei Lin", Initials: "ML", Active: true}, PlannedCost: &cost,
		Workflow: WorkflowRef{ID: chinaFlow, Name: "China BC approval workflow", Active: true},
		Round: RowRound{
			ID: round.ID, Number: 1, Outcome: OutcomeOpen, CurrentLevel: 2, LevelCount: 2, LevelLabel: "Level 2",
			Approvers: []PersonView{{ID: alex.ID, Name: "Alex Morgan", Initials: "AM", Active: true}},
		},
		SubmittedAt: submittedAt, WaitingSince: &approvedAt, StartDate: &start,
	}
	if len(rows) != 1 || !reflect.DeepEqual(rows[0], want) {
		t.Fatalf("rows = %+v\nwant [%+v]", rows, want)
	}

	open := f.inbox(t, alex, TabOpen)
	if len(open) != 2 || open[1].Activity.ID != "A-1010" || open[1].Owner != nil || open[1].Round.LevelLabel != "Level 1" ||
		open[1].WaitingSince == nil || !open[1].WaitingSince.Equal(approvedAt) {
		t.Errorf("open = %+v, want A-1010 without owner, waiting at level 1 since its submission", open)
	}

	f.clock.Advance(21 * 24 * time.Hour) // 2 Apr 2026: the start date has passed
	if rows := f.inbox(t, alex, TabWaiting); len(rows) != 1 || !rows[0].StartDatePassed {
		t.Errorf("rows = %+v, want the start date flagged as passed", rows)
	}
}

// The summary counts equal the tabs' rows; Approved and Declined this month count the rounds that ended so
// this calendar month (FR-028, SC-007).
func TestSummary_MatchesTabs_FR028_SC007(t *testing.T) {
	f := newFixture(t)
	for _, id := range []string{"A-1001", "A-1002", "A-1003", "A-1004"} {
		f.addActivity(t, id, "Activity "+id, priya.ID, lubricantsFlow)
	}
	f.addActivity(t, "A-1005", "Fleet card launch", mei.ID, chinaFlow)
	for _, id := range []string{"A-1001", "A-1002", "A-1003", "A-1004", "A-1005"} {
		f.submit(t, priya, id, "")
	}
	f.approve(t, daniel, "A-1001", "")
	f.decline(t, daniel, "A-1002", "No.")
	f.submit(t, priya, "A-1002", "Again.")
	f.decline(t, daniel, "A-1002", "Still no.")

	for _, person := range []actor.Actor{alex, priya, daniel, carlos} {
		counts, err := f.service.Summary(t.Context(), person, nil, nil)
		if err != nil {
			t.Fatalf("Summary(%s): %v", person.Name, err)
		}
		want := Counts{
			WaitingForMe: len(f.inbox(t, person, TabWaiting)), AllOpen: len(f.inbox(t, person, TabOpen)),
			ApprovedThisMonth: 1, DeclinedThisMonth: 2,
		}
		if counts != want {
			t.Errorf("summary for %s = %+v, want %+v", person.Name, counts, want)
		}
	}
	if counts, _ := f.service.Summary(t.Context(), daniel, nil, nil); counts.WaitingForMe != 2 || counts.AllOpen != 3 {
		t.Errorf("Daniel's summary = %+v, want 2 waiting for him of 3 open", counts)
	}
}

// The month is the viewer's calendar month as two instants, end excluded; without them, the clock's UTC
// month. Only one bound, or an end not after the start, is refused (FR-028).
func TestSummary_MonthBounds_FR028(t *testing.T) {
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1002", "Fleet roadshow", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1001", "")
	f.submit(t, priya, "A-1002", "")
	f.approve(t, daniel, "A-1001", "") // 12 Mar 2026 09:00 UTC
	f.clock.Advance(time.Date(2026, 3, 31, 22, 30, 0, 0, time.UTC).Sub(f.clock.Now()))
	f.decline(t, daniel, "A-1002", "No.") // 31 Mar 22:30 UTC: 1 April in Amsterdam (UTC+2)

	amsterdamMarch := [2]time.Time{time.Date(2026, 2, 28, 23, 0, 0, 0, time.UTC), time.Date(2026, 3, 31, 22, 0, 0, 0, time.UTC)}
	amsterdamApril := [2]time.Time{amsterdamMarch[1], time.Date(2026, 4, 30, 22, 0, 0, 0, time.UTC)}
	for _, c := range []struct {
		name  string
		month *[2]time.Time
		want  Counts
	}{
		{"UTC month of the clock", nil, Counts{ApprovedThisMonth: 1, DeclinedThisMonth: 1}},
		{"March in Amsterdam", &amsterdamMarch, Counts{ApprovedThisMonth: 1}},
		{"April in Amsterdam", &amsterdamApril, Counts{DeclinedThisMonth: 1}},
	} {
		var from, until *time.Time
		if c.month != nil {
			from, until = &c.month[0], &c.month[1]
		}
		counts, err := f.service.Summary(t.Context(), priya, from, until)
		if err != nil || counts != c.want {
			t.Errorf("%s: summary = %+v (%v), want %+v", c.name, counts, err, c.want)
		}
	}

	_, onlyStart := f.service.Summary(t.Context(), priya, &amsterdamMarch[0], nil)
	assertError(t, onlyStart, errs.Validation(errs.FieldError{Field: "monthStart", Message: "Reload the page and try again."}))
	_, backwards := f.service.Summary(t.Context(), priya, &amsterdamMarch[1], &amsterdamMarch[0])
	assertError(t, backwards, errs.Validation(errs.FieldError{Field: "monthEnd", Message: "Reload the page and try again."}))
}

// The preview shows the activity's facts and completeness, its line items, its unassigned planned cost and
// its Approval section; an activity that is gone answers not_found (FR-032).
func TestPreview_FR032(t *testing.T) {
	f := newFixture(t)
	facts := f.addActivity(t, "A-1009", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, mei, "A-1009", "")
	usd := money.USD(mustAmount(t, "13800.00"))
	items := []PreviewLineItem{{
		ID: "L-1007", Name: "Launch event", PlannedCost: money.Money{Amount: mustAmount(t, "100000.00"), Currency: "CNY"},
		PlannedCostUSD: &usd, SpendType: "Events", GLCode: "61000 Marketing", FinanceBudgetItem: "Mobility China events",
	}}
	f.activities.lineItems["A-1009"] = items
	f.activities.missing["A-1009"] = []MissingFieldRef{{ID: "attributes.SA-1003", Label: "Cluster"}}

	preview, err := f.service.Preview(t.Context(), carlos, "A-1009")

	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	stored := f.activities.facts["A-1009"]
	want := PreviewActivity{
		ID: "A-1009", Name: "Fleet card launch", Status: StatusSubmitted, DisplayStatus: StatusSubmitted,
		Missing: []MissingFieldRef{{ID: "attributes.SA-1003", Label: "Cluster"}}, Path: facts.Path,
		Owner:     &PersonView{ID: mei.ID, Name: "Mei Lin", Initials: "ML", Active: true},
		StartDate: stored.StartDate, EndDate: stored.EndDate, PlannedCost: stored.PlannedCost, TypeName: "Event",
	}
	if !reflect.DeepEqual(preview.Activity, want) {
		t.Errorf("activity =\n%+v\nwant\n%+v", preview.Activity, want)
	}
	if !reflect.DeepEqual(preview.LineItems, items) || !reflect.DeepEqual(preview.UnassignedPlannedCost, stored.UnassignedPlannedCost) {
		t.Errorf("line items %+v, unassigned %v; want the activity's", preview.LineItems, preview.UnassignedPlannedCost)
	}
	if len(preview.Approval.Rounds) != 1 || !preview.Approval.Viewer.IsCurrentApprover || !preview.Approval.Actions.Approve.Available {
		t.Errorf("approval = %+v, want round 1 waiting for Carlos with Approve", preview.Approval)
	}
	delete(f.activities.missing, "A-1009")
	if complete, err := f.service.Preview(t.Context(), carlos, "A-1009"); err != nil || complete.Activity.Missing == nil {
		t.Errorf("missing of a complete activity = %v (%v), want an empty list", complete.Activity.Missing, err)
	}

	_, gone := f.service.Preview(t.Context(), carlos, "A-9999")
	assertError(t, gone, errs.NotFound("This activity no longer exists — someone may have deleted it."))
}
