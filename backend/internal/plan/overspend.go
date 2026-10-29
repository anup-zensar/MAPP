package plan

import (
	"cmp"
	"context"
	"fmt"

	"mapp/internal/overspend"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

const fieldOverspend = "overspend"

// OverspendDefaults gives the setting a new budget starts with (037 FR-007); it joins the caller's transaction.
type OverspendDefaults interface {
	DefaultMode(ctx context.Context) (overspend.Mode, error)
}

func (s *Service) UseOverspendDefaults(d OverspendDefaults) {
	s.overspendDefaults = d
}

func (s *Service) defaultOverspend(ctx context.Context) (overspend.Mode, error) {
	if s.overspendDefaults == nil {
		return overspend.Warn, nil
	}
	return s.overspendDefaults.DefaultMode(ctx)
}

// checkOverspend returns the setting to store (037 FR-001 – FR-004): sent, else stored, else the default; a change
// needs the owner or a budget manager, and an allocated plan's changes only through its allocation.
func (s *Service) checkOverspend(ctx context.Context, sent *string, stored *Plan, problems *errs.Problems) (overspend.Mode, error) {
	if sent == nil && stored == nil {
		return s.defaultOverspend(ctx)
	}
	if sent == nil {
		return stored.Overspend.Normalised(), nil
	}
	mode, ok := overspend.ParseMode(*sent)
	if !ok {
		problems.Add(fieldOverspend, overspend.MsgChooseMode)
	}
	if !ok || stored == nil || mode == stored.Overspend.Normalised() {
		return mode, nil
	}
	return mode, s.requireOverspendChange(ctx, *stored)
}

func (s *Service) requireOverspendChange(ctx context.Context, stored Plan) error {
	lock, err := s.budgetLock(ctx, stored)
	switch {
	case err != nil:
		return err
	case lock != "":
		return errs.NotAllowed(lock)
	}
	return overspend.RequireChange(ctx, stored.OwnerID, AccessFacts(stored))
}

func overspendControl(ctx context.Context, p Plan, allocation *PlanAllocation) overspend.Control {
	control := overspend.ControlOf(ctx, p.Overspend, p.OwnerID, AccessFacts(p))
	if allocation != nil {
		control.CanChange = authz.Unavailable(lockReason(allocation))
	}
	return control
}

func lockReason(allocation *PlanAllocation) string {
	ownerName := "the global budget's owner"
	if allocation.Owner != nil {
		ownerName = cmp.Or(allocation.Owner.Name, allocation.Owner.ID)
	}
	return fmt.Sprintf(msgBudgetAllocated, allocation.GlobalBudget.Name, ownerName)
}
