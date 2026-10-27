package financials

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// scope is what a person may see of the transactions (FR-028, 014 FR-007).
type scope struct {
	plans    []string // nil: every plan
	unlinked bool     // not-linked transactions: only for people who may import (D-301)
}

// scopeOf reads a's scope.
func (s *Service) scopeOf(ctx context.Context, a actor.Actor) (scope, error) {
	plans, err := s.access.VisiblePlanIDs(ctx, a)
	if err != nil {
		return scope{}, err
	}
	return scope{plans: plans, unlinked: s.access.Import(ctx, a) == nil}, nil
}

// sees reports whether the plan planID is in the scope.
func (sc scope) sees(planID string) bool {
	return sc.plans == nil || slices.Contains(sc.plans, planID)
}

// Overview returns Home's Financials card (FR-025): the figures of the current year's live plans in the
// Home scope (My teams: the plans with one of a's teams; Everything: every plan) that a may see, the number
// of their activities committed above planned cost and the number of not-linked transactions (for people
// who may import). The year is the API clock's (D-091).
func (s *Service) Overview(ctx context.Context, a actor.Actor, scopeText string) (Overview, error) {
	homeScope, parseErr := ParseScope(scopeText)
	if parseErr != nil {
		return Overview{}, parseErr
	}
	var overview Overview
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		if err := s.access.SeeFinance(ctx, a); err != nil {
			return err
		}
		sc, err := s.scopeOf(ctx, a)
		if err != nil {
			return err
		}
		planIDs, year, err := s.overviewPlans(ctx, a, homeScope, sc)
		if err != nil {
			return err
		}
		b, err := s.loadPlans(ctx, a.ID, planIDs)
		if err != nil {
			return err
		}
		overview = b.overview(b.tree.plans, sc)
		overview.Year, overview.Scope = year, homeScope
		return nil
	})
	if err != nil {
		return Overview{}, err
	}
	return overview, nil
}

// overviewPlans returns the plans of the current year the overview adds up, those sc sees, with My teams
// a's teams' only, and the year.
func (s *Service) overviewPlans(ctx context.Context, a actor.Actor, homeScope Scope, sc scope) ([]string, int, error) {
	year := s.today().Year()
	var mine []string
	if homeScope == ScopeMyTeams {
		var err error
		if mine, err = s.planning.MyTeamPlanIDs(ctx, a.ID); err != nil {
			return nil, year, err
		}
	}
	ofYear, err := s.planning.PlanIDsOfYear(ctx, year)
	if err != nil {
		return nil, year, err
	}
	planIDs := []string{}
	for _, id := range ofYear {
		if sc.sees(id) && (homeScope == ScopeEverything || slices.Contains(mine, id)) {
			planIDs = append(planIDs, id)
		}
	}
	return planIDs, year, nil
}

// overview adds up the plans as one record.
func (b *book) overview(plans []*tnode, sc scope) Overview {
	var facts Facts
	var planned *money.Amount
	var first *tnode
	lineItemsAbove, activitiesAbove := 0, 0
	for _, p := range plans {
		facts = facts.plus(p.facts)
		planned = addAmount(planned, p.planned)
		lineItemsAbove += p.lineItemsAbove
		first = firstByID(first, p.firstAbove)
		activitiesAbove += countActivitiesAbove(p)
	}
	above := planned != nil && facts.Committed.Cmp(*planned) > 0
	o := Overview{
		Figures:                figuresOf(facts, planned, above, lineItemsAbove, first, b.display),
		ActivitiesAbovePlanned: activitiesAbove, Source: b.source(),
	}
	if sc.unlinked {
		for _, t := range b.all {
			if !t.linked() {
				o.NotLinked++
			}
		}
	}
	return o
}

// countActivitiesAbove counts the activities below n committed above planned cost (data model §5).
func countActivitiesAbove(n *tnode) int {
	if n.Level == LevelActivity {
		if n.activityAbove() {
			return 1
		}
		return 0
	}
	count := 0
	for _, c := range n.children {
		count += countActivitiesAbove(c)
	}
	return count
}
