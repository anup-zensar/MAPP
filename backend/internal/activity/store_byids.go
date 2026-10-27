package activity

import (
	"context"
	"fmt"

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
