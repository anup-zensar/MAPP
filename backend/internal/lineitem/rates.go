package lineitem

import (
	"context"

	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// rateTable is every currency with its current rate, and the date and source of that rate (016 FR-002,
// FR-009), read once for all the line items an answer shows: the rates are dated records, so reading them
// line item by line item would read the whole rate history each time.
type rateTable struct {
	currencies map[string]settings.Currency
	infos      map[string]settings.RateInfo
}

// rateTableKey is the context key of a rateTable read for the request.
type rateTableKey struct{}

// withRates returns ctx carrying the current rates, read once now, for every view built with it.
func (s *Service) withRates(ctx context.Context) (context.Context, error) {
	if _, found := ctx.Value(rateTableKey{}).(rateTable); found {
		return ctx, nil
	}
	table, err := s.readRates(ctx)
	if err != nil {
		return nil, err
	}
	return context.WithValue(ctx, rateTableKey{}, table), nil
}

// rates returns the rates ctx carries, or reads them.
func (s *Service) rates(ctx context.Context) (rateTable, error) {
	if table, found := ctx.Value(rateTableKey{}).(rateTable); found {
		return table, nil
	}
	return s.readRates(ctx)
}

// readRates reads every currency with its current rate and where the rate came from.
func (s *Service) readRates(ctx context.Context) (rateTable, error) {
	all, err := s.deps.Currencies.Currencies(ctx)
	if err != nil {
		return rateTable{}, err
	}
	infos, err := s.deps.Currencies.CurrentRates(ctx)
	if err != nil {
		return rateTable{}, err
	}
	table := rateTable{currencies: make(map[string]settings.Currency, len(all)), infos: make(map[string]settings.RateInfo, len(infos))}
	for _, c := range all {
		table.currencies[c.Code] = c
	}
	for _, info := range infos {
		table.infos[info.Currency] = info
	}
	return table, nil
}

// rateOf returns the current rate of code: 1 for USD and for a currency Settings does not know (a plan
// currency is never deleted while a plan uses it, 016 FR-010).
func (t rateTable) rateOf(code string) money.Rate {
	if c, found := t.currencies[code]; found && c.Rate.Micros() > 0 {
		return c.Rate
	}
	return money.One
}

// view returns a currency as the line-item screens show it, with the date and source of its current rate.
func (t rateTable) view(c settings.Currency) CurrencyView {
	view := currencyView(c)
	if info, found := t.infos[c.Code]; found {
		view.RateDate, view.RateSource = info.RateDate, info.Source
	}
	return view
}
