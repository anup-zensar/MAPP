package approval

import (
	"context"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

type DisplayRates interface {
	DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error)
}

func (s *Service) UseDisplayRates(rates DisplayRates) {
	s.displayRates = rates
}

type display struct {
	currency string
	rate     money.Rate
}

var inUSD = display{currency: money.BaseCurrency, rate: money.One}

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

func (d display) optional(usd *money.Money) *money.Money {
	if usd == nil || d.currency == money.BaseCurrency {
		return usd
	}
	converted := money.Money{Amount: money.FromUSD(usd.Amount, d.rate), Currency: d.currency}
	return &converted
}

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
