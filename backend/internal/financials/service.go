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
	"mapp/internal/platform/store"
)

// maxListed is the most problems and not-linked rows a check or an import lists; the totals say how many
// there are.
const maxListed = 200

// latestCount is how many latest transactions a Financials card shows (FR-015).
const latestCount = 5

// Deps are the ports of the service (ports.go), which internal/app fills.
type Deps struct {
	Planning Planning
	Rates    Rates
	Displays Displays
	Access   Access
}

// Service holds the rules of feature 015. Every method takes the actor; the reads run in one read
// transaction, imports, links and unlinks in one Write with their activity-log entry.
type Service struct {
	tx       store.Transactor
	repo     Store
	ids      *ids.Generator
	history  *history.Service
	planning Planning
	rates    Rates
	displays Displays
	access   Access
	feed     Feed // the feed of uploaded files: the simulated GSAP extract
	now      clock.Clock
}

// NewService returns the financials service. It keeps transactions and imports in repo, gives out readable
// IDs, logs, works in transactions and reads the time with tools, and reads the other features through
// deps. Uploaded files are read by the simulated GSAP extract (CSVFeed).
func NewService(tools history.Tools, repo Store, deps Deps) *Service {
	return &Service{
		tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, planning: deps.Planning, rates: deps.Rates,
		displays: deps.Displays, access: deps.Access, feed: CSVFeed{}, now: tools.Now,
	}
}

// UseFeed replaces the feed that reads uploaded files — a real GSAP feed later (FR-003, SC-006). Nothing
// else changes: the checks, matching, figures, handler and contract stay the same.
func (s *Service) UseFeed(f Feed) {
	s.feed = f
}

// today is the calendar date of the clock: "the current year" is its year (D-091).
func (s *Service) today() date.Date {
	return date.Today(s.now())
}

// Link links the not-linked transaction id to the line item req.LineItemID by hand (FR-026, D-253) and
// logs it (FR-027). It refuses: a missing version or a line item ID of the wrong form (validation); a
// transaction that does not exist (not_found); a version that is out of date (version_conflict); a
// transaction already linked, by the rules — corrected in GSAP — or by a person (not_allowed); a line item
// that does not exist or is in the recycle bin (validation on lineItemId); a person who may not link in the
// line item's plan (forbidden).
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

// missingLineItem is the refusal of a line item ID that is not live: in the recycle bin, or unknown.
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

// Unlink undoes a manual link (FR-026): the transaction is matched again by the rules, and usually is Not
// linked again with its import reason. Only links made by a person can be undone (not_allowed otherwise).
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

// rematch matches t again by the rules, against every other stored transaction.
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

// save stores t after a link change by a and writes its one activity-log entry: the line item before and
// after ("" for Not linked).
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

// idValue is a line item id as a stored JSON value: null for none.
func idValue(id string) json.RawMessage {
	if id == "" {
		return json.RawMessage("null")
	}
	encoded, _ := json.Marshal(id) // a string always encodes
	return encoded
}

// dayText writes a moment's UTC date as people read it: "20 Oct 2026".
func dayText(t time.Time) string {
	return t.UTC().Format("2 Jan 2006")
}

// FactsByLineItem returns the figures in USD of every line item with linked transactions, by line item
// id, for All work (010 FR-023): the same rules as the cards — current rates, the latest accrual month,
// currencies no longer in Settings left out (SC-002). It joins the caller's transaction and checks no
// access: the caller does.
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

// DataAsOf returns when the newest GSAP data was loaded (FR-008), nil when none ever was. It joins the
// caller's transaction.
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
