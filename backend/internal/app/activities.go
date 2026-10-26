package app

import (
	"context"
	"maps"
	"slices"
	"strings"

	"mapp/internal/activities"
	"mapp/internal/campaigns"
	"mapp/internal/plans"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// The adapters below hand the activities service (feature 006) to the features above it (D-196): to
// campaigns (005) as its campaigns.Activities, and to plans (004) as the "activity" level of its registry.

// activitiesForCampaigns is campaigns.Activities over the activities service.
type activitiesForCampaigns struct {
	svc *activities.Service
}

// Summaries returns the roll-up summary of every activity under the initiatives.
func (x activitiesForCampaigns) Summaries(ctx context.Context, initiativeIDs []string) ([]campaigns.ActivitySummary, error) {
	return x.svc.Summaries(ctx, initiativeIDs)
}

// CopyAll copies every activity of an initiative, with its line items, into another initiative: the
// copies of a campaign or an initiative (005 FR-027).
func (x activitiesForCampaigns) CopyAll(ctx context.Context, a actor.Actor, fromInitiativeID, toInitiativeID string, yearShift int) (campaigns.ActivityCopyCounts, error) {
	counts, err := x.svc.CopyAll(ctx, a, fromInitiativeID, toInitiativeID, yearShift)
	return campaigns.ActivityCopyCounts{Activities: counts.Activities, LineItems: counts.LineItems}, err
}

// CopyMany copies the activities of many initiatives at once: a campaign copy.
func (x activitiesForCampaigns) CopyMany(ctx context.Context, a actor.Actor, pairs []campaigns.CopyPair, yearShift int) (campaigns.ActivityCopyCounts, error) {
	converted := make([]activities.CopyPair, 0, len(pairs))
	for _, p := range pairs {
		converted = append(converted, activities.CopyPair{FromID: p.FromID, ToID: p.ToID})
	}
	counts, err := x.svc.CopyMany(ctx, a, converted, yearShift)
	return campaigns.ActivityCopyCounts{Activities: counts.Activities, LineItems: counts.LineItems}, err
}

// activityLevel is the activity level of feature 004's registry: the activities' figures and the
// activities that need attention (AddPlanFacts), their copies in a plan copy (the activities of every
// copied initiative; their line items are the next level's), and the activities a new model would not
// allow (004 FR-019).
func activityLevel(svc *activities.Service, activitiesRepo activities.Repository, campaignsRepo campaigns.Repository) plans.Level {
	return plans.Level{
		Name:     "activity",
		AddFacts: svc.AddPlanFacts,
		Copy: func(ctx context.Context, job *plans.CopyJob) error {
			newIDs, err := svc.CopyForInitiatives(ctx, job.Actor, job.NewIDs, job.Years)
			if err != nil {
				return err
			}
			maps.Copy(job.NewIDs, newIDs)
			job.Counts["activity"] += len(newIDs)
			return nil
		},
		ActivitiesWithTypeNotAllowed: func(ctx context.Context, planID string, allowedTypeIDs []string) ([]errs.RecordRef, error) {
			return activitiesWithTypeNotAllowed(ctx, planID, allowedTypeIDs, activitiesRepo, campaignsRepo)
		},
	}
}

// activitiesWithTypeNotAllowed lists the stored activities of the plan whose activity type is not one of
// allowedTypeIDs (006 TypesNotAllowed, for 004 FR-019), ordered by path: campaign, initiative and activity
// names, ignoring letter case. An activity without a type is not listed: it has no type to disallow.
func activitiesWithTypeNotAllowed(ctx context.Context, planID string, allowedTypeIDs []string,
	activitiesRepo activities.Repository, campaignsRepo campaigns.Repository) ([]errs.RecordRef, error) {
	campaignList, err := campaignsRepo.CampaignsOfPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	campaignNames := map[string]string{}
	for _, c := range campaignList {
		campaignNames[c.ID] = c.Name
	}
	initiatives, err := campaignsRepo.InitiativesOfCampaigns(ctx, slices.Collect(maps.Keys(campaignNames)))
	if err != nil {
		return nil, err
	}
	paths := map[string][2]string{} // initiative id → its campaign's name and its own
	for _, i := range initiatives {
		paths[i.ID] = [2]string{campaignNames[i.CampaignID], i.Name}
	}
	list, err := activitiesRepo.ListByInitiatives(ctx, slices.Collect(maps.Keys(paths)))
	if err != nil {
		return nil, err
	}

	type disallowed struct {
		path [3]string // campaign, initiative and activity names
		ref  errs.RecordRef
	}
	var found []disallowed
	for _, a := range list {
		if a.ActivityTypeID == "" || slices.Contains(allowedTypeIDs, a.ActivityTypeID) {
			continue
		}
		path := paths[a.InitiativeID]
		found = append(found, disallowed{
			path: [3]string{path[0], path[1], a.Name},
			ref:  errs.RecordRef{Type: "activity", ID: a.ID, Name: a.Name},
		})
	}
	slices.SortStableFunc(found, func(x, y disallowed) int {
		for i := range x.path {
			if c := strings.Compare(strings.ToLower(x.path[i]), strings.ToLower(y.path[i])); c != 0 {
				return c
			}
		}
		return strings.Compare(x.ref.ID, y.ref.ID)
	})
	refs := make([]errs.RecordRef, 0, len(found))
	for _, d := range found {
		refs = append(refs, d.ref)
	}
	return refs, nil
}
