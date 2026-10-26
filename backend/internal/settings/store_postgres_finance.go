package settings

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/record"
)

func (s *PostgresStore) GetFinanceBudget(ctx context.Context, id string) (FinanceBudget, bool, error) {
	row, err := s.db.Queries(ctx).GetFinanceBudget(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return FinanceBudget{}, false, nil
	}
	if err != nil {
		return FinanceBudget{}, false, fmt.Errorf("select finance budget %s: %w", id, err)
	}
	return financeBudgetFromRow(row), true, nil
}

func (s *PostgresStore) ListFinanceBudgets(ctx context.Context) ([]FinanceBudget, error) {
	rows, err := s.db.Queries(ctx).ListFinanceBudgets(ctx)
	if err != nil {
		return nil, fmt.Errorf("select finance budgets: %w", err)
	}
	budgets := make([]FinanceBudget, 0, len(rows))
	for _, row := range rows {
		budgets = append(budgets, financeBudgetFromRow(row))
	}
	return budgets, nil
}

func (s *PostgresStore) PutFinanceBudget(ctx context.Context, b FinanceBudget) error {
	err := s.db.Queries(ctx).PutFinanceBudget(ctx, sqlcdb.PutFinanceBudgetParams{
		ID: b.ID, Year: b.Year, Name: b.Name, Version: b.Version, CreatedAt: b.CreatedAt,
		CreatedByID: b.CreatedBy.ID, CreatedByName: b.CreatedBy.Name, UpdatedAt: b.UpdatedAt,
		UpdatedByID: b.UpdatedBy.ID, UpdatedByName: b.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save finance budget %s: %w", b.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteFinanceBudget(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteFinanceBudget(ctx, id); err != nil {
		return fmt.Errorf("delete finance budget %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) GetFinanceBudgetItem(ctx context.Context, id string) (FinanceBudgetItem, bool, error) {
	row, err := s.db.Queries(ctx).GetFinanceBudgetItem(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return FinanceBudgetItem{}, false, nil
	}
	if err != nil {
		return FinanceBudgetItem{}, false, fmt.Errorf("select finance budget item %s: %w", id, err)
	}
	return financeBudgetItemFromRow(row), true, nil
}

func (s *PostgresStore) ListFinanceBudgetItems(ctx context.Context) ([]FinanceBudgetItem, error) {
	return s.financeBudgetItems(ctx, nil)
}

func (s *PostgresStore) ItemsOf(ctx context.Context, budgetID string) ([]FinanceBudgetItem, error) {
	return s.financeBudgetItems(ctx, &budgetID)
}

func (s *PostgresStore) PutFinanceBudgetItem(ctx context.Context, i FinanceBudgetItem) error {
	err := s.db.Queries(ctx).PutFinanceBudgetItem(ctx, sqlcdb.PutFinanceBudgetItemParams{
		ID: i.ID, BudgetID: i.BudgetID, Name: i.Name, Code: i.Code, Active: i.Active, Version: i.Version,
		CreatedAt: i.CreatedAt, CreatedByID: i.CreatedBy.ID, CreatedByName: i.CreatedBy.Name,
		UpdatedAt: i.UpdatedAt, UpdatedByID: i.UpdatedBy.ID, UpdatedByName: i.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save finance budget item %s: %w", i.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteFinanceBudgetItem(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteFinanceBudgetItem(ctx, id); err != nil {
		return fmt.Errorf("delete finance budget item %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) GetFieldHelp(ctx context.Context, key string) (FieldHelpText, bool, error) {
	row, err := s.db.Queries(ctx).GetFieldHelp(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return FieldHelpText{}, false, nil
	}
	if err != nil {
		return FieldHelpText{}, false, fmt.Errorf("select field help %s: %w", key, err)
	}
	return fieldHelpFromRow(row), true, nil
}

func (s *PostgresStore) ListFieldHelp(ctx context.Context) ([]FieldHelpText, error) {
	rows, err := s.db.Queries(ctx).ListFieldHelp(ctx)
	if err != nil {
		return nil, fmt.Errorf("select field help: %w", err)
	}
	texts := make([]FieldHelpText, 0, len(rows))
	for _, row := range rows {
		texts = append(texts, fieldHelpFromRow(row))
	}
	return texts, nil
}

func (s *PostgresStore) PutFieldHelp(ctx context.Context, h FieldHelpText) error {
	err := s.db.Queries(ctx).PutFieldHelp(ctx, sqlcdb.PutFieldHelpParams{
		Key: h.Key, Text: h.Text, Version: h.Version, CreatedAt: h.CreatedAt, CreatedByID: h.CreatedBy.ID,
		CreatedByName: h.CreatedBy.Name, UpdatedAt: h.UpdatedAt, UpdatedByID: h.UpdatedBy.ID,
		UpdatedByName: h.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save field help %s: %w", h.Key, err)
	}
	return nil
}

func (s *PostgresStore) DeleteFieldHelp(ctx context.Context, key string) error {
	if err := s.db.Queries(ctx).DeleteFieldHelp(ctx, key); err != nil {
		return fmt.Errorf("delete field help %s: %w", key, err)
	}
	return nil
}

func (s *PostgresStore) financeBudgetItems(ctx context.Context, budgetID *string) ([]FinanceBudgetItem, error) {
	rows, err := s.db.Queries(ctx).ListFinanceBudgetItems(ctx, budgetID)
	if err != nil {
		return nil, fmt.Errorf("select finance budget items: %w", err)
	}
	items := make([]FinanceBudgetItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, financeBudgetItemFromRow(row))
	}
	return items, nil
}

func financeBudgetFromRow(row sqlcdb.FinanceBudget) FinanceBudget {
	return FinanceBudget{
		ID: row.ID, Year: row.Year, Name: row.Name,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}

func financeBudgetItemFromRow(row sqlcdb.FinanceBudgetItem) FinanceBudgetItem {
	return FinanceBudgetItem{
		ID: row.ID, BudgetID: row.BudgetID, Name: row.Name, Code: row.Code, Active: row.Active,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}

func fieldHelpFromRow(row sqlcdb.FieldHelp) FieldHelpText {
	return FieldHelpText{
		Key: row.Key, Text: row.Text,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}
