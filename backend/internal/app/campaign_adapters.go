package app

import (
	"context"
	"maps"

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
// money in the plan currency (016).
func planFacts(p plans.Plan) campaigns.PlanFacts {
	facts := campaigns.PlanFacts{
		ID: p.ID, Name: p.Name, Year: p.Year, Status: string(p.Status), ModelID: p.ModelID, Attributes: p.Attributes,
	}
	if p.OwnerID != "" {
		owner := p.OwnerID
		facts.OwnerID = &owner
	}
	facts.Currency = p.Currency // 016 FR-011: every amount of the plan is in it
	if p.Budget != nil {
		budget := money.Money{Amount: *p.Budget, Currency: p.Currency}
		facts.Budget = &budget
	}
	return facts
}

// campaignLevels are the campaign and initiative levels for feature 004's level registry (D-195, 005 plan
// "Alignment with feature 004 as planned"): they add a plan's campaigns, initiatives, campaign budgets and
// the campaigns and initiatives that block publishing to its facts, and copy the campaigns with their
// initiatives in a plan copy (CopyPlanContents). app.New registers them with planService.RegisterLevel.
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
					f.AmountsSet += r.CampaignBudgetsSet // 016 FR-012
				} else {
					f.Initiatives += r.Initiatives
					f.AmountsSet += r.InitiativeBudgetsSet
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
		{Name: "campaign", AddFacts: addFacts("campaign"), Copy: func(ctx context.Context, job *plans.CopyJob) error {
			counts, newIDs, err := svc.CopyPlanContents(ctx, job.Actor, job.SourcePlanID, job.TargetPlanID, job.Years)
			if err != nil {
				return err
			}
			maps.Copy(job.NewIDs, newIDs)
			job.Counts["campaign"] += counts.Campaigns
			job.Counts["initiative"] += counts.Initiatives
			return nil
		}},
		// The campaign level copies the initiatives with their campaigns, so this level's Copy does nothing.
		{Name: "initiative", AddFacts: addFacts("initiative"), Copy: func(context.Context, *plans.CopyJob) error { return nil }},
	}
}
