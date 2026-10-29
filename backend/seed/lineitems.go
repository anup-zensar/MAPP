package seed

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"mapp/internal/activity"
	"mapp/internal/lineitem"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

type demoLineItem struct {
	plan, activity, name string
	currency             string
	plannedCost          string
	computeCost          func(a activity.Activity, rate money.Rate) (money.Amount, error)
	method               lineitem.SpreadMethod
	custom               func(a activity.Activity) ([]lineitem.MonthAmount, error)
	spendType, glCode    string
	budget, item         string
	requestSwbs          bool
	budgetYear           int
}

// demoLineItems are L-1001 … L-1010, in ID order (L-1009 and L-1010 are the GBP plan's of 016: 15,000.00 GBP and
// 23,000.00 EUR = 19,750.00 GBP, leaving 5,250.00 GBP of "UK launch event" unassigned). Every one has its activity's owner and the finance budget
// year of its plan, the FY2027 ones with an item of 2027 (038 FR-061). Fleet roadshow (Declined) has none: the empty state.
var demoLineItems = []demoLineItem{
	{plan: lubricantsPlan, activity: "Launch film", name: "Agency fee", currency: "USD", plannedCost: "50000.00",
		method: lineitem.SpreadEvenly, spendType: spendDigitalPlatform, glCode: glAdvertisingDigital,
		budget: lubricantsBudget, item: itemEuropeDigital, requestSwbs: true},
	// 46,000.00 EUR = 50,000.00 USD at 0.92: Launch film keeps 20,000.00 USD unassigned; "3 fields needed".
	{plan: lubricantsPlan, activity: "Launch film", name: "Film production", currency: "EUR", plannedCost: "46000.00",
		method: lineitem.SpreadEvenly, requestSwbs: true},
	// 15,800.00 GBP = 20,000.00 USD at 0.79.
	{plan: lubricantsPlan, activity: "Social always-on", name: "Venue hire", currency: "GBP", plannedCost: "15800.00",
		method: lineitem.SpreadFirstMonth, spendType: spendEvent, glCode: glEventsSponsorship,
		budget: lubricantsBudget, item: itemEuropeEvents, requestSwbs: true},
	{plan: lubricantsPlan, activity: "Social always-on", name: "Speaker fees", currency: "USD", plannedCost: "5000.00",
		method: lineitem.SpreadLastMonth, spendType: spendEvent, glCode: glEventsSponsorship,
		budget: lubricantsBudget, item: itemEuropeEvents, requestSwbs: false},
	// A deposit in the month before the activity starts (outside its dates), the rest evenly over its months.
	{plan: lubricantsPlan, activity: "Trade magazine ads", name: "Media booking", currency: "USD", plannedCost: "30000.00",
		method: lineitem.SpreadCustom, custom: depositThenEvenly("6000.00", "24000.00"),
		spendType: spendDigitalPlatform, glCode: glAdvertisingDigital,
		budget: lubricantsBudget, item: itemEuropeDigital, requestSwbs: true},
	{plan: lubricantsPlan, activity: "Billboard winter wave", name: "Stand build", currency: "EUR", plannedCost: "9200.00",
		method: lineitem.SpreadEvenly, spendType: spendEvent, glCode: glEventsSponsorship,
		budget: lubricantsBudget, item: itemEuropeBrand, requestSwbs: true},
	// 900.00 of 1,000.00 spread ("Spread doesn't add up…") and no GL code.
	{plan: mobilityPlan, activity: "Grand opening Shanghai", name: "Station signage", currency: "USD", plannedCost: "1000.00",
		method: lineitem.SpreadCustom, custom: firstMonths("450.00", "450.00"), spendType: spendEvent,
		budget: mobilityBudget, item: itemChinaRetail, requestSwbs: true},
	// (planned cost − 500.00 USD) in CNY: the two line items exceed the activity's planned cost by 500.00 USD.
	{plan: mobilityPlan, activity: "Grand opening Shanghai", name: "Fleet card promotion", currency: "CNY",
		computeCost: exceedBy("500.00", "1000.00"), method: lineitem.SpreadEvenly,
		spendType: spendDigitalPlatform, glCode: glAdvertisingDigital,
		budget: mobilityBudget, item: itemChinaFleet, requestSwbs: true},
	{plan: ukPlan, activity: "UK launch event", name: "Venue", currency: "GBP", plannedCost: "15000.00",
		method: lineitem.SpreadEvenly, spendType: spendEvent, glCode: glEventsSponsorship,
		requestSwbs: true, budgetYear: ukBudgetYear, budget: ukBudget, item: itemUKEvents},
	{plan: ukPlan, activity: "UK launch event", name: "Agency fee (Ireland)", currency: "EUR", plannedCost: "23000.00",
		method: lineitem.SpreadEvenly, spendType: spendAgencyCreative, glCode: glAgencyFees,
		requestSwbs: true, budgetYear: ukBudgetYear, budget: ukBudget, item: itemUKAgency},
}

func (d *demo) addLineItems(ctx context.Context) error {
	return d.putLineItems(ctx, demoLineItems)
}

func (d *demo) putLineItems(ctx context.Context, list []demoLineItem) error {
	repo := d.stores.LineItems
	activityRepo := d.stores.Activities
	for _, l := range list {
		activityID, err := find(d.activities, "activity", l.plan+"/"+l.activity)
		if err != nil {
			return err
		}
		act, found, err := activityRepo.Get(ctx, activityID)
		if err != nil || !found {
			return cmpErr(err, fmt.Errorf("seed: activity %s is not stored", activityID))
		}
		rate, found := d.rates[l.currency]
		if !found {
			return fmt.Errorf("seed: there is no currency %s", l.currency)
		}
		item, err := d.lineItemFields(l, act, rate)
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

func (d *demo) lineItemFields(l demoLineItem, act activity.Activity, rate money.Rate) (lineitem.LineItem, error) {
	year := cmp.Or(l.budgetYear, demoBudgetYear)
	item := lineitem.LineItem{
		ActivityID: act.ID, Name: l.name, OwnerID: act.OwnerID, Currency: l.currency,
		Spread:      lineitem.Spread{Method: l.method, Months: []lineitem.MonthAmount{}},
		RequestSwbs: l.requestSwbs, FinanceBudgetYear: &year, Meta: d.meta(),
	}
	var err error
	if l.plannedCost != "" {
		item.PlannedCost, err = money.ParseAmount(l.plannedCost)
	} else {
		item.PlannedCost, err = l.computeCost(act, rate)
	}
	if err != nil {
		return lineitem.LineItem{}, err
	}
	if l.custom != nil {
		if item.Spread.Months, err = l.custom(act); err != nil {
			return lineitem.LineItem{}, err
		}
	}
	if item.Attributes, err = d.lineItemAttributes(l); err != nil {
		return lineitem.LineItem{}, err
	}
	if l.item != "" {
		key := fmt.Sprintf("%d/%s/%s", year, l.budget, l.item)
		if item.FinanceBudgetItemID, err = find(d.settings.Items, "finance budget item", key); err != nil {
			return lineitem.LineItem{}, err
		}
	}
	return item, nil
}

func (d *demo) lineItemAttributes(l demoLineItem) (settings.AttributeValues, error) {
	values := d.newValues(settings.LevelLineItem)
	if l.spendType != "" {
		values.choice("Spend type", l.spendType)
	}
	if l.glCode != "" {
		values.choice("GL code", l.glCode)
	}
	return values.done()
}

// depositThenEvenly puts deposit in the month before the activity's start, outside its dates.
func depositThenEvenly(deposit, rest string) func(a activity.Activity) ([]lineitem.MonthAmount, error) {
	return func(a activity.Activity) ([]lineitem.MonthAmount, error) {
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
		spread := []lineitem.MonthAmount{{Month: before, Amount: first}}
		for i, part := range remaining.SplitEvenly(len(months)) {
			spread = append(spread, lineitem.MonthAmount{Month: months[i], Amount: part})
		}
		return spread, nil
	}
}

func firstMonths(amounts ...string) func(a activity.Activity) ([]lineitem.MonthAmount, error) {
	return func(a activity.Activity) ([]lineitem.MonthAmount, error) {
		months, err := activityMonths(a)
		if err != nil {
			return nil, err
		}
		if len(months) < len(amounts) {
			return nil, fmt.Errorf("the activity has %d months, the spread needs %d", len(months), len(amounts))
		}
		spread := make([]lineitem.MonthAmount, 0, len(amounts))
		for i, text := range amounts {
			amount, err := money.ParseAmount(text)
			if err != nil {
				return nil, err
			}
			spread = append(spread, lineitem.MonthAmount{Month: months[i], Amount: amount})
		}
		return spread, nil
	}
}

// exceedBy works out the planned cost, in the line item's currency, that makes the activity's line items
// exceed its planned cost by over USD when its other line items cost others USD: planned cost + over −
// others, converted from USD with rate. The seed stops when that would be below 0.00.
func exceedBy(over, others string) func(a activity.Activity, rate money.Rate) (money.Amount, error) {
	return func(a activity.Activity, rate money.Rate) (money.Amount, error) {
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

func activityMonths(a activity.Activity) ([]string, error) {
	if a.StartDate == nil || a.EndDate == nil {
		return nil, fmt.Errorf("activity %s has no dates", a.ID)
	}
	var months []string
	for month := firstOfMonth(*a.StartDate); !month.After(a.EndDate.Time()); month = month.AddDate(0, 1, 0) {
		months = append(months, month.Format("2006-01"))
	}
	return months, nil
}

func firstOfMonth(d date.Date) time.Time {
	return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func cmpErr(err, missing error) error {
	if err != nil {
		return err
	}
	return missing
}
