package purchaserequest

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

const msgAboveSection = "Requested above planned cost by %s."

// LineItemSection is a line item's Purchase requests section: what is requested, what is left and the
// purchase requests the person sees that draw on it, newest first.
func (s *Service) LineItemSection(ctx context.Context, a actor.Actor, lineItemID string) (LineItemSection, error) {
	var section LineItemSection
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		facts, err := s.deps.LineItems.Facts(ctx, []string{lineItemID})
		if err != nil {
			return err
		}
		f, found := facts[lineItemID]
		if !found || !canSee(ctx, f) {
			return errs.NotFound(authz.NotFoundMessage("line item"))
		}
		if section, err = s.lineItemTotals(ctx, f); err != nil {
			return err
		}
		section.PurchaseRequests, err = s.lineItemRows(ctx, a, f)
		return err
	})
	return section, err
}

func (s *Service) lineItemTotals(ctx context.Context, f LineItemFacts) (LineItemSection, error) {
	b, err := s.balanceOf(ctx, []string{f.ID})
	if err != nil {
		return LineItemSection{}, err
	}
	requested, err := b.requested(f, "")
	if err != nil {
		return LineItemSection{}, err
	}
	config, err := s.repo.Configuration(ctx)
	if err != nil {
		return LineItemSection{}, err
	}
	left := f.PlannedCost.Sub(requested)
	section := LineItemSection{
		LineItemID: f.ID, HXKey: HXKey(f.ID), PlannedCost: money.Money{Amount: f.PlannedCost, Currency: f.Currency},
		Requested: money.Money{Amount: requested, Currency: f.Currency}, PurchaseRequests: []LineItemPRRow{},
		AvailableToRequest: money.Money{Amount: left, Currency: f.Currency},
		Create:             authz.Available(ctx, authz.PurchaseRequestsEdit, f.Plan.Access),
	}
	switch {
	case !config.PurchaseSupport:
		section.Create, section.Notice = authz.Unavailable(msgSwitchedOff), msgSwitchedOff
	case left.Sign() < 0:
		section.Create = authz.Unavailable(msgNothingLeft)
		section.Notice = fmt.Sprintf(msgAboveSection, money.Money{Amount: left.Neg(), Currency: f.Currency}.Format())
	case left.IsZero():
		section.Create = authz.Unavailable(msgNothingLeft)
	}
	return section, nil
}

func (s *Service) lineItemRows(ctx context.Context, a actor.Actor, f LineItemFacts) ([]LineItemPRRow, error) {
	prs, err := s.repo.OnLineItems(ctx, viewScope(ctx), []string{f.ID})
	if err != nil {
		return nil, err
	}
	r, err := s.readerFor(ctx, a, prs)
	if err != nil {
		return nil, err
	}
	rows := make([]LineItemPRRow, 0, len(prs))
	for _, pr := range slices.Backward(prs) {
		i := slices.IndexFunc(pr.Lines, func(l Line) bool { return l.LineItemID == f.ID })
		if i < 0 {
			continue
		}
		l := pr.Lines[i]
		converted, err := r.rates.Convert(l.money(), f.Currency)
		if err != nil {
			return nil, err
		}
		rows = append(rows, LineItemPRRow{PurchaseRequestID: pr.ID, Name: pr.Name, Status: pr.Status,
			ChangePending: pr.ChangePending, LineID: l.ID, Amount: l.money(), AmountInLineItemCurrency: converted,
			SentOn: l.SentOn, PONumber: l.PO.Number, Owner: r.person(pr.OwnerID)})
	}
	return rows, nil
}

// ActivitySection is an activity's Purchase requests section: Total requested in the plan currency.
func (s *Service) ActivitySection(ctx context.Context, _ actor.Actor, activityID string) (ActivitySection, error) {
	var section ActivitySection
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		act, found, err := s.deps.LineItems.Activity(ctx, activityID)
		if err != nil {
			return err
		}
		if !found || !authz.CanViewActivity(ctx, act.Access) {
			return errs.NotFound(authz.NotFoundMessage("activity"))
		}
		section, err = s.activityTotals(ctx, act)
		return err
	})
	return section, err
}

func (s *Service) activityTotals(ctx context.Context, act ActivityLineItems) (ActivitySection, error) {
	ids := make([]string, 0, len(act.LineItems))
	for _, f := range act.LineItems {
		ids = append(ids, f.ID)
	}
	b, err := s.balanceOf(ctx, ids)
	if err != nil {
		return ActivitySection{}, err
	}
	section := ActivitySection{ActivityID: act.ID, LineItems: []ActivityLineItem{},
		Requested: money.Money{Currency: act.Plan.Currency}}
	for _, f := range act.LineItems {
		requested, err := b.requested(f, "")
		if err != nil {
			return ActivitySection{}, err
		}
		inPlan, err := b.rates.Convert(money.Money{Amount: requested, Currency: f.Currency}, act.Plan.Currency)
		if err != nil {
			return ActivitySection{}, err
		}
		section.Requested.Amount = section.Requested.Amount.Add(inPlan.Amount)
		section.LineItems = append(section.LineItems, ActivityLineItem{LineItemID: f.ID, Name: f.Name,
			Requested: money.Money{Amount: requested, Currency: f.Currency}, PurchaseRequests: claimsOn(b.claims, f.ID),
			AvailableToRequest: money.Money{Amount: f.PlannedCost.Sub(requested), Currency: f.Currency}})
	}
	return section, nil
}

func claimsOn(claims []Claim, lineItemID string) int {
	var prs []string
	for _, c := range claims {
		if c.LineItemID == lineItemID && !slices.Contains(prs, c.PurchaseRequestID) {
			prs = append(prs, c.PurchaseRequestID)
		}
	}
	return len(prs)
}
