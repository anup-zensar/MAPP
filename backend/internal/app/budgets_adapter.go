package app

import (
	"context"
	"slices"

	"northstar/internal/budgets"
	"northstar/internal/plans"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/money"
)

// The adapters between global budgets (014, package budgets) and plans (004): budgets reads plans and sets a
// plan's budget through budgetPlans; plans learns which plans are allocated through planAllocations.

// budgetPlans is budgets.Plans over the plans service and repository.
type budgetPlans struct {
	plans *plans.Service
	repo  plans.Repository
}

// Plan returns the plan id as budgets needs it.
func (b budgetPlans) Plan(ctx context.Context, id string) (budgets.PlanInfo, bool, error) {
	p, found, err := b.repo.Get(ctx, id)
	if err != nil || !found {
		return budgets.PlanInfo{}, false, err
	}
	return planInfoOf(p), true, nil
}

// PlansOfYear returns the plans of year as budgets needs them.
func (b budgetPlans) PlansOfYear(ctx context.Context, year int) ([]budgets.PlanInfo, error) {
	ofYear, err := b.repo.PlansOfYear(ctx, year)
	if err != nil {
		return nil, err
	}
	infos := make([]budgets.PlanInfo, 0, len(ofYear))
	for _, p := range ofYear {
		infos = append(infos, planInfoOf(p))
	}
	return infos, nil
}

// PlannedCosts returns the planned cost in USD of each plan, as the plan page works it out.
func (b budgetPlans) PlannedCosts(ctx context.Context, planIDs []string) (map[string]*money.Amount, error) {
	return b.plans.PlannedCosts(ctx, planIDs)
}

// SetBudgetFromAllocation sets the plan's budget through the plans service, its one owner.
func (b budgetPlans) SetBudgetFromAllocation(ctx context.Context, a actor.Actor, planID string, amount money.Amount,
	version int, comment string) error {
	return b.plans.SetBudgetFromAllocation(ctx, a, planID, amount, version, comment)
}

// AllocationRemoved logs on the plan that its allocation was removed.
func (b budgetPlans) AllocationRemoved(ctx context.Context, a actor.Actor, planID, comment string) error {
	return b.plans.AllocationRemoved(ctx, a, planID, comment)
}

// planInfoOf returns what budgets needs to know of the stored plan p. Plans are planned in US dollars until
// feature 016 adds plan currencies (D-303).
func planInfoOf(p plans.Plan) budgets.PlanInfo {
	return budgets.PlanInfo{
		ID: p.ID, Name: p.Name, Year: p.Year, Status: string(p.Status), Version: p.Version,
		TeamIDs: slices.Clone(p.TeamIDs), OwnerID: p.OwnerID, CreatedByID: p.CreatedBy.ID,
		Currency: money.BaseCurrency, Budget: p.Budget,
	}
}

// planAllocations is plans.Allocations over the budgets service.
type planAllocations struct {
	budgets *budgets.Service
}

// AllocationOf returns the allocation that sets the budget of the plan planID, and false when it has none.
func (p planAllocations) AllocationOf(ctx context.Context, planID string) (plans.Allocation, bool, error) {
	found, allocated, err := p.budgets.AllocationOf(ctx, planID)
	if err != nil || !allocated {
		return plans.Allocation{}, false, err
	}
	return plans.Allocation{
		ID: found.Allocation.ID, GlobalBudgetID: found.GlobalBudget.ID, GlobalBudgetName: found.GlobalBudget.Name,
		OwnerID: found.GlobalBudget.OwnerID, Amount: found.Allocation.Amount, CanOpen: found.CanOpen,
	}, true, nil
}
