package purchaserequest

import (
	"context"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/money"
)

// Drawn is a purchase request as the SP&A view counts it (feature 013): only its lines on the asked line items.
type Drawn struct {
	ID     string
	Name   string
	Vendor string
	Status Status
	Lines  []DrawnLine
}

type DrawnLine struct {
	LineItemID string
	USD        money.Amount
}

// DrawnOnLineItems returns the purchase requests the person acting sees that draw on the line items, in id
// order, each with its lines on them in USD at today's rates (a currency without a rate counts at 1).
func (s *Service) DrawnOnLineItems(ctx context.Context, lineItemIDs []string) ([]Drawn, error) {
	var list []Drawn
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		prs, err := s.repo.OnLineItems(ctx, viewScope(ctx), lineItemIDs)
		if err != nil {
			return err
		}
		rates, err := s.deps.Currencies.Rates(ctx)
		if err != nil {
			return err
		}
		r, asked := &reader{rates: rates}, make(map[string]bool, len(lineItemIDs))
		for _, id := range lineItemIDs {
			asked[id] = true
		}
		list = make([]Drawn, 0, len(prs))
		for _, pr := range prs {
			list = append(list, drawnOf(r, pr, asked))
		}
		return nil
	})
	return list, err
}

func drawnOf(r *reader, pr PurchaseRequest, asked map[string]bool) Drawn {
	d := Drawn{ID: pr.ID, Name: pr.Name, Vendor: pr.Vendor, Status: pr.Status, Lines: []DrawnLine{}}
	for _, l := range pr.Lines {
		if asked[l.LineItemID] {
			d.Lines = append(d.Lines, DrawnLine{LineItemID: l.LineItemID, USD: r.usd([]Line{l})})
		}
	}
	return d
}

// RaiseAvailability says whether the person acting may raise a purchase request in the plan now, and why not.
func (s *Service) RaiseAvailability(ctx context.Context, plan authz.PlanFacts) (authz.Availability, error) {
	var available authz.Availability
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.repo.Configuration(ctx)
		available = authz.Available(ctx, authz.PurchaseRequestsEdit, plan)
		if err == nil && !c.PurchaseSupport {
			available = authz.Unavailable(msgSwitchedOff)
		}
		return err
	})
	return available, err
}
