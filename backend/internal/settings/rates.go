package settings

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// Dated exchange rates (feature 016, FR-001 – FR-005, data-model §2.1, §3.2; D-258). A currency other than
// USD keeps a rate history: one rate per day at most, each with its source. Its current rate is the one with
// the latest date on or before today (the API's calendar date in UTC, D-091), so a future-dated rate becomes
// current on its date without anyone saving anything. Every conversion in MAPP reads the current rates
// through Rates (or RatesOn for another day) and never stores a converted amount (D-041).

// The history kinds of feature 016 (data-model §7).
const (
	exchangeRateKind = "exchangeRate" // under its currency; deleted rates go to the recycle bin
	rateImportKind   = "rateImport"   // logged only, never deleted
	fieldHelpKind    = "fieldHelp"    // logged only, never in the recycle bin
)

// The nouns of the new records, mid-sentence.
const (
	exchangeRateNoun = "exchange rate"
	rateImportNoun   = "rate import"
	fieldHelpNoun    = "field help"
)

// Messages of 016's "User-visible messages" about rates.
const (
	rateGoneMessage         = "This exchange rate no longer exists — someone may have deleted it."
	usdAlwaysOneMessage     = "USD is always 1."
	rateDateMessage         = "Enter a date such as 2026-10-19."
	rateDateTakenMessage    = "%s already has a rate on %s."
	lastCurrentRateMessage  = "%s needs a rate dated today or earlier."
	firstRateFutureMessage  = "A new currency needs a rate dated today or earlier."
	plansRecalculated       = "%s planned in %s. %s %s amounts stay the same; %s USD values change (%s → %s USD)."
	plansRecalculatedDetail = "%s planned in %s: %s → %s USD."
)

// checkRateInput checks a new dated rate of the currency c against book (FR-001, FR-003): USD has no rates;
// a date "YYYY-MM-DD" that the currency has no rate on yet; a rate above 0 with at most six decimals, at
// most 1,000,000. Every problem is added to problems; the date and rate are returned when they are fine.
func checkRateInput(c Currency, in RateInput, book rateBook, problems *errs.Problems) (date.Date, money.Rate) {
	if c.Code == money.BaseCurrency {
		problems.Add("code", usdAlwaysOneMessage)
		return date.Date{}, money.Rate{}
	}
	day, err := date.Parse(strings.TrimSpace(in.Date))
	if err != nil {
		problems.Add("date", rateDateMessage)
	} else if _, taken := book.onDate(c.Code, day); taken {
		problems.Add("date", fmt.Sprintf(rateDateTakenMessage, c.Code, date.Format(day)))
	}
	rate := checkRate(in.Rate, problems)
	return day, rate
}

// checkRate reads a rate text: above 0 with at most six decimals, at most 1,000,000 (problems on "rate").
func checkRate(text string, problems *errs.Problems) money.Rate {
	rate, err := money.ParseRate(strings.TrimSpace(text))
	switch {
	case errors.Is(err, money.ErrOutOfRange):
		problems.Add("rate", rateTooLargeMessage)
	case err != nil:
		problems.Add("rate", rateMessage)
	}
	return rate
}

// AddRate adds a dated rate to the currency code, entered by hand (FR-003): the date defaults to today in the
// form and may be in the past or the future; one rate per day. The change is logged on the rate, with what
// it recalculates when it becomes the current rate.
func (s *Service) AddRate(ctx context.Context, a actor.Actor, code string, in RateInput) (ExchangeRateView, error) {
	var view ExchangeRateView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
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
		if err := problems.Err(); err != nil {
			return err
		}
		before := withRate(c, book, s.today()).Rate
		r, err := s.putNewRate(ctx, a, c.Code, day, rate, RateManual, "")
		if err != nil {
			return err
		}
		book[c.Code] = append(book[c.Code], r)
		details, err := s.recalculationDetails(ctx, c.Code, before, withRate(c, book.sorted(c.Code), s.today()).Rate)
		if err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: exchangeRateKind, RecordID: r.ID, RecordName: rateName(r), Action: history.ActionCreated, Details: details,
			Changes: []history.Change{{Field: "rate", Old: jsonOf(nil), New: jsonOf(r.Rate)}},
		}); err != nil {
			return err
		}
		view, err = s.rateViewOf(ctx, r)
		return err
	})
	return view, err
}

// putNewRate stores a new dated rate inside the caller's Write.
func (s *Service) putNewRate(ctx context.Context, a actor.Actor, code string, day date.Date, rate money.Rate, source RateSource, importID string) (ExchangeRate, error) {
	id, err := s.ids.Next(ctx, ids.PrefixExchangeRate)
	if err != nil {
		return ExchangeRate{}, err
	}
	r := ExchangeRate{ID: id, Currency: code, Date: day, Rate: rate, Source: source, ImportID: importID, Meta: record.NewMeta(a, s.now())}
	return r, s.repo.PutRate(ctx, r)
}

// UpdateRate changes the value of the rate id from the version the person started at (FR-003): its date
// never changes, and an edited imported rate becomes "Entered by hand".
func (s *Service) UpdateRate(ctx context.Context, a actor.Actor, id string, update RateUpdate) (ExchangeRateView, error) {
	var view ExchangeRateView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.storedRate(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		checkVersion(&problems, update.Version)
		rate := checkRate(update.Rate, &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		if err := stored.CheckVersion(*update.Version, exchangeRateNoun); err != nil {
			return err
		}
		if rate == stored.Rate && stored.Source == RateManual {
			view, err = s.rateViewOf(ctx, stored)
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
		before := withRate(c, book, s.today()).Rate
		changed := stored
		changed.Rate, changed.Source, changed.ImportID = rate, RateManual, ""
		changed.Touch(a, s.now())
		if err := s.repo.PutRate(ctx, changed); err != nil {
			return err
		}
		after, err := s.currentRateOf(ctx, c)
		if err != nil {
			return err
		}
		details, err := s.recalculationDetails(ctx, c.Code, before, after)
		if err != nil {
			return err
		}
		if err := s.logRateChanged(ctx, a, stored, changed, details); err != nil {
			return err
		}
		view, err = s.rateViewOf(ctx, changed)
		return err
	})
	return view, err
}

// currentRateOf returns the current rate of c, read inside the caller's transaction.
func (s *Service) currentRateOf(ctx context.Context, c Currency) (money.Rate, error) {
	book, err := s.rateBook(ctx)
	if err != nil {
		return money.Rate{}, err
	}
	return withRate(c, book, s.today()).Rate, nil
}

// logRateChanged writes the "changed" entry of a rate: the rate and the source, old → new.
func (s *Service) logRateChanged(ctx context.Context, a actor.Actor, before, after ExchangeRate, details string) error {
	changes := []history.Change{}
	if before.Rate != after.Rate {
		changes = append(changes, history.Change{Field: "rate", Old: jsonOf(before.Rate), New: jsonOf(after.Rate)})
	}
	if before.Source != after.Source {
		changes = append(changes, history.Change{Field: "source", Old: jsonOf(before.Source), New: jsonOf(after.Source)})
	}
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: exchangeRateKind, RecordID: after.ID, RecordName: rateName(after), Action: history.ActionChanged,
		Changes: changes, Details: details,
	})
	return err
}

// DeleteRate moves the rate id to the recycle bin (FR-003, D-272), so that it can be undone. The currency's
// last rate dated today or earlier is refused: "EUR needs a rate dated today or earlier."
func (s *Service) DeleteRate(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		r, err := s.storedRate(ctx, id)
		if err != nil {
			return err
		}
		book, err := s.rateBook(ctx)
		if err != nil {
			return err
		}
		today := s.today()
		if !r.Date.After(today) {
			others := 0
			for _, other := range book[r.Currency] {
				if other.ID != r.ID && !other.Date.After(today) {
					others++
				}
			}
			if others == 0 {
				return errs.Validation(errs.FieldError{Field: "id", Message: fmt.Sprintf(lastCurrentRateMessage, r.Currency)})
			}
		}
		entry, err = s.history.MoveToBin(ctx, a, exchangeRateKind, r.ID)
		return err
	})
	return entry, err
}

// setTodaysRate gives the currency c the rate rate from today (FR-004): it adds today's rate, or replaces it,
// entered by hand, and logs the rate's change with what it recalculated. before is the current rate before.
func (s *Service) setTodaysRate(ctx context.Context, a actor.Actor, c Currency, rate, before money.Rate) error {
	book, err := s.rateBook(ctx)
	if err != nil {
		return err
	}
	details, err := s.recalculationDetails(ctx, c.Code, before, rate)
	if err != nil {
		return err
	}
	if existing, found := book.onDate(c.Code, s.today()); found {
		changed := existing
		changed.Rate, changed.Source, changed.ImportID = rate, RateManual, ""
		changed.Touch(a, s.now())
		if err := s.repo.PutRate(ctx, changed); err != nil {
			return err
		}
		// The change people made is the currency's rate: from the rate in use before to the new one.
		existing.Rate = before
		return s.logRateChanged(ctx, a, existing, changed, details)
	}
	r, err := s.putNewRate(ctx, a, c.Code, s.today(), rate, RateManual, "")
	if err != nil {
		return err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{
		Kind: exchangeRateKind, RecordID: r.ID, RecordName: rateName(r), Action: history.ActionCreated, Details: details,
		Changes: []history.Change{{Field: "rate", Old: jsonOf(before), New: jsonOf(rate)}},
	})
	return err
}
