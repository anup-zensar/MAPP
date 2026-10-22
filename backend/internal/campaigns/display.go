package campaigns

import (
	"context"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/money"
)

// The amounts campaigns and initiatives show — the budget, the parent's remaining budget, budget and cost
// and the goals' revenue — and the plan tree's totals are answered a second time in the reader's display
// currency (D-061, D-220): `display` on each campaign and initiative and `displayTotals` on the tree. Each is
// worked out in USD and converted once, half up to the cent. The USD fields stay as they are, because the
// forms type and compare budgets in USD (FR-010) and the web app never converts. The destinations of a move
// or copy only show their amounts, so theirs are converted in place.

// DisplayRates gives the display currency a person shows amounts in, with its rate in units per 1 USD
// (feature 003). internal/app hands people's service to UseDisplayRates; without it amounts stay in USD.
type DisplayRates interface {
	DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error)
}

// UseDisplayRates sets where the display currency of the reader is read (D-220).
func (s *Service) UseDisplayRates(rates DisplayRates) {
	s.displayRates = rates
}

// DisplayAmounts are a campaign's or an initiative's amounts in the reader's display currency
// (ContainerDisplay).
type DisplayAmounts struct {
	Budget                *money.Money  `json:"budget"`
	ParentRemainingBudget *money.Money  `json:"parentRemainingBudget"`
	BudgetAndCost         BudgetAndCost `json:"budgetAndCost"`
	Goals                 GoalSums      `json:"goals"`
}

// displayFor returns the display currency of the person a, read in a transaction of its own, converting the
// plan currencies' amounts with the current rates (016 FR-013): each amount once, from its own currency.
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

// shownTree returns the tree with its amounts also in a's display currency.
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
	tree.Access = authz.ContentAccessOf(ctx, facts) // 014 FR-011
	return tree, err
}

// shownCampaign returns the campaign with its amounts also in a's display currency.
func (s *Service) shownCampaign(ctx context.Context, a actor.Actor, view CampaignView) (CampaignView, error) {
	d, err := s.displayFor(ctx, a)
	if err != nil {
		return CampaignView{}, err
	}
	facts, err := s.planAccess(ctx, view.PlanID)
	view.Access = authz.CampaignAccessOf(ctx, facts) // 014 FR-011
	return displayCampaign(d, view), err
}

// shownInitiative returns the initiative with its amounts also in a's display currency.
func (s *Service) shownInitiative(ctx context.Context, a actor.Actor, view InitiativeView) (InitiativeView, error) {
	d, err := s.displayFor(ctx, a)
	if err != nil {
		return InitiativeView{}, err
	}
	facts, err := s.planAccess(ctx, view.PlanID)
	view.Access = authz.InitiativeAccessOf(ctx, facts) // 014 FR-011
	return displayInitiative(d, view), err
}

// planAccess returns the access facts of the plan planID, read in a transaction of its own (feature 014);
// zero facts when the plan is gone.
func (s *Service) planAccess(ctx context.Context, planID string) (authz.PlanFacts, error) {
	var facts authz.PlanFacts
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		plan, _, err := s.deps.Plans.Plan(ctx, planID)
		facts = plan.Access
		return err
	})
	return facts, err
}

// shownDestinations returns the destinations with what the budget would leave there in a's display
// currency.
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

// displayCampaign fills the campaign's and its initiatives' display amounts from their USD amounts.
func displayCampaign(d money.Display, c CampaignView) CampaignView {
	c.Display = displayAmounts(d, c.Budget, c.ParentRemainingBudget, c.BudgetAndCost, c.Goals)
	initiatives := make([]InitiativeView, len(c.Initiatives))
	for i, view := range c.Initiatives {
		initiatives[i] = displayInitiative(d, view)
	}
	c.Initiatives = initiatives
	return c
}

// displayInitiative fills the initiative's display amounts from its USD amounts.
func displayInitiative(d money.Display, i InitiativeView) InitiativeView {
	i.Display = displayAmounts(d, i.Budget, i.ParentRemainingBudget, i.BudgetAndCost, i.Goals)
	return i
}

// displayAmounts converts a record's USD amounts.
func displayAmounts(d money.Display, budget, parentRemaining *money.Money, bc BudgetAndCost, goals GoalSums) DisplayAmounts {
	return DisplayAmounts{
		Budget:                d.Optional(budget),
		ParentRemainingBudget: d.Optional(parentRemaining),
		BudgetAndCost: BudgetAndCost{
			AllocatedToChildren:  d.Money(bc.AllocatedToChildren),
			RemainingForChildren: d.Optional(bc.RemainingForChildren),
			OverAllocatedBy:      d.Optional(bc.OverAllocatedBy),
			PlannedCost:          d.Optional(bc.PlannedCost),
			BudgetUsedPercent:    bc.BudgetUsedPercent, // worked out in the plan currency
			OverBudget:           bc.OverBudget,
			LeftToPlan:           d.Optional(bc.LeftToPlan),
		},
		Goals: GoalSums{INQ: goals.INQ, MQL: goals.MQL, Revenue: d.Optional(goals.Revenue)},
	}
}

// displayTotals converts the totals row's USD amounts.
func displayTotals(d money.Display, t PlanCampaignTotals) PlanCampaignTotals {
	t.AllocatedToCampaigns = d.Money(t.AllocatedToCampaigns)
	t.PlanBudget = d.Optional(t.PlanBudget)
	t.RemainingPlanBudget = d.Optional(t.RemainingPlanBudget)
	t.OverAllocatedBy = d.Optional(t.OverAllocatedBy)
	t.PlannedCost = d.Optional(t.PlannedCost)
	return t
}
