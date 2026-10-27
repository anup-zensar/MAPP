package plan

import (
	"cmp"
	"context"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// Global-budget allocations (feature 014, BUS.BUD.040, D-235). A global planner shares a global budget out to
// market plans; each allocation becomes the plan's budget, which is then read-only on the plan form. The
// allocations live in package budgets. Plans reads them through the Allocations hook and lets budgets set the
// budget through SetBudgetFromAllocation, so the plan budget keeps one owner and every roll-up of 004–010 reads
// it as before.

// msgBudgetAllocated is the reason the budget of an allocated plan cannot change (014 plan, message 23), on the
// plan form and as the 409 not_allowed answer of a save that changes it (FR-031).
const msgBudgetAllocated = "Allocated from '%s' — ask %s to change it."

// Allocation is the global-budget allocation that sets a plan's budget, as plans needs it (Go only).
type Allocation struct {
	ID               string      // "BA-1001"
	GlobalBudgetID   string      // "GB-1001"
	GlobalBudgetName string      // "FY2026 Global marketing budget"
	OwnerID          string      // the global budget's owner, whom to ask; "" for none
	Amount           money.Money // in the plan's currency; equal to the plan budget (D-303)
	CanOpen          bool        // the reader may open the global budget (014 globalBudgets.view)
}

// Allocations is the hook feature 014 implements (internal/app, over package budgets): it says which plans
// a global budget sets the budget of. Its method joins the caller's transaction, which ctx carries, and ctx
// also carries the signed-in person, for CanOpen.
type Allocations interface {
	// AllocationOf returns the allocation of the plan planID, and false when it has none.
	AllocationOf(ctx context.Context, planID string) (Allocation, bool, error)
}

// GlobalBudgetRef names a global budget (GlobalBudgetRef).
type GlobalBudgetRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PlanAllocation is the allocation as the plan page shows it (PlanAllocation, FR-031, FR-033): where the
// budget comes from and whom to ask.
type PlanAllocation struct {
	AllocationID string            `json:"allocationId"`
	GlobalBudget GlobalBudgetRef   `json:"globalBudget"`
	Amount       money.Money       `json:"amount"`
	Owner        *people.PersonRef `json:"owner"` // null when the global budget has none
	CanOpen      bool              `json:"canOpen"`
}

// UseAllocations tells plans where the allocations of 014 are. internal/app calls it while the API starts;
// until then no plan is allocated.
func (s *Service) UseAllocations(a Allocations) {
	s.allocations = a
}

// allocationOf returns the allocation of the plan planID, and false when it has none or no hook is set.
func (s *Service) allocationOf(ctx context.Context, planID string) (Allocation, bool, error) {
	if s.allocations == nil {
		return Allocation{}, false, nil
	}
	return s.allocations.AllocationOf(ctx, planID)
}

// allocationView returns the allocation of p as the plan page shows it; nil when p has none.
func (s *Service) allocationView(ctx context.Context, p Plan) (*PlanAllocation, error) {
	allocation, found, err := s.allocationOf(ctx, p.ID)
	if err != nil || !found {
		return nil, err
	}
	view := &PlanAllocation{
		AllocationID: allocation.ID,
		GlobalBudget: GlobalBudgetRef{ID: allocation.GlobalBudgetID, Name: allocation.GlobalBudgetName},
		Amount:       allocation.Amount,
		CanOpen:      allocation.CanOpen,
	}
	if allocation.OwnerID != "" {
		refs, err := s.directory.PersonRefs(ctx, []string{allocation.OwnerID})
		if err != nil {
			return nil, err
		}
		owner, found := refs[allocation.OwnerID]
		if !found { // an owner who no longer resolves is shown by their id
			owner = people.PersonRef{ID: allocation.OwnerID, Name: allocation.OwnerID}
		}
		view.Owner = &owner
	}
	return view, nil
}

// budgetLock returns why the budget of p cannot change, message 23 (FR-031); "" when p has no allocation.
// Feature 014's PlanAccess.editBudget shows the same reason.
func (s *Service) budgetLock(ctx context.Context, p Plan) (string, error) {
	allocation, err := s.allocationView(ctx, p)
	if err != nil || allocation == nil {
		return "", err
	}
	ownerName := "the global budget's owner"
	if allocation.Owner != nil {
		ownerName = cmp.Or(allocation.Owner.Name, allocation.Owner.ID)
	}
	return fmt.Sprintf(msgBudgetAllocated, allocation.GlobalBudget.Name, ownerName), nil
}

// sameBudget reports whether two plan budgets are equal: both none, or the same amount.
func sameBudget(x, y *money.Amount) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}
	return x.Cmp(*y) == 0
}

// SetBudgetFromAllocation sets the budget of the plan planID to amount, because an allocation of a global
// budget says so (014 FR-030, D-235): the one write path of the plan budget besides the plan form. It joins
// the caller's Write. version is the plan's version the global planner saw: an out-of-date one is refused
// naming who changed the plan (409 version_conflict), so an allocation never overwrites a newer plan unseen.
// When the budget changes, the plan gets a new version and one "changed" entry with the field Plan budget and
// comment ("Allocated from '<global budget>'"); when it already is amount, only the entry with the comment,
// and the plan keeps its version (014 data model §3).
func (s *Service) SetBudgetFromAllocation(ctx context.Context, a actor.Actor, planID string, amount money.Amount, version int,
	comment string) error {
	return s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.stored(ctx, planID)
		if err != nil {
			return err
		}
		if err := stored.CheckVersion(version, kindSingular); err != nil {
			return err
		}
		changed := copyOfPlan(stored)
		changed.Budget = &amount
		changes, err := planChanges(stored, changed)
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			changed.Touch(a, s.now())
			if err := s.repo.Put(ctx, changed); err != nil {
				return err
			}
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: stored.ID, RecordName: stored.Name, Action: history.ActionChanged,
			Changes: changes, Comment: comment,
		})
		return err
	})
}

// AllocationRemoved logs on the plan planID that its allocation was removed (014 FR-034, data model §3): a
// "changed" entry with comment ("Allocation from '<global budget>' removed") and no changed field. The plan
// keeps its budget and its version; its budget is editable again because no allocation sets it. It joins
// the caller's Write.
func (s *Service) AllocationRemoved(ctx context.Context, a actor.Actor, planID, comment string) error {
	return s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.stored(ctx, planID)
		if err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: stored.ID, RecordName: stored.Name, Action: history.ActionChanged, Comment: comment,
		})
		return err
	})
}

// PlannedCosts returns the planned cost in USD of each of the plans planIDs, nil for a plan below which
// nothing has a planned cost (domain model §4.2), from one call per level below (SC-009). Unknown plans get
// nil. It joins the caller's transaction: for the figures of 014's global budgets.
func (s *Service) PlannedCosts(ctx context.Context, planIDs []string) (map[string]*money.Amount, error) {
	costs := make(map[string]*money.Amount, len(planIDs))
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		facts, err := s.facts(ctx, planIDs)
		if err != nil {
			return err
		}
		for id, f := range facts {
			costs[id] = f.PlannedCost
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return costs, nil
}

// refuseAllocatedBudget refuses a save that changes the budget of an allocated plan with 409 not_allowed and
// message 23 (FR-031); any other save of the plan, the same budget included, goes on.
func (s *Service) refuseAllocatedBudget(ctx context.Context, stored, changed Plan) error {
	if sameBudget(stored.Budget, changed.Budget) {
		return nil
	}
	reason, err := s.budgetLock(ctx, stored)
	if err != nil {
		return err
	}
	if reason != "" {
		return errs.NotAllowed(reason)
	}
	return nil
}
