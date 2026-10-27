package settings

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// CurrencyImpact says, before a change of the currency code is saved, what it would affect (FR-030, US4-3,
// D-044): a new rate recalculates the USD amounts of the live line items in the currency and of the plans
// planned in it (016 FR-005). The body is checked as UpdateCurrency checks it, except the version; nothing is
// saved.
func (s *Service) CurrencyImpact(ctx context.Context, a actor.Actor, code string, update CurrencyUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.storedCurrency(ctx, code)
		if err != nil {
			return err
		}
		currencies, err := s.repo.ListCurrencies(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		_, rate := applyCurrency(stored, update, currencies, stored.Code, &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		if rate == nil || stored.Code == money.BaseCurrency {
			return nil
		}
		current, err := s.currentRateOf(ctx, stored)
		if err != nil {
			return err
		}
		impact.Warnings, err = s.rateWarnings(ctx, stored.Code, current, *rate)
		return err
	})
	return impact, err
}

// rateImpact is what a new rate of a currency changes (FR-030): the live line items in the currency, the
// activities they belong to, and their planned cost in USD with the old and the new rate.
type rateImpact struct {
	lineItems, activities int
	before, after         money.Amount
}

// rateImpactOf works out the rateImpact of changing the rate of the currency code from before to after:
// live line items only — records in the recycle bin are converted when they come back (data model §10).
// Each line item converts on its own, as features 003 and 007 convert it when they read it.
func rateImpactOf(refs []Reference, code string, before, after money.Rate) rateImpact {
	var r rateImpact
	activities := map[string]bool{}
	for _, ref := range refs {
		if ref.InBin || ref.Level != LevelLineItem || ref.Currency != code {
			continue
		}
		r.lineItems++
		activities[ref.ActivityID] = true
		if ref.PlannedCost != nil {
			r.before = r.before.Add(money.ToUSD(*ref.PlannedCost, before))
			r.after = r.after.Add(money.ToUSD(*ref.PlannedCost, after))
		}
	}
	r.activities = len(activities)
	return r
}
