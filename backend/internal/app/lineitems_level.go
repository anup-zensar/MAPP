package app

import (
	"context"
	"maps"
	"slices"

	"mapp/internal/lineitem"
	"mapp/internal/plan"
	"mapp/internal/platform/errs"
	"mapp/internal/stores"
)

// lineItemKind names the line-item level of feature 004's registry, also its history kind.
const lineItemKind = "lineItem"

// lineItemLevel is the line-item level of feature 004's registry: how many line items each plan has and
// those still incomplete, below the plan's activities, and their copies in a plan copy (the line items of
// every activity the activity level copied, each copied once).
func lineItemLevel(svc *lineitem.Service, activitiesRepo stores.ActivityStore, campaignsRepo stores.CampaignStore) plan.Level {
	return plan.Level{
		Name: lineItemKind,
		AddFacts: func(ctx context.Context, planIDs []string, facts map[string]*plan.Facts) error {
			for _, planID := range planIDs {
				activityIDs, err := activitiesOfPlan(ctx, planID, activitiesRepo, campaignsRepo)
				if err != nil {
					return err
				}
				summaries, err := svc.Summaries(ctx, activityIDs)
				if err != nil {
					return err
				}
				var attention []plan.AttentionItem
				for _, id := range activityIDs { // in the activities' order
					s, found := summaries[id]
					if !found {
						continue
					}
					facts[planID].LineItems += s.Count
					for _, item := range s.Incomplete {
						attention = append(attention, plan.AttentionItem{
							Record:  errs.RecordRef{Type: lineItemKind, ID: item.LineItemID, Name: item.LineItemName},
							Missing: item.Missing,
						})
					}
				}
				facts[planID].NeedsAttention = append(facts[planID].NeedsAttention, attention...)
			}
			return nil
		},
		Copy: func(ctx context.Context, job *plan.CopyJob) error {
			newIDs, err := svc.CopyForActivities(ctx, job.Actor, job.NewIDs, job.Years)
			if err != nil {
				return err
			}
			maps.Copy(job.NewIDs, newIDs)
			job.Counts[lineItemKind] += len(newIDs)
			return nil
		},
	}
}

// activitiesOfPlan returns the ids of the activities of the plan, campaign by campaign, initiative by
// initiative.
func activitiesOfPlan(ctx context.Context, planID string, activitiesRepo stores.ActivityStore, campaignsRepo stores.CampaignStore) ([]string, error) {
	campaignList, err := campaignsRepo.CampaignsOfPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	campaignIDs := make([]string, 0, len(campaignList))
	for _, c := range campaignList {
		campaignIDs = append(campaignIDs, c.ID)
	}
	initiatives, err := campaignsRepo.InitiativesOfCampaigns(ctx, campaignIDs)
	if err != nil {
		return nil, err
	}
	initiativeIDs := make([]string, 0, len(initiatives))
	for _, i := range initiatives {
		initiativeIDs = append(initiativeIDs, i.ID)
	}
	list, err := activitiesRepo.ListByInitiatives(ctx, initiativeIDs)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(list))
	for _, a := range list {
		if !slices.Contains(ids, a.ID) {
			ids = append(ids, a.ID)
		}
	}
	return ids, nil
}
