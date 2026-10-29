package overspend

import (
	"context"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

const msgOwnerOrManager = "Only the budget's owner or someone who manages budgets can change this."

// CanChange allows the budget's owner, a manager of global budgets for the plan, or an internal call (FR-003, D-348).
func CanChange(ctx context.Context, ownerID string, plan authz.PlanFacts) authz.Availability {
	g, restricted := authz.From(ctx)
	switch {
	case !restricted, ownerID != "" && g.PersonID == ownerID, g.InScope(authz.GlobalBudgetsManage, plan):
		return authz.Allowed
	}
	return authz.Unavailable(msgOwnerOrManager)
}

func ControlOf(ctx context.Context, mode Mode, ownerID string, plan authz.PlanFacts) Control {
	return Control{Mode: mode.Normalised(), CanChange: CanChange(ctx, ownerID, plan)}
}

// RequireChange refuses with 403 forbidden a change of the setting by someone CanChange does not allow.
func RequireChange(ctx context.Context, ownerID string, plan authz.PlanFacts) error {
	if can := CanChange(ctx, ownerID, plan); !can.Available {
		return errs.Forbidden(can.Reason)
	}
	return nil
}
