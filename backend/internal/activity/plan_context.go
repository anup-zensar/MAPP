package activity

import (
	"context"
	"maps"
	"slices"

	"mapp/internal/campaign"
	"mapp/internal/plan"
	"mapp/internal/platform/errs"
	"mapp/internal/settings"
)

// planContext is where an activity sits: its initiative, campaign and plan, the plan's model and business
// unit.
type planContext struct {
	plan         plan.Plan
	campaign     campaign.Campaign
	initiative   campaign.Initiative
	model        settings.Model
	businessUnit []string // the plan's Business unit option ids
}

// contextOf reads the plan context of the initiative initiativeID, and false when the initiative, its
// campaign or its plan is gone.
func (s *Service) contextOf(ctx context.Context, initiativeID string) (planContext, bool, error) {
	initiative, found, err := s.hierarchy.GetInitiative(ctx, initiativeID)
	if err != nil || !found {
		return planContext{}, false, err
	}
	c, found, err := s.hierarchy.GetCampaign(ctx, initiative.CampaignID)
	if err != nil || !found {
		return planContext{}, false, err
	}
	p, found, err := s.plans.Get(ctx, c.PlanID)
	if err != nil || !found {
		return planContext{}, false, err
	}
	pc, err := s.planFacts(ctx, p)
	pc.campaign, pc.initiative = c, initiative
	return pc, true, err
}

// planFacts reads what every activity of plan shares: its model and its business unit.
func (s *Service) planFacts(ctx context.Context, p plan.Plan) (planContext, error) {
	model, _, err := s.settings.Model(ctx, p.ModelID)
	if err != nil {
		return planContext{}, err
	}
	businessUnit, err := s.settings.BusinessUnit(ctx, p.Attributes)
	if err != nil {
		return planContext{}, err
	}
	return planContext{plan: p, model: model, businessUnit: businessUnit}, nil
}

// load reads the activity id and its plan context, or answers not found.
func (s *Service) load(ctx context.Context, id string) (Activity, planContext, error) {
	activity, found, err := s.repo.Get(ctx, id)
	if err != nil {
		return Activity{}, planContext{}, err
	}
	if !found {
		return Activity{}, planContext{}, errs.NotFound(activityGoneMessage)
	}
	pc, found, err := s.contextOf(ctx, activity.InitiativeID)
	if err != nil {
		return Activity{}, planContext{}, err
	}
	if !found {
		return Activity{}, planContext{}, errs.NotFound(activityGoneMessage)
	}
	return activity, pc, nil
}

// planActivities are every activity of a plan, with the plan's facts and each activity's initiative and
// campaign.
type planActivities struct {
	base        planContext // the plan, its model and business unit
	campaigns   map[string]campaign.Campaign
	initiatives map[string]campaign.Initiative
	activities  []Activity
}

// activitiesOf reads every activity of every initiative of every campaign of plan.
func (s *Service) activitiesOf(ctx context.Context, p plan.Plan) (planActivities, error) {
	base, err := s.planFacts(ctx, p)
	if err != nil {
		return planActivities{}, err
	}
	all := planActivities{base: base, campaigns: map[string]campaign.Campaign{}, initiatives: map[string]campaign.Initiative{}}
	campaignList, err := s.hierarchy.CampaignsOfPlan(ctx, p.ID)
	if err != nil {
		return planActivities{}, err
	}
	for _, c := range campaignList {
		all.campaigns[c.ID] = c
	}
	initiatives, err := s.hierarchy.InitiativesOfCampaigns(ctx, slices.Collect(maps.Keys(all.campaigns)))
	if err != nil {
		return planActivities{}, err
	}
	for _, i := range initiatives {
		all.initiatives[i.ID] = i
	}
	all.activities, err = s.repo.ListByInitiatives(ctx, slices.Collect(maps.Keys(all.initiatives)))
	return all, err
}

// contextOf returns the plan context of a, an activity of the plan.
func (p planActivities) contextOf(a Activity) planContext {
	pc := p.base
	pc.initiative = p.initiatives[a.InitiativeID]
	pc.campaign = p.campaigns[pc.initiative.CampaignID]
	return pc
}
