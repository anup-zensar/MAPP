package settings

import (
	"context"
	"fmt"

	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
)

// PutRates saves rates, replacing those with the same id, in two statements whatever their number: a stored
// rate is deleted and written again, so the date it keeps never clashes with another rate of its currency.
func (s *PostgresStore) PutRates(ctx context.Context, rates []ExchangeRate) error {
	if len(rates) == 0 {
		return nil
	}
	ids := make([]string, 0, len(rates))
	rows := make([]sqlcdb.AddRatesParams, 0, len(rates))
	for _, r := range rates {
		ids = append(ids, r.ID)
		rows = append(rows, sqlcdb.AddRatesParams(rateParams(r)))
	}
	q := s.change(ctx)
	if err := q.DeleteRates(ctx, ids); err != nil {
		return fmt.Errorf("replace exchange rates: %w", err)
	}
	if _, err := q.AddRates(ctx, rows); err != nil {
		return fmt.Errorf("save exchange rates: %w", err)
	}
	return nil
}

// AddOptions stores new options with their parent options in two statements whatever their number.
func (s *PostgresStore) AddOptions(ctx context.Context, options []Option) error {
	if len(options) == 0 {
		return nil
	}
	rows := make([]sqlcdb.AddOptionsParams, 0, len(options))
	var parents []sqlcdb.AddOptionParentsParams
	for _, o := range options {
		rows = append(rows, sqlcdb.AddOptionsParams(optionParams(o)))
		for i, parentID := range o.ParentOptionIDs {
			parents = append(parents, sqlcdb.AddOptionParentsParams{OptionID: o.ID, Position: i + 1, ParentOptionID: parentID})
		}
	}
	q := s.change(ctx)
	if _, err := q.AddOptions(ctx, rows); err != nil {
		return fmt.Errorf("add options: %w", err)
	}
	if len(parents) == 0 {
		return nil
	}
	if _, err := q.AddOptionParents(ctx, parents); err != nil {
		return fmt.Errorf("add option parents: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteOptions(ctx context.Context, ids []string) error {
	if err := s.change(ctx).DeleteOptions(ctx, ids); err != nil {
		return fmt.Errorf("delete options: %w", err)
	}
	return nil
}

func rateParams(r ExchangeRate) sqlcdb.PutRateParams {
	return sqlcdb.PutRateParams{
		ID: r.ID, Currency: r.Currency, Date: r.Date.Time(), RateMicros: r.Rate.Micros(), Source: string(r.Source),
		ImportID: database.NullIfEmpty(r.ImportID), Version: r.Version, CreatedAt: r.CreatedAt,
		CreatedByID: r.CreatedBy.ID, CreatedByName: r.CreatedBy.Name, UpdatedAt: r.UpdatedAt,
		UpdatedByID: r.UpdatedBy.ID, UpdatedByName: r.UpdatedBy.Name,
	}
}

func optionParams(o Option) sqlcdb.PutOptionParams {
	return sqlcdb.PutOptionParams{
		ID: o.ID, AttributeID: o.AttributeID, Label: o.Label, Position: o.Position, Active: o.Active,
		Version: o.Version, CreatedAt: o.CreatedAt, CreatedByID: o.CreatedBy.ID, CreatedByName: o.CreatedBy.Name,
		UpdatedAt: o.UpdatedAt, UpdatedByID: o.UpdatedBy.ID, UpdatedByName: o.UpdatedBy.Name,
	}
}
