package settings

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// The limits of a currency (data model §6).
const (
	maxCurrencyName = 80
	maxSymbol       = 5
)

// Messages of the plan's "User-visible messages" table about currencies.
const (
	currencyGoneMessage = "This currency no longer exists — someone may have deleted it."
	codeLockedMessage   = "The code can't change once the currency is saved."
	usdLockedMessage    = "USD is SP&A One's base currency; its rate is always 1."
	codeMessage         = "Enter a three-letter code such as JPY."
	codeTakenMessage    = "A currency with the code '%s' already exists."
	currencyNameTaken   = "A currency called '%s' already exists."
	rateMessage         = "Enter a rate above 0, with at most six decimals."
	rateTooLargeMessage = "Use a rate of 1,000,000 or less."
	amountsRecalculated = "%s in %s %s %s. Their %s amounts stay the same; their USD amounts will be recalculated (%s → %s USD). Unassigned planned cost of these activities will change. No activity goes back for approval."
	recalculatedDetails = "%s in %s recalculated: %s → %s USD."
)

// CreateCurrency creates an active currency (FR-028, US4-1): a code of three letters A–Z, stored in capitals,
// that no other currency has; a name of at most 80 characters that no other currency has; a symbol of at
// most 5 characters; and a rate in units per 1 USD above 0 with at most six decimals, at most 1,000,000. Every
// problem is reported at once (US4-5). The rate becomes the currency's first dated rate, dated today and
// entered by hand by a (016 FR-003).
func (s *Service) CreateCurrency(ctx context.Context, a actor.Actor, input CurrencyInput) (CurrencyView, error) {
	var view CurrencyView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		currencies, err := s.repo.ListCurrencies(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		c := Currency{Code: strings.ToUpper(strings.TrimSpace(input.Code))}
		if !isCurrencyCode(c.Code) {
			problems.Add("code", codeMessage)
		} else if slices.ContainsFunc(currencies, func(other Currency) bool { return other.Code == c.Code }) {
			problems.Add("code", fmt.Sprintf(codeTakenMessage, c.Code))
		}
		c, rate := applyCurrency(c, CurrencyUpdate{Name: &input.Name, Symbol: &input.Symbol, Rate: &input.Rate}, currencies, "", &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		c.Active, c.Meta = true, record.NewMeta(a, s.now())
		if err := s.repo.PutCurrency(ctx, c); err != nil {
			return err
		}
		if c.Code != money.BaseCurrency && rate != nil {
			if _, err := s.putNewRate(ctx, a, c.Code, s.today(), *rate, RateManual, ""); err != nil {
				return err
			}
		}
		if err := s.logCreated(ctx, a, currencyKind, c.Code, c.Code, ""); err != nil {
			return err
		}
		view, err = s.currencyView(ctx, c)
		return err
	})
	return view, err
}

// UpdateCurrency changes the name, symbol or rate of the currency code (FR-028 – FR-030, US4-3), from the
// version the person started at (FR-007). The code never changes; USD's rate stays 1. A new rate sets
// today's dated rate (016 FR-004: today's rate is added or replaced, entered by hand by a), applies at once
// everywhere (FR-029), and its log entry's details say how many live line items in how many activities and
// how many plans planned in the currency were recalculated, with their USD totals before and after. A save
// that changes nothing writes no version and no log entry.
func (s *Service) UpdateCurrency(ctx context.Context, a actor.Actor, code string, update CurrencyUpdate) (CurrencyView, error) {
	var view CurrencyView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.storedCurrency(ctx, code)
		if err != nil {
			return err
		}
		currencies, err := s.repo.ListCurrencies(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		checkVersion(&problems, update.Version)
		changed, rate := applyCurrency(stored, update, currencies, stored.Code, &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		if err := stored.CheckVersion(*update.Version, currencyNoun); err != nil {
			return err
		}
		current, err := s.currentRateOf(ctx, stored)
		if err != nil {
			return err
		}
		rateChanged := rate != nil && *rate != current && stored.Code != money.BaseCurrency
		changes, err := history.Diff(stored, changed)
		if err != nil {
			return err
		}
		if len(changes) > 0 || rateChanged {
			changed.Touch(a, s.now()) // the currency form was saved: its version goes up
			if err := s.repo.PutCurrency(ctx, changed); err != nil {
				return err
			}
			stored = changed
		}
		if len(changes) > 0 {
			if err := s.logChanged(ctx, a, currencyKind, changed.Code, changed.Code, changes, ""); err != nil {
				return err
			}
		}
		if rateChanged {
			if err := s.setTodaysRate(ctx, a, stored, *rate, current); err != nil {
				return err
			}
		}
		view, err = s.currencyView(ctx, stored)
		return err
	})
	return view, err
}

// SetCurrencyActive sets the currency code inactive or reactivates it (FR-004): line items and people
// cannot choose an inactive currency; those using it keep it, and amounts still convert. USD stays active
// (FR-028). The same status again changes nothing.
func (s *Service) SetCurrencyActive(ctx context.Context, a actor.Actor, code string, update StatusUpdate) (CurrencyView, error) {
	active, version, err := checkStatus(update)
	if err != nil {
		return CurrencyView{}, err
	}
	var view CurrencyView
	err = s.tx.Write(ctx, func(ctx context.Context) error {
		c, err := s.storedCurrency(ctx, code)
		if err != nil {
			return err
		}
		if err := c.CheckVersion(version, currencyNoun); err != nil {
			return err
		}
		if c.Active != active {
			if c.Code == money.BaseCurrency {
				return errs.Validation(errs.FieldError{Field: "active", Message: usdLockedMessage})
			}
			c.Active = active
			c.Touch(a, s.now())
			if err := s.repo.PutCurrency(ctx, c); err != nil {
				return err
			}
			if err := s.logStatus(ctx, a, currencyKind, c.Code, c.Code, active); err != nil {
				return err
			}
		}
		view, err = s.currencyView(ctx, c)
		return err
	})
	return view, err
}

// DeleteCurrency moves the currency code to the recycle bin with its rate history, so that it can be undone
// (FR-004, US4-2). USD is refused with its lock's reason, and a currency in use — by line items, people or
// plans (016 FR-010), also those in the recycle bin — with in_use: set it inactive instead.
func (s *Service) DeleteCurrency(ctx context.Context, a actor.Actor, code string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		c, err := s.storedCurrency(ctx, code)
		if err != nil {
			return err
		}
		if c.Code == money.BaseCurrency {
			return errs.InUse(usdLockedMessage)
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		if usage := u.currency(c.Code); usage.InUse {
			return inUseError(c.Code, usage)
		}
		entry, err = s.history.MoveToBin(ctx, a, currencyKind, c.Code)
		return err
	})
	return entry, err
}

// applyCurrency returns the currency c with the name and symbol update sends, and the rate it sends (nil
// when none), and adds every problem with them to problems (data model §6): a name of at most 80
// characters that no other currency has, a symbol of at most 5 characters and a rate above 0 with at most
// six decimals, at most 1,000,000 — USD's always 1. The currency exceptCode (the one changed; "" for a new
// one) does not count for the name. The rate is not stored on the currency: it is a dated rate (016).
func applyCurrency(c Currency, update CurrencyUpdate, currencies []Currency, exceptCode string, problems *errs.Problems) (Currency, *money.Rate) {
	if update.Name != nil {
		c.Name = strings.TrimSpace(*update.Name)
		checkText(problems, "name", c.Name, enterNameMessage, maxCurrencyName)
		if c.Name != "" && utf8.RuneCountInString(c.Name) <= maxCurrencyName {
			if other, taken := findNamed(currencies, c.Name, exceptCode, func(x Currency) (string, string) { return x.Code, x.Name }); taken {
				problems.Add("name", fmt.Sprintf(currencyNameTaken, other.Name))
			}
		}
	}
	if update.Symbol != nil {
		c.Symbol = strings.TrimSpace(*update.Symbol)
		checkLength(problems, "symbol", c.Symbol, maxSymbol)
	}
	var newRate *money.Rate
	if update.Rate != nil {
		rate, err := money.ParseRate(strings.TrimSpace(*update.Rate))
		switch {
		case errors.Is(err, money.ErrOutOfRange):
			problems.Add("rate", rateTooLargeMessage)
		case err != nil:
			problems.Add("rate", rateMessage)
		case c.Code == money.BaseCurrency && !rate.IsOne():
			problems.Add("rate", usdLockedMessage)
		default:
			newRate = &rate
		}
	}
	return c, newRate
}

// isCurrencyCode reports whether code is three capital letters A–Z.
func isCurrencyCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
