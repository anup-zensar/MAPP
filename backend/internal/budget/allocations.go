package budget

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/overspend"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

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
		mode := checkOverspend(in.Overspend, "", &problems)
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
		if err = s.plans.SetBudgetFromAllocation(ctx, a, PlanBudget{PlanID: plan.ID, Amount: amount, Overspend: mode,
			PlanVersion: *in.PlanVersion, Comment: fmt.Sprintf(commentAllocated, b.Name)}); err != nil {
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
		mode := checkOverspend(update.Overspend, plan.Overspend.Normalised(), &problems)
		checkVersion(update.Version, "version", &problems)
		checkVersion(update.PlanVersion, "planVersion", &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		if err := stored.CheckVersion(*update.Version, kindAllocationSingular); err != nil {
			return err
		}
		change := allocationChange{stored: stored, budget: b, plan: plan, amount: amount, mode: mode, planVersion: *update.PlanVersion}
		if err = s.changeAllocation(ctx, a, change); err != nil {
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

type allocationChange struct {
	stored      Allocation
	budget      GlobalBudget
	plan        PlanInfo
	amount      money.Amount
	mode        overspend.Mode
	planVersion int
}

// changeAllocation saves and logs a new amount, and sets the plan's budget when the amount or the setting moved.
func (s *Service) changeAllocation(ctx context.Context, a actor.Actor, c allocationChange) error {
	amountChanged := c.amount.Cmp(c.stored.Amount.Amount) != 0
	if amountChanged {
		changed := c.stored
		changed.Amount.Amount = c.amount
		changed.Touch(a, s.now())
		if err := s.repo.PutAllocation(ctx, changed); err != nil {
			return err
		}
		if err := s.logAllocation(ctx, a, c.budget, c.plan.ID, &c.stored.Amount, &changed.Amount); err != nil {
			return err
		}
	}
	mode := c.mode
	if mode == c.plan.Overspend.Normalised() {
		mode = ""
	}
	if !amountChanged && mode == "" {
		return nil
	}
	return s.plans.SetBudgetFromAllocation(ctx, a, PlanBudget{PlanID: c.plan.ID, Amount: c.amount, Overspend: mode,
		PlanVersion: c.planVersion, Comment: fmt.Sprintf(commentAllocated, c.budget.Name)})
}

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
	if !found || !s.access.CanViewPlan(ctx, plan) {
		return Allocation{}, GlobalBudget{}, PlanInfo{}, errs.NotFound(msgAllocationGone)
	}
	return allocation, b, plan, nil
}

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

type PlanAllocation struct {
	Allocation   Allocation
	GlobalBudget GlobalBudget
	CanOpen      bool
}

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
