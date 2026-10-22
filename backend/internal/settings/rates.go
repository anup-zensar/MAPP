package settings

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

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

// rateBook is every dated rate, by currency, oldest date first: what the current rate of a day is worked
// out from. It is read once per request.
type rateBook map[string][]ExchangeRate

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

// withoutLegacy returns c without the legacy rate fields, as it is stored since 016.
func withoutLegacy(c Currency) Currency {
	c.Rate, c.RateChangedAt, c.RateChangedBy = money.Rate{}, time.Time{}, actor.Actor{}
	return c
}

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

// RateInfo is a currency's current rate and where it came from (contract RateUsed, FR-009): what the web app
// shows beside every converted amount.
type RateInfo struct {
	Currency       string      `json:"currency"`
	Rate           money.Rate  `json:"rate"`
	InverseRate    money.Rate  `json:"inverseRate"`
	RateDate       *date.Date  `json:"rateDate"`       // null for USD
	Source         *RateSource `json:"source"`         // null for USD
	ImportFileName *string     `json:"importFileName"` // the GSAP file of an imported rate; null otherwise
}

// CurrentRatePage is the answer of GET /api/exchange-rates/current (contract CurrentRatePage).
type CurrentRatePage struct {
	Today date.Date  `json:"today"`
	Items []RateInfo `json:"items"`
	Total int        `json:"total"`
}

// RatesOn returns the rate of every currency current on day d, active or not (D-275): the one way to
// convert at another day than today. USD is 1.
func (s *Service) RatesOn(ctx context.Context, d date.Date) (money.Rates, error) {
	rates := money.Rates{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		currencies, err := s.repo.ListCurrencies(ctx)
		if err != nil {
			return err
		}
		book, err := s.rateBook(ctx)
		if err != nil {
			return err
		}
		for _, c := range currencies {
			rates[c.Code] = withRate(c, book, d).Rate
		}
		return nil
	})
	return rates, err
}

// CurrentRates returns every currency's current rate with its date and source (FR-002, FR-009): USD first,
// then by code.
func (s *Service) CurrentRates(ctx context.Context) ([]RateInfo, error) {
	var infos []RateInfo
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		currencies, err := s.repo.ListCurrencies(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(currencies, compareCurrencies)
		book, err := s.rateBook(ctx)
		if err != nil {
			return err
		}
		files, err := s.importFileNames(ctx)
		if err != nil {
			return err
		}
		infos = make([]RateInfo, 0, len(currencies))
		for _, c := range currencies {
			infos = append(infos, rateInfoOf(c, book, s.today(), files))
		}
		return nil
	})
	return infos, err
}

// CurrentRate returns the current rate of the currency code with its date and source, and false when there
// is no such currency.
func (s *Service) CurrentRate(ctx context.Context, code string) (RateInfo, bool, error) {
	infos, err := s.CurrentRates(ctx)
	if err != nil {
		return RateInfo{}, false, err
	}
	for _, info := range infos {
		if info.Currency == code {
			return info, true, nil
		}
	}
	return RateInfo{}, false, nil
}

// ListCurrentRates answers GET /api/exchange-rates/current. Every signed-in person may read it (014 Table C).
func (s *Service) ListCurrentRates(ctx context.Context, a actor.Actor) (CurrentRatePage, error) {
	infos, err := s.CurrentRates(ctx)
	if err != nil {
		return CurrentRatePage{}, err
	}
	return CurrentRatePage{Today: s.today(), Items: infos, Total: len(infos)}, nil
}

// rateInfoOf returns the current rate of c on day with where it came from.
func rateInfoOf(c Currency, book rateBook, day date.Date, files map[string]string) RateInfo {
	info := RateInfo{Currency: c.Code, Rate: money.One, InverseRate: money.One}
	if c.Code == money.BaseCurrency {
		return info
	}
	if r, found := book.on(c.Code, day); found {
		info.Rate, info.InverseRate = r.Rate, r.Rate.Inverse()
		rateDate, source := r.Date, r.Source
		info.RateDate, info.Source = &rateDate, &source
		if name, imported := files[r.ImportID]; imported && r.Source == RateGSAPImport {
			info.ImportFileName = &name
		}
		return info
	}
	if c.Rate.Micros() > 0 { // stored before 016 and not migrated yet
		rateDate, source := date.Today(c.RateChangedAt), RateManual
		info.Rate, info.InverseRate, info.RateDate, info.Source = c.Rate, c.Rate.Inverse(), &rateDate, &source
	}
	return info
}

// importFileNames returns the file name of every rate import, by id.
func (s *Service) importFileNames(ctx context.Context) (map[string]string, error) {
	imports, err := s.repo.ListRateImports(ctx)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(imports))
	for _, i := range imports {
		names[i.ID] = i.FileName
	}
	return names, nil
}

// The API types of the rate history (contract ExchangeRateRecord, ExchangeRatePage, ExchangeRateCreate,
// ExchangeRateUpdate).

// ExchangeRateView is one dated rate as the API answers it (ExchangeRateRecord).
type ExchangeRateView struct {
	ID             string     `json:"id"`
	Currency       string     `json:"currency"`
	Date           date.Date  `json:"date"`
	Rate           money.Rate `json:"rate"`
	InverseRate    money.Rate `json:"inverseRate"`
	Source         RateSource `json:"source"`
	ImportID       *string    `json:"importId"`
	ImportFileName *string    `json:"importFileName"`
	Current        bool       `json:"current"` // the rate conversions use today
	Future         bool       `json:"future"`  // dated after today ("From <date>")
	record.Meta
}

// ExchangeRatePage is a currency's rate history, newest date first, with its current rate (ExchangeRatePage).
type ExchangeRatePage struct {
	Items   []ExchangeRateView `json:"items"`
	Total   int                `json:"total"`
	Current RateInfo           `json:"current"`
}

// RateInput is the body of POST /api/currencies/{code}/rates and its impact check (ExchangeRateCreate).
type RateInput struct {
	Date string `json:"date"`
	Rate string `json:"rate"`
}

// RateUpdate is the body of PATCH /api/exchange-rates/{rateId} and its impact check (ExchangeRateUpdate).
type RateUpdate struct {
	Rate    string `json:"rate"`
	Version *int   `json:"version"`
}

// rateView returns the rate r as the API answers it on day.
func rateView(r ExchangeRate, book rateBook, day date.Date, files map[string]string) ExchangeRateView {
	view := ExchangeRateView{
		ID: r.ID, Currency: r.Currency, Date: r.Date, Rate: r.Rate, InverseRate: r.Rate.Inverse(), Source: r.Source,
		Future: r.Date.After(day), Meta: r.Meta,
	}
	if current, found := book.on(r.Currency, day); found && current.ID == r.ID {
		view.Current = true
	}
	if r.ImportID != "" {
		importID := r.ImportID
		view.ImportID = &importID
		if name, found := files[r.ImportID]; found {
			view.ImportFileName = &name
		}
	}
	return view
}

// rateName is how the activity log and the recycle bin name a rate: "EUR on 19 Oct 2026".
func rateName(r ExchangeRate) string {
	return r.Currency + " on " + date.Format(r.Date)
}

// ListRates returns the rate history of the currency code, newest date first, with its current rate (FR-001,
// FR-003). USD has none. Every signed-in person may read it.
func (s *Service) ListRates(ctx context.Context, a actor.Actor, code string) (ExchangeRatePage, error) {
	var page ExchangeRatePage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.storedCurrency(ctx, code)
		if err != nil {
			return err
		}
		book, err := s.rateBook(ctx)
		if err != nil {
			return err
		}
		files, err := s.importFileNames(ctx)
		if err != nil {
			return err
		}
		day := s.today()
		page = ExchangeRatePage{Items: []ExchangeRateView{}, Current: rateInfoOf(c, book, day, files)}
		rates := book[c.Code]
		for i := len(rates) - 1; i >= 0; i-- {
			page.Items = append(page.Items, rateView(rates[i], book, day, files))
		}
		page.Total = len(page.Items)
		return nil
	})
	return page, err
}

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

// sorted returns the book with the rates of code in date order again, after one was added.
func (b rateBook) sorted(code string) rateBook {
	slices.SortStableFunc(b[code], func(x, y ExchangeRate) int { return cmp.Or(x.Date.Compare(y.Date), compareIDs(x.ID, y.ID)) })
	return b
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

// rateViewOf returns the stored rate r as the API answers it, reading the other rates of its currency.
func (s *Service) rateViewOf(ctx context.Context, r ExchangeRate) (ExchangeRateView, error) {
	book, err := s.rateBook(ctx)
	if err != nil {
		return ExchangeRateView{}, err
	}
	files, err := s.importFileNames(ctx)
	if err != nil {
		return ExchangeRateView{}, err
	}
	return rateView(r, book, s.today(), files), nil
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

// storedRate reads the rate id, or answers not found.
func (s *Service) storedRate(ctx context.Context, id string) (ExchangeRate, error) {
	r, found, err := s.repo.GetRate(ctx, id)
	if err != nil {
		return ExchangeRate{}, err
	}
	if !found {
		return ExchangeRate{}, errs.NotFound(rateGoneMessage)
	}
	return r, nil
}

// AddRateImpact says, before a new dated rate of the currency code is saved, what it would change (FR-005,
// D-044): when it becomes the current rate, the line items in the currency and the plans planned in it. A
// future-dated rate changes nothing before its date, so it answers no warning. It is checked as AddRate
// checks it; nothing is saved.
func (s *Service) AddRateImpact(ctx context.Context, a actor.Actor, code string, in RateInput) (ChangeImpact, error) {
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
		if err := problems.Err(); err != nil {
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
func (s *Service) UpdateRateImpact(ctx context.Context, a actor.Actor, id string, update RateUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.storedRate(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		rate := checkRate(update.Rate, &problems)
		if err := problems.Err(); err != nil {
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
