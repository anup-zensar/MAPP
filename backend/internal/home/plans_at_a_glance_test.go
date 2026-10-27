package home

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/platform/money"
)

// User story 2 — See how my teams' plans are doing.

// Each row shows the plan with its status, teams, campaign and activity counts, approved activities, and its
// budget, planned cost and budget used in the display currency: red above 100 %, "—" without a budget, and
// a plan with no activities shows 0 of 0 (FR-008, FR-009, US2 scenarios 1–3).
func TestPlansAtAGlance_RowFigures_FR008_FR009(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	lubricants := newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope)
	lubricants.Status, lubricants.Budget, lubricants.PlannedCost = "published", usdPtr(t, "1200000.00"), usdPtr(t, "460000.00")
	lubricants.CampaignCount, lubricants.ActivityCount, lubricants.ApprovedActivityCount = 2, 7, 4
	over := newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina)
	over.Budget, over.PlannedCost, over.CampaignCount, over.ActivityCount = usdPtr(t, "800000.00"), usdPtr(t, "880000.00"), 1, 3
	empty := newPlan("P-1003", "FY2027 Global Brand", 2027, globalMarketingOps)
	empty.StrategicAOP = true
	f.plans = []Plan{lubricants, over, empty}
	f.currencies[sara.ID] = Currency{Code: "EUR", Rate: rate(t, "0.92")}
	s := newService(t, f)

	page, err := s.PlansAtAGlance(t.Context(), sara, "everything")

	if err != nil || page.Total != 3 || len(page.Items) != 3 {
		t.Fatalf("PlansAtAGlance = %+v, %v; want 3 rows", page, err)
	}
	eur := func(text string) money.Money { return money.Money{Amount: usd(t, text), Currency: "EUR"} }
	wantLubricants := PlanRow{ID: "P-1001", Name: "FY2026 Lubricants Europe", Year: 2026, Status: "published",
		Teams:         []TeamView{{ID: lubricantsEurope.ID, Name: "Lubricants Europe", Colour: "blue", Active: true, Mine: true}},
		CampaignCount: 2, ActivityCount: 7, ApprovedActivityCount: 4, Budget: ptr(eur("1104000.00")),
		PlannedCost: ptr(eur("423200.00")), BudgetUsedPercent: ptr(38)}
	if !reflect.DeepEqual(page.Items[1], wantLubricants) {
		t.Errorf("Lubricants row = %+v\nwant %+v", page.Items[1], wantLubricants)
	}
	if row := page.Items[2]; row.BudgetUsedPercent == nil || *row.BudgetUsedPercent != 110 || !row.OverBudget {
		t.Errorf("Mobility row = %+v, want 110 %% and over budget", row)
	}
	wantEmpty := PlanRow{ID: "P-1003", Name: "FY2027 Global Brand", Year: 2027, StrategicAOP: true, Status: "draft",
		Teams: []TeamView{{ID: globalMarketingOps.ID, Name: "Global Marketing Ops", Colour: "violet", Active: true}}}
	if !reflect.DeepEqual(page.Items[0], wantEmpty) {
		t.Errorf("Global Brand row = %+v\nwant no budget, no activities %+v", page.Items[0], wantEmpty)
	}
}

// A plan's teams list the person's own teams first, then the others, each by name (FR-008).
func TestPlansAtAGlance_OwnTeamsFirst_FR008(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	f.plans = []Plan{newPlan("P-1004", "FY2026 Joint plan", 2026, globalMarketingOps, lubricantsEurope, mobilityChina)}
	s := newService(t, f)

	page, err := s.PlansAtAGlance(t.Context(), sara, "myTeams")

	want := []TeamView{{ID: lubricantsEurope.ID, Name: "Lubricants Europe", Colour: "blue", Active: true, Mine: true},
		{ID: mobilityChina.ID, Name: "Mobility China", Colour: "teal", Active: true, Mine: true},
		{ID: globalMarketingOps.ID, Name: "Global Marketing Ops", Colour: "violet", Active: true}}
	if err != nil || len(page.Items) != 1 || !reflect.DeepEqual(page.Items[0].Teams, want) {
		t.Errorf("teams = %+v, %v\nwant %+v", page.Items, err, want)
	}
}

// Rows are ordered by year, newest first, then by name ignoring letter case, then by ID; at most 10 are
// shown, and the total counts every plan in scope (FR-010).
func TestPlansAtAGlance_OrderAndLimit_FR010(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	f.plans = []Plan{
		newPlan("P-1001", "beta", 2026, lubricantsEurope), newPlan("P-1002", "Alpha", 2026, lubricantsEurope),
		newPlan("P-1003", "Zulu", 2027, lubricantsEurope), newPlan("P-1004", "alpha", 2026, lubricantsEurope),
		newPlan("P-1005", "Old", 2025, lubricantsEurope), newPlan("P-1006", "Other team", 2027, mobilityChina),
	}
	for i := range 8 {
		f.plans = append(f.plans, newPlan(fmt.Sprintf("P-20%02d", i), fmt.Sprintf("Plan %d", i), 2024, lubricantsEurope))
	}
	s := newService(t, f)

	page, err := s.PlansAtAGlance(t.Context(), priya, "myTeams")

	var ids []string
	for _, row := range page.Items {
		ids = append(ids, row.ID)
	}
	want := []string{"P-1003", "P-1002", "P-1004", "P-1001", "P-1005", "P-2000", "P-2001", "P-2002", "P-2003", "P-2004"}
	if err != nil || !slices.Equal(ids, want) || page.Total != 13 {
		t.Errorf("rows = %q of %d, %v\nwant %q of 13", ids, page.Total, err, want)
	}
	_, err = s.PlansAtAGlance(t.Context(), priya, "")
	assertError(t, err, scopeRefused)
}
