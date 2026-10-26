package campaigns

import "context"

// planCampaigns builds the campaign tree of plan (buildTree). It runs inside the caller's transaction.
func (s *Service) planCampaigns(ctx context.Context, plan PlanFacts) (PlanCampaigns, error) {
	trees, err := s.planTrees(ctx, []PlanFacts{plan})
	if err != nil {
		return PlanCampaigns{}, err
	}
	return trees[0], nil
}

// planTrees builds the campaign tree of every plan, in the order of plans. The campaigns, the initiatives and
// the activities below are read once for all the plans, so a page of many plans reads no more than one plan.
func (s *Service) planTrees(ctx context.Context, plans []PlanFacts) ([]PlanCampaigns, error) {
	planIDs := make([]string, 0, len(plans))
	for _, p := range plans {
		planIDs = append(planIDs, p.ID)
	}
	campaigns, err := s.repo.CampaignsOfPlans(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	campaignsOf, campaignIDs := map[string][]Campaign{}, make([]string, 0, len(campaigns))
	for _, c := range campaigns {
		campaignsOf[c.PlanID] = append(campaignsOf[c.PlanID], c)
		campaignIDs = append(campaignIDs, c.ID)
	}
	below, err := s.below(ctx, campaignIDs)
	if err != nil {
		return nil, err
	}
	trees := make([]PlanCampaigns, 0, len(plans))
	for _, p := range plans {
		tree, err := s.buildTree(ctx, p, campaignsOf[p.ID], below)
		if err != nil {
			return nil, err
		}
		trees = append(trees, tree)
	}
	return trees, nil
}

// treeBelow is the initiatives of some campaigns by campaign, and their activities by initiative, in id order.
type treeBelow struct {
	initiativesOf map[string][]Initiative
	activitiesOf  map[string][]ActivitySummary
}

func (s *Service) below(ctx context.Context, campaignIDs []string) (treeBelow, error) {
	initiatives, err := s.repo.InitiativesOfCampaigns(ctx, campaignIDs)
	if err != nil {
		return treeBelow{}, err
	}
	initiativesOf, initiativeIDs := map[string][]Initiative{}, make([]string, 0, len(initiatives))
	for _, i := range initiatives {
		initiativesOf[i.CampaignID] = append(initiativesOf[i.CampaignID], i)
		initiativeIDs = append(initiativeIDs, i.ID)
	}
	summaries, err := s.deps.Activities.Summaries(ctx, initiativeIDs)
	if err != nil {
		return treeBelow{}, err
	}
	activitiesOf := map[string][]ActivitySummary{}
	for _, a := range summaries {
		activitiesOf[a.InitiativeID] = append(activitiesOf[a.InitiativeID], a)
	}
	return treeBelow{initiativesOf: initiativesOf, activitiesOf: activitiesOf}, nil
}
