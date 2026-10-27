package financials

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// Transaction returns one transaction with every field of the details panel (FR-022). A linked
// transaction needs access to its plan's financial data; a not-linked one needs the right to import
// (D-301): otherwise it is forbidden, or not_found when its plan is outside a's access.
func (s *Service) Transaction(ctx context.Context, a actor.Actor, id string) (TransactionView, error) {
	var view TransactionView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		if err := s.access.SeeFinance(ctx, a); err != nil {
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
		if err := s.mayOpen(ctx, a, b, t); err != nil {
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

// mayOpen checks that a may open t (D-301).
func (s *Service) mayOpen(ctx context.Context, a actor.Actor, b *book, t *Transaction) error {
	var err error
	if li := b.lineItemOf(t); li != nil {
		err = s.access.SeeFinanceIn(ctx, a, li.PlanID)
	} else {
		err = s.access.Import(ctx, a)
	}
	switch {
	case err == nil:
		return nil
	case isHidden(err):
		return errs.NotFound(msgTransactionGone)
	default:
		return errs.Forbidden(fmt.Sprintf(msgNoAccessRecord, transactionNoun))
	}
}

// isHidden reports whether an access check answered not_found: the record is outside the person's access,
// which must look exactly like a missing record (constitution VII).
func isHidden(err error) bool {
	var apiErr *errs.Error
	return errors.As(err, &apiErr) && apiErr.Code == errs.CodeNotFound
}

// orGone answers gone, the answer for a missing record, when an access check found the record outside the
// person's access, and err otherwise.
func orGone(err, gone error) error {
	if isHidden(err) {
		return gone
	}
	return err
}

// find returns the transaction id, nil when there is none.
func (b *book) find(id string) *Transaction {
	i := slices.IndexFunc(b.all, func(t *Transaction) bool { return t.ID == id })
	if i < 0 {
		return nil
	}
	return b.all[i]
}

// transactionView returns t with every field of the details panel, and what a may do with it.
func (s *Service) transactionView(ctx context.Context, a actor.Actor, b *book, t *Transaction) TransactionView {
	r := b.row(t)
	v := TransactionView{
		ID: r.ID, Type: r.Type, Name: r.Name, DocumentNumber: r.DocumentNumber, LineNumber: r.LineNumber,
		PONumber: r.PONumber, POLineNumber: r.POLineNumber, PostingDate: r.PostingDate, AccrualMonth: r.AccrualMonth,
		Amount: r.Amount, AmountDisplay: r.AmountDisplay, CurrencyMissing: r.CurrencyMissing, Vendor: r.Vendor,
		Description: r.Description, SwbsCode: r.SwbsCode, TrackingKey: t.TrackingKey, SentLineItemID: t.SentLineItemID,
		Link: r.Link, Version: t.Version, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, UpdatedBy: t.UpdatedBy,
	}
	if usd := b.usd[t.ID]; usd != nil {
		amount := money.USD(*usd)
		v.AmountUSD = &amount
	}
	for _, i := range b.imports {
		if i.ID == t.ImportID {
			v.Import = i.ref()
		}
	}
	planID := ""
	if li := b.lineItemOf(t); li != nil {
		planID = li.PlanID
	}
	v.CanLink = !t.linked() && s.access.Link(ctx, a, "") == nil
	v.CanUnlink = t.MatchedBy == MatchedByManual && s.access.Link(ctx, a, planID) == nil
	return v
}
