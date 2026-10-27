package settings

import (
	"context"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// MigrateRates turns the one stored rate of every currency of before 016 into its first dated rate, once
// (data-model §2.2): for every currency other than USD without a dated rate, a rate dated on the UTC day the
// stored rate was last changed, source "Entered by hand", recorded by whoever changed it then; then it writes
// the currency back without the legacy fields. It writes no activity-log entry and changes nothing on a
// second start. internal/app calls it once the store is open.
func (s *Service) MigrateRates(ctx context.Context) error {
	return s.tx.Write(ctx, func(ctx context.Context) error {
		currencies, err := s.repo.ListCurrencies(ctx)
		if err != nil {
			return err
		}
		book, err := s.rateBook(ctx)
		if err != nil {
			return err
		}
		for _, c := range currencies {
			if err = s.migrateCurrency(ctx, c, book); err != nil {
				return err
			}
		}
		return nil
	})
}

// migrateCurrency gives c its first dated rate when it has none yet, and stores it without the legacy fields.
func (s *Service) migrateCurrency(ctx context.Context, c Currency, book rateBook) error {
	if c.Code != money.BaseCurrency && len(book[c.Code]) == 0 && c.Rate.Micros() > 0 {
		if err := s.putFirstRate(ctx, c); err != nil {
			return err
		}
	}
	if hasLegacy(c) {
		return s.repo.PutCurrency(ctx, withoutLegacy(c))
	}
	return nil
}

// putFirstRate stores the legacy rate of c as its first dated rate, recorded by whoever last changed it.
func (s *Service) putFirstRate(ctx context.Context, c Currency) error {
	changedAt, changedBy := c.RateChangedAt, c.RateChangedBy
	if changedAt.IsZero() {
		changedAt, changedBy = c.CreatedAt, c.CreatedBy
	}
	id, err := s.ids.Next(ctx, ids.PrefixExchangeRate)
	if err != nil {
		return err
	}
	first := ExchangeRate{ID: id, Currency: c.Code, Date: date.Today(changedAt), Rate: c.Rate, Source: RateManual,
		Meta: record.NewMeta(changedBy, changedAt)}
	return s.repo.PutRate(ctx, first)
}

// hasLegacy reports whether c is stored with one of the legacy rate fields.
func hasLegacy(c Currency) bool {
	return c.Rate != (money.Rate{}) || !c.RateChangedAt.IsZero() || c.RateChangedBy != (actor.Actor{})
}

// withoutLegacy returns c without the legacy rate fields, as it is stored since 016.
func withoutLegacy(c Currency) Currency {
	c.Rate, c.RateChangedAt, c.RateChangedBy = money.Rate{}, time.Time{}, actor.Actor{}
	return c
}
