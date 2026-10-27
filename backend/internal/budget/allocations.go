package budget

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// AllocationOptions returns the plans of the global budget's year that no global budget allocates to and that
// the person may allocate to (globalBudgets.manage in the plan's scope), by name: the Allocate dialog (FR-030).
func (s *Service) AllocationOptions(ctx context.Context, a actor.Actor, id string) (AllocationOptionPage, error) {
	page := AllocationOptionPage{Items: []AllocationOption{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		b, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		unallocated, err := s.unallocatedPlans(ctx, b.Year)
		if err != nil {
			return err
		}
		l, err := s.lookupsFor(ctx, unallocated)
		if err != nil {
			return err
		}
		for _, p := range unallocated {
			option, err := s.option(ctx, p, l)
			if err != nil {
				return err
			}
			if option.CanAllocate.Available {
				page.Items = append(page.Items, option)
			}
		}
		page.Total = len(page.Items)
		return nil
	})
	if err != nil {
		return AllocationOptionPage{}, err
	}
	return page, nil
}

// Allocate gives part of the global budget id to a plan of its year (FR-030): the plan's budget becomes the
// amount in the same Write. It needs globalBudgets.manage in the plan's scope; a plan has at most one
// allocation across all global budgets (message 29); the amount is 0 or more in the plan's currency (D-303);
// planVersion is the plan's version the person saw (409 version_conflict when the plan changed since). Every
// field problem is reported at once. The change is logged on the global budget ("Allocation to '<plan>'") and
// on the plan (data model §3). Over-allocating the global budget never blocks (D-015).
func (s *Service) Allocate(ctx context.Context, a actor.Actor, id string, in AllocationInput) (GlobalBudgetView, error) {
	var view GlobalBudgetView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		b, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		plan, found, err := s.allocatablePlan(ctx, b, strings.TrimSpace(in.PlanID), &problems)
		if err != nil {
			return err
		}
		currency := "USD"
		if found {
			currency = plan.Currency
		}
		amount, _ := checkAmount(in.Amount, currency, fmt.Sprintf(msgAmountInPlanCurrency, currency), "amount", &problems)
		checkVersion(in.PlanVersion, "planVersion", &problems)
		if err := problems.Err(); err != nil {
			return err
		}

		allocation := Allocation{
			GlobalBudgetID: b.ID, PlanID: plan.ID, Amount: money.Money{Amount: amount, Currency: currency},
			Meta: record.NewMeta(a, s.now()),
		}
		if allocation.ID, err = s.ids.Next(ctx, PrefixAllocation); err != nil {
			return err
		}
		if err := s.repo.PutAllocation(ctx, allocation); err != nil {
			return err
		}
		if err := s.logAllocation(ctx, a, b, plan.ID, nil, &allocation.Amount); err != nil {
			return err
		}
		if err := s.plans.SetBudgetFromAllocation(ctx, a, plan.ID, amount, *in.PlanVersion,
			fmt.Sprintf(commentAllocated, b.Name)); err != nil {
			return err
		}
		view, err = s.view(ctx, b)
		return err
	})
	if err != nil {
		return GlobalBudgetView{}, err
	}
	return view, nil
}

// allocatablePlan checks the plan planID chosen for an allocation from b: chosen, a plan of b's year
// (message 28) and without an allocation (message 29). A person who may not manage global budgets in the
// plan's scope is refused at once (403). It returns the plan, and false when it is not a plan of the year.
func (s *Service) allocatablePlan(ctx context.Context, b GlobalBudget, planID string, problems *errs.Problems) (PlanInfo, bool, error) {
	if planID == "" {
		problems.Add("planId", msgChoosePlan)
		return PlanInfo{}, false, nil
	}
	plan, found, err := s.plans.Plan(ctx, planID)
	if err != nil {
		return PlanInfo{}, false, err
	}
	if !found || plan.Year != b.Year {
		problems.Add("planId", fmt.Sprintf(msgPlanOfYear, b.Year))
		return PlanInfo{}, false, nil
	}
	if err := s.access.ManageIn(ctx, plan); err != nil {
		return PlanInfo{}, false, err
	}
	existing, taken, err := s.repo.AllocationOfPlan(ctx, plan.ID)
	if err != nil {
		return PlanInfo{}, false, err
	}
	if taken {
		other, _, err := s.repo.GetBudget(ctx, existing.GlobalBudgetID)
		if err != nil {
			return PlanInfo{}, false, err
		}
		problems.Add("planId", fmt.Sprintf(msgPlanAllocated, plan.Name, other.Name))
	}
	return plan, true, nil
}

// UpdateAllocation changes the amount of the allocation id from the versions the person saw: the
// allocation's (version) and its plan's (planVersion), each refused with 409 version_conflict when out of
// date (FR-030). The plan's budget follows in the same Write. It needs globalBudgets.manage in the plan's
// scope. An unchanged amount changes nothing and logs nothing.
func (s *Service) UpdateAllocation(ctx context.Context, a actor.Actor, id string, update AllocationUpdate) (GlobalBudgetView, error) {
	var view GlobalBudgetView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, b, plan, err := s.storedAllocation(ctx, id)
		if err != nil {
			return err
		}
		if err := s.access.ManageIn(ctx, plan); err != nil {
			return err
		}
		var problems errs.Problems
		currency := stored.Amount.Currency
		amount, _ := checkAmount(update.Amount, currency, fmt.Sprintf(msgAmountInPlanCurrency, currency), "amount", &problems)
		checkVersion(update.Version, "version", &problems)
		checkVersion(update.PlanVersion, "planVersion", &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		if err := stored.CheckVersion(*update.Version, kindAllocationSingular); err != nil {
			return err
		}
		if amount.Cmp(stored.Amount.Amount) != 0 {
			changed := stored
			changed.Amount.Amount = amount
			changed.Touch(a, s.now())
			if err := s.repo.PutAllocation(ctx, changed); err != nil {
				return err
			}
			if err := s.logAllocation(ctx, a, b, plan.ID, &stored.Amount, &changed.Amount); err != nil {
				return err
			}
			if err := s.plans.SetBudgetFromAllocation(ctx, a, plan.ID, amount, *update.PlanVersion,
				fmt.Sprintf(commentAllocated, b.Name)); err != nil {
				return err
			}
		}
		view, err = s.view(ctx, b)
		return err
	})
	if err != nil {
		return GlobalBudgetView{}, err
	}
	return view, nil
}

// RemoveAllocation removes the allocation id (FR-034): its plan keeps its budget, which becomes editable
// again. The row is deleted outright, not moved to the recycle bin: the log keeps its amount on the global
// budget, and the plan's log says the allocation was removed. It needs globalBudgets.manage in the plan's
// scope.
func (s *Service) RemoveAllocation(ctx context.Context, a actor.Actor, id string) (GlobalBudgetView, error) {
	var view GlobalBudgetView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, b, plan, err := s.storedAllocation(ctx, id)
		if err != nil {
			return err
		}
		if err := s.access.ManageIn(ctx, plan); err != nil {
			return err
		}
		if err := s.repo.DeleteAllocation(ctx, stored.ID); err != nil {
			return err
		}
		if err := s.logAllocation(ctx, a, b, plan.ID, &stored.Amount, nil); err != nil {
			return err
		}
		if err := s.plans.AllocationRemoved(ctx, a, plan.ID, fmt.Sprintf(commentRemoved, b.Name)); err != nil {
			return err
		}
		view, err = s.view(ctx, b)
		return err
	})
	if err != nil {
		return GlobalBudgetView{}, err
	}
	return view, nil
}

// storedAllocation reads the allocation id with its global budget and its plan inside the caller's
// transaction; an allocation that does not exist, or whose plan the person cannot see, is not_found.
func (s *Service) storedAllocation(ctx context.Context, id string) (Allocation, GlobalBudget, PlanInfo, error) {
	allocation, found, err := s.repo.GetAllocation(ctx, id)
	if err != nil {
		return Allocation{}, GlobalBudget{}, PlanInfo{}, err
	}
	if !found {
		return Allocation{}, GlobalBudget{}, PlanInfo{}, errs.NotFound(msgAllocationGone)
	}
	b, err := s.stored(ctx, allocation.GlobalBudgetID)
	if err != nil {
		return Allocation{}, GlobalBudget{}, PlanInfo{}, err
	}
	plan, found, err := s.plans.Plan(ctx, allocation.PlanID)
	if err != nil {
		return Allocation{}, GlobalBudget{}, PlanInfo{}, err
	}
	// A deleted plan takes its allocation to the recycle bin; a plan the person cannot see hides it.
	if !found || !s.access.CanViewPlan(ctx, plan) {
		return Allocation{}, GlobalBudget{}, PlanInfo{}, errs.NotFound(msgAllocationGone)
	}
	return allocation, b, plan, nil
}

// logAllocation logs on the global budget b that its allocation to the plan planID was added, changed or
// removed (data model §3): field "allocations.<planID>", labelled "Allocation to '<plan>'", from before to after
// (before or after nil: not allocated).
func (s *Service) logAllocation(ctx context.Context, a actor.Actor, b GlobalBudget, planID string, before, after *money.Money) error {
	oldValue, err := json.Marshal(before)
	if err != nil {
		return err
	}
	newValue, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{
		Kind: kindBudget, RecordID: b.ID, RecordName: b.Name, Action: history.ActionChanged,
		Changes: []history.Change{{Field: allocationField + planID, Old: oldValue, New: newValue}},
	})
	return err
}

// PlanAllocation is the allocation that sets one plan's budget, with its global budget, for package plans
// (through internal/app): the plan page's note and the budget lock (FR-031, FR-033).
type PlanAllocation struct {
	Allocation   Allocation
	GlobalBudget GlobalBudget
	CanOpen      bool // the person may open the global budget (globalBudgets.view)
}

// AllocationOf returns the allocation of the plan planID, and false when it has none. It joins the caller's
// transaction.
func (s *Service) AllocationOf(ctx context.Context, planID string) (PlanAllocation, bool, error) {
	var found PlanAllocation
	var allocated bool
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		allocation, ok, err := s.repo.AllocationOfPlan(ctx, planID)
		if err != nil || !ok {
			return err
		}
		b, ok, err := s.repo.GetBudget(ctx, allocation.GlobalBudgetID)
		if err != nil || !ok {
			return err
		}
		found = PlanAllocation{Allocation: allocation, GlobalBudget: b, CanOpen: s.access.SeeGlobalBudgets(ctx)}
		allocated = true
		return nil
	})
	return found, allocated, err
}
