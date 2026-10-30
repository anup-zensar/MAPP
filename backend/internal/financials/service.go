package financials

import (
	"context"
	"fmt"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
)

// maxListed caps the problems and not-linked rows listed; the totals still count them all.
const maxListed = 200

const latestCount = 5

type Deps struct {
	Planning Planning
	Rates    Rates
	Displays Displays
	Access   Access
}

// Service reads in one read transaction; imports, links and unlinks write with their log entry in one Write.
type Service struct {
	tx       Transactor
	repo     Store
	ids      *ids.Generator
	history  *history.Service
	planning Planning
	rates    Rates
	displays Displays
	access   Access
	feed     Feed
	now      clock.Clock
}

func NewService(tools history.Tools, repo Store, deps Deps) *Service {
	return &Service{
		tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, planning: deps.Planning, rates: deps.Rates,
		displays: deps.Displays, access: deps.Access, feed: CSVFeed{}, now: tools.Now,
	}
}

func (s *Service) UseFeed(f Feed) {
	s.feed = f
}

func (s *Service) today() date.Date {
	return date.Today(s.now())
}

// FactsByLineItem gives All work the cards' USD figures by line item. It checks no access: the caller does.
func (s *Service) FactsByLineItem(ctx context.Context) (map[string]Facts, error) {
	facts := map[string]Facts{}
	err := s.eachLinked(ctx, func(t *Transaction, usd *money.Amount, month string) {
		f := facts[t.LineItemID]
		f.add(t, usd, month)
		facts[t.LineItemID] = f
	})
	return facts, err
}

// FactsByLineItemMonth splits FactsByLineItem by posting month, "2026-03". It checks no access: the caller does.
func (s *Service) FactsByLineItemMonth(ctx context.Context) (map[string]map[string]Facts, error) {
	facts := map[string]map[string]Facts{}
	err := s.eachLinked(ctx, func(t *Transaction, usd *money.Amount, month string) {
		byMonth := facts[t.LineItemID]
		if byMonth == nil {
			byMonth = map[string]Facts{}
			facts[t.LineItemID] = byMonth
		}
		key := fmt.Sprintf("%04d-%02d", t.PostingDate.Year(), t.PostingDate.Month())
		f := byMonth[key]
		f.add(t, usd, month)
		byMonth[key] = f
	})
	return facts, err
}

func (s *Service) eachLinked(ctx context.Context, add func(t *Transaction, usd *money.Amount, month string)) error {
	return s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.repo.ListTransactions(ctx)
		if err != nil {
			return err
		}
		rates, err := s.rates.Rates(ctx)
		if err != nil {
			return err
		}
		all := make([]*Transaction, len(stored))
		for i := range stored {
			all[i] = &stored[i]
		}
		month := latestAccrualMonth(all)
		for _, t := range all {
			if t.linked() {
				add(t, usdOf(t, rates), month)
			}
		}
		return nil
	})
}

// DataAsOf is nil when no GSAP data was ever loaded.
func (s *Service) DataAsOf(ctx context.Context) (*time.Time, error) {
	var asOf *time.Time
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		imports, err := s.repo.ListImports(ctx)
		if err != nil {
			return err
		}
		asOf = (&book{imports: imports}).source().DataAsOf
		return nil
	})
	return asOf, err
}
