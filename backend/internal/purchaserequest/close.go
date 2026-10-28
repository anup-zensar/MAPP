package purchaserequest

import (
	"context"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// ClosePreview says what Close purchase request would do, line by line; nothing is saved.
func (s *Service) ClosePreview(ctx context.Context, a actor.Actor, id string) (ClosePreview, error) {
	var preview ClosePreview
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		pr, r, err := s.closable(ctx, a, id)
		if err != nil {
			return err
		}
		preview = closePreview(pr, r)
		return nil
	})
	return preview, err
}

// Close reduces an Approved purchase request to what GSAP invoiced on each line, releases the rest, makes it
// Closed and routes it as Close. Its PO stays as GSAP made it (033 D).
func (s *Service) Close(ctx context.Context, a actor.Actor, id string, in VersionInput) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		pr, r, err := s.closable(ctx, a, id)
		if err != nil {
			return err
		}
		preview := closePreview(pr, r)
		if !preview.Available {
			return errs.NotAllowed(preview.Reason)
		}
		if err = pr.CheckVersion(in.Version, recordNoun); err != nil {
			return err
		}
		if pr, err = s.closed(ctx, a, pr, preview); err != nil {
			return err
		}
		view, err = s.viewOf(ctx, a, pr)
		return err
	})
	return view, err
}

func (s *Service) closable(ctx context.Context, a actor.Actor, id string) (PurchaseRequest, *reader, error) {
	pr, err := s.visible(ctx, id)
	if err != nil {
		return PurchaseRequest{}, nil, err
	}
	r, err := s.readerFor(ctx, a, []PurchaseRequest{pr})
	if err != nil {
		return PurchaseRequest{}, nil, err
	}
	if err = authz.RequireIn(ctx, authz.PurchaseRequestsEdit, pr.planFacts(r.facts)); err != nil {
		return PurchaseRequest{}, nil, err
	}
	return pr, r, nil
}

func closePreview(pr PurchaseRequest, r *reader) ClosePreview {
	p := ClosePreview{PurchaseRequestID: pr.ID, Version: pr.Version, Lines: []CloseLine{}}
	switch {
	case !r.config.PurchaseSupport:
		p.Reason = msgSwitchedOff
	case pr.ChangePending:
		p.Reason = msgChangeInReview
	case pr.Status != StatusApproved:
		p.Reason = msgOnlyApprovedClose
	default:
		p.Available = true
	}
	var released []Line
	for _, l := range pr.Lines {
		invoiced := r.lineInvoiced(l)
		closedAmount := l.Amount
		if invoiced.Amount.Cmp(l.Amount) < 0 {
			closedAmount = max0(invoiced.Amount)
		}
		gone := l.Amount.Sub(closedAmount)
		p.Lines = append(p.Lines, CloseLine{LineID: l.ID, LineItemID: l.LineItemID, HXKey: HXKey(l.LineItemID),
			Amount: l.money(), Invoiced: invoiced, ClosedAmount: money.Money{Amount: closedAmount, Currency: l.Currency},
			Released: money.Money{Amount: gone, Currency: l.Currency}})
		released = append(released, Line{Amount: gone, Currency: l.Currency})
	}
	p.ReleasedDisplay = r.shown(r.usd(released))
	return p
}

// closed applies a close: each line at its closed amount, the purchase request Closed and routed as Close.
func (s *Service) closed(ctx context.Context, a actor.Actor, pr PurchaseRequest, preview ClosePreview) (PurchaseRequest, error) {
	now := s.now()
	var released []string
	for i, c := range preview.Lines {
		if c.Released.Amount.Sign() > 0 {
			released = append(released, c.Released.Format())
		}
		pr.Lines[i].Amount = c.ClosedAmount.Amount
		pr.Lines[i].Touch(a, now)
	}
	pr.Status, pr.ClosedAt = StatusClosed, &now
	pr.Touch(a, now)
	if err := s.route(ctx, a, &pr, routing{typ: RoutingClose}); err != nil {
		return PurchaseRequest{}, err
	}
	if err := s.repo.Put(ctx, pr); err != nil {
		return PurchaseRequest{}, err
	}
	details := "Nothing released"
	if len(released) > 0 {
		details = "Released " + strings.Join(released, " + ")
	}
	_, err := s.history.Log(ctx, a, history.LogInput{Kind: recordKind, RecordID: pr.ID, RecordName: pr.Name,
		Action: history.ActionStatusChanged, Verb: "closed", Details: details})
	return pr, err
}
