package campaign

import (
	"context"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/money"
)

type DisplayRates interface {
	DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error)
}

func (s *Service) UseDisplayRates(rates DisplayRates) {
	s.displayRates = rates
}

type DisplayAmounts struct {
	Budget                *money.Money  `json:"budget"`
	ParentRemainingBudget *money.Money  `json:"parentRemainingBudget"`
	BudgetAndCost         BudgetAndCost `json:"budgetAndCost"`
	Goals                 GoalSums      `json:"goals"`
}

func (s *Service) displayFor(ctx context.Context, a actor.Actor) (money.Display, error) {
	d := money.DisplayUSD
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		rates, err := s.deps.Settings.Rates(ctx)
		if err != nil {
			return err
		}
		if s.displayRates != nil {
			rate, err := s.displayRates.DisplayRateOf(ctx, a.ID)
			if err != nil {
				return err
			}
			d = money.DisplayIn(rate.Code, rate.Rate)
		}
		d = d.WithRates(rates)
		return nil
	})
	return d, err
}

func (s *Service) shownTree(ctx context.Context, a actor.Actor, tree PlanCampaigns) (PlanCampaigns, error) {
	d, err := s.displayFor(ctx, a)
	if err != nil {
		return PlanCampaigns{}, err
	}
	items := make([]CampaignView, len(tree.Items))
	for i, c := range tree.Items {
		items[i] = displayCampaign(d, c)
	}
	tree.Items = items
	tree.DisplayTotals = displayTotals(d, tree.Totals)
	facts, err := s.planAccess(ctx, tree.Plan.ID)
	tree.Access = authz.ContentAccessOf(ctx, facts)
	for i, c := range tree.Items {
		tree.Items[i] = withOverspendAccess(ctx, c, facts)
	}
	return tree, err
}

func (s *Service) shownCampaign(ctx context.Context, a actor.Actor, view CampaignView) (CampaignView, error) {
	d, err := s.displayFor(ctx, a)
	if err != nil {
		return CampaignView{}, err
	}
	facts, err := s.planAccess(ctx, view.PlanID)
	view.Access = authz.CampaignAccessOf(ctx, facts)
	view = withOverspendAccess(ctx, view, facts)
	return displayCampaign(d, view), err
}

func (s *Service) shownInitiative(ctx context.Context, a actor.Actor, view InitiativeView) (InitiativeView, error) {
	d, err := s.displayFor(ctx, a)
	if err != nil {
		return InitiativeView{}, err
	}
	facts, err := s.planAccess(ctx, view.PlanID)
	view.Access = authz.InitiativeAccessOf(ctx, facts)
	view = withInitiativeAccess(ctx, view, facts)
	return displayInitiative(d, view), err
}

func (s *Service) planAccess(ctx context.Context, planID string) (authz.PlanFacts, error) {
	var facts authz.PlanFacts
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		plan, _, err := s.deps.Plans.Plan(ctx, planID)
		facts = plan.Access
		return err
	})
	return facts, err
}

func (s *Service) shownDestinations(ctx context.Context, a actor.Actor, list DestinationList) (DestinationList, error) {
	d, err := s.displayFor(ctx, a)
	if err != nil {
		return DestinationList{}, err
	}
	items := make([]Destination, len(list.Items))
	for i, item := range list.Items {
		item.RemainingAfter = d.Optional(item.RemainingAfter)
		item.OverAllocatedBy = d.Optional(item.OverAllocatedBy)
		items[i] = item
	}
	list.Items = items
	return list, nil
}

func displayCampaign(d money.Display, c CampaignView) CampaignView {
	c.Display = displayAmounts(d, c.Budget, c.ParentRemainingBudget, c.BudgetAndCost, c.Goals)
	initiatives := make([]InitiativeView, len(c.Initiatives))
	for i, view := range c.Initiatives {
		initiatives[i] = displayInitiative(d, view)
	}
	c.Initiatives = initiatives
	return c
}

func displayInitiative(d money.Display, i InitiativeView) InitiativeView {
	i.Display = displayAmounts(d, i.Budget, i.ParentRemainingBudget, i.BudgetAndCost, i.Goals)
	return i
}

func displayAmounts(d money.Display, budget, parentRemaining *money.Money, bc BudgetAndCost, goals GoalSums) DisplayAmounts {
	return DisplayAmounts{
		Budget:                d.Optional(budget),
		ParentRemainingBudget: d.Optional(parentRemaining),
		BudgetAndCost: BudgetAndCost{
			AllocatedToChildren:  d.Money(bc.AllocatedToChildren),
			RemainingForChildren: d.Optional(bc.RemainingForChildren),
			OverAllocatedBy:      d.Optional(bc.OverAllocatedBy),
			PlannedCost:          d.Optional(bc.PlannedCost),
			BudgetUsedPercent:    bc.BudgetUsedPercent,
			OverBudget:           bc.OverBudget,
			LeftToPlan:           d.Optional(bc.LeftToPlan),
		},
		Goals: GoalSums{INQ: goals.INQ, MQL: goals.MQL, Revenue: d.Optional(goals.Revenue)},
	}
}

func displayTotals(d money.Display, t PlanCampaignTotals) PlanCampaignTotals {
	t.AllocatedToCampaigns = d.Money(t.AllocatedToCampaigns)
	t.PlanBudget = d.Optional(t.PlanBudget)
	t.RemainingPlanBudget = d.Optional(t.RemainingPlanBudget)
	t.OverAllocatedBy = d.Optional(t.OverAllocatedBy)
	t.PlannedCost = d.Optional(t.PlannedCost)
	return t
}
