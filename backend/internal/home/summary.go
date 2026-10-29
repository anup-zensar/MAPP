package home

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
)

// Summary returns the four summary cards for the scope (FR-005, FR-006, FR-017): the plans in scope by
// status; the activities a owns, in the years of their plans (038 FR-032), with how many are Declined and Submitted; how many wait for
// a's approval and since when the oldest waits; and the planned cost of the plans in scope against their
// plan budgets, added up in US dollars and converted once.
func (s *Service) Summary(ctx context.Context, a actor.Actor, scopeText string) (SummaryView, error) {
	scope, scopeErr := ParseScope(scopeText)
	if scopeErr != nil {
		return SummaryView{}, scopeErr
	}
	var view SummaryView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.converter(ctx, a)
		if err != nil {
			return err
		}
		ps, err := s.planScope(ctx, a)
		if err != nil {
			return err
		}
		owned, err := s.src.Activities(ctx, a, a.ID)
		if err != nil {
			return err
		}
		waiting, err := s.src.WaitingForMe(ctx, a)
		if err != nil {
			return err
		}
		inScope := ps.in(scope)
		view = SummaryView{
			Scope: scope, DisplayCurrency: c.code, GettingStarted: gettingStarted(ps),
			Plans: planCounts(inScope), ActivitiesOwned: activityCounts(owned, ps.all), WaitingForMe: waitingCounts(waiting),
			PlannedCost: plannedCostOf(inScope, c),
		}
		return nil
	})
	if err != nil {
		return SummaryView{}, err
	}
	return view, nil
}

// The reasons to show the Getting started card (HomeSummary.gettingStarted, FR-020).
const (
	notInTeam   = "notInTeam"
	noTeamPlans = "noTeamPlans"
)

// gettingStarted says why a new person sees the Getting started card (FR-020, D-170): they belong to no
// team, or none of their teams has a plan. It is nil for everyone else, whatever the scope.
func gettingStarted(ps planScope) *string {
	var reason string
	switch {
	case len(ps.myTeams) == 0:
		reason = notInTeam
	case !slices.ContainsFunc(ps.all, ps.isTeamPlan):
		reason = noTeamPlans
	default:
		return nil
	}
	return &reason
}

// planCounts counts plans by status (FR-005 card 1).
func planCounts(plans []Plan) PlanCounts {
	counts := PlanCounts{Count: len(plans)}
	for _, p := range plans {
		if p.Status == planPublished {
			counts.Published++
		} else {
			counts.Draft++
		}
	}
	return counts
}

// maxYears is how many plan years All work shows at most (016 FR-004), so the card covers no more.
const maxYears = 5

// activityCounts counts the person's activities in the plan years All work opens on — the earliest year of
// their plans and at most four more — and the Declined and Submitted ones by stored status (FR-005 card 2,
// 038 FR-032). An activity of a plan the person cannot see is not on that list, so not counted.
func activityCounts(owned []Activity, plans []Plan) ActivityCounts {
	yearOf := make(map[string]int, len(plans))
	for _, p := range plans {
		yearOf[p.ID] = p.Year
	}
	years := make([]int, 0, len(owned))
	for _, act := range owned {
		if year, found := yearOf[act.Plan.ID]; found {
			years = append(years, year)
		}
	}
	var counts ActivityCounts
	if len(years) == 0 {
		return counts
	}
	from := slices.Min(years)
	to := slices.Max(slices.DeleteFunc(years, func(y int) bool { return y >= from+maxYears }))
	counts.YearFrom, counts.YearTo = &from, &to
	for _, act := range owned {
		if year, found := yearOf[act.Plan.ID]; found && year <= to {
			counts.add(act.Status)
		}
	}
	return counts
}

// add counts one activity of the stored status.
func (c *ActivityCounts) add(status string) {
	c.Count++
	switch status {
	case activityDeclined:
		c.Declined++
	case activitySubmitted:
		c.Submitted++
	}
}

// waitingCounts counts what waits for the person and finds the earliest "waiting since" (FR-005 card 3,
// D-175).
func waitingCounts(waiting []Waiting) WaitingCounts {
	counts := WaitingCounts{Count: len(waiting)}
	for _, w := range waiting {
		if counts.OldestWaitingSince == nil || w.WaitingSince.Before(*counts.OldestWaitingSince) {
			since := w.WaitingSince
			counts.OldestWaitingSince = &since
		}
	}
	return counts
}

// plannedCostOf adds up the planned cost and the plan budgets of plans in US dollars and converts the totals
// once (FR-005 card 4, FR-006, FR-017): budget used is the total planned cost over the total budget of the
// plans that have one; plans without a budget are counted.
func plannedCostOf(plans []Plan, c converter) PlannedCostTotal {
	total := PlannedCostTotal{PlanCount: len(plans)}
	var planned money.Amount // a plan with nothing planned adds 0.00
	var budget *money.Amount
	for _, p := range plans {
		if p.PlannedCost != nil {
			planned = planned.Add(*p.PlannedCost)
		}
		if p.Budget == nil {
			total.PlansWithoutBudget++
			continue
		}
		sum := *p.Budget
		if budget != nil {
			sum = budget.Add(sum)
		}
		budget = &sum
	}
	total.PlannedCost, total.PlanBudget = c.of(planned), c.optional(budget)
	total.BudgetUsedPercent, total.OverBudget = budgetUsed(planned, budget)
	return total
}

// budgetUsed is planned × 100 ÷ budget rounded half up to a whole percent, and whether planned is above the
// budget, compared exactly so that 100.4 % shows "100 %" in red (domain model §4.2, D-176) — the rule of
// the plan page. Without a budget there is neither; a budget of 0.00 has no percent.
func budgetUsed(planned money.Amount, budget *money.Amount) (*int, bool) {
	if budget == nil {
		return nil, false
	}
	plannedUSD, budgetUSD := money.USD(planned), money.USD(*budget)
	return rollup.BudgetUsedPercent(&plannedUSD, &budgetUSD), planned.Cmp(*budget) > 0
}
