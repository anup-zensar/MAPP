package lineitem

import (
	"cmp"
	"context"
	"fmt"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// shownView converts the USD planned cost to a's display currency as well (D-213).
func (s *Service) shownView(ctx context.Context, a actor.Actor, f ActivityFacts, li LineItem) (View, error) {
	v, err := s.view(ctx, f, li)
	if err != nil {
		return View{}, err
	}
	d, err := s.displayOf(ctx, a)
	if err != nil {
		return View{}, err
	}
	return d.view(v), nil
}

func (s *Service) views(ctx context.Context, f ActivityFacts, items []LineItem) ([]View, error) {
	ctx, err := s.withRates(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]View, 0, len(items))
	for _, li := range items {
		v, err := s.view(ctx, f, li)
		if err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	return views, nil
}

func (s *Service) view(ctx context.Context, f ActivityFacts, li LineItem) (View, error) {
	rates, err := s.rates(ctx)
	if err != nil {
		return View{}, err
	}
	currency, found := rates.currencies[li.Currency]
	if !found {
		return View{}, fmt.Errorf("lineitems: line item %s has the unknown currency %q", li.ID, li.Currency)
	}
	planRate := rates.rateOf(f.planCurrency())
	spread := spreadView(li, f, currency.Rate, planRate)
	v := View{
		ID: li.ID, ActivityID: li.ActivityID, Name: li.Name, Description: li.Description,
		Currency:          rates.view(currency),
		PlannedCost:       money.Money{Amount: li.PlannedCost, Currency: li.Currency},
		PlannedCostUSD:    money.USD(PlannedCostUSD(li, currency.Rate)),
		PlannedCostPlan:   money.Money{Amount: money.Convert(li.PlannedCost, currency.Rate, planRate), Currency: f.planCurrency()},
		Spread:            spread,
		Attributes:        orNoValues(li.Attributes),
		RequestSwbs:       li.RequestSwbs,
		SwbsStatus:        SwbsStatusOf(li, f),
		Swbs:              li.Swbs,
		HXKey:             li.HXKey(),
		FinanceBudgetYear: li.FinanceBudgetYear,
		Meta:              li.Meta,
	}
	v.PlannedCostDisplay = v.PlannedCostUSD // in USD until an answer converts it (display.view)
	if li.ContractedAmount != nil {
		v.ContractedAmount = &money.Money{Amount: *li.ContractedAmount, Currency: li.Currency}
	}
	if v.Owner, err = s.ownerOf(ctx, li.OwnerID); err != nil {
		return View{}, err
	}
	item, err := s.budgetItemOf(ctx, li.FinanceBudgetItemID)
	if err != nil {
		return View{}, err
	}
	if item != nil {
		ref := itemRef(*item)
		v.FinanceBudgetItem = &ref
	}
	attributeMissing, err := s.deps.Attributes.Missing(ctx, li.Attributes, f.BusinessUnit)
	if err != nil {
		return View{}, err
	}
	v.Missing = Missing(li, attributeMissing, item, spread.AddsUp)
	return v, nil
}

func (s *Service) ownerOf(ctx context.Context, id string) (*people.PersonRef, error) {
	if id == "" {
		return nil, nil
	}
	owner, found, err := s.deps.People.Person(ctx, id)
	if err != nil || !found {
		return nil, err
	}
	return &owner, nil
}

func (s *Service) budgetItemOf(ctx context.Context, id string) (*BudgetItem, error) {
	if id == "" {
		return nil, nil
	}
	item, found, err := s.deps.FinanceBudgets.Item(ctx, id)
	if err != nil || !found {
		return nil, err
	}
	return &item, nil
}

// spreadView marks months outside the activity's dates and, for a custom spread, whether it adds up (FR-017).
func spreadView(li LineItem, f ActivityFacts, rate, planRate money.Rate) SpreadView {
	months, spread := SpreadMonths(li, f)
	usd := USDMonths(months, rate)
	plan := ConvertMonths(months, rate, planRate)
	view := SpreadView{Method: li.Spread.Method, State: StateSpread, Months: []SpreadMonth{}, AddsUp: true}
	if !spread {
		view.State = StateNotSpread
	}
	var total money.Amount
	for i, m := range months {
		view.Months = append(view.Months, SpreadMonth{
			Month:                m.Month,
			Amount:               money.Money{Amount: m.Amount, Currency: li.Currency},
			AmountUSD:            money.USD(usd[i]),
			OutsideActivityDates: OutsideActivityDates(m.Month, f),
			AmountPlan:           money.Money{Amount: plan[i], Currency: f.planCurrency()},
		})
		total = total.Add(m.Amount)
	}
	if li.Spread.Method == SpreadCustom {
		view.CustomTotal = &money.Money{Amount: total, Currency: li.Currency}
		view.AddsUp = total.Cmp(li.PlannedCost) == 0
	}
	return view
}

func balanceOf(f ActivityFacts, views []View) UnassignedBalance {
	planned := make([]money.Amount, 0, len(views))
	currencies := make([]string, 0, len(views))
	for _, v := range views {
		planned = append(planned, v.PlannedCostPlan.Amount)
		currencies = append(currencies, v.Currency.Code)
	}
	return Balance(f, planned, currencies)
}

func activityView(f ActivityFacts) ActivityView {
	v := ActivityView{
		ID: f.ID, Name: f.Name, Status: f.Status,
		StartDate: f.StartDate, EndDate: f.EndDate, Ongoing: f.Ongoing,
		Path: orEmpty(f.Path), PlanYear: f.PlanYear, DefaultSpreadMethod: defaultSpread(f),
		BusinessUnitOptionIDs: orEmpty(f.BusinessUnit),
	}
	if f.PlannedCost != nil {
		planned := money.Money{Amount: *f.PlannedCost, Currency: f.planCurrency()}
		v.PlannedCost = &planned
	}
	if f.DefaultFinanceBudget != "" {
		name := f.DefaultFinanceBudget
		v.DefaultFinanceBudget = &name
	}
	return v
}

func defaultSpread(f ActivityFacts) SpreadMethod {
	return cmp.Or(f.DefaultSpread, SpreadEvenly)
}

func currencyView(c settings.Currency) CurrencyView {
	return CurrencyView{Code: c.Code, Name: c.Name, Symbol: c.Symbol, RatePerUSD: c.Rate, Active: c.Active}
}

func itemRef(item BudgetItem) FinanceBudgetItemRef {
	return FinanceBudgetItemRef(item)
}
