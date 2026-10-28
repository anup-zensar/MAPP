package purchaserequest

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// Create raises a purchase request, Created and reserved on its line items at once whatever their activities'
// status. Raising needs purchaseRequests.edit in the plan of the lines.
func (s *Service) Create(ctx context.Context, a actor.Actor, in Input) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		pr, err := s.created(ctx, a, in)
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, a, pr)
		return err
	})
	return view, err
}

func (s *Service) created(ctx context.Context, a actor.Actor, in Input) (PurchaseRequest, error) {
	ck, err := s.checkCreate(ctx, in)
	if err != nil {
		return PurchaseRequest{}, err
	}
	pr, err := s.newRecord(ctx, a, ck.pr)
	if err != nil {
		return PurchaseRequest{}, err
	}
	if err = s.repo.Put(ctx, pr); err != nil {
		return PurchaseRequest{}, err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{Kind: recordKind, RecordID: pr.ID, RecordName: pr.Name,
		Action: history.ActionCreated, Details: linesText(pr.Lines)})
	return pr, err
}

// checkCreate refuses a person without purchaseRequests.edit in the plan of the lines (403) before it answers
// the problems of the purchase request all at once (400).
func (s *Service) checkCreate(ctx context.Context, in Input) (checked, error) {
	if _, err := s.switchedOn(ctx); err != nil {
		return checked{}, err
	}
	ck, err := s.check(ctx, in, PurchaseRequest{Attributes: settings.AttributeValues{}})
	if err != nil {
		return checked{}, err
	}
	if ck.plan.Access.ID != "" {
		if err = authz.RequireIn(ctx, authz.PurchaseRequestsEdit, ck.plan.Access); err != nil {
			return checked{}, err
		}
	}
	if in.Then != ThenSave {
		ck.problems.Add("then", fmt.Sprintf(msgChooseThen, "Save"))
	}
	return ck, ck.problems.Err()
}

// newRecord gives the purchase request and its lines their IDs and bookkeeping: Created, no GSAP answer yet.
func (s *Service) newRecord(ctx context.Context, a actor.Actor, pr PurchaseRequest) (PurchaseRequest, error) {
	var err error
	if pr.ID, err = s.ids.Next(ctx, ids.PrefixPurchaseRequest); err != nil {
		return PurchaseRequest{}, err
	}
	pr.Status, pr.Meta = StatusCreated, record.NewMeta(a, s.now())
	for i := range pr.Lines {
		if pr.Lines[i], err = s.newLine(ctx, a, pr.ID, pr.Lines[i]); err != nil {
			return PurchaseRequest{}, err
		}
	}
	return pr, nil
}

func (s *Service) newLine(ctx context.Context, a actor.Actor, prID string, l Line) (Line, error) {
	id, err := s.ids.Next(ctx, ids.PrefixPurchaseRequestLine)
	if err != nil {
		return Line{}, err
	}
	l.ID, l.PurchaseRequestID, l.Meta = id, prID, record.NewMeta(a, s.now())
	return l, nil
}

// linesText is the log's summary of lines: "1 line, 9,200.00 EUR", or per currency "2 lines, 100.00 EUR + 5.00 USD".
func linesText(lines []Line) string {
	sums := map[string]money.Amount{}
	var order []string
	for _, l := range lines {
		if _, seen := sums[l.Currency]; !seen {
			order = append(order, l.Currency)
		}
		sums[l.Currency] = sums[l.Currency].Add(l.Amount)
	}
	amounts := make([]string, 0, len(order))
	for _, code := range order {
		amounts = append(amounts, money.Money{Amount: sums[code], Currency: code}.Format())
	}
	return lineCount(len(lines)) + ", " + strings.Join(amounts, " + ")
}
