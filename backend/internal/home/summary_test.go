package home

import (
	"fmt"
	"reflect"
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

func TestSummary_OwnedActivitiesCoverAllPlans_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	f.plans = []Plan{newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope),
		newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina)}
	outside := errs.RecordRef{Type: "plan", ID: "P-1002", Name: "FY2026 Mobility China"}
	f.activities[priya.ID] = []Activity{{ID: "A-1008", Plan: outside, Status: "submitted"}}
	s := newService(t, f)

	for _, scope := range []string{"myTeams", "everything"} {
		got, err := s.Summary(t.Context(), priya, scope)

		if c := countsText(got.ActivitiesOwned); err != nil || c != "1 (0 declined, 1 submitted) in 2026–2026" {
			t.Errorf("%s: activities owned = %s, %v; want the activity outside her teams", scope, c, err)
		}
	}
}

func countsText(c ActivityCounts) string {
	years := "no years"
	if c.YearFrom != nil && c.YearTo != nil {
		years = fmt.Sprintf("%d–%d", *c.YearFrom, *c.YearTo)
	}
	return fmt.Sprintf("%d (%d declined, %d submitted) in %s", c.Count, c.Declined, c.Submitted, years)
}

// The card covers at most five plan years from the earliest, as All work on those years lists them (D-363).
func TestSummary_OwnedActivitiesYears_038_FR032_M5(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	f.plans = []Plan{newPlan("P-1", "FY2026", 2026, lubricantsEurope), newPlan("P-2", "FY2027", 2027, lubricantsEurope),
		newPlan("P-3", "FY2031", 2031, lubricantsEurope)}
	in := func(id string) errs.RecordRef { return errs.RecordRef{Type: "plan", ID: id} }
	f.activities[priya.ID] = []Activity{{ID: "A-1", Plan: in("P-2"), Status: "declined"},
		{ID: "A-2", Plan: in("P-1"), Status: "created"}, {ID: "A-3", Plan: in("P-3"), Status: "submitted"}}
	s := newService(t, f)

	got, err := s.Summary(t.Context(), priya, "myTeams")
	none, noneErr := s.Summary(t.Context(), daniel, "myTeams")

	if c := countsText(got.ActivitiesOwned); err != nil || c != "2 (1 declined, 0 submitted) in 2026–2027" {
		t.Errorf("activities owned = %s, %v; want 2 in 2026–2027 (2031 is more than four years later)", c, err)
	}
	if c := countsText(none.ActivitiesOwned); noneErr != nil || c != "0 (0 declined, 0 submitted) in no years" {
		t.Errorf("no activities = %s, %v", c, noneErr)
	}
}

// Budget used rounds half up to a whole percent, but over budget compares exactly: 100.4 % shows 100 and is over.
func TestSummary_BudgetUsed_FR006(t *testing.T) {
	t.Parallel()
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

func TestSummary_PlansWithoutBudget_FR006(t *testing.T) {
	t.Parallel()
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

func TestSummary_NoBudgetAtAll_FR006(t *testing.T) {
	t.Parallel()
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

// 1,000.05 + 1,000.05 USD at 0.92 is 1,840.09 EUR, not the 1,840.10 of two converted amounts.
func TestSummary_TotalsConvertedOnce_FR017(t *testing.T) {
	t.Parallel()
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

func TestSummary_GettingStarted_FR020(t *testing.T) {
	t.Parallel()
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

func TestSummary_InactiveDisplayCurrencyStaysInUse(t *testing.T) {
	t.Parallel()
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
