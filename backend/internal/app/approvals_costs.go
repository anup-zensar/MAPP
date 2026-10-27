package app

import (
	"context"

	"mapp/internal/activity"
	"mapp/internal/platform/money"
)

// unassigned is the activity's planned cost (0.00 when empty) minus its line items' planned costs in the plan
// currency at the current rates (domain model §4.3, 016 FR-015), converted to USD once — approvals works in
// USD and shows the reader's display currency (D-213); nil when it has neither a planned cost nor a line item.
func (p *approvalsActivities) unassigned(ctx context.Context, act activity.Activity) (*money.Money, error) {
	items, err := p.lineItems.ListByActivity(ctx, act.ID)
	if err != nil {
		return nil, err
	}
	if act.PlannedCost == nil && len(items) == 0 {
		return nil, nil
	}
	rates, err := p.currentRates(ctx)
	if err != nil {
		return nil, err
	}
	currency, err := p.keptPlanCurrency(ctx, act.InitiativeID)
	if err != nil {
		return nil, err
	}
	var balance money.Amount
	if act.PlannedCost != nil {
		balance = act.PlannedCost.Amount
	}
	for _, li := range items {
		inPlan, err := rates.Convert(money.Money{Amount: li.PlannedCost, Currency: li.Currency}, currency)
		if err != nil {
			return nil, err
		}
		balance = balance.Sub(inPlan.Amount)
	}
	return p.inUSD(ctx, &money.Money{Amount: balance, Currency: currency})
}

// inUSD converts an amount of a plan currency to USD at the current rate (016 FR-014); nil stays nil.
func (p *approvalsActivities) inUSD(ctx context.Context, m *money.Money) (*money.Money, error) {
	if m == nil || m.Currency == money.BaseCurrency {
		return m, nil
	}
	rates, err := p.currentRates(ctx)
	if err != nil {
		return nil, err
	}
	usd, err := rates.Convert(*m, money.BaseCurrency)
	if err != nil {
		return nil, err
	}
	return &usd, nil
}

// planCurrency returns the plan currency of the initiative initiativeID's plan (016 FR-011).
func (p *approvalsActivities) planCurrency(ctx context.Context, initiativeID string) (string, error) {
	initiative, found, err := p.campaigns.GetInitiative(ctx, initiativeID)
	if err != nil || !found {
		return money.BaseCurrency, err
	}
	campaign, found, err := p.campaigns.GetCampaign(ctx, initiative.CampaignID)
	if err != nil || !found {
		return money.BaseCurrency, err
	}
	plan, found, err := p.plans.Get(ctx, campaign.PlanID)
	if err != nil || !found {
		return money.BaseCurrency, err
	}
	return plan.Currency, nil
}
