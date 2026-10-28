package purchaserequest

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// Get returns the purchase request's details page; one the person cannot see answers not found.
func (s *Service) Get(ctx context.Context, a actor.Actor, id string) (View, error) {
	var view View
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		pr, err := s.visible(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, a, pr)
		return err
	})
	return view, err
}

// Defaults is what New purchase request from lineItemID is pre-filled with: "PR for <name>",
// the acting person as owner and one line for everything the line item has left to request.
func (s *Service) Defaults(ctx context.Context, a actor.Actor, lineItemID string) (Defaults, error) {
	var d Defaults
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.raisable(ctx, lineItemID)
		if err != nil {
			return err
		}
		left, err := s.leftToRequest(ctx, f)
		if err != nil {
			return err
		}
		if left.Sign() <= 0 {
			return errs.Validation(errs.FieldError{Field: "lineItemId", Message: msgNothingLeft})
		}
		found, err := s.deps.People.People(ctx, []string{a.ID})
		if err != nil {
			return err
		}
		amount := money.Money{Amount: left, Currency: f.Currency}
		d = Defaults{Name: "PR for " + f.Name, Owner: found[a.ID].PersonRef, CompanyCode: f.CompanyCode,
			Plan: planRef(f.Plan), Line: DefaultLine{LineItem: lineItemRefOf(f), HXKey: HXKey(f.ID), Currency: f.Currency,
				Amount: amount, OverallLimit: amount}}
		return nil
	})
	return d, err
}

// raisable returns the live line item when the person sees it (404 otherwise), may raise purchase requests in its
// plan (403) and Purchase Support is on (409).
func (s *Service) raisable(ctx context.Context, lineItemID string) (LineItemFacts, error) {
	facts, err := s.deps.LineItems.Facts(ctx, []string{lineItemID})
	if err != nil {
		return LineItemFacts{}, err
	}
	f, found := facts[lineItemID]
	if !found || !canSee(ctx, f) {
		return LineItemFacts{}, errs.NotFound(authz.NotFoundMessage("line item"))
	}
	if err = authz.RequireIn(ctx, authz.PurchaseRequestsEdit, f.Plan.Access); err != nil {
		return LineItemFacts{}, err
	}
	_, err = s.switchedOn(ctx)
	return f, err
}

func (s *Service) leftToRequest(ctx context.Context, f LineItemFacts) (money.Amount, error) {
	b, err := s.balanceOf(ctx, []string{f.ID})
	if err != nil {
		return money.Amount{}, err
	}
	return b.available(f, "")
}

func (s *Service) balanceOf(ctx context.Context, lineItemIDs []string) (balance, error) {
	rates, err := s.deps.Currencies.Rates(ctx)
	if err != nil {
		return balance{}, err
	}
	claims, err := s.repo.Claims(ctx, lineItemIDs)
	return balance{rates: rates, claims: claims}, err
}

func planRef(p PlanFacts) errs.RecordRef {
	return errs.RecordRef{Type: planType, ID: p.Access.ID, Name: p.Name}
}

func lineItemRefOf(f LineItemFacts) LineItemRef {
	r := reader{facts: map[string]LineItemFacts{f.ID: f}}
	return r.lineItemRef(f.ID)
}
