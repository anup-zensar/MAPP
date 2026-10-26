package financials

import (
	"context"
	"slices"
)

func (f *fakePlanning) PlanNodes(_ context.Context, planIDs []string) ([]Node, error) {
	var nodes []Node
	for _, n := range f.nodes {
		if slices.Contains(planIDs, n.PlanID) {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func (f *fakePlanning) PlanOf(_ context.Context, recordID string) (string, bool, error) {
	i := slices.IndexFunc(f.nodes, func(n Node) bool { return n.ID == recordID })
	if i < 0 {
		return "", false, nil
	}
	return f.nodes[i].PlanID, true, nil
}

func (f *fakePlanning) PlanIDsOfYear(_ context.Context, year int) ([]string, error) {
	var planIDs []string
	for _, n := range f.nodes {
		if n.Level == LevelPlan && n.PlanYear == year {
			planIDs = append(planIDs, n.ID)
		}
	}
	return planIDs, nil
}
