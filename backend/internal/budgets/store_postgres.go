package budgets

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// PostgresStore is the Repository on PostgreSQL. Global budgets belong to the organisation; allocations
// belong to their plan, and AllocationsInScope keeps only those of the plans the person acting may see.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) GetBudget(ctx context.Context, id string) (GlobalBudget, bool, error) {
	row, err := s.db.Queries(ctx).GetGlobalBudget(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return GlobalBudget{}, false, nil
	}
	if err != nil {
		return GlobalBudget{}, false, fmt.Errorf("select global budget %s: %w", id, err)
	}
	return budgetFromRow(row), true, nil
}

func (s *PostgresStore) ListBudgets(ctx context.Context) ([]GlobalBudget, error) {
	rows, err := s.db.Queries(ctx).ListGlobalBudgets(ctx)
	if err != nil {
		return nil, fmt.Errorf("select global budgets: %w", err)
	}
	budgets := make([]GlobalBudget, 0, len(rows))
	for _, row := range rows {
		budgets = append(budgets, budgetFromRow(row))
	}
	return budgets, nil
}

func (s *PostgresStore) PutBudget(ctx context.Context, b GlobalBudget) error {
	err := s.db.Queries(ctx).PutGlobalBudget(ctx, sqlcdb.PutGlobalBudgetParams{
		ID: b.ID, Name: b.Name, Year: b.Year, AmountMinor: b.Amount.Cents(), OwnerID: b.OwnerID,
		Description: b.Description, Version: b.Version, CreatedAt: b.CreatedAt, CreatedByID: b.CreatedBy.ID,
		CreatedByName: b.CreatedBy.Name, UpdatedAt: b.UpdatedAt, UpdatedByID: b.UpdatedBy.ID,
		UpdatedByName: b.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save global budget %s: %w", b.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteBudget(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteGlobalBudget(ctx, id); err != nil {
		return fmt.Errorf("delete global budget %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) GetAllocation(ctx context.Context, id string) (Allocation, bool, error) {
	return firstAllocation(s.allocations(ctx, sqlcdb.ListAllocationsParams{ID: &id}))
}

func (s *PostgresStore) ListAllocations(ctx context.Context) ([]Allocation, error) {
	return s.allocations(ctx, sqlcdb.ListAllocationsParams{})
}

func (s *PostgresStore) AllocationsOf(ctx context.Context, globalBudgetID string) ([]Allocation, error) {
	return s.allocations(ctx, sqlcdb.ListAllocationsParams{GlobalBudgetID: &globalBudgetID})
}

func (s *PostgresStore) AllocationOfPlan(ctx context.Context, planID string) (Allocation, bool, error) {
	return firstAllocation(s.allocations(ctx, sqlcdb.ListAllocationsParams{PlanID: &planID}))
}

// AllocationsInScope returns the allocations of globalBudgetID ("" for every global budget) to the plans the
// scope sees.
func (s *PostgresStore) AllocationsInScope(ctx context.Context, scope authz.Scope, globalBudgetID string) ([]Allocation, error) {
	rows, err := s.db.Queries(ctx).ListAllocationsInScope(ctx, sqlcdb.ListAllocationsInScopeParams{
		GlobalBudgetID: database.NullIfEmpty(globalBudgetID), SeesEverything: scope.Everything,
		ViewerID: scope.PersonID, ViewsPlans: scope.ViewsPlans, ViewerTeamIds: scope.TeamIDs,
	})
	return allocationsFromRows(rows, err)
}

func (s *PostgresStore) PutAllocation(ctx context.Context, a Allocation) error {
	err := s.db.Queries(ctx).PutAllocation(ctx, sqlcdb.PutAllocationParams{
		ID: a.ID, GlobalBudgetID: a.GlobalBudgetID, PlanID: a.PlanID, AmountMinor: a.Amount.Amount.Cents(),
		AmountCurrency: a.Amount.Currency, Version: a.Version, CreatedAt: a.CreatedAt, CreatedByID: a.CreatedBy.ID,
		CreatedByName: a.CreatedBy.Name, UpdatedAt: a.UpdatedAt, UpdatedByID: a.UpdatedBy.ID,
		UpdatedByName: a.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save budget allocation %s: %w", a.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteAllocation(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteAllocation(ctx, id); err != nil {
		return fmt.Errorf("delete budget allocation %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) allocations(ctx context.Context, filter sqlcdb.ListAllocationsParams) ([]Allocation, error) {
	rows, err := s.db.Queries(ctx).ListAllocations(ctx, filter)
	return allocationsFromRows(rows, err)
}

func firstAllocation(list []Allocation, err error) (Allocation, bool, error) {
	if err != nil || len(list) == 0 {
		return Allocation{}, false, err
	}
	return list[0], true, nil
}

func allocationsFromRows(rows []sqlcdb.BudgetAllocation, err error) ([]Allocation, error) {
	if err != nil {
		return nil, fmt.Errorf("select budget allocations: %w", err)
	}
	list := make([]Allocation, 0, len(rows))
	for _, row := range rows {
		list = append(list, Allocation{
			ID: row.ID, GlobalBudgetID: row.GlobalBudgetID, PlanID: row.PlanID,
			Amount: money.Money{Amount: money.FromCents(row.AmountMinor), Currency: row.AmountCurrency},
			Meta: record.Meta{
				Version:   row.Version,
				CreatedAt: row.CreatedAt,
				CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
				UpdatedAt: row.UpdatedAt,
				UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
			},
		})
	}
	return list, nil
}

func budgetFromRow(row sqlcdb.GlobalBudget) GlobalBudget {
	return GlobalBudget{
		ID: row.ID, Name: row.Name, Year: row.Year, Amount: money.FromCents(row.AmountMinor), OwnerID: row.OwnerID,
		Description: row.Description,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}
