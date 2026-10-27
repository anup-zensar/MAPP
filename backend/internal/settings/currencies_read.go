package settings

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// ListCurrencies returns every currency, active and inactive, USD first and then by code, each with its
// current rate both ways and where it came from, its usage and its locks (FR-028, US4-2; 016 FR-003). Every
// signed-in person may see them (FR-008).
func (s *Service) ListCurrencies(ctx context.Context, _ actor.Actor) ([]CurrencyView, error) {
	var views []CurrencyView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		currencies, err := s.repo.ListCurrencies(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(currencies, compareCurrencies)
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		book, err := s.rateBook(ctx)
		if err != nil {
			return err
		}
		files, err := s.importFileNames(ctx)
		if err != nil {
			return err
		}
		views = make([]CurrencyView, 0, len(currencies))
		for _, c := range currencies {
			views = append(views, currencyView(c, u, book, s.today(), files))
		}
		return nil
	})
	return views, err
}

// GetCurrency returns the currency with the code, with its usage, its locks and who last changed its rate
// (US4-2).
func (s *Service) GetCurrency(ctx context.Context, _ actor.Actor, code string) (CurrencyView, error) {
	var view CurrencyView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.storedCurrency(ctx, code)
		if err != nil {
			return err
		}
		view, err = s.currencyView(ctx, c)
		return err
	})
	return view, err
}

// storedCurrency reads the currency code as stored, or answers not found.
func (s *Service) storedCurrency(ctx context.Context, code string) (Currency, error) {
	c, found, err := s.repo.GetCurrency(ctx, code)
	if err != nil {
		return Currency{}, err
	}
	if !found {
		return Currency{}, errs.NotFound(currencyGoneMessage)
	}
	return c, nil
}

// currencyView returns the currency c as the API answers it, reading its usage and its rates.
func (s *Service) currencyView(ctx context.Context, c Currency) (CurrencyView, error) {
	u, err := s.usage(ctx)
	if err != nil {
		return CurrencyView{}, err
	}
	book, err := s.rateBook(ctx)
	if err != nil {
		return CurrencyView{}, err
	}
	files, err := s.importFileNames(ctx)
	if err != nil {
		return CurrencyView{}, err
	}
	return currencyView(c, u, book, s.today(), files), nil
}

// currencyView returns the stored currency c as the API answers it on day: its current rate also inverted
// ("1 EUR = 1.086957 USD") with its date and source (016 FR-003), its usage and its locks — the code always,
// and USD's rate, status and delete (FR-028).
func currencyView(c Currency, u *usageIndex, book rateBook, day date.Date, files map[string]string) CurrencyView {
	locks := []Lock{{Field: "code", Reason: codeLockedMessage}}
	if c.Code == money.BaseCurrency {
		for _, field := range []string{"rate", "active", "delete"} {
			locks = append(locks, Lock{Field: field, Reason: usdLockedMessage})
		}
	}
	current := withRate(c, book, day)
	info := rateInfoOf(c, book, day, files)
	rateDate, source := date.Today(current.RateChangedAt), RateManual // USD: since it exists, by hand
	if info.RateDate != nil {
		rateDate, source = *info.RateDate, *info.Source
	}
	return CurrencyView{
		Code: c.Code, Name: c.Name, Symbol: c.Symbol, Rate: current.Rate, InverseRate: current.Rate.Inverse(),
		RateChangedAt: current.RateChangedAt, RateChangedBy: current.RateChangedBy, RateDate: rateDate, RateSource: source,
		RateImportFileName: info.ImportFileName, Active: c.Active, Usage: u.currency(c.Code), Locks: locks, Meta: c.Meta,
	}
}
