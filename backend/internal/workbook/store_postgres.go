package workbook

import (
	"context"
	"fmt"
	"math"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
)

type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) AddImport(ctx context.Context, imp Import) error {
	q := s.db.Queries(ctx)
	err := q.AddPlanImport(ctx, sqlcdb.AddPlanImportParams{
		ID: imp.ID, PlanID: imp.PlanID, FileName: imp.FileName, ImportedAt: imp.ImportedAt, ImportedByID: imp.ImportedBy.ID,
		ImportedByName: imp.ImportedBy.Name, RowsRead: imp.RowsRead, RowsLeftOut: imp.RowsLeftOut,
	})
	if err != nil {
		return fmt.Errorf("add import %s: %w", imp.ID, err)
	}
	records := make([]sqlcdb.AddPlanImportRecordsParams, 0, len(imp.Records))
	for i, r := range imp.Records {
		records = append(records, sqlcdb.AddPlanImportRecordsParams{ImportID: imp.ID, Position: i + 1, Level: string(r.Level),
			RecordID: r.RecordID, RecordName: r.Name, ParentID: r.ParentID, Action: r.Action})
	}
	if _, err := q.AddPlanImportRecords(ctx, records); err != nil {
		return fmt.Errorf("add the records of import %s: %w", imp.ID, err)
	}
	problems := make([]sqlcdb.AddPlanImportProblemsParams, 0, len(imp.Problems))
	for i, p := range imp.Problems {
		problems = append(problems, sqlcdb.AddPlanImportProblemsParams{ImportID: imp.ID, Position: i + 1, Sheet: p.Sheet,
			RowNumber: p.Row, ColumnHeading: p.Column, Value: p.Value, Reason: p.Reason})
	}
	if _, err := q.AddPlanImportProblems(ctx, problems); err != nil {
		return fmt.Errorf("add the problems of import %s: %w", imp.ID, err)
	}
	return nil
}

func (s *PostgresStore) ListImports(ctx context.Context, scope authz.Scope, planID string, limit int) ([]Import, error) {
	return s.inScope(ctx, scope, importFilter{planID: &planID, limit: limit})
}

func (s *PostgresStore) GetImport(ctx context.Context, scope authz.Scope, id string) (Import, bool, error) {
	imports, err := s.inScope(ctx, scope, importFilter{id: &id, limit: 1})
	if err != nil || len(imports) == 0 {
		return Import{}, false, err
	}
	rows, err := s.db.Queries(ctx).ListPlanImportProblems(ctx, id)
	if err != nil {
		return Import{}, false, fmt.Errorf("select the problems of import %s: %w", id, err)
	}
	imp := imports[0]
	imp.Problems = make([]Problem, 0, len(rows))
	for _, p := range rows {
		imp.Problems = append(imp.Problems, Problem{Sheet: p.Sheet, Row: p.RowNumber, Column: p.ColumnHeading, Value: p.Value, Reason: p.Reason})
	}
	return imp, true, nil
}

func (s *PostgresStore) MarkUndone(ctx context.Context, id string, by actor.Actor, at time.Time) error {
	err := s.db.Queries(ctx).MarkPlanImportUndone(ctx, sqlcdb.MarkPlanImportUndoneParams{
		ID: id, UndoneAt: &at, UndoneByID: &by.ID, UndoneByName: &by.Name,
	})
	if err != nil {
		return fmt.Errorf("mark import %s undone: %w", id, err)
	}
	return nil
}

type importFilter struct {
	id, planID *string
	limit      int
}

func (s *PostgresStore) inScope(ctx context.Context, scope authz.Scope, f importFilter) ([]Import, error) {
	q := s.db.Queries(ctx)
	rows, err := q.ListPlanImportsInScope(ctx, sqlcdb.ListPlanImportsInScopeParams{
		ID: f.id, PlanID: f.planID, SeesEverything: scope.Everything, ViewerID: scope.PersonID, ViewsPlans: scope.Holds,
		ViewerTeamIds: scope.TeamIDs, MaxImports: int32(min(f.limit, math.MaxInt32)), //nolint:gosec // G115: min keeps it in range
	})
	if err != nil {
		return nil, fmt.Errorf("select imports in scope: %w", err)
	}
	imports := make([]Import, 0, len(rows))
	ids := make([]string, 0, len(rows))
	byID := map[string]int{}
	for _, r := range rows {
		byID[r.ID] = len(imports)
		ids = append(ids, r.ID)
		imports = append(imports, importOf(r))
	}
	records, err := q.ListPlanImportRecords(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("select the records of imports: %w", err)
	}
	for _, r := range records {
		i := byID[r.ImportID]
		imports[i].Records = append(imports[i].Records, ImportRecord{Level: Level(r.Level), RecordID: r.RecordID, Name: r.RecordName,
			ParentID: r.ParentID, Action: r.Action})
	}
	return imports, nil
}

func importOf(r sqlcdb.PlanImport) Import {
	imp := Import{ID: r.ID, PlanID: r.PlanID, FileName: r.FileName, ImportedAt: r.ImportedAt,
		ImportedBy: actor.Actor{ID: r.ImportedByID, Name: r.ImportedByName}, RowsRead: r.RowsRead, RowsLeftOut: r.RowsLeftOut,
		Records: []ImportRecord{}, Problems: []Problem{}, UndoneAt: r.UndoneAt}
	if r.UndoneByID != nil && r.UndoneByName != nil {
		imp.UndoneBy = &actor.Actor{ID: *r.UndoneByID, Name: *r.UndoneByName}
	}
	return imp
}
