package activities

import (
	"context"

	"mapp/internal/approvals"
	"mapp/internal/campaigns"
	"mapp/internal/plans"
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
		plan, err := jsonPlanOf(ctx, a.InitiativeID)
		if err != nil {
			return nil, err
		}
		if scope.SeesActivity(authz.ActivityFacts{Plan: plan, OwnerID: a.OwnerID, ApproverIDs: approvers[a.ID]}) {
			seen = append(seen, a)
		}
	}
	return seen, nil
}

func jsonApprovers(ctx context.Context) (map[string][]string, error) {
	rounds, err := approvals.NewJSONStore().ListRounds(ctx)
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
	hierarchy := campaigns.NewJSONStore()
	initiative, _, err := hierarchy.GetInitiative(ctx, initiativeID)
	if err != nil {
		return authz.PlanFacts{}, err
	}
	campaign, _, err := hierarchy.GetCampaign(ctx, initiative.CampaignID)
	if err != nil {
		return authz.PlanFacts{}, err
	}
	plan, _, err := plans.NewJSONStore().Get(ctx, campaign.PlanID)
	return plans.AccessFacts(plan), err
}
