package settings

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// AddRateImpact says, before a new dated rate of the currency code is saved, what it would change (FR-005,
// D-044): when it becomes the current rate, the line items in the currency and the plans planned in it. A
// future-dated rate changes nothing before its date, so it answers no warning. It is checked as AddRate
// checks it; nothing is saved.
func (s *Service) AddRateImpact(ctx context.Context, _ actor.Actor, code string, in RateInput) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.storedCurrency(ctx, code)
		if err != nil {
			return err
		}
		book, err := s.rateBook(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		day, rate := checkRateInput(c, in, book, &problems)
		if err = problems.Err(); err != nil {
			return err
		}
		today := s.today()
		before := withRate(c, book, today).Rate
		book[c.Code] = append(slices.Clone(book[c.Code]), ExchangeRate{ID: "new", Currency: c.Code, Date: day, Rate: rate})
		after := withRate(c, book.sorted(c.Code), today).Rate
		impact.Warnings, err = s.rateWarnings(ctx, c.Code, before, after)
		return err
	})
	return impact, err
}

// UpdateRateImpact says, before a changed rate is saved, what it would change: only a change of the current
// rate changes conversions (FR-005). Nothing is saved.
func (s *Service) UpdateRateImpact(ctx context.Context, _ actor.Actor, id string, update RateUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.storedRate(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		rate := checkRate(update.Rate, &problems)
		if err = problems.Err(); err != nil {
			return err
		}
		c, err := s.storedCurrency(ctx, stored.Currency)
		if err != nil {
			return err
		}
		book, err := s.rateBook(ctx)
		if err != nil {
			return err
		}
		current, found := book.on(c.Code, s.today())
		if !found || current.ID != stored.ID {
			return nil // not the rate in use: nothing converts differently
		}
		impact.Warnings, err = s.rateWarnings(ctx, c.Code, stored.Rate, rate)
		return err
	})
	return impact, err
}

// rateWarnings states what a new current rate of the currency code changes (002 FR-030, 016 FR-005): the
// line items in it and the plans planned in it, with their USD values before and after. None when the rate
// stays the same.
func (s *Service) rateWarnings(ctx context.Context, code string, before, after money.Rate) ([]ImpactWarning, error) {
	warnings := []ImpactWarning{}
	if before == after {
		return warnings, nil
	}
	refs, err := s.references(ctx)
	if err != nil {
		return nil, err
	}
	if r := rateImpactOf(refs, code, before, after); r.lineItems > 0 {
		verb := "use"
		if r.lineItems == 1 {
			verb = "uses"
		}
		warnings = append(warnings, ImpactWarning{
			Kind: ImpactAmountsRecalculated, Count: r.lineItems,
			Message: fmt.Sprintf(amountsRecalculated, plural(r.lineItems, "line item", "line items"), plural(r.activities, "activity", "activities"),
				verb, code, code, r.before.Format(), r.after.Format()),
		})
	}
	if p := planImpactOf(refs, code, before, after); p.plans > 0 {
		subject, possessive := "Their", "their"
		if p.plans == 1 {
			subject, possessive = "Its", "its"
		}
		verb := "are"
		if p.plans == 1 {
			verb = "is"
		}
		warnings = append(warnings, ImpactWarning{
			Kind: ImpactAmountsRecalculated, Count: p.plans,
			Message: fmt.Sprintf(plansRecalculated, plural(p.plans, "plan", "plans")+" "+verb, code, subject, code, possessive,
				p.before.Format(), p.after.Format()),
		})
	}
	return warnings, nil
}

// recalculationDetails is what a new current rate recalculated, for the details of its log entry: the line
// items and the plans in the currency; "" when the current rate did not change or nothing uses it.
func (s *Service) recalculationDetails(ctx context.Context, code string, before, after money.Rate) (string, error) {
	if before == after {
		return "", nil
	}
	refs, err := s.references(ctx)
	if err != nil {
		return "", err
	}
	var sentences []string
	if r := rateImpactOf(refs, code, before, after); r.lineItems > 0 {
		sentences = append(sentences, fmt.Sprintf(recalculatedDetails, plural(r.lineItems, "line item", "line items"),
			plural(r.activities, "activity", "activities"), r.before.Format(), r.after.Format()))
	}
	if p := planImpactOf(refs, code, before, after); p.plans > 0 {
		sentences = append(sentences, fmt.Sprintf(plansRecalculatedDetail, plural(p.plans, "plan", "plans"), code,
			p.before.Format(), p.after.Format()))
	}
	return strings.Join(sentences, " "), nil
}

// planImpact is what a new rate of a currency changes for the plans planned in it (FR-005): how many live
// plans, and their budgets in USD with the old and the new rate, each plan converted once.
type planImpact struct {
	plans         int
	before, after money.Amount
}

// planImpactOf works out the planImpact of changing the rate of code from before to after.
func planImpactOf(refs []Reference, code string, before, after money.Rate) planImpact {
	var p planImpact
	for _, ref := range refs {
		if ref.InBin || ref.Level != LevelPlan || ref.Currency != code {
			continue
		}
		p.plans++
		if ref.Budget != nil {
			p.before = p.before.Add(money.ToUSD(*ref.Budget, before))
			p.after = p.after.Add(money.ToUSD(*ref.Budget, after))
		}
	}
	return p
}
