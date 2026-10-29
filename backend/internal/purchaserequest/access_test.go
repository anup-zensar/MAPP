package purchaserequest

import (
	"testing"

	"mapp/internal/platform/authz"
)

func TestRequisitionerReason_038_FR025_M7(t *testing.T) {
	t.Parallel()
	plan := authz.PlanFacts{ID: "P-1001", TeamIDs: []string{"T-1001"}}
	perms := map[string][]authz.Permission{
		"RL-viewer":  {authz.PlansView, authz.PurchaseRequestsList},
		"RL-planner": {authz.PlansView, authz.PurchaseRequestsList, authz.PurchaseRequestsEdit, authz.PurchaseRequestsSubmit},
	}
	tests := []struct {
		name, role string
		editToo    bool
	}{
		{name: "a person who may not change it hears why, not how to change it", role: "RL-viewer"},
		{name: "a person who may change it is told to edit and submit changes", role: "RL-planner", editToo: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			roles := []authz.GrantedRole{{ID: tt.role, Scope: authz.ScopeTeams, TeamIDs: []string{"T-1001"}}}
			ctx := authz.With(t.Context(), authz.NewGrants("U-1006", roles, []string{"T-1001"}, perms, nil, authz.Rules{}))
			f := accessFacts{pr: PurchaseRequest{Status: StatusApproved}, config: DefaultConfiguration(), plan: plan}

			got := f.submit(ctx)

			permission := authz.Available(ctx, authz.PurchaseRequestsEdit, plan)
			if got.Available || (got.Reason == msgEditToChange) != tt.editToo ||
				(!tt.editToo && got.Reason != permission.Reason) {
				t.Errorf("submit = %+v, want the permission reason %q first", got, permission.Reason)
			}
		})
	}
}
