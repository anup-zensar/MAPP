package app

import (
	"context"

	"mapp/internal/activity"
	"mapp/internal/platform/money"
)

// unassigned is unassignedInPlan converted to USD once: approvals works in USD and shows the display currency.
func (p *approvalsActivities) unassigned(ctx context.Context, act activity.Activity) (*money.Money, error) {
	inPlan, err := p.unassignedInPlan(ctx, act)
	if err != nil {
		return nil, err
	}
	return p.inUSD(ctx, inPlan)
}

func (p *approvalsActivities) unassignedInPlan(ctx context.Context, act activity.Activity) (*money.Money, error) {
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
	return &money.Money{Amount: balance, Currency: currency}, nil
}

// overPlannedCost is a negative unassigned planned cost made positive; nil when the line items fit.
func overPlannedCost(unassigned *money.Money) *money.Money {
	if unassigned == nil || unassigned.Amount.Sign() >= 0 {
		return nil
	}
	over := money.Money{Amount: unassigned.Amount.Neg(), Currency: unassigned.Currency}
	return &over
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
