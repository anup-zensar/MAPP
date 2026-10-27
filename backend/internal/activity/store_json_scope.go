package activity

import (
	"context"

	"mapp/internal/approval"
	"mapp/internal/campaign"
	"mapp/internal/plan"
	"mapp/internal/platform/authz"
)

// ListInScope reads every activity and keeps those the scope sees, as activity_in_scope() does in SQL.
func (r JSONStore) ListInScope(ctx context.Context, scope authz.Scope) ([]Activity, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	approvers, err := jsonApprovers(ctx)
	if err != nil {
		return nil, err
	}
	seen := []Activity{}
	for _, a := range all {
		p, err := jsonPlanOf(ctx, a.InitiativeID)
		if err != nil {
			return nil, err
		}
		if scope.SeesActivity(authz.ActivityFacts{Plan: p, OwnerID: a.OwnerID, ApproverIDs: approvers[a.ID]}) {
			seen = append(seen, a)
		}
	}
	return seen, nil
}

func jsonApprovers(ctx context.Context) (map[string][]string, error) {
	rounds, err := approval.NewJSONStore().ListRounds(ctx)
	if err != nil {
		return nil, err
	}
	ids := map[string][]string{}
	for _, r := range rounds {
		for _, level := range r.Levels {
			for _, approver := range level.Approvers {
				ids[r.ActivityID] = append(ids[r.ActivityID], approver.ID)
			}
		}
	}
	return ids, nil
}

func jsonPlanOf(ctx context.Context, initiativeID string) (authz.PlanFacts, error) {
	hierarchy := campaign.NewJSONStore()
	initiative, _, err := hierarchy.GetInitiative(ctx, initiativeID)
	if err != nil {
		return authz.PlanFacts{}, err
	}
	c, _, err := hierarchy.GetCampaign(ctx, initiative.CampaignID)
	if err != nil {
		return authz.PlanFacts{}, err
	}
	p, _, err := plan.NewJSONStore().Get(ctx, c.PlanID)
	return plan.AccessFacts(p), err
}
