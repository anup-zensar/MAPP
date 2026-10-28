package app

import (
	"context"
	"strings"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/bulk"
	"mapp/internal/campaign"
	"mapp/internal/platform/actor"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// bulkPlanning is bulk.Planning on the planning areas: reads through their services, so a record the person
// cannot see is not found, and changes through their single operations (012 D-338).
type bulkPlanning struct {
	planningEdits
	plans     stores.PlanStore
	people    stores.PeopleStore
	settings  *settings.Service
	workflows activityWorkflows
	approvals *approval.Service
}

var _ bulk.Planning = bulkPlanning{}

var shownStatus = map[activity.DisplayStatus]string{ //nolint:gochecknoglobals // the status words of design system §4
	activity.DisplayCreated: "Created", activity.DisplaySubmitted: "Submitted", activity.DisplayDeclined: "Declined",
	activity.DisplayApproved: "Approved", activity.DisplayInProgress: "In progress", activity.DisplayComplete: "Complete",
}

func (p bulkPlanning) Find(ctx context.Context, a actor.Actor, id string) (bulk.Record, error) {
	switch prefix, _, _ := strings.Cut(id, "-"); prefix {
	case "C":
		v, err := p.campaigns.GetCampaign(ctx, a, id)
		return bulk.Record{ID: v.ID, Level: bulk.LevelCampaign, Name: v.Name, PlanID: v.PlanID, PlanName: v.Plan.Name,
			Year: v.Plan.Year, Ancestors: []string{v.PlanID}}, err
	case "I":
		v, err := p.campaigns.GetInitiative(ctx, a, id)
		return bulk.Record{ID: v.ID, Level: bulk.LevelInitiative, Name: v.Name, PlanID: v.PlanID, PlanName: v.Plan.Name,
			Year: v.Plan.Year, Ancestors: []string{v.CampaignID}}, err
	}
	v, err := p.activities.Get(ctx, a, id)
	if err != nil {
		return bulk.Record{}, err
	}
	plan, _, err := p.plans.Get(ctx, v.Plan.ID)
	return bulk.Record{ID: v.ID, Level: bulk.LevelActivity, Name: v.Name, PlanID: v.Plan.ID, PlanName: v.Plan.Name,
		Year: plan.Year, Ancestors: []string{v.Initiative.ID, v.Campaign.ID}, Status: string(v.Status),
		Shown: shownStatus[v.DisplayStatus]}, err
}

// Below counts the records under r and the Submitted or Approved activities among r and them.
func (p bulkPlanning) Below(ctx context.Context, r bulk.Record) (bulk.Below, error) {
	var below bulk.Below
	initiativeIDs := []string{}
	var activities []activity.Activity
	var err error
	switch r.Level {
	case bulk.LevelCampaign:
		var initiatives []campaign.Initiative
		if initiatives, err = p.campaignsRepo.InitiativesOfCampaign(ctx, r.ID); err != nil {
			return below, err
		}
		for _, i := range initiatives {
			initiativeIDs = append(initiativeIDs, i.ID)
		}
		below.Counts.Initiatives = len(initiatives)
	case bulk.LevelInitiative:
		initiativeIDs = append(initiativeIDs, r.ID)
	case bulk.LevelActivity:
		stored, _, getErr := p.activitiesRepo.Get(ctx, r.ID)
		activities, err = []activity.Activity{stored}, getErr
	}
	if len(initiativeIDs) > 0 {
		activities, err = p.activitiesRepo.ListByInitiatives(ctx, initiativeIDs)
		below.Counts.Activities = len(activities)
	}
	if err != nil {
		return below, err
	}
	return p.withLineItems(ctx, below, activities)
}

func (p bulkPlanning) withLineItems(ctx context.Context, below bulk.Below, activities []activity.Activity) (bulk.Below, error) {
	ids := make([]string, 0, len(activities))
	for _, a := range activities {
		ids = append(ids, a.ID)
		if a.Status == activity.StatusSubmitted || a.Status == activity.StatusApproved {
			below.InApproval++
		}
	}
	lineItems, err := p.lineItemsRepo.ListByActivities(ctx, ids)
	below.Counts.LineItems = len(lineItems)
	return below, err
}

func (p bulkPlanning) Person(ctx context.Context, id string) (string, bool, bool, error) {
	person, found, err := p.people.Get(ctx, id)
	return person.Name, person.Active, found, err
}
