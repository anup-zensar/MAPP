package overspend

import (
	"cmp"
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// PostgresStore reads budgets and keeps the Overspend control setting on PostgreSQL. The budget reads are not
// scoped: they serve the guard's rule, never a list a person reads.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) PlanningBudgets(ctx context.Context) ([]Budget, error) {
	rows, err := s.db.Queries(ctx).ListOverspendBudgets(ctx)
	if err != nil {
		return nil, fmt.Errorf("select budgets with their planned cost: %w", err)
	}
	budgets := make([]Budget, 0, len(rows))
	for _, r := range rows {
		budgets = append(budgets, Budget{Kind: Kind(r.Kind), ID: r.ID, Name: r.Name, Mode: Mode(r.Overspend),
			Amount:  money.Money{Amount: money.FromCents(r.AmountMinor), Currency: r.Currency},
			Planned: money.Money{Amount: money.FromCents(r.PlannedMinor), Currency: r.Currency}})
	}
	return budgets, nil
}

func (s *PostgresStore) GlobalBudgets(ctx context.Context) ([]GlobalBudget, error) {
	rows, err := s.db.Queries(ctx).ListOverspendGlobalBudgets(ctx)
	if err != nil {
		return nil, fmt.Errorf("select global budgets with their plans: %w", err)
	}
	globals := []GlobalBudget{}
	for _, r := range rows {
		if len(globals) == 0 || globals[len(globals)-1].ID != r.ID {
			globals = append(globals, GlobalBudget{ID: r.ID, Name: r.Name, Mode: Mode(r.Overspend),
				Amount: money.FromCents(r.AmountMinor), Plans: []PlanCost{}})
		}
		if r.PlanID == nil || r.PlanCurrency == nil {
			continue
		}
		last := &globals[len(globals)-1]
		last.Plans = append(last.Plans, PlanCost{PlanID: *r.PlanID,
			Planned: money.Money{Amount: money.FromCents(r.PlannedMinor), Currency: *r.PlanCurrency}})
	}
	return globals, nil
}

func (s *PostgresStore) Configuration(ctx context.Context) (Configuration, error) {
	rows, err := s.db.Queries(ctx).GetOverspendConfiguration(ctx)
	if err != nil {
		return Configuration{}, fmt.Errorf("select the overspend configuration: %w", err)
	}
	if len(rows) == 0 {
		return DefaultConfiguration(), nil
	}
	r := rows[0]
	return Configuration{DefaultMode: Mode(r.DefaultMode), Meta: record.Meta{Version: r.Version, CreatedAt: r.CreatedAt,
		CreatedBy: actor.Actor{ID: r.CreatedByID, Name: r.CreatedByName}, UpdatedAt: r.UpdatedAt,
		UpdatedBy: actor.Actor{ID: r.UpdatedByID, Name: r.UpdatedByName}}}, nil
}

func (s *PostgresStore) PutConfiguration(ctx context.Context, c Configuration) error {
	created, by := cmp.Or(c.CreatedAt, c.UpdatedAt), cmp.Or(c.CreatedBy, c.UpdatedBy)
	err := s.db.Queries(ctx).PutOverspendConfiguration(ctx, sqlcdb.PutOverspendConfigurationParams{
		DefaultMode: string(c.DefaultMode.Normalised()), Version: c.Version, CreatedAt: created, CreatedByID: by.ID,
		CreatedByName: by.Name, UpdatedAt: c.UpdatedAt, UpdatedByID: c.UpdatedBy.ID, UpdatedByName: c.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save the overspend configuration: %w", err)
	}
	return nil
}

// Stored is the mode as a column keeps it: "" (a record built before feature 037) is Warn.
func Stored(m Mode) string {
	return string(m.Normalised())
}
