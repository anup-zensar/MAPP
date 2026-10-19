package app

import (
	"context"
	"errors"

	"northstar/internal/activities"
	"northstar/internal/campaigns"
	"northstar/internal/plans"
	"northstar/internal/platform/actor"
)

// The adapters below hand the activities service (feature 006) to the features above it (D-196): to
// campaigns (005) as its campaigns.Activities, and to plans (004) as the "activity" level of its registry.

// errCopyNotBuilt is the answer of the copy seams: copying plans, campaigns and initiatives (004 US6, 005
// US6, 006 copy) is not built yet, and no served operation calls them until it is.
var errCopyNotBuilt = errors.New("app: copying activities is not built yet")

// activitiesForCampaigns is campaigns.Activities over the activities service.
type activitiesForCampaigns struct {
	svc *activities.Service
}

// Summaries returns the roll-up summary of every activity under the initiatives.
func (a activitiesForCampaigns) Summaries(ctx context.Context, initiativeIDs []string) ([]campaigns.ActivitySummary, error) {
	return a.svc.Summaries(ctx, initiativeIDs)
}

// CopyAll is not built yet (see errCopyNotBuilt).
func (activitiesForCampaigns) CopyAll(context.Context, actor.Actor, string, string, int) (campaigns.ActivityCopyCounts, error) {
	return campaigns.ActivityCopyCounts{}, errCopyNotBuilt
}

// activityLevel is the activity level of feature 004's registry: the activities' figures and the
// activities that need attention (AddPlanFacts). Plan copies are not built yet.
func activityLevel(svc *activities.Service) plans.Level {
	return plans.Level{
		Name:     "activity",
		AddFacts: svc.AddPlanFacts,
		Copy:     func(context.Context, *plans.CopyJob) error { return errCopyNotBuilt },
	}
}
