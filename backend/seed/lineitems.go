package seed

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"mapp/internal/activities"
	"mapp/internal/lineitems"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// demoLineItem is one line item of 007 data model §8. plannedCost and custom may be worked out from the
// seeded activity, so the stated balances hold whatever the activity's planned cost is.
type demoLineItem struct {
	plan, activity, name string
	currency             string
	plannedCost          string                                                             // in the currency; "" when computed
	computeCost          func(a activities.Activity, rate money.Rate) (money.Amount, error) // when plannedCost is ""
	method               lineitems.SpreadMethod
	custom               func(a activities.Activity) ([]lineitems.MonthAmount, error) // custom spreads only
	spendType, glCode    string                                                       // "" when empty
	budget, item         string                                                       // finance budget and item names; "" when empty
	requestSwbs          bool
	budgetYear           int // the finance budget year; 0 for demoBudgetYear (016: the 2027 plan's are 2027)
}

// demoLineItems are L-1001 … L-1010, in ID order (L-1009 and L-1010 are the GBP plan's of 016: 15,000.00 GBP and
// 23,000.00 EUR = 19,750.00 GBP, leaving 5,250.00 GBP of "UK launch event" unassigned). Every one has its activity's owner and the finance budget
// year 2026. The Declined activity (Fleet roadshow) has none: it shows the empty state.
var demoLineItems = []demoLineItem{
	{plan: lubricantsPlan, activity: "Launch film", name: "Agency fee", currency: "USD", plannedCost: "50000.00",
		method: lineitems.SpreadEvenly, spendType: "Digital – Platform development", glCode: "7250080 – Advertising digital",
		budget: "2026 Lubricants Budget", item: "Europe – Digital", requestSwbs: true},
	// 46,000.00 EUR = 50,000.00 USD at 0.92: Launch film keeps 20,000.00 USD unassigned; "3 fields needed".
	{plan: lubricantsPlan, activity: "Launch film", name: "Film production", currency: "EUR", plannedCost: "46000.00",
		method: lineitems.SpreadEvenly, requestSwbs: true},
	// 15,800.00 GBP = 20,000.00 USD at 0.79.
	{plan: lubricantsPlan, activity: "Social always-on", name: "Venue hire", currency: "GBP", plannedCost: "15800.00",
		method: lineitems.SpreadFirstMonth, spendType: "BTL – Event", glCode: "7250040 – Events and sponsorship",
		budget: "2026 Lubricants Budget", item: "Europe – Events", requestSwbs: true},
	{plan: lubricantsPlan, activity: "Social always-on", name: "Speaker fees", currency: "USD", plannedCost: "5000.00",
		method: lineitems.SpreadLastMonth, spendType: "BTL – Event", glCode: "7250040 – Events and sponsorship",
		budget: "2026 Lubricants Budget", item: "Europe – Events", requestSwbs: false},
	// A deposit in the month before the activity starts (outside its dates), the rest evenly over its months.
	{plan: lubricantsPlan, activity: "Trade magazine ads", name: "Media booking", currency: "USD", plannedCost: "30000.00",
		method: lineitems.SpreadCustom, custom: depositThenEvenly("6000.00", "24000.00"),
		spendType: "Digital – Platform development", glCode: "7250080 – Advertising digital",
		budget: "2026 Lubricants Budget", item: "Europe – Digital", requestSwbs: true},
	{plan: lubricantsPlan, activity: "Billboard winter wave", name: "Stand build", currency: "EUR", plannedCost: "9200.00",
		method: lineitems.SpreadEvenly, spendType: "BTL – Event", glCode: "7250040 – Events and sponsorship",
		budget: "2026 Lubricants Budget", item: "Europe – Brand", requestSwbs: true},
	// 900.00 of 1,000.00 spread ("Spread doesn't add up…") and no GL code.
	{plan: mobilityPlan, activity: "Grand opening Shanghai", name: "Station signage", currency: "USD", plannedCost: "1000.00",
		method: lineitems.SpreadCustom, custom: firstMonths("450.00", "450.00"), spendType: "BTL – Event",
		budget: "2026 Mobility Budget", item: "China – Retail", requestSwbs: true},
	// (planned cost − 500.00 USD) in CNY: the two line items exceed the activity's planned cost by 500.00 USD.
	{plan: mobilityPlan, activity: "Grand opening Shanghai", name: "Fleet card promotion", currency: "CNY",
		computeCost: exceedBy("500.00", "1000.00"), method: lineitems.SpreadEvenly,
		spendType: "Digital – Platform development", glCode: "7250080 – Advertising digital",
		budget: "2026 Mobility Budget", item: "China – Fleet", requestSwbs: true},
	{plan: ukPlan, activity: "UK launch event", name: "Venue", currency: "GBP", plannedCost: "15000.00",
		method: lineitems.SpreadEvenly, spendType: "BTL – Event", glCode: "7250040 – Events and sponsorship",
		requestSwbs: true, budgetYear: 2027},
	{plan: ukPlan, activity: "UK launch event", name: "Agency fee (Ireland)", currency: "EUR", plannedCost: "23000.00",
		method: lineitems.SpreadEvenly, spendType: "Agency – Creative", glCode: "7250060 – Agency fees",
		requestSwbs: true, budgetYear: 2027},
}

// addLineItems creates the demo line items through the line-items repository, looking up their activities
// by plan and name, currencies by code, finance budget items by year, budget and name, and Shell attribute
// options by label.
func (d *demo) addLineItems(ctx context.Context) error {
	repo := d.stores.LineItems
	activityRepo := d.stores.Activities
	for _, l := range demoLineItems {
		activityID, err := find(d.activities, "activity", l.plan+"/"+l.activity)
		if err != nil {
			return err
		}
		activity, found, err := activityRepo.Get(ctx, activityID)
		if err != nil || !found {
			return cmpErr(err, fmt.Errorf("seed: activity %s is not stored", activityID))
		}
		rate, found := d.rates[l.currency]
		if !found {
			return fmt.Errorf("seed: there is no currency %s", l.currency)
		}
		item, err := d.lineItemFields(l, activity, rate)
		if err != nil {
			return fmt.Errorf("seed: line item %s: %w", l.name, err)
		}
		if item.ID, err = d.next(ctx, ids.PrefixLineItem); err != nil {
			return err
		}
		if err := repo.Put(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

// lineItemFields builds the stored line item of l under activity, whose currency has rate.
func (d *demo) lineItemFields(l demoLineItem, activity activities.Activity, rate money.Rate) (lineitems.LineItem, error) {
	year := cmp.Or(l.budgetYear, demoBudgetYear)
	item := lineitems.LineItem{
		ActivityID: activity.ID, Name: l.name, OwnerID: activity.OwnerID, Currency: l.currency,
		Spread:      lineitems.Spread{Method: l.method, Months: []lineitems.MonthAmount{}},
		RequestSwbs: l.requestSwbs, FinanceBudgetYear: &year, Meta: d.meta(),
	}
	var err error
	if l.plannedCost != "" {
		item.PlannedCost, err = money.ParseAmount(l.plannedCost)
	} else {
		item.PlannedCost, err = l.computeCost(activity, rate)
	}
	if err != nil {
		return lineitems.LineItem{}, err
	}
	if l.custom != nil {
		if item.Spread.Months, err = l.custom(activity); err != nil {
			return lineitems.LineItem{}, err
		}
	}
	values := d.newValues(settings.LevelLineItem)
	if l.spendType != "" {
		values.choice("Spend type", l.spendType)
	}
	if l.glCode != "" {
		values.choice("GL code", l.glCode)
	}
	if item.Attributes, err = values.done(); err != nil {
		return lineitems.LineItem{}, err
	}
	if l.item != "" {
		key := fmt.Sprintf("%d/%s/%s", demoBudgetYear, l.budget, l.item)
		if item.FinanceBudgetItemID, err = find(d.settings.Items, "finance budget item", key); err != nil {
			return lineitems.LineItem{}, err
		}
	}
	return item, nil
}

// depositThenEvenly puts deposit in the month before the activity's start month and spreads rest evenly
// over the activity's months.
func depositThenEvenly(deposit, rest string) func(a activities.Activity) ([]lineitems.MonthAmount, error) {
	return func(a activities.Activity) ([]lineitems.MonthAmount, error) {
		months, err := activityMonths(a)
		if err != nil {
			return nil, err
		}
		first, err := money.ParseAmount(deposit)
		if err != nil {
			return nil, err
		}
		remaining, err := money.ParseAmount(rest)
		if err != nil {
			return nil, err
		}
		before := a.StartDate.Time().AddDate(0, -1, 0).Format("2006-01")
		spread := []lineitems.MonthAmount{{Month: before, Amount: first}}
		for i, part := range remaining.SplitEvenly(len(months)) {
			spread = append(spread, lineitems.MonthAmount{Month: months[i], Amount: part})
		}
		return spread, nil
	}
}

// firstMonths puts the amounts in the activity's first months, one each.
func firstMonths(amounts ...string) func(a activities.Activity) ([]lineitems.MonthAmount, error) {
	return func(a activities.Activity) ([]lineitems.MonthAmount, error) {
		months, err := activityMonths(a)
		if err != nil {
			return nil, err
		}
		if len(months) < len(amounts) {
			return nil, fmt.Errorf("the activity has %d months, the spread needs %d", len(months), len(amounts))
		}
		spread := make([]lineitems.MonthAmount, 0, len(amounts))
		for i, text := range amounts {
			amount, err := money.ParseAmount(text)
			if err != nil {
				return nil, err
			}
			spread = append(spread, lineitems.MonthAmount{Month: months[i], Amount: amount})
		}
		return spread, nil
	}
}

// exceedBy works out the planned cost, in the line item's currency, that makes the activity's line items
// exceed its planned cost by over USD when its other line items cost others USD: planned cost + over −
// others, converted from USD with rate. The seed stops when that would be below 0.00.
func exceedBy(over, others string) func(a activities.Activity, rate money.Rate) (money.Amount, error) {
	return func(a activities.Activity, rate money.Rate) (money.Amount, error) {
		excess, err := money.ParseAmount(over)
		if err != nil {
			return money.Amount{}, err
		}
		rest, err := money.ParseAmount(others)
		if err != nil {
			return money.Amount{}, err
		}
		if a.PlannedCost == nil {
			return money.Amount{}, errors.New("the activity has no planned cost")
		}
		usd := a.PlannedCost.Amount.Add(excess).Sub(rest)
		if usd.Sign() < 0 {
			return money.Amount{}, fmt.Errorf("the activity's planned cost must be at least %s USD", rest.Sub(excess).Format())
		}
		return money.FromUSD(usd, rate), nil
	}
}

// activityMonths returns the calendar months (YYYY-MM) of an activity with a start and an end date.
func activityMonths(a activities.Activity) ([]string, error) {
	if a.StartDate == nil || a.EndDate == nil {
		return nil, fmt.Errorf("activity %s has no dates", a.ID)
	}
	var months []string
	for month := firstOfMonth(*a.StartDate); !month.After(a.EndDate.Time()); month = month.AddDate(0, 1, 0) {
		months = append(months, month.Format("2006-01"))
	}
	return months, nil
}

// firstOfMonth returns the first day of d's month.
func firstOfMonth(d date.Date) time.Time {
	return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// cmpErr returns err when it is set, otherwise missing.
func cmpErr(err, missing error) error {
	if err != nil {
		return err
	}
	return missing
}
