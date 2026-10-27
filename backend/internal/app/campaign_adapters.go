package app

import (
	"context"
	"maps"

	"mapp/internal/campaign"
	"mapp/internal/plan"
	"mapp/internal/platform/money"
	"mapp/internal/stores"
)

// The adapters below give feature 005's campaigns package what it needs from feature 004 (campaign.Deps,
// D-196); stores.PeopleStore is its PersonSource as it is, and 002's *settings.Service its Settings.
// campaignLevels are registered with 004's RegisterLevel in app.New.

// campaignPlans is the campaign.PlanSource over the plans repository (feature 004).
type campaignPlans struct {
	repo stores.PlanStore
}

// Plan returns the plan id as campaigns needs it.
func (p campaignPlans) Plan(ctx context.Context, id string) (campaign.PlanFacts, bool, error) {
	pl, found, err := p.repo.Get(ctx, id)
	if err != nil || !found {
		return campaign.PlanFacts{}, false, err
	}
	return planFacts(pl), true, nil
}

// PlansOfYear returns every plan of year as campaigns needs them.
func (p campaignPlans) PlansOfYear(ctx context.Context, year int) ([]campaign.PlanFacts, error) {
	list, err := p.repo.PlansOfYear(ctx, year)
	if err != nil {
		return nil, err
	}
	facts := make([]campaign.PlanFacts, 0, len(list))
	for _, p := range list {
		facts = append(facts, planFacts(p))
	}
	return facts, nil
}

// The names of the campaign and initiative levels of feature 004's level registry, also their history kinds.
const (
	campaignLevel   = "campaign"
	initiativeLevel = "initiative"
)

// publishProblems are the problems of the records of level that block publishing, in tree order.
func publishProblems(problems []campaign.RecordProblems, level string) []plan.AttentionItem {
	var items []plan.AttentionItem
	for _, p := range problems {
		if p.Record.Type == level {
			items = append(items, plan.AttentionItem{Record: p.Record, Missing: p.Missing, BlocksPublish: true})
		}
	}
	return items
}

// planFacts turns a stored plan into campaigns' PlanFacts: no owner and no budget become nil, the budget
// money in the plan currency (016).
func planFacts(p plan.Plan) campaign.PlanFacts {
	facts := campaign.PlanFacts{
		ID: p.ID, Name: p.Name, Year: p.Year, Status: string(p.Status), ModelID: p.ModelID, Attributes: p.Attributes,
		Access: plan.AccessFacts(p),
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
func campaignLevels(svc *campaign.Service) []plan.Level {
	addFacts := func(ctx context.Context, planIDs []string, facts map[string]*plan.Facts) error {
		rollups, err := svc.PlanRollups(ctx, planIDs)
		if err != nil {
			return err
		}
		for _, planID := range planIDs {
			r, f := rollups[planID], facts[planID]
			f.Campaigns += r.Campaigns
			f.CampaignBudgets = f.CampaignBudgets.Add(r.AllocatedToCampaigns.Amount)
			f.Initiatives += r.Initiatives
			f.AmountsSet += r.CampaignBudgetsSet + r.InitiativeBudgetsSet // 016 FR-012
			f.NeedsAttention = append(f.NeedsAttention, publishProblems(r.PublishProblems, campaignLevel)...)
			f.NeedsAttention = append(f.NeedsAttention, publishProblems(r.PublishProblems, initiativeLevel)...)
		}
		return nil
	}
	return []plan.Level{
		{Name: campaignLevel, AddFacts: addFacts, Copy: func(ctx context.Context, job *plan.CopyJob) error {
			counts, newIDs, err := svc.CopyPlanContents(ctx, job.Actor, job.SourcePlanID, job.TargetPlanID, job.Years)
			if err != nil {
				return err
			}
			maps.Copy(job.NewIDs, newIDs)
			job.Counts[campaignLevel] += counts.Campaigns
			job.Counts[initiativeLevel] += counts.Initiatives
			return nil
		}},
		// The campaign level adds the initiatives' facts and copies them with their campaigns, so this level
		// does nothing: the roll-ups of the plans are worked out once.
		{Name: initiativeLevel, AddFacts: func(context.Context, []string, map[string]*plan.Facts) error { return nil },
			Copy: func(context.Context, *plan.CopyJob) error { return nil }},
	}
}
