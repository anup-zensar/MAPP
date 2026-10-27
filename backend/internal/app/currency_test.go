package app

import (
	"context"
	"testing"
	"time"

	"mapp/internal/activities"
	"mapp/internal/allwork"
	"mapp/internal/approvals"
	"mapp/internal/campaigns"
	"mapp/internal/history"
	"mapp/internal/lineitems"
	"mapp/internal/plans"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
	"mapp/internal/stores"
	"mapp/seed"
)

// P-1004 is planned in GBP at 0.79 per USD: 600,000.00 GBP = 759,493.67 USD; A-1011 40,000.00 GBP = 50,632.91
// USD; its line item L-1010 23,000.00 EUR = 25,000.00 USD = 19,750.00 GBP.
func TestAllWorkSource_ConvertsPlanCurrency_FR014(t *testing.T) {
	t.Parallel()
	built, db := newSeededApp(t)
	var data allwork.YearData
	err := db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		data, err = built.allWork.Year(ctx, 2027, date.MustParse("2026-03-12"), everyColumn)
		return err
	})
	if err != nil {
		t.Fatalf("Year: %v", err)
	}
	byID := map[string]allwork.Node{}
	for _, n := range data.Nodes {
		byID[n.ID] = n
	}

	plan, launch, agency := byID["P-1004"], byID["A-1011"], byID["L-1010"]
	if plan.Budget == nil || plan.Budget.String() != "759493.67" || plan.BudgetPlan == nil || plan.BudgetPlan.Format() != "600,000.00 GBP" ||
		plan.PlanCurrency != "GBP" || plan.PlanType != "crossMarket" || plan.PlanYear != 2027 || plan.PhasingMethod != "evenly" {
		t.Errorf("P-1004 = budget %v (%v), %s %s %d %s", plan.Budget, plan.BudgetPlan, plan.PlanCurrency, plan.PlanType, plan.PlanYear, plan.PhasingMethod)
	}
	if len(plan.PhasedBudgetUSD) != 12 || plan.PhasedBudgetUSD[0].Month != "2027-01" || plan.PhasedBudgetUSD[0].USD.String() != "63291.14" ||
		plan.PhasedBudgetUSD[11].USD.String() != "63291.13" {
		t.Errorf("phased budget in USD = %+v, want 63,291.14 a month and the rest in December", plan.PhasedBudgetUSD)
	}
	var phasedSum money.Amount
	for _, m := range plan.PhasedBudgetUSD {
		phasedSum = phasedSum.Add(m.USD)
	}
	if phasedSum.String() != "759493.67" {
		t.Errorf("the phased months add up to %s USD, want the budget 759,493.67", phasedSum)
	}
	if launch.PlannedCost == nil || launch.PlannedCost.String() != "50632.91" || launch.PlannedCostPlan == nil ||
		launch.PlannedCostPlan.Format() != "40,000.00 GBP" || launch.PlanCurrency != "GBP" || launch.PlanYear != 2027 {
		t.Errorf("A-1011 = %v (%v), %s %d", launch.PlannedCost, launch.PlannedCostPlan, launch.PlanCurrency, launch.PlanYear)
	}
	if agency.PlannedCost == nil || agency.PlannedCost.String() != "25000.00" || agency.PlannedCostPlan == nil ||
		agency.PlannedCostPlan.Format() != "19,750.00 GBP" || agency.LocalPlannedCost.Format() != "23,000.00 EUR" {
		t.Errorf("L-1010 = %v USD, %v, %v", agency.PlannedCost, agency.PlannedCostPlan, agency.LocalPlannedCost)
	}
	campaign := byID[launchCampaignID(data)]
	if campaign.Budget == nil || campaign.Budget.String() != "506329.11" || campaign.BudgetPlan.Format() != "400,000.00 GBP" ||
		campaign.PhasingMethod != "custom" || len(campaign.PhasedBudgetUSD) != 12 || !campaign.PhasedBudgetUSD[0].USD.IsZero() {
		t.Errorf("Helix UK & Ireland = %v (%v), %s, %+v", campaign.Budget, campaign.BudgetPlan, campaign.PhasingMethod, campaign.PhasedBudgetUSD)
	}
}

func launchCampaignID(data allwork.YearData) string {
	for _, n := range data.Nodes {
		if n.Level == allwork.LevelCampaign && n.Name == "Helix UK & Ireland" {
			return n.ID
		}
	}
	return ""
}

// A-1011's unassigned 40,000.00 − (15,000.00 + 19,750.00) = 5,250.00 GBP = 6,645.57 USD (D-213).
func TestApprovals_DisplayFromPlanCurrency(t *testing.T) {
	t.Parallel()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	now := func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }
	if err = seed.Load(t.Context(), stores.JSON(db), now); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	settingsService := settings.NewService(history.Tools{
		Tx: db, IDs: generator, Log: history.NewService(db, history.NewJSONStore(), generator, now), Now: now,
	}, settings.NewJSONStore())
	adapter := &approvalsActivities{
		activities: activities.NewJSONStore(), lineItems: lineitems.NewJSONStore(), campaigns: campaigns.NewJSONStore(),
		plans: plans.NewJSONStore(), settings: settings.NewJSONStore(), rates: settingsService.Rates, now: now,
	}

	var launch, usdActivity approvals.ActivityFacts
	err = db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		if launch, _, err = adapter.Activity(ctx, "A-1011"); err != nil {
			return err
		}
		usdActivity, _, err = adapter.Activity(ctx, "A-1001")
		return err
	})
	if err != nil {
		t.Fatalf("Activity: %v", err)
	}

	if launch.PlannedCost == nil || launch.PlannedCost.Format() != "50,632.91 USD" || launch.UnassignedPlannedCost == nil ||
		launch.UnassignedPlannedCost.Format() != "6,645.57 USD" {
		t.Errorf("A-1011 = planned %v, unassigned %v; want 50,632.91 USD and 6,645.57 USD", launch.PlannedCost, launch.UnassignedPlannedCost)
	}
	if usdActivity.PlannedCost == nil || usdActivity.PlannedCost.Currency != "USD" {
		t.Errorf("A-1001 of a USD plan = %v, want USD unchanged", usdActivity.PlannedCost)
	}
}
