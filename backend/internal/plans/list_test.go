package plans

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/platform/money"
)

// putDemoPlans stores the three plans of the demo data, with planned costs below two of them.
func (f *fixture) putDemoPlans(t *testing.T) {
	t.Helper()
	f.put(t,
		seedPlan(t, "P-1001", "FY2026 Lubricants Europe", 2026, teamLubricants, priya.ID, "1200000.00", StatusPublished),
		seedPlan(t, "P-1002", "FY2026 Mobility China", 2026, teamMobility, "U-1004", "800000.00", StatusDraft),
		seedPlan(t, "P-1003", "FY2027 Global Brand", 2027, teamGlobal, "U-1001", "500000.00", StatusDraft))
	planned := func(text string) *money.Amount { amount := usd(t, text); return &amount }
	f.addLevel(t, "activity", map[string]Facts{
		"P-1001": {PlannedCost: planned("960000.00")}, "P-1003": {PlannedCost: planned("100000.00")},
	})
}

// itemIDs returns the IDs of a list's plans, in order.
func itemIDs(list PlanList) []string {
	planIDs := []string{}
	for _, item := range list.Items {
		planIDs = append(planIDs, item.ID)
	}
	return planIDs
}

// The Plans list is sorted by year, newest first, then by name; status and year narrow it; total counts
// the plans in the team scope, and the summary covers the scope and the chosen year whatever the status
// (FR-001, FR-003, D-080, US2 scenarios 1 and 2).
func TestList_FiltersSortAndSummary_FR001_FR003(t *testing.T) { //nolint:gocyclo // 10: two lists, then the rows, totals and summary of each checked in turn
	t.Parallel()
	f := newFixture(t)
	f.putDemoPlans(t)
	usdOf := func(text string) *money.Money { m := money.USD(usd(t, text)); return &m }

	all, err := f.service.List(t.Context(), priya, ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	draft2026, err := f.service.List(t.Context(), priya, ListFilter{Status: "draft", Year: ptr(2026)})
	if err != nil {
		t.Fatalf("List(draft, 2026): %v", err)
	}

	if ids := itemIDs(all); !slices.Equal(ids, []string{"P-1003", "P-1001", "P-1002"}) || all.Total != 3 ||
		!slices.Equal(all.Years, []int{2027, 2026}) {
		t.Errorf("all: %q, total %d, years %v", ids, all.Total, all.Years)
	}
	wantAll := ListSummary{Plans: 3, Draft: 2, Published: 1, PlannedCost: usdOf("1060000.00"), Budget: usdOf("2500000.00")}
	if !reflect.DeepEqual(all.Summary, wantAll) {
		t.Errorf("summary = %+v, want %+v", all.Summary, wantAll)
	}
	if ids := itemIDs(draft2026); !slices.Equal(ids, []string{"P-1002"}) || draft2026.Total != 3 {
		t.Errorf("draft 2026: %q, total %d; want P-1002 of 3", ids, draft2026.Total)
	}
	want2026 := ListSummary{Plans: 2, Draft: 1, Published: 1, PlannedCost: usdOf("960000.00"), Budget: usdOf("2000000.00")}
	if !reflect.DeepEqual(draft2026.Summary, want2026) {
		t.Errorf("summary of 2026 = %+v, want %+v", draft2026.Summary, want2026)
	}
	assertLubricantsEuropeRow(t, all.Items[1])

	_, err = f.service.List(t.Context(), priya, ListFilter{Scope: "mine", Status: "open"})
	assertError(t, err, problems("scope", "Choose all or myTeams.", "status", "Choose draft or published."))
}

// assertLubricantsEuropeRow checks the list row of the demo plan FY2026 Lubricants Europe: its owner, 80 % of
// its budget used, nothing missing and its team.
func assertLubricantsEuropeRow(t *testing.T, item PlanSummary) {
	t.Helper()
	if item.Owner == nil || item.Owner.Name != "Priya Raman" || item.Figures.BudgetUsedPercent == nil ||
		*item.Figures.BudgetUsedPercent != 80 || len(item.Completeness.Missing) != 0 || item.Teams[0].Name != "Lubricants Europe" {
		t.Errorf("FY2026 Lubricants Europe = %+v", item)
	}
}

// My teams keeps the plans with one of the person's teams; All plans is every plan (FR-004, Q-006, US2
// scenario 4).
func TestList_MyTeams_FR004(t *testing.T) { //nolint:gocyclo // 10: three people's lists, each checked in one condition
	t.Parallel()
	f := newFixture(t)
	f.putDemoPlans(t)

	mei, err := f.service.List(t.Context(), actorOf("U-1004", "Mei Lin"), ListFilter{Scope: ScopeMyTeams})
	carlos, _ := f.service.List(t.Context(), actorOf("U-1005", "Carlos Mendes"), ListFilter{Scope: ScopeMyTeams})
	alone, _ := f.service.List(t.Context(), nina, ListFilter{Scope: ScopeMyTeams})

	if ids := itemIDs(mei); err != nil || !slices.Equal(ids, []string{"P-1002"}) || mei.Total != 1 || !slices.Equal(mei.Years, []int{2026}) {
		t.Errorf("Mei: %q, total %d, years %v, %v", ids, mei.Total, mei.Years, err)
	}
	if ids := itemIDs(carlos); !slices.Equal(ids, []string{"P-1003", "P-1002"}) {
		t.Errorf("Carlos: %q, want both of his teams' plans", ids)
	}
	if len(alone.Items) != 0 || alone.Total != 0 || alone.Summary.Budget != nil || alone.Summary.PlannedCost != nil {
		t.Errorf("Nina: %+v, want nothing and no totals", alone)
	}
}

// The search finds any part of the name or of the ID, ignoring letter case and surrounding spaces (FR-003,
// US2 scenario 3).
func TestList_SearchNameOrID_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.putDemoPlans(t)
	for _, q := range []string{"MOBILITY", "p-1002", " 1002 "} {
		list, err := f.service.List(t.Context(), priya, ListFilter{Q: q})

		if ids := itemIDs(list); err != nil || !slices.Equal(ids, []string{"P-1002"}) || list.Total != 3 {
			t.Errorf("q %q: %q, total %d, %v; want P-1002 of 3", q, ids, list.Total, err)
		}
	}
}

// With 200 plans the list asks each level below once for all of them, never once per plan (SC-009).
func TestList_200PlansOneFactsCallPerLevel_SC009(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	many := make([]Plan, 0, 200)
	for i := range 200 {
		many = append(many, seedPlan(t, fmt.Sprintf("P-%d", 1001+i), fmt.Sprintf("Plan %03d", i), 2026, teamLubricants, priya.ID, "1000.00", StatusDraft))
	}
	f.put(t, many...)
	campaigns := f.addLevel(t, "campaign", nil)
	activities := f.addLevel(t, "activity", nil)

	list, err := f.service.List(t.Context(), priya, ListFilter{})

	if err != nil || len(list.Items) != 200 || campaigns.factCalls != 1 || activities.factCalls != 1 {
		t.Errorf("%d plans, %v; facts asked %d and %d times, want once per level", len(list.Items), err, campaigns.factCalls, activities.factCalls)
	}
}
