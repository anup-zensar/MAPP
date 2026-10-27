package app

import (
	"context"

	"mapp/internal/allwork"
	"mapp/internal/plan"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
)

// PlanYear is Year for the plan planID alone, when the person acting may see it, without the GSAP figures.
func (s allWorkSource) PlanYear(ctx context.Context, planID string, year int, _ date.Date) (allwork.YearData, error) {
	p, found, err := s.plansRepo.GetInScope(ctx, authz.ScopeOf(ctx), planID)
	if err != nil || !found || p.Year != year {
		return allwork.YearData{Nodes: []allwork.Node{}}, err
	}
	return s.nodesOf(ctx, []plan.Plan{p}, year, nil)
}
