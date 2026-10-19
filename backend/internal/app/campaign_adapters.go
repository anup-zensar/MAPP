package app

import (
	"context"
	"errors"

	"northstar/internal/campaigns"
	"northstar/internal/plans"
	"northstar/internal/platform/money"
)

// The adapters below give feature 005's campaigns package what it needs from feature 004 (campaigns.Deps,
// D-196); people.Repository is its PersonSource as it is, and 002's *settings.Service its Settings.
// campaignLevels are registered with 004's RegisterLevel in app.New.

// campaignPlans is the campaigns.PlanSource over the plans repository (feature 004).
type campaignPlans struct {
	repo plans.Repository
}

// Plan returns the plan id as campaigns needs it.
func (p campaignPlans) Plan(ctx context.Context, id string) (campaigns.PlanFacts, bool, error) {
	plan, found, err := p.repo.Get(ctx, id)
	if err != nil || !found {
		return campaigns.PlanFacts{}, false, err
	}
	return planFacts(plan), true, nil
}

// PlansOfYear returns every plan of year as campaigns needs them.
func (p campaignPlans) PlansOfYear(ctx context.Context, year int) ([]campaigns.PlanFacts, error) {
	list, err := p.repo.PlansOfYear(ctx, year)
	if err != nil {
		return nil, err
	}
	facts := make([]campaigns.PlanFacts, 0, len(list))
	for _, plan := range list {
		facts = append(facts, planFacts(plan))
	}
	return facts, nil
}

// planFacts turns a stored plan into campaigns' PlanFacts: no owner and no budget become nil, the budget
// USD money.
func planFacts(p plans.Plan) campaigns.PlanFacts {
	facts := campaigns.PlanFacts{
		ID: p.ID, Name: p.Name, Year: p.Year, Status: string(p.Status), ModelID: p.ModelID, Attributes: p.Attributes,
	}
	if p.OwnerID != "" {
		owner := p.OwnerID
		facts.OwnerID = &owner
	}
	if p.Budget != nil {
		budget := money.USD(*p.Budget)
		facts.Budget = &budget
	}
	return facts
}

// campaignLevels are the campaign and initiative levels for feature 004's level registry (D-195, 005 plan
// "Alignment with feature 004 as planned"): they add a plan's campaigns, initiatives, campaign budgets and
// the campaigns and initiatives that block publishing to its facts. app.New registers them with
// planService.RegisterLevel; plan copies (Copy, through CopyPlanContents) arrive with US6.
func campaignLevels(svc *campaigns.Service) []plans.Level {
	addFacts := func(level string) func(ctx context.Context, planIDs []string, facts map[string]*plans.Facts) error {
		return func(ctx context.Context, planIDs []string, facts map[string]*plans.Facts) error {
			rollups, err := svc.PlanRollups(ctx, planIDs)
			if err != nil {
				return err
			}
			for _, planID := range planIDs {
				r, f := rollups[planID], facts[planID]
				if level == "campaign" {
					f.Campaigns += r.Campaigns
					f.CampaignBudgets = f.CampaignBudgets.Add(r.AllocatedToCampaigns.Amount)
				} else {
					f.Initiatives += r.Initiatives
				}
				for _, p := range r.PublishProblems {
					if p.Record.Type == level {
						f.NeedsAttention = append(f.NeedsAttention, plans.AttentionItem{Record: p.Record, Missing: p.Missing, BlocksPublish: true})
					}
				}
			}
			return nil
		}
	}
	return []plans.Level{
		// Plan copies (US6) are not built yet: no plan operation calls Copy until they are.
		{Name: "campaign", AddFacts: addFacts("campaign"), Copy: func(context.Context, *plans.CopyJob) error {
			return errors.New("app: copying a plan's campaigns is not built yet (005 US6)")
		}},
		// The campaign level copies the initiatives with their campaigns, so this level's Copy does nothing.
		{Name: "initiative", AddFacts: addFacts("initiative"), Copy: func(context.Context, *plans.CopyJob) error { return nil }},
	}
}
