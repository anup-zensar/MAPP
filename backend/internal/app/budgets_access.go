package app

import (
	"context"
	"slices"

	"mapp/internal/access"
	"mapp/internal/budgets"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// budgetAccess is budgets.Access (014 plan Table C; FR-029, FR-030, FR-033) over the grants the request's
// context carries and, for who may own a global budget, over the access service. An internal call (no grants)
// may do everything.
type budgetAccess struct {
	access *access.Service
}

// The compiler checks here that budgetAccess implements budgets.Access.
var _ budgets.Access = budgetAccess{}

// budgetPlanFacts returns what an access decision needs to know of the plan.
func budgetPlanFacts(plan budgets.PlanInfo) authz.PlanFacts {
	return authz.PlanFacts{ID: plan.ID, TeamIDs: plan.TeamIDs, OwnerID: plan.OwnerID, CreatedByID: plan.CreatedByID}
}

// ManageAll needs globalBudgets.manage for All markets.
func (budgetAccess) ManageAll(ctx context.Context) error {
	g, restricted := authz.From(ctx)
	if !restricted {
		return nil
	}
	if all, _ := g.Teams(authz.GlobalBudgetsManage); all && g.Has(authz.GlobalBudgetsManage) {
		return nil
	}
	if !g.Has(authz.GlobalBudgetsManage) {
		return errs.Forbidden(g.Refusal(authz.GlobalBudgetsManage, nil))
	}
	return errs.Forbidden(g.Refusal(authz.GlobalBudgetsManage, &authz.PlanFacts{}))
}

// Manage needs globalBudgets.manage in some scope.
func (budgetAccess) Manage(ctx context.Context) error {
	return authz.Require(ctx, authz.GlobalBudgetsManage)
}

// ManageIn needs globalBudgets.manage in the plan's scope.
func (budgetAccess) ManageIn(ctx context.Context, plan budgets.PlanInfo) error {
	return authz.RequireIn(ctx, authz.GlobalBudgetsManage, budgetPlanFacts(plan))
}

// SeeGlobalBudgets reports whether the person holds globalBudgets.view in some scope.
func (budgetAccess) SeeGlobalBudgets(ctx context.Context) bool {
	return authz.Has(ctx, authz.GlobalBudgetsView)
}

// CanViewPlan reports whether the person sees the plan.
func (budgetAccess) CanViewPlan(ctx context.Context, plan budgets.PlanInfo) bool {
	return authz.CanView(ctx, budgetPlanFacts(plan))
}

// CanManageAll reports whether the person personID holds globalBudgets.manage for All markets.
func (b budgetAccess) CanManageAll(ctx context.Context, personID string) (bool, error) {
	holders, err := b.access.HoldersOf(ctx, authz.GlobalBudgetsManage, true)
	return slices.Contains(holders, personID), err
}
