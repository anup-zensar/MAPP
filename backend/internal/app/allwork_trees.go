package app

import (
	"context"

	"mapp/internal/plan"
)

// readTrees reads the campaigns, initiatives, activities and line items of every plan of a view, one level at
// a time, so that the stores keep them for the transaction and each plan's tree reads none of them again.
func (s allWorkSource) readTrees(ctx context.Context, list []plan.Plan) error {
	planIDs := make([]string, 0, len(list))
	for _, p := range list {
		planIDs = append(planIDs, p.ID)
	}
	campaignList, err := s.campaignsRepo.CampaignsOfPlans(ctx, planIDs)
	if err != nil {
		return err
	}
	campaignIDs := make([]string, 0, len(campaignList))
	for _, c := range campaignList {
		campaignIDs = append(campaignIDs, c.ID)
	}
	initiatives, err := s.campaignsRepo.InitiativesOfCampaigns(ctx, campaignIDs)
	if err != nil {
		return err
	}
	initiativeIDs := make([]string, 0, len(initiatives))
	for _, i := range initiatives {
		initiativeIDs = append(initiativeIDs, i.ID)
	}
	activityList, err := s.activitiesRepo.ListByInitiatives(ctx, initiativeIDs)
	if err != nil {
		return err
	}
	activityIDs := make([]string, 0, len(activityList))
	for _, a := range activityList {
		activityIDs = append(activityIDs, a.ID)
	}
	_, err = s.lineItemsRepo.ListByActivities(ctx, activityIDs)
	return err
}
