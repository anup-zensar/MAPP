package home

import (
	"context"

	"mapp/internal/platform/actor"
)

// maxRecentChanges is how many entries Recent changes shows (FR-015).
const maxRecentChanges = 10

// RecentChanges returns Recent changes for the scope (FR-015, FR-016): the 10 newest activity-log entries
// about plans, campaigns, initiatives, activities and line items, for My teams only those of the plans in
// scope and whatever is or was inside them, for Everything those of every plan, and how many match. A
// deleted record keeps its entries, with where the record is now (in the recycle bin). For My teams without
// team plans the page is empty and the log is not read.
func (s *Service) RecentChanges(ctx context.Context, a actor.Actor, scopeText string) (RecentChangesPage, error) {
	scope, scopeErr := ParseScope(scopeText)
	if scopeErr != nil {
		return RecentChangesPage{}, scopeErr
	}
	page := RecentChangesPage{Items: []any{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var planIDs []string // nil: every plan
		if scope == ScopeMyTeams {
			ps, err := s.planScope(ctx, a)
			if err != nil {
				return err
			}
			for _, p := range ps.in(scope) {
				planIDs = append(planIDs, p.ID)
			}
			if len(planIDs) == 0 {
				return nil
			}
		}
		found, err := s.src.RecentChanges(ctx, a, planIDs, maxRecentChanges)
		if err != nil {
			return err
		}
		page.Total = found.Total
		if found.Items != nil {
			page.Items = found.Items
		}
		return nil
	})
	if err != nil {
		return RecentChangesPage{}, err
	}
	return page, nil
}
