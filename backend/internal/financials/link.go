package financials

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Link links a not-linked transaction by hand: one the rules linked is corrected in GSAP instead. The checks
// run in the order of 038 FR-007: a transaction a may not see is missing; then import and link in the target
// line item's plan, a target a may not see answering as a missing one; then the version; then the state.
func (s *Service) Link(ctx context.Context, a actor.Actor, id string, req LinkRequest) (TransactionView, error) {
	if err := checkLinkRequest(req); err != nil {
		return TransactionView{}, err
	}
	var view TransactionView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		var err error
		view, err = s.link(ctx, a, id, req)
		return err
	})
	if err != nil {
		return TransactionView{}, err
	}
	return view, nil
}

func checkLinkRequest(req LinkRequest) error {
	var problems errs.Problems
	if req.Version == nil {
		problems.Add("version", msgEnterVersion)
	}
	if !lineItemIDForm.MatchString(req.LineItemID) {
		problems.Add(lineItemIDField, msgLineItemIDForm)
	}
	return problems.Err()
}

func (s *Service) link(ctx context.Context, a actor.Actor, id string, req LinkRequest) (TransactionView, error) {
	b, t, err := s.visible(ctx, a, id)
	if err != nil {
		return TransactionView{}, err
	}
	target, err := s.linkTarget(ctx, a, b, req.LineItemID)
	if err != nil {
		return TransactionView{}, err
	}
	if err := t.CheckVersion(*req.Version, transactionNoun); err != nil {
		return TransactionView{}, err
	}
	switch {
	case t.MatchedBy == MatchedByManual:
		return TransactionView{}, errs.NotAllowed(fmt.Sprintf(msgLinkedByPerson, t.ManualLink.By.Name, dayText(t.ManualLink.At)))
	case t.linked():
		return TransactionView{}, errs.NotAllowed(fmt.Sprintf(msgLinkedByGSAP, matchedWords(*t)))
	}
	t.LineItemID, t.MatchedBy, t.NotLinkedReason = target.ID, MatchedByManual, ""
	t.ManualLink = &ManualLink{By: a, At: s.now()}
	if err := s.save(ctx, a, t, linkChange{verb: linkedVerb, after: target.ID}); err != nil {
		return TransactionView{}, err
	}
	return s.transactionView(ctx, a, b, t), nil
}

// visible loads the book and finds the transaction id, which answers as missing when a may not open it.
func (s *Service) visible(ctx context.Context, a actor.Actor, id string) (*book, *Transaction, error) {
	b, err := s.load(ctx, a.ID)
	if err != nil {
		return nil, nil, err
	}
	t := b.find(id)
	if t == nil {
		return nil, nil, errs.NotFound(msgTransactionGone)
	}
	if err := s.mayOpen(ctx, a, b, t); err != nil {
		return nil, nil, err
	}
	return b, t, nil
}

// linkTarget checks that a may link into the line item lineItemID and returns it (038 FR-005).
func (s *Service) linkTarget(ctx context.Context, a actor.Actor, b *book, lineItemID string) (*tnode, error) {
	if err := s.access.Import(ctx, a); err != nil {
		return nil, err
	}
	if err := s.access.Link(ctx, a, ""); err != nil {
		return nil, err
	}
	target := b.tree.byID[lineItemID]
	if target == nil || target.Level != LevelLineItem {
		return nil, s.missingLineItem(ctx, lineItemID)
	}
	if err := s.access.Link(ctx, a, target.PlanID); err != nil {
		if isHidden(err) {
			return nil, s.missingLineItem(ctx, lineItemID)
		}
		return nil, err
	}
	return target, nil
}

func (s *Service) missingLineItem(ctx context.Context, id string) error {
	inBin, err := s.planning.LineItemsInBin(ctx)
	if err != nil {
		return err
	}
	if slices.Contains(inBin, id) {
		return fieldProblem(lineItemIDField, fmt.Sprintf(msgLineItemInBin, id))
	}
	return fieldProblem(lineItemIDField, fmt.Sprintf(msgNoLineItem, id))
}

// Unlink undoes only a link made by a person; the rules then match the transaction again. The checks run in
// the order of 038 FR-007: visible, link in the linked line item's plan, version, state.
func (s *Service) Unlink(ctx context.Context, a actor.Actor, id string, req UnlinkRequest) (TransactionView, error) {
	if req.Version == nil {
		return TransactionView{}, fieldProblem("version", msgEnterVersion)
	}
	var view TransactionView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		var err error
		view, err = s.unlink(ctx, a, id, req)
		return err
	})
	if err != nil {
		return TransactionView{}, err
	}
	return view, nil
}

func (s *Service) unlink(ctx context.Context, a actor.Actor, id string, req UnlinkRequest) (TransactionView, error) {
	b, t, err := s.visible(ctx, a, id)
	if err != nil {
		return TransactionView{}, err
	}
	planID := ""
	if li := b.lineItemOf(t); li != nil {
		planID = li.PlanID
	}
	if err := s.access.Link(ctx, a, planID); err != nil {
		return TransactionView{}, orGone(err, errs.NotFound(msgTransactionGone))
	}
	if err := t.CheckVersion(*req.Version, transactionNoun); err != nil {
		return TransactionView{}, err
	}
	if t.MatchedBy != MatchedByManual {
		return TransactionView{}, errs.NotAllowed(msgOnlyManualUnlink)
	}
	before := t.LineItemID
	t.LineItemID, t.MatchedBy, t.ManualLink = "", MatchedByNone, nil
	if err := s.rematch(ctx, b, t); err != nil {
		return TransactionView{}, err
	}
	if err := s.save(ctx, a, t, linkChange{verb: unlinkedVerb, before: before, after: t.LineItemID}); err != nil {
		return TransactionView{}, err
	}
	return s.transactionView(ctx, a, b, t), nil
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

const lineItemIDField = "lineItemId"

// linkChange is the history entry of a link or unlink: its verb and the line item before and after.
type linkChange struct {
	verb, before, after string
}

func (s *Service) save(ctx context.Context, a actor.Actor, t *Transaction, change linkChange) error {
	t.Touch(a, s.now())
	if err := s.repo.PutTransaction(ctx, *t); err != nil {
		return err
	}
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: transactionKind, RecordID: t.ID, RecordName: t.Name(), Action: history.ActionChanged, Verb: change.verb,
		Changes: []history.Change{{Field: lineItemIDField, Old: idValue(change.before), New: idValue(change.after)}},
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
