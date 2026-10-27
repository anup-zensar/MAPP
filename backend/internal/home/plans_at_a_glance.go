package home

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// maxPlanRows is how many plans Plans at a glance shows (FR-010).
const maxPlanRows = 10

// PlansAtAGlance returns Plans at a glance for the scope (FR-008 – FR-010): the plans in scope, newest year
// first, then by name ignoring letter case, then by ID, at most 10, and how many are in scope. Each row has
// the plan's teams, the person's own first, its campaigns and activities, how many activities are approved
// (Approved, In progress or Complete, domain model §2.2), and its budget, planned cost and budget used as
// the plan page shows them, in the person's display currency.
func (s *Service) PlansAtAGlance(ctx context.Context, a actor.Actor, scopeText string) (PlanPage, error) {
	scope, scopeErr := ParseScope(scopeText)
	if scopeErr != nil {
		return PlanPage{}, scopeErr
	}
	var page PlanPage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.converter(ctx, a)
		if err != nil {
			return err
		}
		ps, err := s.planScope(ctx, a)
		if err != nil {
			return err
		}
		inScope := ps.in(scope)
		slices.SortFunc(inScope, func(x, y Plan) int {
			return cmp.Or(cmp.Compare(y.Year, x.Year), compareNames(x.Name, y.Name), strings.Compare(x.ID, y.ID))
		})
		shown := inScope[:min(len(inScope), maxPlanRows)]
		page = PlanPage{Items: make([]PlanRow, 0, len(shown)), Total: len(inScope)}
		for _, p := range shown {
			page.Items = append(page.Items, ps.row(p, c))
		}
		return nil
	})
	if err != nil {
		return PlanPage{}, err
	}
	return page, nil
}

// row returns the plan p as a row of Plans at a glance, its amounts converted with c.
func (ps planScope) row(p Plan, c converter) PlanRow {
	var planned money.Amount // nothing planned yet counts as 0.00 for the budget used, as on the plan page
	if p.PlannedCost != nil {
		planned = *p.PlannedCost
	}
	percent, over := budgetUsed(planned, p.Budget)
	return PlanRow{
		ID: p.ID, Name: p.Name, Year: p.Year, StrategicAOP: p.StrategicAOP, Status: p.Status, Teams: ps.teamViews(p),
		CampaignCount: p.CampaignCount, ActivityCount: p.ActivityCount, ApprovedActivityCount: p.ApprovedActivityCount,
		Budget: c.optional(p.Budget), PlannedCost: c.optional(p.PlannedCost), BudgetUsedPercent: percent, OverBudget: over,
	}
}

// teamViews returns p's teams, the person's own first, then by name ignoring letter case (FR-008).
func (ps planScope) teamViews(p Plan) []TeamView {
	teams := make([]TeamView, 0, len(p.Teams))
	for _, t := range p.Teams {
		teams = append(teams, TeamView{ID: t.ID, Name: t.Name, Colour: t.Colour, Active: t.Active, Mine: ps.myTeams[t.ID]})
	}
	slices.SortFunc(teams, func(x, y TeamView) int {
		if x.Mine != y.Mine {
			if x.Mine {
				return -1
			}
			return 1
		}
		return cmp.Or(compareNames(x.Name, y.Name), strings.Compare(x.ID, y.ID))
	})
	return teams
}
