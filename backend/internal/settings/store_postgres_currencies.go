package settings

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

func (s *PostgresStore) GetCurrency(ctx context.Context, code string) (Currency, bool, error) {
	row, err := s.db.Queries(ctx).GetCurrency(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return Currency{}, false, nil
	}
	if err != nil {
		return Currency{}, false, fmt.Errorf("select currency %s: %w", code, err)
	}
	return currencyFromRow(row), true, nil
}

func (s *PostgresStore) ListCurrencies(ctx context.Context) ([]Currency, error) {
	rows, err := s.db.Queries(ctx).ListCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("select currencies: %w", err)
	}
	currencies := make([]Currency, 0, len(rows))
	for _, row := range rows {
		currencies = append(currencies, currencyFromRow(row))
	}
	return currencies, nil
}

func (s *PostgresStore) PutCurrency(ctx context.Context, c Currency) error {
	params := sqlcdb.PutCurrencyParams{
		Code: c.Code, Name: c.Name, Symbol: c.Symbol, Active: c.Active, Version: c.Version, CreatedAt: c.CreatedAt,
		CreatedByID: c.CreatedBy.ID, CreatedByName: c.CreatedBy.Name, UpdatedAt: c.UpdatedAt,
		UpdatedByID: c.UpdatedBy.ID, UpdatedByName: c.UpdatedBy.Name,
	}
	if micros := c.Rate.Micros(); micros != 0 {
		params.RateMicros = &micros
	}
	if !c.RateChangedAt.IsZero() {
		params.RateChangedAt = &c.RateChangedAt
	}
	if c.RateChangedBy != (actor.Actor{}) {
		params.RateChangedByID, params.RateChangedByName = &c.RateChangedBy.ID, &c.RateChangedBy.Name
	}
	if err := s.db.Queries(ctx).PutCurrency(ctx, params); err != nil {
		return fmt.Errorf("save currency %s: %w", c.Code, err)
	}
	return nil
}

func (s *PostgresStore) DeleteCurrency(ctx context.Context, code string) error {
	if err := s.db.Queries(ctx).DeleteCurrency(ctx, code); err != nil {
		return fmt.Errorf("delete currency %s: %w", code, err)
	}
	return nil
}

func (s *PostgresStore) GetRate(ctx context.Context, id string) (ExchangeRate, bool, error) {
	row, err := s.db.Queries(ctx).GetRate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExchangeRate{}, false, nil
	}
	if err != nil {
		return ExchangeRate{}, false, fmt.Errorf("select exchange rate %s: %w", id, err)
	}
	return rateFromRow(row), true, nil
}

func (s *PostgresStore) ListRates(ctx context.Context) ([]ExchangeRate, error) {
	rows, err := s.db.Queries(ctx).ListRates(ctx)
	return ratesFromRows(rows, err)
}

func (s *PostgresStore) RatesOf(ctx context.Context, code string) ([]ExchangeRate, error) {
	rows, err := s.db.Queries(ctx).RatesOf(ctx, code)
	return ratesFromRows(rows, err)
}

func (s *PostgresStore) PutRate(ctx context.Context, r ExchangeRate) error {
	err := s.db.Queries(ctx).PutRate(ctx, sqlcdb.PutRateParams{
		ID: r.ID, Currency: r.Currency, Date: r.Date.Time(), RateMicros: r.Rate.Micros(), Source: string(r.Source),
		ImportID: database.NullIfEmpty(r.ImportID), Version: r.Version, CreatedAt: r.CreatedAt,
		CreatedByID: r.CreatedBy.ID, CreatedByName: r.CreatedBy.Name, UpdatedAt: r.UpdatedAt,
		UpdatedByID: r.UpdatedBy.ID, UpdatedByName: r.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save exchange rate %s: %w", r.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteRate(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteRate(ctx, id); err != nil {
		return fmt.Errorf("delete exchange rate %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) PutRateImport(ctx context.Context, i RateImport) error {
	err := s.db.Queries(ctx).PutRateImport(ctx, sqlcdb.PutRateImportParams{
		ID: i.ID, FileName: i.FileName, Feed: i.Feed, FirstDate: database.DateColumn(i.FirstDate), LastDate: database.DateColumn(i.LastDate),
		Added: i.Added, Replaced: i.Replaced, Unchanged: i.Unchanged, NotUsed: i.NotUsed, Problems: i.Problems,
		Version: i.Version, CreatedAt: i.CreatedAt, CreatedByID: i.CreatedBy.ID, CreatedByName: i.CreatedBy.Name,
		UpdatedAt: i.UpdatedAt, UpdatedByID: i.UpdatedBy.ID, UpdatedByName: i.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save rate import %s: %w", i.ID, err)
	}
	return nil
}

func (s *PostgresStore) GetRateImport(ctx context.Context, id string) (RateImport, bool, error) {
	row, err := s.db.Queries(ctx).GetRateImport(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return RateImport{}, false, nil
	}
	if err != nil {
		return RateImport{}, false, fmt.Errorf("select rate import %s: %w", id, err)
	}
	return rateImportFromRow(row), true, nil
}

func (s *PostgresStore) ListRateImports(ctx context.Context) ([]RateImport, error) {
	rows, err := s.db.Queries(ctx).ListRateImports(ctx)
	if err != nil {
		return nil, fmt.Errorf("select rate imports: %w", err)
	}
	imports := make([]RateImport, 0, len(rows))
	for _, row := range rows {
		imports = append(imports, rateImportFromRow(row))
	}
	return imports, nil
}

func currencyFromRow(row sqlcdb.Currency) Currency {
	c := Currency{
		Code: row.Code, Name: row.Name, Symbol: row.Symbol, Active: row.Active,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
	if row.RateMicros != nil {
		c.Rate = money.RateFromMicros(*row.RateMicros)
	}
	if row.RateChangedAt != nil {
		c.RateChangedAt = *row.RateChangedAt
	}
	if row.RateChangedByID != nil && row.RateChangedByName != nil {
		c.RateChangedBy = actor.Actor{ID: *row.RateChangedByID, Name: *row.RateChangedByName}
	}
	return c
}

func ratesFromRows(rows []sqlcdb.ExchangeRate, err error) ([]ExchangeRate, error) {
	if err != nil {
		return nil, fmt.Errorf("select exchange rates: %w", err)
	}
	rates := make([]ExchangeRate, 0, len(rows))
	for _, row := range rows {
		rates = append(rates, rateFromRow(row))
	}
	return rates, nil
}

func rateFromRow(row sqlcdb.ExchangeRate) ExchangeRate {
	return ExchangeRate{
		ID: row.ID, Currency: row.Currency, Date: date.Today(row.Date), Rate: money.RateFromMicros(row.RateMicros),
		Source: RateSource(row.Source), ImportID: database.EmptyIfNull(row.ImportID),
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}

func rateImportFromRow(row sqlcdb.RateImport) RateImport {
	return RateImport{
		ID: row.ID, FileName: row.FileName, Feed: row.Feed, FirstDate: database.DateOf(row.FirstDate),
		LastDate: database.DateOf(row.LastDate), Added: row.Added, Replaced: row.Replaced, Unchanged: row.Unchanged,
		NotUsed: row.NotUsed, Problems: row.Problems,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}
