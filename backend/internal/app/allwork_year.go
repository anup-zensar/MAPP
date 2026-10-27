package app

import (
	"context"
	"slices"

	"mapp/internal/allwork"
	"mapp/internal/people"
	"mapp/internal/plans"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// yearReader collects the nodes of one year.
type yearReader struct {
	s             allWorkSource
	columns       []string // the columns the view shows or sorts by
	nodes         []allwork.Node
	owners        map[string]*allwork.Owner // read once each
	attributes    map[settings.Level][]settings.Attribute
	options       map[string]settings.Option // the options of the choice attributes, by id
	initiativeIDs []string
	activities    map[string]int    // activity id → index of its node
	stored        map[string]string // activity id → its stored status (for the SWBS request status)
	rates         money.Rates       // the current rates (016)
	plans         map[string]plans.Plan
}

// planRate returns the current rate of the plan currency of the plan planID (016 FR-014); 1 for USD.
func (y *yearReader) planRate(planID string) money.Rate {
	if rate, err := y.rates.Of(y.plans[planID].Currency); err == nil {
		return rate
	}
	return money.One
}

// usd converts an amount of the plan planID's currency to USD (016 FR-014): once, half up; nil stays nil.
func (y *yearReader) usd(planID string, amount *money.Amount) *money.Amount {
	if amount == nil {
		return nil
	}
	usd := money.Convert(*amount, y.planRate(planID), money.One)
	return &usd
}

// inPlan returns an amount of the plan planID's currency as money; nil stays nil.
func (y *yearReader) inPlan(planID string, amount *money.Amount) *money.Money {
	if amount == nil {
		return nil
	}
	return &money.Money{Amount: *amount, Currency: y.plans[planID].Currency}
}

// phased returns the 12 phased monthly budgets of the plan planID's year, in the plan currency and in USD
// (016 FR-024, FR-038); nil, nil when not phased.
func (y *yearReader) phased(planID string, monthly []money.Amount) ([]money.Amount, []allwork.MonthAmount) {
	if len(monthly) != 12 {
		return nil, nil
	}
	keys := rollup.MonthKeys(y.plans[planID].Year)
	usd := make([]allwork.MonthAmount, 0, 12)
	for i, amount := range money.ConvertParts(monthly, y.planRate(planID), money.One) {
		usd = append(usd, allwork.MonthAmount{Month: keys[i], USD: amount})
	}
	return monthly, usd
}

// planFacts sets the plan fields every node of the plan planID carries (016 data-model §5.1).
func (y *yearReader) planFacts(node *allwork.Node) {
	p := y.plans[node.PlanID]
	node.PlanYear, node.PlanType, node.PlanCurrency = p.Year, string(p.PlanType), p.Currency
}

// needs reports whether the view shows or sorts by one of the columns keys.
func (y *yearReader) needs(keys ...string) bool {
	return slices.ContainsFunc(keys, func(key string) bool { return slices.Contains(y.columns, key) })
}

// owner returns the person id as an owner, nil for "". A person who is not found is shown by their ID.
func (y *yearReader) owner(ctx context.Context, id string) (*allwork.Owner, error) {
	if id == "" {
		return nil, nil
	}
	if owner, found := y.owners[id]; found {
		return owner, nil
	}
	owner := &allwork.Owner{ID: id, Name: id}
	p, found, err := y.s.peopleRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if found {
		owner = &allwork.Owner{ID: p.ID, Name: p.Name, Inactive: !p.Active}
	}
	y.owners[id] = owner
	return owner, nil
}

// ownerOf returns a person as an owner, nil for none.
func ownerOf(ref *people.PersonRef) *allwork.Owner {
	if ref == nil {
		return nil
	}
	return &allwork.Owner{ID: ref.ID, Name: ref.Name, Inactive: !ref.Active}
}
