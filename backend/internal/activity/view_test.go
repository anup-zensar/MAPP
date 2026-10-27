package activity_test

import (
	"slices"
	"testing"

	"mapp/internal/activity"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// User story 2 — see an activity's status and everything it still needs.

// An activity is answered with its path, owner, type and workflow (each with its status), display status,
// unassigned planned cost, line-item totals, completeness and the ready-to-submit check (FR-019 – FR-024).
func TestView_DerivedValues_FR019_FR020(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.lineItems.facts["A-1001"] = activity.LineItemFacts{
		Count: 3, PlannedCostPlan: money.FromCents(10_000_000), AutoSpread: 2, CustomSpread: 1,
		Problems: []activity.LineItemProblem{{LineItemID: "L-1002", LineItemName: "Edit", Code: "missingField",
			Field: "attributes.SA-1033", Message: "GL code — needed to submit"}},
	}
	f.deactivateWorkflow(t, lubricantsFlow)

	view := f.get(t, "A-1001")

	if view.Plan.Name != "FY2026 Lubricants Europe" || view.Campaign.Name != "Helix launch" || view.Initiative.Name != "Digital & social" {
		t.Errorf("path = %+v › %+v › %+v", view.Plan, view.Campaign, view.Initiative)
	}
	if view.Owner.Name != "Priya Raman" || view.ActivityType != (activity.TypeRef{ID: contentType, Name: "Content production", Active: true, Allowed: true}) ||
		*view.ApprovalWorkflow != (activity.WorkflowRef{ID: lubricantsFlow, Name: "Lubricants EU approval", Active: false}) {
		t.Errorf("owner %+v, type %+v, workflow %+v", view.Owner, view.ActivityType, view.ApprovalWorkflow)
	}
	if view.UnassignedPlannedCost.Format() != "20,000.00 USD" || view.LineItems != (activity.LineItemTotals{
		Count: 3, PlannedCost: money.USD(money.FromCents(10_000_000)), AutoSpread: 2, CustomSpread: 1}) {
		t.Errorf("unassigned %v, line items %+v", view.UnassignedPlannedCost, view.LineItems)
	}
	if view.Completeness != (activity.Completeness{MissingFields: 4, LineItemsNeedingAttention: 1}) {
		t.Errorf("completeness = %+v, want 4 fields (3 attributes, inactive workflow) and 1 line item", view.Completeness)
	}
	r := view.Readiness
	if r.Ready || r.ProblemCount != 5 || r.Activity[3].Code != activity.ProblemInactiveWorkflow ||
		r.LineItems[0].Message != "Line item 'Edit' (L-1002): GL code — needed to submit" ||
		r.Advice[0].Message != "20,000.00 USD not yet assigned to a line item" {
		t.Errorf("readiness = %+v", r)
	}
	_, err := f.service.Get(t.Context(), priya, "A-9999")
	assertError(t, err, errs.NotFound("This activity no longer exists — someone may have deleted it."))
}

// Only the attributes shown for the plan's business unit count: on the Mobility China plan, which has no
// business unit, the Lubricants group is neither needed nor listed; a split not adding up to 100.00 % is
// listed with its total (FR-015, FR-017, US2 independent test).
func TestView_HiddenGroupsNotCountedSplitListed_FR015_FR017(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view := f.get(t, "A-1008")

	var messages []string
	for _, p := range view.Readiness.Activity {
		messages = append(messages, p.Field+": "+p.Message)
	}
	want := []string{
		"attributes.SA-1013: Country / OU shares add up to 90.00 % — they must add up to 100.00 %",
		"attributes.SA-1015: Cost centre — needed to submit",
	}
	if !slices.Equal(messages, want) || view.Completeness.MissingFields != 2 {
		t.Errorf("problems = %q, want %q", messages, want)
	}
}

// The plan's rows are every activity of every initiative of the plan, ordered by start date, then name
// ignoring letter case, activities without a start date last (FR-033, D-111).
func TestListForPlan_Order_FR033(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.create(t, "I-1005", activity.Input{Name: "b promo", ActivityTypeID: eventType, StartDate: text("2026-04-01")})
	f.create(t, "I-1005", activity.Input{Name: "A promo", ActivityTypeID: eventType, StartDate: text("2026-04-01")})
	f.create(t, fleetCards, activity.Input{Name: "Another undated", ActivityTypeID: eventType})

	rows, err := f.service.ListForPlan(t.Context(), priya, "P-1002")

	if err != nil {
		t.Fatalf("ListForPlan: %v", err)
	}
	var names []string
	for _, r := range rows {
		names = append(names, r.Name)
	}
	want := []string{"A promo", "b promo", "Grand opening Shanghai", "Fleet card launch", "Another undated", "Lunar new year promo"}
	if !slices.Equal(names, want) {
		t.Errorf("rows = %q, want %q", names, want)
	}
	if rows[2].Completeness.MissingFields != 2 || rows[2].Version != 1 || rows[3].DisplayStatus != activity.DisplaySubmitted {
		t.Errorf("rows carry completeness, version and status: %+v", rows[2:4])
	}
	_, err = f.service.ListForPlan(t.Context(), priya, "P-9999")
	assertError(t, err, errs.NotFound("This plan no longer exists — someone may have deleted it."))
}

// Approved activities show In progress from their start and Complete after their end, never when ongoing
// (FR-021): the demo data on 12 Mar 2026.
func TestView_DisplayStatusOfDemoActivities_FR021(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	want := map[string]activity.DisplayStatus{
		"A-1002": activity.DisplayInProgress, // 1 Jan – 30 Jun
		"A-1003": activity.DisplayComplete,   // ended 28 Feb
		"A-1006": activity.DisplayApproved,   // starts 1 Nov
		"A-1007": activity.DisplayInProgress, // ongoing
		"A-1004": activity.DisplaySubmitted,
		"A-1005": activity.DisplayDeclined,
	}
	for id, status := range want {
		if got := f.get(t, id).DisplayStatus; got != status {
			t.Errorf("%s shows %s, want %s", id, got, status)
		}
	}
}
