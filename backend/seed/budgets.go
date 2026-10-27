package seed

import (
	"context"
	"fmt"

	"mapp/internal/budget"
	"mapp/internal/platform/money"
)

// The demo global budget of 014 plan "Demo data" (FR-040): GB-1001, owned by Carlos Mendes, the global
// planner, allocated to the two plans of 2026 with exactly their seeded budgets, so the plans do not change.
const (
	demoGlobalBudget            = "FY2026 Global marketing budget"
	demoGlobalBudgetYear        = 2026
	demoGlobalBudgetAmount      = "2500000.00"
	demoGlobalBudgetOwner       = "Carlos Mendes"
	demoGlobalBudgetDescription = "Shared out to market plans by Global Marketing Ops."
)

// demoAllocations are BA-1001 and BA-1002: plan name → amount in USD, in this order.
var demoAllocations = []struct{ plan, amount string }{
	{"FY2026 Lubricants Europe", "1200000.00"},
	{"FY2026 Mobility China", "800000.00"},
}

// addGlobalBudgets creates the demo global budget and its allocations through the budgets repository, after
// the plans (D-306). Each allocation equals its plan's seeded budget.
func (d *demo) addGlobalBudgets(ctx context.Context) error {
	repo := d.stores.Budgets
	var r resolver
	ownerID := r.id(d.people, "person", demoGlobalBudgetOwner)
	if r.err != nil {
		return r.err
	}
	amount, err := money.ParseAmount(demoGlobalBudgetAmount)
	if err != nil {
		return fmt.Errorf("seed: amount of %s: %w", demoGlobalBudget, err)
	}
	b := budget.GlobalBudget{
		Name: demoGlobalBudget, Year: demoGlobalBudgetYear, Amount: amount, OwnerID: ownerID,
		Description: demoGlobalBudgetDescription, Meta: d.meta(),
	}
	if b.ID, err = d.next(ctx, budget.PrefixGlobalBudget); err != nil {
		return err
	}
	if err := repo.PutBudget(ctx, b); err != nil {
		return err
	}
	for _, a := range demoAllocations {
		planID := r.id(d.plans, "plan", a.plan)
		if r.err != nil {
			return r.err
		}
		amount, err := usd(a.amount)
		if err != nil {
			return err
		}
		allocation := budget.Allocation{GlobalBudgetID: b.ID, PlanID: planID, Amount: amount, Meta: d.meta()}
		if allocation.ID, err = d.next(ctx, budget.PrefixAllocation); err != nil {
			return err
		}
		if err := repo.PutAllocation(ctx, allocation); err != nil {
			return err
		}
	}
	return nil
}
