package settings

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// PostgresStore is the Store on PostgreSQL: models and activity types here, the other records in the
// store_postgres_*.go files. Lists a record owns are child tables and read back as empty lists, never nil.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) GetModel(ctx context.Context, id string) (Model, bool, error) {
	models, err := s.models(ctx)
	row, found := models.Find(id)
	if err != nil || !found {
		return Model{}, false, err
	}
	return modelFromRow(row.Model, row.ActivityTypeIds), true, nil
}

func (s *PostgresStore) ListModels(ctx context.Context) ([]Model, error) {
	rows, err := s.models(ctx)
	if err != nil {
		return nil, err
	}
	models := make([]Model, 0, len(rows.Rows))
	for _, row := range rows.Rows {
		models = append(models, modelFromRow(row.Model, row.ActivityTypeIds))
	}
	return models, nil
}

func (s *PostgresStore) PutModel(ctx context.Context, m Model) error {
	q := s.change(ctx)
	err := q.PutModel(ctx, sqlcdb.PutModelParams{
		ID: m.ID, Name: m.Name, InqToMqlRateHundredths: m.InqToMqlRate.Hundredths(),
		MqlToDealRateHundredths: m.MqlToDealRate.Hundredths(), AverageDealSizeMinor: m.AverageDealSize.Amount.Cents(),
		AverageDealSizeCurrency: m.AverageDealSize.Currency, Active: m.Active, Version: m.Version,
		CreatedAt: m.CreatedAt, CreatedByID: m.CreatedBy.ID, CreatedByName: m.CreatedBy.Name,
		UpdatedAt: m.UpdatedAt, UpdatedByID: m.UpdatedBy.ID, UpdatedByName: m.UpdatedBy.Name,
	})
	if err == nil {
		err = q.DeleteModelActivityTypes(ctx, m.ID)
	}
	if err == nil {
		err = q.InsertModelActivityTypes(ctx, sqlcdb.InsertModelActivityTypesParams{ModelID: m.ID, ActivityTypeIds: m.ActivityTypeIDs})
	}
	if err != nil {
		return fmt.Errorf("save model %s: %w", m.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteModel(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteModel(ctx, id); err != nil {
		return fmt.Errorf("delete model %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) GetActivityType(ctx context.Context, id string) (ActivityType, bool, error) {
	types, err := s.activityTypes(ctx)
	row, found := types.Find(id)
	if err != nil || !found {
		return ActivityType{}, false, err
	}
	return activityTypeFromRow(row), true, nil
}

func (s *PostgresStore) ListActivityTypes(ctx context.Context) ([]ActivityType, error) {
	rows, err := s.activityTypes(ctx)
	if err != nil {
		return nil, err
	}
	types := make([]ActivityType, 0, len(rows.Rows))
	for _, row := range rows.Rows {
		types = append(types, activityTypeFromRow(row))
	}
	return types, nil
}

func (s *PostgresStore) PutActivityType(ctx context.Context, t ActivityType) error {
	params := sqlcdb.PutActivityTypeParams{
		ID: t.ID, Name: t.Name, Active: t.Active, Version: t.Version, CreatedAt: t.CreatedAt,
		CreatedByID: t.CreatedBy.ID, CreatedByName: t.CreatedBy.Name, UpdatedAt: t.UpdatedAt,
		UpdatedByID: t.UpdatedBy.ID, UpdatedByName: t.UpdatedBy.Name,
	}
	if cost := t.DefaultPlannedCost; cost != nil {
		cents := cost.Amount.Cents()
		params.DefaultPlannedCostMinor, params.DefaultPlannedCostCurrency = &cents, &cost.Currency
	}
	if err := s.change(ctx).PutActivityType(ctx, params); err != nil {
		return fmt.Errorf("save activity type %s: %w", t.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteActivityType(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteActivityType(ctx, id); err != nil {
		return fmt.Errorf("delete activity type %s: %w", id, err)
	}
	return nil
}

func modelFromRow(row sqlcdb.Model, activityTypeIDs []string) Model {
	return Model{
		ID: row.ID, Name: row.Name, InqToMqlRate: money.PercentFromHundredths(row.InqToMqlRateHundredths),
		MqlToDealRate: money.PercentFromHundredths(row.MqlToDealRateHundredths),
		AverageDealSize: money.Money{Amount: money.FromCents(row.AverageDealSizeMinor),
			Currency: row.AverageDealSizeCurrency},
		ActivityTypeIDs: slices.Clone(activityTypeIDs), Active: row.Active,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}

func activityTypeFromRow(row sqlcdb.ActivityType) ActivityType {
	t := ActivityType{
		ID: row.ID, Name: row.Name, Active: row.Active,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
	if row.DefaultPlannedCostMinor != nil && row.DefaultPlannedCostCurrency != nil {
		t.DefaultPlannedCost = &money.Money{Amount: money.FromCents(*row.DefaultPlannedCostMinor),
			Currency: *row.DefaultPlannedCostCurrency}
	}
	return t
}
