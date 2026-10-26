package settings

import (
	"bytes"
	"cmp"
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// The GSAP rate import (016 FR-006 – FR-008, data-model §2.3, §2.4; D-259): review a rate file, then import
// the rates to add and replace in one step. The rows come from a RateFeed; ApplyRateRows is the core a later
// automatic feed calls with the same rules.

// RateImportUpload is the body of the two import operations (contract RateImportUpload): the file in base64.
type RateImportUpload struct {
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

// RateOutcome is what importing one row does (contract RateImportOutcome).
type RateOutcome string

// The outcomes of a row (data-model §2.4).
const (
	OutcomeAdd     RateOutcome = "add"     // a new rate
	OutcomeReplace RateOutcome = "replace" // a different rate for the same currency and day
	OutcomePresent RateOutcome = "present" // the same rate is stored already: skipped
	OutcomeNotUsed RateOutcome = "notUsed" // another rate type, not quoted from USD, USD itself, or a currency not set up
	OutcomeProblem RateOutcome = "problem" // an unreadable date or rate, or a row twice in the file
)

// RateImportRow is one reviewed row (contract RateImportRow).
type RateImportRow struct {
	Row          int         `json:"row"`
	Outcome      RateOutcome `json:"outcome"`
	Currency     *string     `json:"currency"`
	Date         *date.Date  `json:"date"`
	Rate         *money.Rate `json:"rate"`
	PreviousRate *money.Rate `json:"previousRate"` // replace only
	Message      *string     `json:"message"`      // why a row is not used or has a problem; old → new for replace
}

// RateImportCounts counts the rows of a review by outcome (contract RateImportCounts).
type RateImportCounts struct {
	Add     int `json:"add"`
	Replace int `json:"replace"`
	Present int `json:"present"`
	NotUsed int `json:"notUsed"`
	Problem int `json:"problem"`
}

// RateImportCheck is the review of a rate file (contract RateImportCheck): every row, in file order.
type RateImportCheck struct {
	FileName string           `json:"fileName"`
	Feed     string           `json:"feed"`
	Rows     []RateImportRow  `json:"rows"`
	Counts   RateImportCounts `json:"counts"`
}

// RateImportView is a rate import as the API answers it (contract RateImport).
type RateImportView struct {
	ID         string      `json:"id"`
	FileName   string      `json:"fileName"`
	Feed       string      `json:"feed"`
	FirstDate  *date.Date  `json:"firstDate"`
	LastDate   *date.Date  `json:"lastDate"`
	Added      int         `json:"added"`
	Replaced   int         `json:"replaced"`
	Unchanged  int         `json:"unchanged"`
	NotUsed    int         `json:"notUsed"`
	Problems   int         `json:"problems"`
	ImportedAt time.Time   `json:"importedAt"`
	ImportedBy actor.Actor `json:"importedBy"`
}

// RateImportPage is the latest imports, newest first (contract RateImportPage).
type RateImportPage struct {
	Items []RateImportView `json:"items"`
	Total int              `json:"total"`
}

// Messages of a reviewed row and of an import (data-model §2.4, plan "User-visible messages").
const (
	rateTypeNotUsedMessage  = "Rate type %s is not used — only M rates are imported."
	notFromUSDMessage       = "Only rates quoted from USD are used."
	currencyNotSetUpMessage = "%s is not set up in Settings › Currencies."
	rowDateMessage          = "Row %d: 'valid_from' must be a date such as 2026-10-19."
	rowRateMessage          = "Row %d: the rate must be above 0 with at most 6 decimals."
	rowTwiceMessage         = "Row %d: %s on %s appears twice (also row %d)."
	nothingToImportMessage  = "Nothing to import: every rate is already present or not used."
	chooseRateFileMessage   = "Choose a file to import."
	fileNameTooLongMessage  = "Use a file name of 200 characters or fewer."
	rateImportedVerb        = "imported"
	rateImportDetails       = "%s (%d added, %d replaced) for %s"
	maxRateFileName         = 200
	sampleRateType          = "M"
	gsapSampleNamePattern   = "gsap_rates_%s.csv"
	defaultRecentImports    = 10
	maxRecentImports        = 50
)

// rateFeed is the feed the import reads files with: the simulated GSAP daily rate file unless app wires
// another (FR-008).
func (s *Service) rateFeed() RateFeed {
	if s.feed == nil {
		return GSAPRateFile{}
	}
	return s.feed
}

// UseRateFeed sets the feed rate files are read with (FR-008). internal/app wires GSAPRateFile.
func (s *Service) UseRateFeed(feed RateFeed) {
	s.feed = feed
}

// readUpload checks the upload and reads its rows with feed: a file name of 1–200 characters and content in
// base64 the feed can read. Every problem is a validation error.
func readUpload(feed RateFeed, upload RateImportUpload) (string, []FeedRow, error) {
	name := strings.TrimSpace(upload.FileName)
	var problems errs.Problems
	switch {
	case name == "":
		problems.Add("fileName", chooseRateFileMessage)
	case utf8.RuneCountInString(name) > maxRateFileName:
		problems.Add("fileName", fileNameTooLongMessage)
	}
	content, err := base64.StdEncoding.DecodeString(upload.Content)
	if err != nil {
		problems.Add("content", rateFileUnreadableMessage)
	}
	if err := problems.Err(); err != nil {
		return "", nil, err
	}
	rows, err := feed.Read(name, content)
	return name, rows, err
}

// CheckRateImport reviews a rate file without saving anything (FR-006): every row's outcome with its row
// number. A file that can't be read at all answers validation_failed on content.
func (s *Service) CheckRateImport(ctx context.Context, a actor.Actor, upload RateImportUpload) (RateImportCheck, error) {
	feed := s.rateFeed()
	name, rows, err := readUpload(feed, upload)
	if err != nil {
		return RateImportCheck{}, err
	}
	var check RateImportCheck
	err = s.tx.Read(ctx, func(ctx context.Context) error {
		review, err := s.reviewRates(ctx, rows)
		check = RateImportCheck{FileName: name, Feed: feed.Name(), Rows: review.rows, Counts: review.counts}
		return err
	})
	return check, err
}

// ApplyRateImport imports a rate file (FR-006, FR-007): it reviews the file again, as CheckRateImport does,
// inside one transaction, saves the rows to add and replace as GSAP imported rates, records the import and
// writes one activity-log entry. With nothing to add or replace it is refused and nothing is saved.
func (s *Service) ApplyRateImport(ctx context.Context, a actor.Actor, upload RateImportUpload) (RateImportView, error) {
	feed := s.rateFeed()
	name, rows, err := readUpload(feed, upload)
	if err != nil {
		return RateImportView{}, err
	}
	return s.ApplyRateRows(ctx, a, feed.Name(), name, rows)
}

// ApplyRateRows is the shared core of every rate feed (FR-008): it reviews rows with the rules of
// data-model §2.4 and saves the rows to add and replace in one Write, with the import record named source
// (a file name) and one log entry. A later automatic GSAP feed calls it with its own rows.
func (s *Service) ApplyRateRows(ctx context.Context, a actor.Actor, feedName, source string, rows []FeedRow) (RateImportView, error) {
	var view RateImportView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		review, err := s.reviewRates(ctx, rows)
		if err != nil {
			return err
		}
		if review.counts.Add+review.counts.Replace == 0 {
			return contentProblem(nothingToImportMessage)
		}
		importID, err := s.ids.Next(ctx, ids.PrefixRateImport)
		if err != nil {
			return err
		}
		var first, last *date.Date
		for _, r := range review.write {
			r.Source, r.ImportID = RateGSAPImport, importID
			if r.ID == "" {
				id, err := s.ids.Next(ctx, ids.PrefixExchangeRate)
				if err != nil {
					return err
				}
				r.ID, r.Meta = id, record.NewMeta(a, s.now())
			} else {
				r.Touch(a, s.now())
			}
			if err := s.repo.PutRate(ctx, r); err != nil {
				return err
			}
			day := r.Date
			if first == nil || day.Before(*first) {
				first = &day
			}
			if last == nil || day.After(*last) {
				last = &day
			}
		}
		imp := RateImport{
			ID: importID, FileName: source, Feed: feedName, FirstDate: first, LastDate: last,
			Added: review.counts.Add, Replaced: review.counts.Replace, Unchanged: review.counts.Present,
			NotUsed: review.counts.NotUsed, Problems: review.counts.Problem, Meta: record.NewMeta(a, s.now()),
		}
		if err := s.repo.PutRateImport(ctx, imp); err != nil {
			return err
		}
		saved := imp.Added + imp.Replaced
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: rateImportKind, RecordID: imp.ID, RecordName: imp.FileName, Action: history.ActionCreated, Verb: rateImportedVerb,
			Details: fmt.Sprintf(rateImportDetails, plural(saved, "exchange rate", "exchange rates"), imp.Added, imp.Replaced,
				dateSpan(*first, *last)),
		}); err != nil {
			return err
		}
		view = rateImportView(imp)
		return nil
	})
	return view, err
}

// dateSpan writes the dates of an import: "19 Oct 2026", or "19 Sep – 19 Oct 2026".
func dateSpan(first, last date.Date) string {
	if first == last {
		return date.Format(first)
	}
	return date.FormatRange(first, last)
}

// rateReview is the outcome of every row, with the rates to write: new ones without an id, replaced ones
// with their stored id and meta.
type rateReview struct {
	rows   []RateImportRow
	counts RateImportCounts
	write  []ExchangeRate
}

// reviewRates classifies rows against the stored currencies and rates (data-model §2.4), inside the caller's
// transaction. Inactive currencies are imported; currencies are never created.
func (s *Service) reviewRates(ctx context.Context, rows []FeedRow) (rateReview, error) {
	currencies, err := s.repo.ListCurrencies(ctx)
	if err != nil {
		return rateReview{}, err
	}
	known := map[string]bool{}
	for _, c := range currencies {
		known[c.Code] = true
	}
	book, err := s.rateBook(ctx)
	if err != nil {
		return rateReview{}, err
	}
	review := rateReview{rows: make([]RateImportRow, 0, len(rows))}
	firstRow := map[string]int{} // "EUR 2026-10-19" → the row it was first seen on
	for _, in := range rows {
		row := RateImportRow{Row: in.Row}
		code := strings.ToUpper(strings.TrimSpace(in.To))
		if code != "" {
			row.Currency = &code
		}
		day, dayErr := parseRateDate(in.ValidFrom)
		if dayErr == nil {
			row.Date = &day
		}
		rate, rateErr := money.ParseRate(strings.TrimSpace(in.Rate))
		if rateErr == nil {
			row.Rate = &rate
		}
		outcome, message := OutcomeAdd, ""
		switch {
		case !strings.EqualFold(strings.TrimSpace(in.RateType), sampleRateType):
			outcome, message = OutcomeNotUsed, fmt.Sprintf(rateTypeNotUsedMessage, strings.TrimSpace(in.RateType))
		case !strings.EqualFold(strings.TrimSpace(in.From), money.BaseCurrency):
			outcome, message = OutcomeNotUsed, notFromUSDMessage
		case code == money.BaseCurrency:
			outcome, message = OutcomeNotUsed, usdAlwaysOneMessage
		case !known[code]:
			outcome, message = OutcomeNotUsed, fmt.Sprintf(currencyNotSetUpMessage, code)
		case dayErr != nil:
			outcome, message = OutcomeProblem, fmt.Sprintf(rowDateMessage, in.Row)
		case rateErr != nil:
			outcome, message = OutcomeProblem, fmt.Sprintf(rowRateMessage, in.Row)
		}
		if outcome == OutcomeAdd {
			key := code + " " + day.String()
			if earlier, twice := firstRow[key]; twice {
				outcome, message = OutcomeProblem, fmt.Sprintf(rowTwiceMessage, in.Row, code, date.Format(day), earlier)
			} else {
				firstRow[key] = in.Row
				if stored, found := book.onDate(code, day); found {
					if stored.Rate == rate {
						outcome = OutcomePresent
					} else {
						outcome, message = OutcomeReplace, stored.Rate.Format()+" → "+rate.Format()
						previous := stored.Rate
						row.PreviousRate = &previous
						changed := stored
						changed.Rate = rate
						review.write = append(review.write, changed)
					}
				} else {
					review.write = append(review.write, ExchangeRate{Currency: code, Date: day, Rate: rate})
				}
			}
		}
		row.Outcome = outcome
		if message != "" {
			row.Message = &message
		}
		review.count(outcome)
		review.rows = append(review.rows, row)
	}
	return review, nil
}

// count adds a row of outcome to the counts.
func (r *rateReview) count(outcome RateOutcome) {
	switch outcome {
	case OutcomeAdd:
		r.counts.Add++
	case OutcomeReplace:
		r.counts.Replace++
	case OutcomePresent:
		r.counts.Present++
	case OutcomeNotUsed:
		r.counts.NotUsed++
	case OutcomeProblem:
		r.counts.Problem++
	}
}

// parseRateDate reads a valid_from: "2026-10-19" or "20261019".
func parseRateDate(text string) (date.Date, error) {
	text = strings.TrimSpace(text)
	if len(text) == 8 && !strings.Contains(text, "-") {
		text = text[:4] + "-" + text[4:6] + "-" + text[6:]
	}
	return date.Parse(text)
}

// rateImportView returns the import i as the API answers it.
func rateImportView(i RateImport) RateImportView {
	return RateImportView{
		ID: i.ID, FileName: i.FileName, Feed: i.Feed, FirstDate: i.FirstDate, LastDate: i.LastDate, Added: i.Added,
		Replaced: i.Replaced, Unchanged: i.Unchanged, NotUsed: i.NotUsed, Problems: i.Problems,
		ImportedAt: i.CreatedAt, ImportedBy: i.CreatedBy,
	}
}

// ListRateImports returns the latest limit imports, newest first (FR-007): 10 when limit is 0, at most 50.
func (s *Service) ListRateImports(ctx context.Context, a actor.Actor, limit int) (RateImportPage, error) {
	if limit <= 0 {
		limit = defaultRecentImports
	}
	limit = min(limit, maxRecentImports)
	page := RateImportPage{Items: []RateImportView{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		imports, err := s.repo.ListRateImports(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(imports, func(x, y RateImport) int {
			return cmp.Or(y.CreatedAt.Compare(x.CreatedAt), compareIDs(y.ID, x.ID))
		})
		for _, i := range imports[:min(limit, len(imports))] {
			page.Items = append(page.Items, rateImportView(i))
		}
		page.Total = len(page.Items)
		return nil
	})
	return page, err
}

// ImportSample returns a sample GSAP rate file (FR-006, data-model §2.4): one M row per active currency other
// than USD, dated today, at its current rate × 1.005 rounded half up to six decimals, named
// gsap_rates_<today>.csv.
func (s *Service) ImportSample(ctx context.Context) (string, []byte, error) {
	var buffer bytes.Buffer
	day := s.today()
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		currencies, err := s.currenciesWithRates(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(currencies, compareCurrencies)
		w := csv.NewWriter(&buffer)
		if err := w.Write(gsapColumns); err != nil {
			return err
		}
		for _, c := range currencies {
			if !c.Active || c.Code == money.BaseCurrency || c.Rate.Micros() <= 0 {
				continue
			}
			sample := money.RateFromMicros((c.Rate.Micros()*1005 + 500) / 1000)
			if err := w.Write([]string{sampleRateType, money.BaseCurrency, c.Code, day.String(), sample.String()}); err != nil {
				return err
			}
		}
		w.Flush()
		return w.Error()
	})
	return fmt.Sprintf(gsapSampleNamePattern, day.String()), buffer.Bytes(), err
}
