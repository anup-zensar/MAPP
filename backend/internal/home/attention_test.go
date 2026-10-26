package home

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// User story 3 — Know what needs me.

// The records of the demo story and their places.
var (
	lubricantsRef = errs.RecordRef{Type: "plan", ID: "P-1001", Name: "FY2026 Lubricants Europe"}
	helixRef      = errs.RecordRef{Type: "campaign", ID: "C-1001", Name: "Helix launch"}
	digitalRef    = errs.RecordRef{Type: "initiative", ID: "I-1001", Name: "Digital & social"}
)

// activityIn returns an activity of Digital & social, Created, changed at changed.
func activityIn(id, name string, changed time.Time) Activity {
	return Activity{ID: id, Name: name, Plan: lubricantsRef, Campaign: helixRef, Initiative: digitalRef,
		Status: "created", DisplayStatus: "created", Missing: []MissingField{}, UpdatedAt: changed}
}

// reasonKinds returns the kinds of an item's reasons, in order.
func reasonKinds(item HomeAttentionItem) []string {
	kinds := []string{}
	for _, r := range item.Reasons {
		kinds = append(kinds, r.Kind)
	}
	return kinds
}

// itemIDs returns the IDs of the items, in order.
func itemIDs(page HomeAttentionPage) []string {
	ids := []string{}
	for _, item := range page.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

// Each reason of FR-011 with its data, amounts in the display currency: the decline with who, when and why;
// line items above the planned cost; children's budgets above a budget; planned cost above a budget; fields
// still needed with the records below that need attention.
func TestAttention_EachReason_FR011(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	declinedAt := now.Add(-5 * 24 * time.Hour)
	roadshow := activityIn("A-1005", "Fleet roadshow", now)
	roadshow.Status, roadshow.DisplayStatus = "declined", "declined"
	roadshow.Decline = &Decline{By: daniel, At: declinedAt, Comment: "Please reduce it."}
	launchFilm := activityIn("A-1001", "Launch film", now)
	launchFilm.Unassigned = usdPtr(t, "-1200.00")
	f.activities[priya.ID] = []Activity{roadshow, launchFilm}
	helix := Container{Type: "campaign", ID: helixRef.ID, Name: helixRef.Name, Plan: lubricantsRef, Path: []errs.RecordRef{lubricantsRef},
		Budget: usdPtr(t, "300000.00"), RemainingBudget: usdPtr(t, "-50000.00"), Missing: []MissingField{}, UpdatedAt: now}
	digital := Container{Type: "initiative", ID: digitalRef.ID, Name: digitalRef.Name, Plan: lubricantsRef,
		Path: []errs.RecordRef{lubricantsRef, helixRef}, Budget: usdPtr(t, "100.00"), PlannedCost: usdPtr(t, "150.00"),
		Missing: []MissingField{}, UpdatedAt: now}
	f.containers[priya.ID] = []Container{helix, digital}
	plan := newPlan(lubricantsRef.ID, lubricantsRef.Name, 2026, lubricantsEurope)
	plan.OwnerID, plan.Missing, plan.ItemsBelow = priya.ID, []MissingField{{Field: "goalStage", Label: "Goal stage"}}, 2
	f.plans = []Plan{plan}
	f.currencies[priya.ID] = Currency{Code: "EUR", Rate: rate(t, "0.92")}
	s := newService(t, f)

	page, err := s.Attention(t.Context(), priya)

	if err != nil || page.Total != 5 || len(page.Items) != 5 {
		t.Fatalf("Attention = %d items of %d, %v; want 5", len(page.Items), page.Total, err)
	}
	reasonOf := func(id string) HomeAttentionReason {
		for _, item := range page.Items {
			if item.ID == id {
				return item.Reasons[0]
			}
		}
		t.Fatalf("%s is not listed", id)
		return HomeAttentionReason{}
	}
	eur := func(text string) *money.Money { return &money.Money{Amount: usd(t, text), Currency: "EUR"} }
	comment := "Please reduce it."
	want := map[string]HomeAttentionReason{
		"A-1005": {Kind: "declined", DeclinedBy: &daniel, DeclinedAt: &declinedAt, Comment: &comment, Missing: []MissingField{}},
		"A-1001": {Kind: "negativeBalance", Amount: eur("1104.00"), Missing: []MissingField{}},
		"C-1001": {Kind: "overAllocated", Amount: eur("46000.00"), Missing: []MissingField{}},
		"I-1001": {Kind: "overBudget", Amount: eur("46.00"), Missing: []MissingField{}},
		"P-1001": {Kind: "missingFields", Missing: []MissingField{{Field: "goalStage", Label: "Goal stage"}}, ItemsBelow: 2},
	}
	for id, w := range want {
		if got := reasonOf(id); !reflect.DeepEqual(got, w) {
			t.Errorf("%s: reason = %+v\nwant %+v", id, got, w)
		}
	}
	var planItem, activity HomeAttentionItem
	for _, item := range page.Items {
		switch item.ID {
		case "P-1001":
			planItem = item
		case "A-1001":
			activity = item
		}
	}
	if planItem.RecordType != "plan" || planItem.RecordTypeLabel != "plan" || planItem.Plan != lubricantsRef || len(planItem.Path) != 0 ||
		planItem.Path == nil {
		t.Errorf("plan item = %+v, want the plan itself and an empty path", planItem)
	}
	if activity.RecordType != "activity" || activity.Plan != lubricantsRef ||
		!reflect.DeepEqual(activity.Path, []errs.RecordRef{lubricantsRef, helixRef, digitalRef}) || !activity.ChangedAt.Equal(now) {
		t.Errorf("activity item = %+v, want its plan, path and last change", activity)
	}
}

// Only records the person owns are listed: someone else's plan is not, nor is a record without any reason
// (FR-011).
func TestAttention_OnlyOwnedRecords_FR011(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	mine := newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope)
	mine.OwnerID, mine.ItemsBelow = priya.ID, 1
	theirs := newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina)
	theirs.OwnerID, theirs.ItemsBelow = mei.ID, 3
	fine := newPlan("P-1003", "FY2027 Priya's fine plan", 2027, lubricantsEurope)
	fine.OwnerID, fine.Budget, fine.PlannedCost = priya.ID, usdPtr(t, "100.00"), usdPtr(t, "100.00")
	f.plans = []Plan{mine, theirs, fine}
	f.activities[priya.ID] = []Activity{activityIn("A-1002", "Complete and fine", now)}
	unassignedLeft := activityIn("A-1003", "Money left to assign", now)
	unassignedLeft.Unassigned = usdPtr(t, "500.00") // a positive balance is advice only (D-204)
	f.activities[priya.ID] = append(f.activities[priya.ID], unassignedLeft)
	s := newService(t, f)

	page, err := s.Attention(t.Context(), priya)

	if ids := itemIDs(page); err != nil || !slices.Equal(ids, []string{"P-1001"}) || page.Total != 1 {
		t.Errorf("items = %q of %d, %v; want only her plan with a problem", ids, page.Total, err)
	}
	nothing, err := s.Attention(t.Context(), nina)
	if err != nil || len(nothing.Items) != 0 || nothing.Items == nil || nothing.Total != 0 {
		t.Errorf("Nina = %+v, %v; want an empty list", nothing, err)
	}
}

// A record with several problems appears once with all its reasons, in the order of FR-011 (FR-012, US3
// scenario 4).
func TestAttention_OneItemAllReasons_FR012(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	p := newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope)
	p.OwnerID, p.Budget, p.PlannedCost, p.RemainingBudget = priya.ID, usdPtr(t, "1000.00"), usdPtr(t, "1500.00"), usdPtr(t, "-200.00")
	p.Missing = []MissingField{{Field: "teamIds", Label: "Teams"}}
	f.plans = []Plan{p}
	act := activityIn("A-1005", "Fleet roadshow", now)
	act.Status, act.Unassigned, act.LineItemsNeedingAttention = "declined", usdPtr(t, "-10.00"), 1
	f.activities[priya.ID] = []Activity{act}
	s := newService(t, f)

	page, err := s.Attention(t.Context(), priya)

	if err != nil || len(page.Items) != 2 {
		t.Fatalf("Attention = %+v, %v; want 2 items", page, err)
	}
	if kinds := reasonKinds(page.Items[0]); !slices.Equal(kinds, []string{"declined", "negativeBalance", "missingFields"}) {
		t.Errorf("activity reasons = %q", kinds)
	}
	if kinds := reasonKinds(page.Items[1]); !slices.Equal(kinds, []string{"overAllocated", "overBudget", "missingFields"}) {
		t.Errorf("plan reasons = %q", kinds)
	}
	if r := page.Items[0].Reasons[0]; r.DeclinedBy != nil || r.Comment != nil {
		t.Errorf("decline without a recorded decision = %+v, want null details", r)
	}
}

// Items are ordered by their first reason: declined, then balance, then budget, then missing fields; within
// a group the most recently changed first, then by ID (FR-012).
func TestAttention_Order_FR012(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	at := func(hoursAgo int) time.Time { return now.Add(-time.Duration(hoursAgo) * time.Hour) }
	missingOld := activityIn("A-1001", "Missing, old", at(30))
	missingOld.Missing = []MissingField{{Field: "ownerId", Label: "Owner"}}
	missingNew := activityIn("A-1002", "Missing, new", at(1))
	missingNew.Missing = missingOld.Missing
	balance := activityIn("A-1003", "Balance", at(40))
	balance.Unassigned = usdPtr(t, "-1.00")
	declined := activityIn("A-1004", "Declined", at(50))
	declined.Status = "declined"
	sameTimeB := activityIn("A-1006", "Missing, same time B", at(5))
	sameTimeB.Missing = missingOld.Missing
	sameTimeA := activityIn("A-1005", "Missing, same time A", at(5))
	sameTimeA.Missing = missingOld.Missing
	f.activities[priya.ID] = []Activity{missingOld, missingNew, balance, declined, sameTimeB, sameTimeA}
	budget := Container{Type: "initiative", ID: "I-1001", Name: "Over budget", Plan: lubricantsRef,
		Path: []errs.RecordRef{lubricantsRef, helixRef}, Budget: usdPtr(t, "1.00"), PlannedCost: usdPtr(t, "2.00"), UpdatedAt: at(100)}
	f.containers[priya.ID] = []Container{budget}
	s := newService(t, f)

	page, err := s.Attention(t.Context(), priya)

	want := []string{"A-1004", "A-1003", "I-1001", "A-1002", "A-1005", "A-1006", "A-1001"}
	if ids := itemIDs(page); err != nil || !slices.Equal(ids, want) {
		t.Errorf("order = %q, %v\nwant %q", ids, err, want)
	}
}

// Line items never appear on their own: their problems count in their activity as line items needing
// attention (Assumptions).
func TestAttention_LineItemsThroughActivity_Assumption(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	act := activityIn("A-1001", "Launch film", now)
	act.LineItemsNeedingAttention = 2
	f.activities[priya.ID] = []Activity{act}
	s := newService(t, f)

	page, err := s.Attention(t.Context(), priya)

	want := []HomeAttentionReason{{Kind: "missingFields", Missing: []MissingField{}, ItemsBelow: 2}}
	if err != nil || len(page.Items) != 1 || page.Items[0].RecordType != "activity" || !reflect.DeepEqual(page.Items[0].Reasons, want) {
		t.Errorf("items = %+v, %v; want the activity with 2 items below", page.Items, err)
	}
}

// User story 4 — Follow my activities.

// My activities groups the person's activities by displayed status in the order Declined, Created,
// Submitted, Approved, In progress, Complete, leaving empty groups out; within a group by start date (none
// last), then name, then ID; each row with its place, dates, planned cost in the display currency, the
// fields still needed and the line items needing attention (FR-014, US4 scenario 1).
func TestMyActivities_Groups_FR014(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	d := func(text string) *date.Date { day := date.MustParse(text); return &day }
	withStatus := func(id, name, status string, start *date.Date) Activity {
		act := activityIn(id, name, now)
		act.Status, act.DisplayStatus, act.StartDate = status, status, start
		return act
	}
	launchFilm := withStatus("A-1001", "Launch film", "created", d("2026-03-01"))
	launchFilm.EndDate, launchFilm.PlannedCost = d("2026-05-31"), usdPtr(t, "120000.00")
	launchFilm.Missing, launchFilm.LineItemsNeedingAttention = []MissingField{{Field: "ownerId", Label: "Owner"}}, 2
	f.activities[priya.ID] = []Activity{
		launchFilm,
		withStatus("A-1010", "No dates yet", "created", nil),
		withStatus("A-1011", "early bird", "created", d("2026-01-01")),
		withStatus("A-1012", "Early bird", "created", d("2026-01-01")),
		withStatus("A-1005", "Fleet roadshow", "declined", d("2026-07-01")),
		withStatus("A-1006", "Billboard winter wave", "complete", d("2026-01-05")),
	}
	f.currencies[priya.ID] = Currency{Code: "EUR", Rate: rate(t, "0.92")}
	s := newService(t, f)

	view, err := s.MyActivities(t.Context(), priya)

	if err != nil || view.Total != 6 {
		t.Fatalf("MyActivities = %+v, %v; want 6 activities", view, err)
	}
	var groups []string
	for _, g := range view.Groups {
		groups = append(groups, fmt.Sprintf("%s %d", g.Status, g.Count))
	}
	if want := []string{"declined 1", "created 4", "complete 1"}; !slices.Equal(groups, want) {
		t.Errorf("groups = %q, want %q (empty ones left out)", groups, want)
	}
	var created []string
	for _, row := range view.Groups[1].Items {
		created = append(created, row.ID)
	}
	if want := []string{"A-1011", "A-1012", "A-1001", "A-1010"}; !slices.Equal(created, want) {
		t.Errorf("created = %q, want %q (by start date, no date last, then name, then ID)", created, want)
	}
	want := MyActivityRow{ID: "A-1001", Name: "Launch film", Plan: lubricantsRef, Campaign: helixRef, Initiative: digitalRef,
		Status: "created", StartDate: d("2026-03-01"), EndDate: d("2026-05-31"),
		PlannedCost: &money.Money{Amount: usd(t, "110400.00"), Currency: "EUR"},
		Missing:     []MissingField{{Field: "ownerId", Label: "Owner"}}, ItemsBelow: 2}
	if got := view.Groups[1].Items[2]; !reflect.DeepEqual(got, want) {
		t.Errorf("Launch film row = %+v\nwant %+v", got, want)
	}
	nothing, err := s.MyActivities(t.Context(), nina)
	if err != nil || nothing.Total != 0 || nothing.Groups == nil || len(nothing.Groups) != 0 {
		t.Errorf("Nina = %+v, %v; want no groups", nothing, err)
	}
}

// The group is the status the activity's page shows, not the stored one: an approved activity shows In
// progress or Complete, and an ongoing one is never Complete (edge case, domain model §2.3).
func TestMyActivities_UsesDisplayedStatus_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	running := activityIn("A-1002", "Social always-on", now)
	running.Status, running.DisplayStatus = "approved", "inProgress"
	ongoing := activityIn("A-1007", "Partner co-marketing fund", now)
	ongoing.Status, ongoing.DisplayStatus, ongoing.Ongoing = "approved", "inProgress", true
	done := activityIn("A-1003", "Billboard winter wave", now)
	done.Status, done.DisplayStatus = "approved", "complete"
	f.activities[priya.ID] = []Activity{running, ongoing, done}
	s := newService(t, f)

	view, err := s.MyActivities(t.Context(), priya)

	if err != nil || len(view.Groups) != 2 || view.Groups[0].Status != "inProgress" || view.Groups[0].Count != 2 ||
		view.Groups[1].Status != "complete" || view.Groups[1].Items[0].ID != "A-1003" || !view.Groups[0].Items[0].Ongoing {
		t.Errorf("groups = %+v, %v; want In progress (2, one ongoing) and Complete (1)", view.Groups, err)
	}
}
