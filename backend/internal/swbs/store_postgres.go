package swbs

import (
	"context"
	"fmt"
	"math"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/record"
)

// PostgresStore keeps runs, which hold no plan-owned record of their own: who may read them is decided by the
// service (finance.import, which applies to All markets).
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) AddRun(ctx context.Context, run Run) error {
	q := s.db.Queries(ctx)
	err := q.AddSwbsRun(ctx, sqlcdb.AddSwbsRunParams{
		ID: run.ID, Source: string(run.Source), Version: run.Version, CreatedAt: run.CreatedAt,
		CreatedByID: run.CreatedBy.ID, CreatedByName: run.CreatedBy.Name, UpdatedAt: run.UpdatedAt,
		UpdatedByID: run.UpdatedBy.ID, UpdatedByName: run.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("add SWBS run %s: %w", run.ID, err)
	}
	if len(run.Codes) == 0 {
		return nil
	}
	rows := make([]sqlcdb.AddSwbsRunCodesParams, 0, len(run.Codes))
	for i, c := range run.Codes {
		rows = append(rows, sqlcdb.AddSwbsRunCodesParams{
			RunID: run.ID, Position: i + 1, LineItemID: c.LineItemID, LineItemName: c.LineItemName,
			CompanyCode: c.CompanyCode, SwbsCode: c.SwbsCode,
		})
	}
	if _, err := q.AddSwbsRunCodes(ctx, rows); err != nil {
		return fmt.Errorf("add the codes of SWBS run %s: %w", run.ID, err)
	}
	return nil
}

func (s *PostgresStore) GetRun(ctx context.Context, id string) (Run, bool, error) {
	runs, err := s.list(ctx, &id, 1)
	if err != nil || len(runs) == 0 {
		return Run{}, false, err
	}
	return runs[0], true, nil
}

func (s *PostgresStore) DeleteRun(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteSwbsRun(ctx, id); err != nil {
		return fmt.Errorf("delete SWBS run %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) ListRuns(ctx context.Context, limit int) ([]Run, int, error) {
	runs, err := s.list(ctx, nil, limit)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.db.Queries(ctx).CountSwbsRuns(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count SWBS runs: %w", err)
	}
	return runs, total, nil
}

func (s *PostgresStore) CountCodes(ctx context.Context, prefix string) (int, error) {
	n, err := s.db.Queries(ctx).CountSwbsCodes(ctx, prefix)
	if err != nil {
		return 0, fmt.Errorf("count SWBS codes starting %s: %w", prefix, err)
	}
	return n, nil
}

func (s *PostgresStore) list(ctx context.Context, id *string, limit int) ([]Run, error) {
	rows, err := s.db.Queries(ctx).ListSwbsRuns(ctx, sqlcdb.ListSwbsRunsParams{ID: id, MaxRuns: int32(min(limit, math.MaxInt32))}) //nolint:gosec // G115: min keeps it in range
	if err != nil {
		return nil, fmt.Errorf("select SWBS runs: %w", err)
	}
	runs := make([]Run, 0, len(rows))
	for _, row := range rows {
		r := row.SwbsRun
		run := Run{ID: r.ID, Source: Source(r.Source), Codes: make([]Code, 0, len(row.SwbsCodes)), Meta: record.Meta{
			Version: r.Version, CreatedAt: r.CreatedAt, CreatedBy: actor.Actor{ID: r.CreatedByID, Name: r.CreatedByName},
			UpdatedAt: r.UpdatedAt, UpdatedBy: actor.Actor{ID: r.UpdatedByID, Name: r.UpdatedByName},
		}}
		for i, code := range row.SwbsCodes {
			run.Codes = append(run.Codes, Code{
				LineItemID: row.LineItemIds[i], LineItemName: row.LineItemNames[i], CompanyCode: row.CompanyCodes[i], SwbsCode: code,
			})
		}
		runs = append(runs, run)
	}
	return runs, nil
}
