package campaigns

import (
	"context"

	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// PlanRollup is what one plan holds below it, as features 004 (plan figures, completeness and publish
// check) and 009 need it. Every figure is computed by the same view builder as the plan tree, so the views
// never disagree (D-090).
type PlanRollup struct {
	Campaigns, Initiatives, Activities, ApprovedActivities, LineItems int
	AllocatedToCampaigns                                              money.Money  // sum of the campaign budgets (plan currency, 016)
	PlannedCost                                                       *money.Money // nil when nothing below has one
	Goals                                                             rollup.Goals
	ItemsNeedingAttention                                             int              // incomplete campaigns, initiatives, activities and line items
	PublishProblems                                                   []RecordProblems // campaigns and initiatives with missing fields, in tree order (004 FR-024)
	// CampaignBudgetsSet and InitiativeBudgetsSet count the budgets that are set: amounts in the plan currency
	// that lock it (016 FR-012).
	CampaignBudgetsSet, InitiativeBudgetsSet int
}

// RecordProblems is a campaign or initiative with its missing needed-to-publish fields.
type RecordProblems struct {
	Record  errs.RecordRef
	Missing []settings.MissingField
}

// InitiativeFacts is an initiative as feature 006 needs it (R3, defaults of a new activity, move targets).
type InitiativeFacts struct {
	ID, Name                         string
	OwnerID                          *string
	StartDate, EndDate               *date.Date
	CampaignID, CampaignName, PlanID string
}

// PlanRollups returns the roll-up of every plan in planIDs; a plan that is gone or has no campaigns gets
// zeros. It joins the caller's transaction.
func (s *Service) PlanRollups(ctx context.Context, planIDs []string) (map[string]PlanRollup, error) {
	rollups := make(map[string]PlanRollup, len(planIDs))
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var found []PlanFacts
		for _, planID := range planIDs {
			plan, stored, err := s.deps.Plans.Plan(ctx, planID)
			if err != nil {
				return err
			}
			if !stored {
				rollups[planID] = PlanRollup{AllocatedToCampaigns: money.USD(money.Amount{}), PublishProblems: []RecordProblems{}}
				continue
			}
			found = append(found, plan)
		}
		trees, err := s.planTrees(ctx, found) // every plan at once, so many plans cost what one does
		for i, tree := range trees {
			rollups[found[i].ID] = rollupOf(tree)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return rollups, nil
}

// rollupOf sums up a plan's campaign tree.
func rollupOf(tree PlanCampaigns) PlanRollup {
	r := PlanRollup{
		Campaigns:            tree.Totals.Campaigns,
		Initiatives:          tree.Totals.Initiatives,
		Activities:           tree.Totals.Activities,
		AllocatedToCampaigns: tree.Totals.AllocatedToCampaigns,
		PlannedCost:          tree.Totals.PlannedCost,
		PublishProblems:      []RecordProblems{},
	}
	goals := make([]rollup.Goals, 0, len(tree.Items))
	for _, c := range tree.Items {
		if c.Budget != nil {
			r.CampaignBudgetsSet++
		}
		for _, i := range c.Initiatives {
			if i.Budget != nil {
				r.InitiativeBudgetsSet++
			}
		}
		r.ApprovedActivities += c.Counts.ApprovedActivities
		r.LineItems += c.Counts.LineItems
		r.ItemsNeedingAttention += c.Completeness.ItemsBelowNeedingAttention
		goals = append(goals, rollup.Goals{INQ: c.Goals.INQ, MQL: c.Goals.MQL, Revenue: c.Goals.Revenue})
		if len(c.Completeness.Missing) > 0 {
			r.ItemsNeedingAttention++
			r.PublishProblems = append(r.PublishProblems, RecordProblems{
				Record: errs.RecordRef{Type: campaignKind, ID: c.ID, Name: c.Name}, Missing: c.Completeness.Missing,
			})
		}
		for _, i := range c.Initiatives {
			if len(i.Completeness.Missing) > 0 {
				r.PublishProblems = append(r.PublishProblems, RecordProblems{
					Record: errs.RecordRef{Type: initiativeKind, ID: i.ID, Name: i.Name}, Missing: i.Completeness.Missing,
				})
			}
		}
	}
	r.Goals = rollup.AddGoals(goals)
	return r
}

// InitiativeFacts returns the initiative id with its campaign and plan, and false when there is none. It
// joins the caller's transaction.
func (s *Service) InitiativeFacts(ctx context.Context, id string) (InitiativeFacts, bool, error) {
	var facts InitiativeFacts
	var found bool
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		i, ok, err := s.repo.GetInitiative(ctx, id)
		if err != nil || !ok {
			return err
		}
		c, ok, err := s.repo.GetCampaign(ctx, i.CampaignID)
		if err != nil || !ok {
			return err
		}
		facts, found = initiativeFacts(i, c), true
		return nil
	})
	return facts, found, err
}

// InitiativesOfPlan returns every initiative of the plan planID, campaign by campaign in tree order. It
// joins the caller's transaction.
func (s *Service) InitiativesOfPlan(ctx context.Context, planID string) ([]InitiativeFacts, error) {
	list := []InitiativeFacts{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		campaigns, err := s.repo.CampaignsOfPlan(ctx, planID)
		if err != nil {
			return err
		}
		sortInTree(campaigns, func(c Campaign) (*date.Date, string, string) { return c.StartDate, c.Name, c.ID })
		for _, c := range campaigns {
			initiatives, err := s.repo.InitiativesOfCampaign(ctx, c.ID)
			if err != nil {
				return err
			}
			sortInTree(initiatives, func(i Initiative) (*date.Date, string, string) { return i.StartDate, i.Name, i.ID })
			for _, i := range initiatives {
				list = append(list, initiativeFacts(i, c))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

// initiativeFacts returns initiative i of campaign c as feature 006 needs it.
func initiativeFacts(i Initiative, c Campaign) InitiativeFacts {
	return InitiativeFacts{
		ID: i.ID, Name: i.Name, OwnerID: i.OwnerID, StartDate: i.StartDate, EndDate: i.EndDate,
		CampaignID: c.ID, CampaignName: c.Name, PlanID: c.PlanID,
	}
}
