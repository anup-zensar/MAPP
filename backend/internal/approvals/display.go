package approvals

import (
	"context"

	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/money"
)

// The amounts the Approvals page shows — a row's planned cost, the preview's planned cost, line items and
// unassigned planned cost — and the submit dialog's unassigned planned cost are answered in the reader's
// display currency (008 FR-030, 007 FR-005, D-061, D-213): worked out in USD, then each converted once at
// the display currency's current rate, half up to the cent. Inbox itself answers USD, because Home (009)
// adds those amounts up before it converts.

// DisplayRates gives the display currency a person shows amounts in, with its rate in units per 1 USD
// (feature 003). internal/app hands people's service to UseDisplayRates; without it amounts stay in USD.
type DisplayRates interface {
	DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error)
}

// UseDisplayRates sets where the display currency of the reader is read (D-213).
func (s *Service) UseDisplayRates(rates DisplayRates) {
	s.displayRates = rates
}

// display converts USD amounts into one display currency.
type display struct {
	currency string
	rate     money.Rate
}

// inUSD shows amounts as they are, in US dollars.
var inUSD = display{currency: money.BaseCurrency, rate: money.One}

// displayOf returns the display currency of the person a, with its current rate; without display rates,
// or with an unusable rate, amounts stay in USD.
func (s *Service) displayOf(ctx context.Context, a actor.Actor) (display, error) {
	if s.displayRates == nil {
		return inUSD, nil
	}
	d, err := s.displayRates.DisplayRateOf(ctx, a.ID)
	if err != nil {
		return display{}, err
	}
	rate, err := money.ParseRate(d.Rate)
	if err != nil || d.Code == "" || d.Code == money.BaseCurrency {
		return inUSD, nil
	}
	return display{currency: d.Code, rate: rate}, nil
}

// optional converts usd, an amount in US dollars, or gives nil when there is no amount.
func (d display) optional(usd *money.Money) *money.Money {
	if usd == nil || d.currency == money.BaseCurrency {
		return usd
	}
	converted := money.Money{Amount: money.FromUSD(usd.Amount, d.rate), Currency: d.currency}
	return &converted
}

// Rows returns the rows of the tab as the Approvals page shows them to a (FR-029, FR-030): Inbox's rows
// with their planned cost in a's display currency (D-213).
func (s *Service) Rows(ctx context.Context, a actor.Actor, tab string) ([]Row, error) {
	rows, err := s.Inbox(ctx, a, tab)
	if err != nil {
		return nil, err
	}
	var d display
	err = s.tx.Read(ctx, func(ctx context.Context) error {
		d, err = s.displayOf(ctx, a)
		return err
	})
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].PlannedCost = d.optional(rows[i].PlannedCost)
	}
	return rows, nil
}
