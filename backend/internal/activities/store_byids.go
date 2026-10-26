package activities

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/database/sqlcdb"
)

func (s *PostgresStore) ListInScopeOf(ctx context.Context, scope authz.Scope, ids []string) ([]Activity, error) {
	if len(ids) == 0 {
		return []Activity{}, nil
	}
	rows, err := s.db.Queries(ctx).ListActivitiesInScopeOf(ctx, sqlcdb.ListActivitiesInScopeOfParams{
		Ids: ids, SeesEverything: scope.Everything, ViewerID: scope.PersonID, ViewsPlans: scope.Holds,
		ViewerTeamIds: scope.TeamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("select activities in scope: %w", err)
	}
	s.records().Keep(ctx, rows)
	return activitiesFromRows(rows, nil)
}

func (s *PostgresStore) ListByIDs(ctx context.Context, ids []string) ([]Activity, error) {
	return activitiesFromRows(s.records().ByID(ctx, ids))
}

func (s *PostgresStore) WorkflowIDs(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.Queries(ctx).ListActivityWorkflows(ctx)
	if err != nil {
		return nil, fmt.Errorf("select activity workflows: %w", err)
	}
	fields := make(map[string]string, len(rows))
	for _, row := range rows {
		fields[row.ID] = row.ApprovalWorkflowID
	}
	return fields, nil
}

func (r jsonRepository) WorkflowIDs(ctx context.Context) (map[string]string, error) {
	all, err := r.List(ctx)
	fields := make(map[string]string, len(all))
	for _, a := range all {
		fields[a.ID] = a.ApprovalWorkflowID
	}
	return fields, err
}

func (r jsonRepository) ListInScopeOf(ctx context.Context, scope authz.Scope, ids []string) ([]Activity, error) {
	seen, err := r.ListInScope(ctx, scope)
	return among(seen, ids), err
}

func (r jsonRepository) ListByIDs(ctx context.Context, ids []string) ([]Activity, error) {
	all, err := r.List(ctx)
	return among(all, ids), err
}

func among(list []Activity, ids []string) []Activity {
	kept := []Activity{}
	for _, a := range list {
		if slices.Contains(ids, a.ID) {
			kept = append(kept, a)
		}
	}
	return kept
}
