package settingstest

import (
	"context"
	"slices"

	"mapp/internal/settings"
)

func (s Store) GetCurrency(_ context.Context, code string) (settings.Currency, bool, error) {
	return s.currencies.Get(code)
}

func (s Store) ListCurrencies(context.Context) ([]settings.Currency, error) {
	return s.currencies.List()
}

func (s Store) PutCurrency(ctx context.Context, c settings.Currency) error {
	return s.currencies.Put(ctx, c)
}

func (s Store) DeleteCurrency(ctx context.Context, code string) error {
	return s.currencies.Delete(ctx, code)
}

func (s Store) GetRate(_ context.Context, id string) (settings.ExchangeRate, bool, error) {
	return s.rates.Get(id)
}

func (s Store) ListRates(context.Context) ([]settings.ExchangeRate, error) {
	return s.rates.List()
}

// RatesOf lists oldest date first; the stable sort keeps Where's id order within a date, as the SQL does.
func (s Store) RatesOf(_ context.Context, code string) ([]settings.ExchangeRate, error) {
	rates, err := s.rates.Where(func(r settings.ExchangeRate) bool { return r.Currency == code })
	slices.SortStableFunc(rates, func(a, b settings.ExchangeRate) int { return a.Date.Compare(b.Date) })
	return rates, err
}

func (s Store) PutRate(ctx context.Context, r settings.ExchangeRate) error {
	return s.rates.Put(ctx, r)
}

func (s Store) PutRates(ctx context.Context, rates []settings.ExchangeRate) error {
	return s.rates.Put(ctx, rates...)
}

func (s Store) DeleteRate(ctx context.Context, id string) error {
	return s.rates.Delete(ctx, id)
}

func (s Store) PutRateImport(ctx context.Context, i settings.RateImport) error {
	return s.imports.Put(ctx, i)
}

func (s Store) GetRateImport(_ context.Context, id string) (settings.RateImport, bool, error) {
	return s.imports.Get(id)
}

func (s Store) ListRateImports(context.Context) ([]settings.RateImport, error) {
	return s.imports.List()
}
