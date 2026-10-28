package purchaserequest

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

const (
	maxOptions    = 50
	maxOptionRead = 500
	maxSearch     = 120
)

// OptionQuery is the line-item picker's question (contract listPurchaseRequestLineItemOptions).
type OptionQuery struct {
	Search, PurchaseRequestID, CompanyCode, PlanID string
}

// pickerContext is the purchase request the picker adds to: its plan, company code and line items.
type pickerContext struct {
	planID, companyCode string
	onPR                []string
}

// Options lists at most 50 live line items the person may raise purchase requests for, matching the search, with
// what each has left and why it cannot be chosen.
func (s *Service) Options(ctx context.Context, _ actor.Actor, q OptionQuery) (OptionPage, error) {
	if utf8.RuneCountInString(q.Search) > maxSearch {
		return OptionPage{}, errs.Validation(errs.FieldError{Field: "search", Message: msgNameTooLong})
	}
	page := OptionPage{Items: []LineItemOption{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		pc, err := s.pickerOf(ctx, q)
		if err != nil {
			return err
		}
		found, err := s.deps.LineItems.Search(ctx, authz.PermissionScopeOf(ctx, authz.PurchaseRequestsEdit),
			LineItemSearch{Text: strings.TrimSpace(q.Search), PlanID: pc.planID, Limit: maxOptionRead})
		if err != nil {
			return err
		}
		found = slices.DeleteFunc(found, func(f LineItemFacts) bool { return !canSee(ctx, f) })
		page.Total = len(found)
		page.Items, err = s.options(ctx, pc, found[:min(maxOptions, len(found))])
		return err
	})
	return page, err
}

func (s *Service) pickerOf(ctx context.Context, q OptionQuery) (pickerContext, error) {
	pc := pickerContext{planID: q.PlanID, companyCode: q.CompanyCode}
	if q.PurchaseRequestID == "" {
		return pc, nil
	}
	pr, err := s.visible(ctx, q.PurchaseRequestID)
	if err != nil {
		return pickerContext{}, err
	}
	facts, err := s.factsOf(ctx, pr)
	if err != nil {
		return pickerContext{}, err
	}
	pc.planID, pc.onPR = pr.PlanID, pr.lineItemIDs()
	for _, l := range pr.Lines {
		if code := facts[l.LineItemID].CompanyCode; code != "" && pc.companyCode == "" {
			pc.companyCode = code
		}
	}
	return pc, nil
}

func (s *Service) options(ctx context.Context, pc pickerContext, found []LineItemFacts) ([]LineItemOption, error) {
	ids := make([]string, 0, len(found))
	for _, f := range found {
		ids = append(ids, f.ID)
	}
	b, err := s.balanceOf(ctx, ids)
	if err != nil {
		return nil, err
	}
	list := make([]LineItemOption, 0, len(found))
	for _, f := range found {
		left, err := b.available(f, "")
		if err != nil {
			return nil, err
		}
		reason := pc.reason(f, left)
		list = append(list, LineItemOption{LineItem: lineItemRefOf(f), Plan: planRef(f.Plan), Path: f.Path,
			HXKey: HXKey(f.ID), AvailableToRequest: money.Money{Amount: left, Currency: f.Currency}, Available: reason == "",
			Reason: reason})
	}
	return list, nil
}

func (pc pickerContext) reason(f LineItemFacts, left money.Amount) string {
	switch {
	case slices.Contains(pc.onPR, f.ID):
		return msgAlreadyOnPR
	case left.Sign() <= 0:
		return msgNothingLeft
	case pc.companyCode != "" && f.CompanyCode != "" && f.CompanyCode != pc.companyCode:
		return fmt.Sprintf(msgTwoCompanyCodes, pc.companyCode, f.CompanyCode)
	}
	return ""
}
