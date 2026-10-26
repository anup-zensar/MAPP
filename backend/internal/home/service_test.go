package home

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/race"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// My teams covers the plans with at least one of the person's teams, each once even when two of its teams
// are the person's; Everything covers every plan (FR-003, FR-004).
func TestPlansInScope_MyTeamsAndEverything_FR003_FR004(t *testing.T) {
	f := newFakeSources()
	f.plans = []Plan{
		newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope),
		newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina),
		newPlan("P-1003", "FY2027 Global Brand", 2027, globalMarketingOps),
		newPlan("P-1004", "FY2026 Joint plan", 2026, lubricantsEurope, mobilityChina),
	}
	s := newService(t, f)
	inScope := func(a actor.Actor, scope Scope) []string {
		ps, err := s.planScope(t.Context(), a)
		if err != nil {
			t.Fatalf("planScope: %v", err)
		}
		return planIDs(ps.in(scope))
	}

	if got := inScope(sara, ScopeMyTeams); !slices.Equal(got, []string{"P-1001", "P-1002", "P-1004"}) {
		t.Errorf("Sara, My teams = %q, want her two teams' plans, the joint plan once", got)
	}
	if got := inScope(priya, ScopeMyTeams); !slices.Equal(got, []string{"P-1001", "P-1004"}) {
		t.Errorf("Priya, My teams = %q", got)
	}
	if got := inScope(nina, ScopeMyTeams); len(got) != 0 {
		t.Errorf("Nina (no team), My teams = %q, want none", got)
	}
	if got := inScope(nina, ScopeEverything); !slices.Equal(got, []string{"P-1001", "P-1002", "P-1003", "P-1004"}) {
		t.Errorf("Everything = %q, want every plan", got)
	}
}

// Only myTeams and everything are scopes; anything else is refused on the field scope.
func TestScope_Invalid(t *testing.T) {
	for _, text := range []string{"myTeams", "everything"} {
		if scope, err := ParseScope(text); err != nil || string(scope) != text {
			t.Errorf("ParseScope(%q) = %q, %v", text, scope, err)
		}
	}
	for _, text := range []string{"", "all", "MyTeams", "mine"} {
		_, err := ParseScope(text)

		assertError(t, err, scopeRefused)
	}
	s := newService(t, newFakeSources())
	_, err := s.Summary(t.Context(), priya, "all")
	assertError(t, err, scopeRefused)
}

// User story 1 — See my work at a glance.

// The cards count the plans in scope by status, the activities the person owns with the Declined and
// Submitted ones, and what waits for their approval with the oldest "waiting since" (FR-005, D-175).
func TestSummary_Counts_FR005(t *testing.T) {
	f := newFakeSources()
	published := newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope)
	published.Status = "published"
	f.plans = []Plan{published, newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina),
		newPlan("P-1003", "FY2027 Global Brand", 2027, globalMarketingOps)}
	f.activities[priya.ID] = []Activity{
		{ID: "A-1001", Status: "created"}, {ID: "A-1004", Status: "submitted"}, {ID: "A-1005", Status: "declined"},
		{ID: "A-1002", Status: "approved", DisplayStatus: "inProgress"},
	}
	f.waiting[priya.ID] = []Waiting{
		{ActivityID: "A-1009", WaitingSince: now.Add(-2 * time.Hour)}, // submitted first, but reached its level later
		{ActivityID: "A-1004", WaitingSince: now.Add(-50 * time.Hour)},
	}
	s := newService(t, f)

	mine, err := s.Summary(t.Context(), priya, "myTeams")
	all, err2 := s.Summary(t.Context(), priya, "everything")

	if err != nil || err2 != nil {
		t.Fatalf("Summary: %v, %v", err, err2)
	}
	if mine.Scope != ScopeMyTeams || mine.Plans != (PlanCounts{Count: 1, Published: 1, Draft: 0}) {
		t.Errorf("My teams: scope %q, plans %+v; want 1 published", mine.Scope, mine.Plans)
	}
	if all.Scope != ScopeEverything || all.Plans != (PlanCounts{Count: 3, Published: 1, Draft: 2}) {
		t.Errorf("Everything: scope %q, plans %+v; want 1 published, 2 draft", all.Scope, all.Plans)
	}
	if mine.ActivitiesOwned != (ActivityCounts{Count: 4, Declined: 1, Submitted: 1}) {
		t.Errorf("activities owned = %+v, want 4 with 1 declined and 1 submitted", mine.ActivitiesOwned)
	}
	if w := mine.WaitingForMe; w.Count != 2 || w.OldestWaitingSince == nil || !w.OldestWaitingSince.Equal(now.Add(-50*time.Hour)) {
		t.Errorf("waiting for me = %+v, want 2, the oldest waiting for 50 hours", w)
	}
	empty, _ := s.Summary(t.Context(), daniel, "myTeams")
	if empty.WaitingForMe != (WaitingCounts{}) || empty.ActivitiesOwned != (ActivityCounts{}) {
		t.Errorf("Daniel: waiting %+v, owned %+v; want nothing", empty.WaitingForMe, empty.ActivitiesOwned)
	}
}

// The person's own activities count in every plan, also outside their teams and whatever the scope (edge
// case, FR-003 hint).
func TestSummary_OwnedActivitiesCoverAllPlans_EdgeCase(t *testing.T) {
	f := newFakeSources()
	f.plans = []Plan{newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope),
		newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina)}
	outside := errs.RecordRef{Type: "plan", ID: "P-1002", Name: "FY2026 Mobility China"}
	f.activities[priya.ID] = []Activity{{ID: "A-1008", Plan: outside, Status: "submitted"}}
	s := newService(t, f)

	for _, scope := range []string{"myTeams", "everything"} {
		got, err := s.Summary(t.Context(), priya, scope)

		if err != nil || got.ActivitiesOwned != (ActivityCounts{Count: 1, Submitted: 1}) {
			t.Errorf("%s: activities owned = %+v, %v; want the activity outside her teams", scope, got.ActivitiesOwned, err)
		}
	}
}

// Budget used is the total planned cost of the plans in scope over their total plan budget, half up to a
// whole percent; over budget is compared exactly, so 100.4 % shows 100 and is over (FR-006, D-176).
func TestSummary_BudgetUsed_FR006(t *testing.T) {
	f := newFakeSources()
	lubricants := newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope)
	lubricants.Budget, lubricants.PlannedCost = usdPtr(t, "1200000.00"), usdPtr(t, "460000.00")
	joint := newPlan("P-1004", "FY2026 Joint plan", 2026, lubricantsEurope, mobilityChina)
	joint.Budget, joint.PlannedCost = usdPtr(t, "800000.00"), usdPtr(t, "500000.00")
	f.plans = []Plan{lubricants, joint}
	s := newService(t, f)

	got, err := s.Summary(t.Context(), sara, "myTeams")

	want := PlannedCostTotal{PlanCount: 2, PlannedCost: money.USD(usd(t, "960000.00")),
		PlanBudget: ptr(money.USD(usd(t, "2000000.00"))), BudgetUsedPercent: ptr(48)}
	if err != nil || !reflect.DeepEqual(got.PlannedCost, want) {
		t.Errorf("planned cost = %+v, %v\nwant %+v", got.PlannedCost, err, want)
	}

	lubricants.PlannedCost = usdPtr(t, "1204800.00") // 100.4 % of 1,200,000.00
	f.plans = []Plan{lubricants}
	over, _ := s.Summary(t.Context(), priya, "myTeams")
	if p := over.PlannedCost; p.BudgetUsedPercent == nil || *p.BudgetUsedPercent != 100 || !p.OverBudget {
		t.Errorf("100.4 %%: %+v; want 100 %% and over budget", p)
	}
	lubricants.PlannedCost = usdPtr(t, "1200000.00")
	f.plans = []Plan{lubricants}
	full, _ := s.Summary(t.Context(), priya, "myTeams")
	if p := full.PlannedCost; p.BudgetUsedPercent == nil || *p.BudgetUsedPercent != 100 || p.OverBudget {
		t.Errorf("exactly the budget: %+v; want 100 %% and not over", p)
	}
}

// Plans without a budget add their planned cost but no budget, and are counted (FR-006, edge case).
func TestSummary_PlansWithoutBudget_FR006(t *testing.T) {
	f := newFakeSources()
	withBudget := newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope)
	withBudget.Budget, withBudget.PlannedCost = usdPtr(t, "1000000.00"), usdPtr(t, "400000.00")
	noBudget := newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina)
	noBudget.PlannedCost = usdPtr(t, "100000.00")
	nothing := newPlan("P-1003", "FY2027 Global Brand", 2027, globalMarketingOps)
	f.plans = []Plan{withBudget, noBudget, nothing}
	s := newService(t, f)

	got, err := s.Summary(t.Context(), priya, "everything")

	want := PlannedCostTotal{PlanCount: 3, PlannedCost: money.USD(usd(t, "500000.00")),
		PlanBudget: ptr(money.USD(usd(t, "1000000.00"))), BudgetUsedPercent: ptr(50), PlansWithoutBudget: 2}
	if err != nil || !reflect.DeepEqual(got.PlannedCost, want) {
		t.Errorf("planned cost = %+v, %v\nwant %+v", got.PlannedCost, err, want)
	}
}

// Without any plan budget the card shows "—" and "No plan budgets set"; without plans, nothing at all
// (FR-006).
func TestSummary_NoBudgetAtAll_FR006(t *testing.T) {
	f := newFakeSources()
	p := newPlan("P-1003", "FY2027 Global Brand", 2027, globalMarketingOps)
	p.PlannedCost = usdPtr(t, "250000.00")
	f.plans = []Plan{p}
	s := newService(t, f)

	got, err := s.Summary(t.Context(), alex, "myTeams")
	none, err2 := s.Summary(t.Context(), nina, "myTeams")

	want := PlannedCostTotal{PlanCount: 1, PlannedCost: money.USD(usd(t, "250000.00")), PlansWithoutBudget: 1}
	if err != nil || !reflect.DeepEqual(got.PlannedCost, want) {
		t.Errorf("no budget = %+v, %v\nwant %+v", got.PlannedCost, err, want)
	}
	wantNone := PlannedCostTotal{PlannedCost: money.USD(money.Amount{})}
	if err2 != nil || !reflect.DeepEqual(none.PlannedCost, wantNone) {
		t.Errorf("no plans = %+v, %v\nwant %+v", none.PlannedCost, err2, wantNone)
	}
}

// Totals are added in US dollars and converted once: 1,000.05 + 1,000.05 USD at 0.92 is 1,840.09 EUR, not
// the 1,840.10 of two converted amounts (FR-017, D-010).
func TestSummary_TotalsConvertedOnce_FR017(t *testing.T) {
	f := newFakeSources()
	one := newPlan("P-1001", "Plan one", 2026, lubricantsEurope)
	one.Budget, one.PlannedCost = usdPtr(t, "1000.05"), usdPtr(t, "1000.05")
	two := newPlan("P-1002", "Plan two", 2026, lubricantsEurope)
	two.Budget, two.PlannedCost = usdPtr(t, "1000.05"), usdPtr(t, "1000.05")
	f.plans = []Plan{one, two}
	f.currencies[priya.ID] = Currency{Code: "EUR", Rate: rate(t, "0.92")}
	s := newService(t, f)

	got, err := s.Summary(t.Context(), priya, "myTeams")

	if err != nil || got.DisplayCurrency != "EUR" || got.PlannedCost.PlannedCost.Format() != "1,840.09 EUR" ||
		got.PlannedCost.PlanBudget.Format() != "1,840.09 EUR" || *got.PlannedCost.BudgetUsedPercent != 100 {
		t.Errorf("in euros: %s, planned %v, budget %v, %v", got.DisplayCurrency, got.PlannedCost.PlannedCost,
			got.PlannedCost.PlanBudget, err)
	}
}

// User story 5 — Get started as a new user.

// Getting started shows for a person in no team (notInTeam) and for one whose teams have no plan
// (noTeamPlans), in both scopes; once one of their teams has a plan it is gone (FR-020, D-170, US5).
func TestSummary_GettingStarted_FR020(t *testing.T) {
	f := newFakeSources()
	f.plans = []Plan{newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope)}
	s := newService(t, f)
	reason := func(a actor.Actor, scope string) string {
		t.Helper()
		got, err := s.Summary(t.Context(), a, scope)
		if err != nil {
			t.Fatalf("Summary: %v", err)
		}
		if got.GettingStarted == nil {
			return "none"
		}
		return *got.GettingStarted
	}

	for _, scope := range []string{"myTeams", "everything"} {
		if got := reason(nina, scope); got != "notInTeam" {
			t.Errorf("Nina (no team), %s: %s, want notInTeam", scope, got)
		}
		if got := reason(mei, scope); got != "noTeamPlans" {
			t.Errorf("Mei (team without a plan), %s: %s, want noTeamPlans", scope, got)
		}
		if got := reason(priya, scope); got != "none" {
			t.Errorf("Priya (team plan), %s: %s, want none", scope, got)
		}
	}
	f.plans = append(f.plans, newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina))
	if got := reason(mei, "myTeams"); got != "none" {
		t.Errorf("Mei once her team has a plan: %s, want none", got)
	}
}

// User story 2 — See how my teams' plans are doing.

// Each row shows the plan with its status, teams, campaign and activity counts, approved activities, and its
// budget, planned cost and budget used in the display currency: red above 100 %, "—" without a budget, and
// a plan with no activities shows 0 of 0 (FR-008, FR-009, US2 scenarios 1–3).
func TestPlansAtAGlance_RowFigures_FR008_FR009(t *testing.T) {
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

// A display currency set inactive in Settings stays in use with its rate, as on every other page (D-201,
// D-063, D-217).
func TestSummary_InactiveDisplayCurrencyStaysInUse(t *testing.T) {
	f := newFakeSources()
	p := newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope)
	p.PlannedCost = usdPtr(t, "100.00")
	f.plans = []Plan{p}
	f.currencies[priya.ID] = Currency{Code: "GBP", Rate: rate(t, "0.79")} // GBP is inactive: the source still gives it
	s := newService(t, f)

	got, err := s.Summary(t.Context(), priya, "myTeams")

	if err != nil || got.DisplayCurrency != "GBP" || got.PlannedCost.PlannedCost.Format() != "79.00 GBP" {
		t.Errorf("display currency %s, planned %v, %v; want GBP in use", got.DisplayCurrency, got.PlannedCost.PlannedCost, err)
	}
}

// Waiting for my approval is the Waiting for me tab of the Approvals page, in its order (oldest first): the
// first 5 with "Level n of m", owner, plan, planned cost in the display currency and since when each waits,
// and how many wait in all (FR-013, US3 scenario 5).
func TestWaiting_SameListOldestFirst_FR013(t *testing.T) {
	f := newFakeSources()
	for i := range 7 {
		f.waiting[daniel.ID] = append(f.waiting[daniel.ID], Waiting{
			ActivityID: fmt.Sprintf("A-10%02d", i), ActivityName: fmt.Sprintf("Activity %d", i), Plan: lubricantsRef,
			Owner: &priya, PlannedCost: usdPtr(t, "1000.00"), Level: 1, Levels: 1,
			WaitingSince: now.Add(-time.Duration(48-i) * time.Hour),
		})
	}
	f.waiting[daniel.ID][1].PlannedCost, f.waiting[daniel.ID][1].Owner = nil, nil
	f.currencies[daniel.ID] = Currency{Code: "EUR", Rate: rate(t, "0.92")}
	s := newService(t, f)

	page, err := s.Waiting(t.Context(), daniel)
	nothing, err2 := s.Waiting(t.Context(), priya)

	if err != nil || page.Total != 7 || len(page.Items) != 5 {
		t.Fatalf("Waiting = %d items of %d, %v; want 5 of 7", len(page.Items), page.Total, err)
	}
	want := WaitingItem{ID: "A-1000", Name: "Activity 0", Plan: lubricantsRef, Owner: &priya,
		PlannedCost: &money.Money{Amount: usd(t, "920.00"), Currency: "EUR"}, Level: 1, Levels: 1, WaitingSince: now.Add(-48 * time.Hour)}
	if !reflect.DeepEqual(page.Items[0], want) {
		t.Errorf("first = %+v\nwant %+v", page.Items[0], want)
	}
	if second := page.Items[1]; second.ID != "A-1001" || second.Owner != nil || second.PlannedCost != nil {
		t.Errorf("second = %+v, want no owner and no planned cost", second)
	}
	if err2 != nil || nothing.Total != 0 || nothing.Items == nil || len(nothing.Items) != 0 {
		t.Errorf("Priya = %+v, %v; want an empty list", nothing, err2)
	}
}

// With 50 plans in scope and 5,000 activities, all owned by the person and each needing attention, every
// Home operation answers within 300 ms on the fakes: each is one pass over its inputs (SC-004).
func TestHome_LargeScope_SC004(t *testing.T) {
	if testing.Short() || race.Enabled {
		t.Skip("a timing test: it runs without -short and without the race detector, in CI's Timing tests step")
	}
	f := newFakeSources()
	for i := range 50 {
		p := newPlan(fmt.Sprintf("P-%d", 1001+i), fmt.Sprintf("Plan %02d", i), 2026, lubricantsEurope, mobilityChina)
		p.OwnerID, p.Budget, p.PlannedCost, p.RemainingBudget = sara.ID, usdPtr(t, "1000.00"), usdPtr(t, "1500.00"), usdPtr(t, "-1.00")
		p.ActivityCount, p.ItemsBelow = 100, 100
		f.plans = append(f.plans, p)
		f.containers[sara.ID] = append(f.containers[sara.ID], Container{Type: "campaign", ID: fmt.Sprintf("C-%d", 1001+i),
			Name: "Campaign", Plan: lubricantsRef, Path: []errs.RecordRef{lubricantsRef}, Budget: usdPtr(t, "1.00"),
			PlannedCost: usdPtr(t, "2.00"), UpdatedAt: now})
	}
	statuses := []string{"created", "submitted", "declined", "approved"}
	for i := range 5000 {
		act := Activity{ID: fmt.Sprintf("A-%d", 10001+i), Name: fmt.Sprintf("Activity %04d", i%997), Plan: lubricantsRef,
			Status: statuses[i%4], DisplayStatus: statuses[i%4], PlannedCost: usdPtr(t, "10.00"), Unassigned: usdPtr(t, "-1.00"),
			Missing: []MissingField{{Field: "ownerId", Label: "Owner"}}, UpdatedAt: now.Add(-time.Duration(i) * time.Minute)}
		if i%3 == 0 {
			act.StartDate = ptr(date.MustParse("2026-03-01").AddDays(i % 300))
		}
		f.activities[sara.ID] = append(f.activities[sara.ID], act)
		if i < 200 {
			f.waiting[sara.ID] = append(f.waiting[sara.ID], Waiting{ActivityID: act.ID, Level: 1, Levels: 1, WaitingSince: now})
		}
	}
	s := newService(t, f)

	operations := map[string]func() error{
		"summary":       func() error { _, err := s.Summary(t.Context(), sara, "myTeams"); return err },
		"plans":         func() error { _, err := s.PlansAtAGlance(t.Context(), sara, "everything"); return err },
		"attention":     func() error { _, err := s.Attention(t.Context(), sara); return err },
		"waiting":       func() error { _, err := s.Waiting(t.Context(), sara); return err },
		"my activities": func() error { _, err := s.MyActivities(t.Context(), sara); return err },
	}
	for name, operation := range operations {
		started := time.Now()
		err := operation()
		if took := time.Since(started); err != nil || took > 300*time.Millisecond {
			t.Errorf("%s took %v, %v; want under 300 ms", name, took, err)
		}
	}
}

// Recent changes asks the activity log for the 10 newest entries of the plans in scope: My teams names the
// person's team plans, Everything every plan (nil); My teams without team plans answers an empty page
// without asking; another scope is refused (FR-015).
func TestRecentChanges_Scope_FR015(t *testing.T) {
	f := newFakeSources()
	f.plans = []Plan{
		newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope),
		newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina),
		newPlan("P-1003", "FY2026 Global brand", 2026, lubricantsEurope, mobilityChina),
	}
	entry := map[string]any{"id": "LOG-1001"}
	f.logAnswer = RecentChangesPage{Items: []any{entry}, Total: 12}
	service := newService(t, f)

	mine, err := service.RecentChanges(t.Context(), priya, "myTeams")
	if err != nil || mine.Total != 12 || len(mine.Items) != 1 {
		t.Fatalf("My teams = %+v, %v; want the answer of the log", mine, err)
	}
	if _, err := service.RecentChanges(t.Context(), sara, "everything"); err != nil {
		t.Fatalf("Everything: %v", err)
	}
	if want := [][]string{{"P-1001", "P-1003"}, nil}; !reflect.DeepEqual(f.logAsked, want) || f.logLimit != 10 {
		t.Errorf("asked the log for %v (limit %d), want %v (limit 10)", f.logAsked, f.logLimit, want)
	}

	none, err := service.RecentChanges(t.Context(), nina, "myTeams") // in no team
	if err != nil || none.Total != 0 || none.Items == nil || len(none.Items) != 0 {
		t.Errorf("My teams without team plans = %+v, %v; want an empty page", none, err)
	}
	if len(f.logAsked) != 2 {
		t.Errorf("the log was asked %d times, want 2 (not for a person without team plans)", len(f.logAsked))
	}

	_, err = service.RecentChanges(t.Context(), priya, "all")
	assertError(t, err, scopeRefused)
}
