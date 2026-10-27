package settings

import (
	"cmp"
	"context"
	"slices"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

// rateBook is every dated rate, by currency, oldest date first: what the current rate of a day is worked
// out from. It is read once per request.
type rateBook map[string][]ExchangeRate

// rateDay is what the current rate of a currency on a day and its source are read from: every dated rate and
// the file name of every import.
type rateDay struct {
	book  rateBook
	day   date.Date
	files map[string]string
}

// rateBook reads every dated rate inside the caller's transaction.
func (s *Service) rateBook(ctx context.Context) (rateBook, error) {
	all, err := s.repo.ListRates(ctx)
	if err != nil {
		return nil, err
	}
	book := rateBook{}
	for _, r := range all {
		book[r.Currency] = append(book[r.Currency], r)
	}
	for code := range book {
		slices.SortStableFunc(book[code], func(a, b ExchangeRate) int { return cmp.Or(a.Date.Compare(b.Date), compareIDs(a.ID, b.ID)) })
	}
	return book, nil
}

// on returns the rate of code current on day d: the one with the latest date on or before d (FR-002), and
// false when there is none.
func (b rateBook) on(code string, d date.Date) (ExchangeRate, bool) {
	rates := b[code]
	for i := len(rates) - 1; i >= 0; i-- {
		if !rates[i].Date.After(d) {
			return rates[i], true
		}
	}
	return ExchangeRate{}, false
}

// onDate returns the rate of code dated exactly d, and false when there is none.
func (b rateBook) onDate(code string, d date.Date) (ExchangeRate, bool) {
	for _, r := range b[code] {
		if r.Date == d {
			return r, true
		}
	}
	return ExchangeRate{}, false
}

// sorted returns the book with the rates of code in date order again, after one was added.
func (b rateBook) sorted(code string) rateBook {
	slices.SortStableFunc(b[code], func(x, y ExchangeRate) int { return cmp.Or(x.Date.Compare(y.Date), compareIDs(x.ID, y.ID)) })
	return b
}

// today is the API's calendar date (UTC, D-091).
func (s *Service) today() date.Date {
	return date.Today(s.now())
}

// withRate returns c with its current rate on day filled in (FR-002): Rate, and who recorded it and when.
// USD is always 1, recorded when USD was created. A currency without dated rates keeps the rate it was
// stored with before 016 (MigrateRates turns it into its first dated rate at the next start).
func withRate(c Currency, book rateBook, day date.Date) Currency {
	if c.Code == money.BaseCurrency {
		c.Rate = money.One
		if c.RateChangedAt.IsZero() {
			c.RateChangedAt, c.RateChangedBy = c.CreatedAt, c.CreatedBy
		}
		return c
	}
	if r, found := book.on(c.Code, day); found {
		c.Rate, c.RateChangedAt, c.RateChangedBy = r.Rate, r.UpdatedAt, r.UpdatedBy
	}
	return c
}
