package plans

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// The plan currency and the plan type (feature 016, FR-011 – FR-014, FR-032; D-260, D-261, D-264, D-268). A
// plan's budget, phasing, and the budgets and planned costs of everything in it are typed and kept in its plan
// currency; the API answers them in the plan currency and, beside them, in the reader's display currency.

// The messages of the plan currency and the plan type (016 plan "User-visible messages").
const (
	msgAmountInPlanCurrency = "Enter the amount in %s, the plan's currency."
	msgChooseCurrency       = "Choose a currency from the list."
	msgInactiveCurrency     = "'%s' is inactive — choose an active currency."
	msgChoosePlanType       = "Choose Operational, Strategic or Cross-market."
	msgCurrencyLocked       = "The plan currency can't change once campaigns, initiatives or activities have amounts in it (%s)."
)

// checkCurrency checks the plan currency a form sends (FR-011): an active currency, or the one the plan
// already has even when it became inactive. Not sent keeps the plan's currency, or USD for a new plan.
func (s *Service) checkCurrency(ctx context.Context, sent *string, stored *Plan, problems *errs.Problems) (string, error) {
	keep := money.BaseCurrency
	if stored != nil {
		keep = stored.Currency
	}
	code := strings.ToUpper(text(sent))
	if code == "" || code == keep {
		return keep, nil
	}
	c, found, err := s.settings.Currency(ctx, code)
	switch {
	case err != nil:
		return "", err
	case !found:
		problems.Add("currency", msgChooseCurrency)
		return keep, nil
	case !c.Active:
		problems.Add("currency", fmt.Sprintf(msgInactiveCurrency, c.Code))
		return keep, nil
	}
	return c.Code, nil
}

// checkPlanType checks the plan type a form sends (FR-032): one of the three types. Not sent keeps the plan's
// type, or Operational for a new plan.
func checkPlanType(sent *string, stored *Plan, problems *errs.Problems) PlanType {
	keep := TypeOperational
	if stored != nil {
		keep = stored.PlanType
	}
	t := PlanType(text(sent))
	switch {
	case t == "":
		return keep
	case !slices.Contains(planTypes, t):
		problems.Add("planType", msgChoosePlanType)
		return keep
	}
	return t
}

// currencyLock says why the plan currency can't change (FR-012): once campaigns, initiatives or activities
// have amounts in it, "… (4 amounts)". nil when it can.
func currencyLock(f Facts) *string {
	if f.AmountsSet == 0 {
		return nil
	}
	reason := fmt.Sprintf(msgCurrencyLocked, amountsText(f.AmountsSet))
	return &reason
}

// amountsText writes a number of amounts: "1 amount", "4 amounts".
func amountsText(n int) string {
	if n == 1 {
		return "1 amount"
	}
	return fmt.Sprintf("%d amounts", n)
}

// usdOf converts an amount of a plan planned in currency to USD with the current rates (FR-014): once, half
// up. A currency without a rate counts as USD, so a total is never lost.
func usdOf(amount money.Amount, currency string, rates money.Rates) money.Amount {
	r, err := rates.Of(currency)
	if err != nil {
		return amount
	}
	return money.Convert(amount, r, money.One)
}
