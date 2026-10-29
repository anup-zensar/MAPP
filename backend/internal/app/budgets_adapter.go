package app

import (
	"context"
	"slices"

	"mapp/internal/budget"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
	"mapp/internal/stores"
)

// budgetPlans is budget.Plans over the plans service and repository; planAllocations tells plans which
// plans are allocated (014).
type budgetPlans struct {
	plans *plan.Service
	repo  stores.PlanStore
	rates func(context.Context) (money.Rates, error) // the current rates (016), for planned costs in USD
}

func (b budgetPlans) Plan(ctx context.Context, id string) (budget.PlanInfo, bool, error) {
	p, found, err := b.repo.Get(ctx, id)
	if err != nil || !found {
		return budget.PlanInfo{}, false, err
	}
	return planInfoOf(p), true, nil
}

func (b budgetPlans) PlansOfYear(ctx context.Context, year int) ([]budget.PlanInfo, error) {
	ofYear, err := b.repo.PlansOfYear(ctx, year)
	if err != nil {
		return nil, err
	}
	infos := make([]budget.PlanInfo, 0, len(ofYear))
	for _, p := range ofYear {
		infos = append(infos, planInfoOf(p))
	}
	return infos, nil
}

// PlannedCosts converts each plan's planned cost to USD at the current rate; every plan is read once first,
// so the levels below find it in the transaction.
func (b budgetPlans) PlannedCosts(ctx context.Context, planIDs []string) (map[string]*money.Amount, error) {
	all, err := b.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	costs, err := b.plans.PlannedCosts(ctx, planIDs)
	if err != nil || b.rates == nil {
		return costs, err
	}
	currencyOf := make(map[string]string, len(all))
	for _, p := range all {
		currencyOf[p.ID] = p.Normalised().Currency
	}
	var rates money.Rates
	for id, cost := range costs {
		currency, found := currencyOf[id]
		if cost == nil || !found || currency == money.BaseCurrency {
			continue
		}
		if rates == nil {
			if rates, err = b.rates(ctx); err != nil {
				return nil, err
			}
		}
		usd, err := rates.ToUSD(money.Money{Amount: *cost, Currency: currency})
		if err != nil {
			return nil, err
		}
		costs[id] = &usd
	}
	return costs, nil
}

func (b budgetPlans) SetBudgetFromAllocation(ctx context.Context, a actor.Actor, in budget.PlanBudget) error {
	return b.plans.SetBudgetFromAllocation(ctx, a, plan.BudgetFromAllocation{PlanID: in.PlanID, Amount: in.Amount,
		Overspend: in.Overspend, Version: in.PlanVersion, Comment: in.Comment})
}

func (b budgetPlans) AllocationRemoved(ctx context.Context, a actor.Actor, planID, comment string) error {
	return b.plans.AllocationRemoved(ctx, a, planID, comment)
}

func planInfoOf(p plan.Plan) budget.PlanInfo {
	return budget.PlanInfo{
		ID: p.ID, Name: p.Name, Year: p.Year, Status: string(p.Status), Version: p.Version,
		TeamIDs: slices.Clone(p.TeamIDs), OwnerID: p.OwnerID, CreatedByID: p.CreatedBy.ID,
		Currency: p.Normalised().Currency, Budget: p.Budget, Overspend: p.Overspend.Normalised(),
	}
}

type planAllocations struct {
	budgets *budget.Service
}

func (p planAllocations) AllocationOf(ctx context.Context, planID string) (plan.Allocation, bool, error) {
	found, allocated, err := p.budgets.AllocationOf(ctx, planID)
	if err != nil || !allocated {
		return plan.Allocation{}, false, err
	}
	return plan.Allocation{
		ID: found.Allocation.ID, GlobalBudgetID: found.GlobalBudget.ID, GlobalBudgetName: found.GlobalBudget.Name,
		OwnerID: found.GlobalBudget.OwnerID, Amount: found.Allocation.Amount, CanOpen: found.CanOpen,
	}, true, nil
}
