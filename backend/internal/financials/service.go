package financials

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
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

// UseFeed replaces the simulated GSAP extract, later with a real GSAP feed; nothing else changes.
func (s *Service) UseFeed(f Feed) {
	s.feed = f
}

func (s *Service) today() date.Date {
	return date.Today(s.now())
}

// Link links a not-linked transaction by hand: one the rules linked is corrected in GSAP instead.
func (s *Service) Link(ctx context.Context, a actor.Actor, id string, req LinkRequest) (TransactionView, error) {
	var problems errs.Problems
	if req.Version == nil {
		problems.Add("version", msgEnterVersion)
	}
	if !lineItemIDForm.MatchString(req.LineItemID) {
		problems.Add("lineItemId", msgLineItemIDForm)
	}
	if err := problems.Err(); err != nil {
		return TransactionView{}, err
	}
	var view TransactionView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if err := s.access.Link(ctx, a, ""); err != nil {
			return err
		}
		b, err := s.load(ctx, a.ID)
		if err != nil {
			return err
		}
		t := b.find(id)
		if t == nil {
			return errs.NotFound(msgTransactionGone)
		}
		if err := t.CheckVersion(*req.Version, transactionNoun); err != nil {
			return err
		}
		switch {
		case t.MatchedBy == MatchedByManual:
			return errs.NotAllowed(fmt.Sprintf(msgLinkedByPerson, t.ManualLink.By.Name, dayText(t.ManualLink.At)))
		case t.linked():
			return errs.NotAllowed(fmt.Sprintf(msgLinkedByGSAP, matchedWords(*t)))
		}
		target := b.tree.byID[req.LineItemID]
		if target == nil || target.Level != LevelLineItem {
			return s.missingLineItem(ctx, req.LineItemID)
		}
		if err := s.access.Link(ctx, a, target.PlanID); err != nil {
			return err
		}
		t.LineItemID, t.MatchedBy, t.NotLinkedReason = target.ID, MatchedByManual, ""
		t.ManualLink = &ManualLink{By: a, At: s.now()}
		if err := s.save(ctx, a, t, linkedVerb, "", target.ID); err != nil {
			return err
		}
		view = s.transactionView(ctx, a, b, t)
		return nil
	})
	if err != nil {
		return TransactionView{}, err
	}
	return view, nil
}

func (s *Service) missingLineItem(ctx context.Context, id string) error {
	inBin, err := s.planning.LineItemsInBin(ctx)
	if err != nil {
		return err
	}
	if slices.Contains(inBin, id) {
		return fieldProblem("lineItemId", fmt.Sprintf(msgLineItemInBin, id))
	}
	return fieldProblem("lineItemId", fmt.Sprintf(msgNoLineItem, id))
}

// Unlink undoes only a link made by a person; the rules then match the transaction again.
func (s *Service) Unlink(ctx context.Context, a actor.Actor, id string, req UnlinkRequest) (TransactionView, error) {
	if req.Version == nil {
		return TransactionView{}, fieldProblem("version", msgEnterVersion)
	}
	var view TransactionView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if err := s.access.Link(ctx, a, ""); err != nil {
			return err
		}
		b, err := s.load(ctx, a.ID)
		if err != nil {
			return err
		}
		t := b.find(id)
		if t == nil {
			return errs.NotFound(msgTransactionGone)
		}
		if err := t.CheckVersion(*req.Version, transactionNoun); err != nil {
			return err
		}
		if t.MatchedBy != MatchedByManual {
			return errs.NotAllowed(msgOnlyManualUnlink)
		}
		planID := ""
		if li := b.lineItemOf(t); li != nil {
			planID = li.PlanID
		}
		if err := s.access.Link(ctx, a, planID); err != nil {
			return err
		}
		before := t.LineItemID
		t.LineItemID, t.MatchedBy, t.ManualLink = "", MatchedByNone, nil
		if err := s.rematch(ctx, b, t); err != nil {
			return err
		}
		if err := s.save(ctx, a, t, unlinkedVerb, before, t.LineItemID); err != nil {
			return err
		}
		view = s.transactionView(ctx, a, b, t)
		return nil
	})
	if err != nil {
		return TransactionView{}, err
	}
	return view, nil
}

func (s *Service) rematch(ctx context.Context, b *book, t *Transaction) error {
	inBin, err := s.planning.LineItemsInBin(ctx)
	if err != nil {
		return err
	}
	items := storedSet{}
	for id, n := range b.tree.byID {
		if n.Level == LevelLineItem {
			items[id] = true
		}
	}
	for _, id := range inBin {
		items[id] = true
	}
	stored := make([]Transaction, 0, len(b.all))
	for _, other := range b.all {
		if other.ID != t.ID {
			stored = append(stored, *other)
		}
	}
	x := newIndex(stored)
	l := match(t, x, items)
	t.LineItemID, t.MatchedBy, t.NotLinkedReason = l.lineItemID, l.by, l.reason
	return nil
}

func (s *Service) save(ctx context.Context, a actor.Actor, t *Transaction, verb, before, after string) error {
	t.Touch(a, s.now())
	if err := s.repo.PutTransaction(ctx, *t); err != nil {
		return err
	}
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: transactionKind, RecordID: t.ID, RecordName: t.Name(), Action: history.ActionChanged, Verb: verb,
		Changes: []history.Change{{Field: "lineItemId", Old: idValue(before), New: idValue(after)}},
	})
	return err
}

func idValue(id string) json.RawMessage {
	if id == "" {
		return json.RawMessage("null")
	}
	encoded, _ := json.Marshal(id) // a string always encodes
	return encoded
}

func dayText(t time.Time) string {
	return t.UTC().Format("2 Jan 2006")
}

// FactsByLineItem gives All work the cards' USD figures by line item: current rates, the latest accrual
// month, currencies no longer in Settings left out. It checks no access: the caller does.
func (s *Service) FactsByLineItem(ctx context.Context) (map[string]Facts, error) {
	var facts map[string]Facts
	err := s.tx.Read(ctx, func(ctx context.Context) error {
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
		facts = map[string]Facts{}
		for _, t := range all {
			if t.linked() {
				f := facts[t.LineItemID]
				f.add(t, usdOf(t, rates), month)
				facts[t.LineItemID] = f
			}
		}
		return nil
	})
	return facts, err
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
