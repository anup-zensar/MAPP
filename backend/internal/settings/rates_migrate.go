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
			if c.Code != money.BaseCurrency && len(book[c.Code]) == 0 && c.Rate.Micros() > 0 {
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
				if err := s.repo.PutRate(ctx, first); err != nil {
					return err
				}
			}
			if hasLegacy(c) {
				if err := s.repo.PutCurrency(ctx, withoutLegacy(c)); err != nil {
					return err
				}
			}
		}
		return nil
	})
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
