package plan

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/overspend"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

const msgBudgetAllocated = "Allocated from '%s' — ask %s to change it."

type Allocation struct {
	ID               string
	GlobalBudgetID   string
	GlobalBudgetName string
	OwnerID          string
	Amount           money.Money
	CanOpen          bool
}

type Allocations interface {
	AllocationOf(ctx context.Context, planID string) (Allocation, bool, error)
}

type GlobalBudgetRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PlanAllocation struct {
	AllocationID string            `json:"allocationId"`
	GlobalBudget GlobalBudgetRef   `json:"globalBudget"`
	Amount       money.Money       `json:"amount"`
	Owner        *people.PersonRef `json:"owner"`
	CanOpen      bool              `json:"canOpen"`
}

type BudgetFromAllocation struct {
	PlanID    string
	Amount    money.Amount
	Overspend overspend.Mode
	Version   int
	Comment   string
}

func (s *Service) UseAllocations(a Allocations) {
	s.allocations = a
}

func (s *Service) allocationOf(ctx context.Context, planID string) (Allocation, bool, error) {
	if s.allocations == nil {
		return Allocation{}, false, nil
	}
	return s.allocations.AllocationOf(ctx, planID)
}

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
		if !found {
			owner = people.PersonRef{ID: allocation.OwnerID, Name: allocation.OwnerID}
		}
		view.Owner = &owner
	}
	return view, nil
}

func (s *Service) budgetLock(ctx context.Context, p Plan) (string, error) {
	allocation, err := s.allocationView(ctx, p)
	if err != nil || allocation == nil {
		return "", err
	}
	return lockReason(allocation), nil
}

func sameBudget(x, y *money.Amount) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}
	return x.Cmp(*y) == 0
}

// SetBudgetFromAllocation sets the plan's budget and overspend setting from an allocation in the caller's Write
// (014 FR-030, 037 FR-004); the plan keeps its version when nothing changes, and the comment is logged anyway.
func (s *Service) SetBudgetFromAllocation(ctx context.Context, a actor.Actor, in BudgetFromAllocation) error {
	return s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.stored(ctx, in.PlanID)
		if err != nil {
			return err
		}
		if err = stored.CheckVersion(in.Version, kindSingular); err != nil {
			return err
		}
		changed := copyOfPlan(stored)
		changed.Budget = &in.Amount
		if in.Overspend != "" {
			changed.Overspend = in.Overspend.Normalised()
		}
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
			Changes: changes, Comment: in.Comment,
		})
		return err
	})
}

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

// refuseAllocatedBudget refuses with 409 not_allowed a save that changes an allocated plan's budget (FR-031).
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
