package activitytest

import (
	"context"
	"slices"

	"mapp/internal/activity"
	"mapp/internal/plan"
	"mapp/internal/platform/authz"
)

// ListInScope keeps the activities the scope sees through their plan, owner and approvers, as
// activity_in_scope() does in SQL.
func (s Store) ListInScope(ctx context.Context, scope authz.Scope) ([]activity.Activity, error) {
	all, err := s.activities.List()
	if err != nil {
		return nil, err
	}
	approvers, err := s.approverIDs(ctx)
	if err != nil {
		return nil, err
	}
	seen := []activity.Activity{}
	for _, a := range all {
		p, planErr := s.planOf(ctx, a.InitiativeID)
		if planErr != nil {
			return nil, planErr
		}
		if scope.SeesActivity(authz.ActivityFacts{Plan: p, OwnerID: a.OwnerID, ApproverIDs: approvers[a.ID]}) {
			seen = append(seen, a)
		}
	}
	return seen, nil
}

func (s Store) ListInScopeOf(ctx context.Context, scope authz.Scope, ids []string) ([]activity.Activity, error) {
	seen, err := s.ListInScope(ctx, scope)
	return slices.DeleteFunc(seen, func(a activity.Activity) bool { return !slices.Contains(ids, a.ID) }), err
}

func (s Store) approverIDs(ctx context.Context) (map[string][]string, error) {
	rounds, err := s.approvals.ListRounds(ctx)
	if err != nil {
		return nil, err
	}
	ids := map[string][]string{}
	for _, r := range rounds {
		for _, level := range r.Levels {
			for _, approver := range level.Approvers {
				ids[r.SubjectID] = append(ids[r.SubjectID], approver.ID)
			}
		}
	}
	return ids, nil
}

func (s Store) planOf(ctx context.Context, initiativeID string) (authz.PlanFacts, error) {
	initiative, _, err := s.campaigns.GetInitiative(ctx, initiativeID)
	if err != nil {
		return authz.PlanFacts{}, err
	}
	c, _, err := s.campaigns.GetCampaign(ctx, initiative.CampaignID)
	if err != nil {
		return authz.PlanFacts{}, err
	}
	p, _, err := s.plans.Get(ctx, c.PlanID)
	return plan.AccessFacts(p), err
}
