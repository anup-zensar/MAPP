package overspend

import (
	"errors"
	"net/http"
	"testing"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

func TestCanChange_OwnerOrBudgetManager_037_FR003(t *testing.T) {
	t.Parallel()
	plan := authz.PlanFacts{ID: "P-1001", TeamIDs: []string{"T-1001"}}
	planner := authz.GrantedRole{ID: "R-planner", Scope: authz.ScopeTeams, TeamIDs: []string{"T-1001"}}
	manager := authz.GrantedRole{ID: "R-global", Scope: authz.ScopeAll}
	perms := map[string][]authz.Permission{"R-planner": {authz.PlansEdit}, "R-global": {authz.GlobalBudgetsManage}}
	grants := func(person string, roles ...authz.GrantedRole) authz.Grants {
		return authz.NewGrants(person, roles, nil, perms, nil, authz.Rules{})
	}
	tests := []struct {
		name string
		ctx  authz.Grants
		want bool
	}{
		{name: "the budget's owner", ctx: grants("U-1002", planner), want: true},
		{name: "a person who manages global budgets for the plan", ctx: grants("U-1001", manager), want: true},
		{name: "another planner", ctx: grants("U-1004", planner)},
	}
	for _, tt := range tests {
		t.Run("037 FR-003 "+tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := authz.With(t.Context(), tt.ctx)
			got := CanChange(ctx, "U-1002", plan)
			err := RequireChange(ctx, "U-1002", plan)
			var refused *errs.Error
			if got.Available != tt.want || (err == nil) != tt.want {
				t.Fatalf("CanChange = %+v, RequireChange = %v; want available %v", got, err, tt.want)
			}
			if !tt.want && (!errors.As(err, &refused) || refused.Status != http.StatusForbidden || got.Reason != msgOwnerOrManager) {
				t.Errorf("refusal = %v, reason %q", err, got.Reason)
			}
		})
	}
	if !CanChange(t.Context(), "", plan).Available {
		t.Error("an internal call may change every setting")
	}
}
