package settings

import (
	"cmp"
	"context"
	"slices"

	"mapp/internal/platform/store"
)

// The JSON store tables of feature 016 (016 data-model §1).
const (
	exchangeRatesTable = "exchange_rates"
	rateImportsTable   = "rate_imports"
	fieldHelpTable     = "field_help"
)

// GetRate reads one dated exchange rate.
func (JSONStore) GetRate(ctx context.Context, id string) (ExchangeRate, bool, error) {
	return store.Get[ExchangeRate](ctx, exchangeRatesTable, id)
}

// ListRates reads the whole exchange_rates table.
func (JSONStore) ListRates(ctx context.Context) ([]ExchangeRate, error) {
	return store.List[ExchangeRate](ctx, exchangeRatesTable)
}

// RatesOf reads every rate to find those of one currency, oldest date first: a few hundred rows per
// currency and year. PostgreSQL would use an index on (currency, date).
func (r JSONStore) RatesOf(ctx context.Context, code string) ([]ExchangeRate, error) {
	all, err := r.ListRates(ctx)
	if err != nil {
		return nil, err
	}
	rates := filter(all, func(x ExchangeRate) bool { return x.Currency == code })
	slices.SortStableFunc(rates, func(a, b ExchangeRate) int { return cmp.Or(a.Date.Compare(b.Date), compareIDs(a.ID, b.ID)) })
	return rates, nil
}

// PutRate saves a rate under its id.
func (JSONStore) PutRate(ctx context.Context, r ExchangeRate) error {
	return store.Put(ctx, exchangeRatesTable, r.ID, r)
}

// DeleteRate removes one rate.
func (JSONStore) DeleteRate(ctx context.Context, id string) error {
	return store.Delete(ctx, exchangeRatesTable, id)
}

// PutRateImport saves a rate import under its id.
func (JSONStore) PutRateImport(ctx context.Context, i RateImport) error {
	return store.Put(ctx, rateImportsTable, i.ID, i)
}

// GetRateImport reads one rate import.
func (JSONStore) GetRateImport(ctx context.Context, id string) (RateImport, bool, error) {
	return store.Get[RateImport](ctx, rateImportsTable, id)
}

// ListRateImports reads the whole rate_imports table, oldest first (ids only go up).
func (JSONStore) ListRateImports(ctx context.Context) ([]RateImport, error) {
	return store.List[RateImport](ctx, rateImportsTable)
}

// GetFieldHelp reads the edited help text of one key.
func (JSONStore) GetFieldHelp(ctx context.Context, key string) (FieldHelpText, bool, error) {
	return store.Get[FieldHelpText](ctx, fieldHelpTable, key)
}

// ListFieldHelp reads the whole field_help table.
func (JSONStore) ListFieldHelp(ctx context.Context) ([]FieldHelpText, error) {
	return store.List[FieldHelpText](ctx, fieldHelpTable)
}

// PutFieldHelp saves an edited help text under its key.
func (JSONStore) PutFieldHelp(ctx context.Context, h FieldHelpText) error {
	return store.Put(ctx, fieldHelpTable, h.Key, h)
}

// DeleteFieldHelp removes the edited help text of one key.
func (JSONStore) DeleteFieldHelp(ctx context.Context, key string) error {
	return store.Delete(ctx, fieldHelpTable, key)
}
