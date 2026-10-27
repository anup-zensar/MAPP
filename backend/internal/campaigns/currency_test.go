package campaigns

import (
	"testing"

	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// gbp returns the GBP amount text as money.
func gbp(text string) *money.Money {
	amount, err := money.ParseAmount(text)
	if err != nil {
		panic(err)
	}
	return &money.Money{Amount: amount, Currency: "GBP"}
}

// withCurrencyPlans adds the 2027 plans of the 016 examples: P-1004 planned in GBP (600,000.00 GBP) and
// P-1005 planned in EUR.
func withCurrencyPlans(f *fixture) {
	f.plans.plans["P-1004"] = PlanFacts{ID: "P-1004", Name: "FY2027 Lubricants UK & Ireland", Year: 2027, Status: "draft",
		OwnerID: ptr("U-1002"), Budget: gbp("600000.00"), Currency: "GBP", ModelID: "MOD-1001", Attributes: settings.AttributeValues{}}
	f.plans.plans["P-1005"] = PlanFacts{ID: "P-1005", Name: "FY2027 Lubricants Germany", Year: 2027, Status: "draft",
		Currency: "EUR", ModelID: "MOD-1001", Attributes: settings.AttributeValues{}}
}

// Campaign and initiative budgets are typed in the plan currency; another currency is refused (FR-011).
func TestCampaignBudget_MustBePlanCurrency_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	withCurrencyPlans(f)

	_, usdErr := f.service.CreateCampaign(t.Context(), priya, "P-1004", CampaignInput{Name: "Helix", Budget: input("400000")})
	campaign := f.createCampaign(t, priya, "P-1004", CampaignInput{Name: "Helix UK & Ireland", Budget: &money.Input{Amount: "400000", Currency: "GBP"}})
	_, initiativeErr := f.service.CreateInitiative(t.Context(), priya, campaign.ID, InitiativeInput{Name: "UK", Budget: &money.Input{Amount: "1", Currency: "EUR"}})
	initiative := f.createInitiative(t, priya, campaign.ID, InitiativeInput{Name: "UK", Budget: &money.Input{Amount: "250000", Currency: "GBP"}})
	edit := CampaignUpdate{Name: campaign.Name, Budget: input("1"), Version: ptr(1)}
	_, updateErr := f.service.UpdateCampaign(t.Context(), priya, campaign.ID, edit)

	assertError(t, usdErr, problems("budget", "Enter the amount in GBP, the plan's currency."))
	assertError(t, initiativeErr, problems("budget", "Enter the amount in GBP, the plan's currency."))
	assertError(t, updateErr, problems("budget", "Enter the amount in GBP, the plan's currency."))
	if campaign.Budget.Format() != "400,000.00 GBP" || initiative.Budget.Format() != "250,000.00 GBP" {
		t.Errorf("budgets = %v, %v", campaign.Budget, initiative.Budget)
	}
}

// Inside a plan the amounts are in the plan currency and, in display, in the reader's display currency, each
// converted once from the plan currency (FR-013, D-261).
func TestContainerDisplay_FromPlanCurrency_FR013(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	withCurrencyPlans(f)
	campaign := f.createCampaign(t, priya, "P-1004", CampaignInput{Name: "Helix UK & Ireland", Budget: &money.Input{Amount: "400000", Currency: "GBP"}})
	initiative := f.createInitiative(t, priya, campaign.ID, InitiativeInput{Name: "UK", Budget: &money.Input{Amount: "250000", Currency: "GBP"}})
	f.activities.add(ActivitySummary{ID: "A-1012", InitiativeID: initiative.ID, Name: "UK launch event", PlannedCost: gbp("40000.00")})

	tree := f.list(t, "P-1004")

	c := tree.Items[0]
	if c.BudgetAndCost.AllocatedToChildren.Format() != "250,000.00 GBP" || c.BudgetAndCost.PlannedCost.Format() != "40,000.00 GBP" ||
		c.ParentRemainingBudget.Format() != "200,000.00 GBP" || c.BudgetAndCost.LeftToPlan.Format() != "360,000.00 GBP" {
		t.Errorf("plan-currency amounts = %+v, parent remaining %v", c.BudgetAndCost, c.ParentRemainingBudget)
	}
	if c.Display.Budget.Format() != "506,329.11 USD" || c.Display.BudgetAndCost.PlannedCost.Format() != "50,632.91 USD" ||
		c.Display.ParentRemainingBudget.Format() != "253,164.56 USD" {
		t.Errorf("display = %+v", c.Display)
	}
	if tree.Totals.AllocatedToCampaigns.Format() != "400,000.00 GBP" || tree.DisplayTotals.AllocatedToCampaigns.Format() != "506,329.11 USD" ||
		tree.DisplayTotals.PlanBudget.Format() != "759,493.67 USD" {
		t.Errorf("totals = %+v, display %+v", tree.Totals, tree.DisplayTotals)
	}
	empty := f.createCampaign(t, priya, "P-1004", CampaignInput{Name: "Empty"})
	if empty.BudgetAndCost.AllocatedToChildren.Format() != "0.00 GBP" {
		t.Errorf("nothing allocated = %v, want 0.00 GBP", empty.BudgetAndCost.AllocatedToChildren)
	}
}

// A plan with another plan currency is listed but not selectable for a move or a copy, and the move and the
// copy refuse it (FR-017, US2-6).
func TestDestinations_OtherCurrencyUnavailable_FR017(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	withCurrencyPlans(f)
	campaign := f.createCampaign(t, priya, "P-1004", CampaignInput{Name: "Helix UK & Ireland", Budget: &money.Input{Amount: "400000", Currency: "GBP"}})

	moves, err := f.service.CampaignDestinations(t.Context(), priya, campaign.ID, PurposeMove)
	copies, copyErr := f.service.CampaignDestinations(t.Context(), priya, campaign.ID, PurposeCopy)
	_, moveErr := f.service.MoveCampaign(t.Context(), priya, campaign.ID, CampaignMove{PlanID: "P-1005", Version: ptr(1)})
	_, copyIntoErr := f.service.CopyCampaign(t.Context(), priya, campaign.ID, CampaignCopy{PlanID: "P-1005", Name: "Helix (copy)"})

	reasons := map[string]string{}
	for _, d := range moves.Items {
		if d.UnavailableReason != nil {
			reasons[d.ID] = *d.UnavailableReason
		}
		if d.RemainingAfter != nil {
			t.Errorf("%s: remaining after %v across currencies", d.ID, d.RemainingAfter)
		}
	}
	if err != nil || len(moves.Items) != 2 || reasons["P-1005"] != "Planned in EUR — this campaign's amounts are in GBP." ||
		reasons["P-1003"] != "Planned in USD — this campaign's amounts are in GBP." {
		t.Errorf("move destinations = %+v, %v", reasons, err)
	}
	if copyErr != nil || len(copies.Items) != 3 || !copies.Items[0].Current || !copies.Items[0].Available || copies.Items[1].Available {
		t.Errorf("copy destinations = %+v, %v", copies.Items, copyErr)
	}
	assertError(t, moveErr, problems("planId", "Planned in EUR — this campaign's amounts are in GBP."))
	assertError(t, copyIntoErr, problems("planId", "Planned in EUR — this campaign's amounts are in GBP."))
}
