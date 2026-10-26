package lineitems

import (
	"context"
	"maps"
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/memo"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// Provided to other features (plan, "Provided to other features"): what activities (006), plans (004),
// campaigns (005) and approvals (008) read about line items, through adapters in internal/app.

// Summary is one activity's line items in short (data model §3.9).
type Summary struct {
	Count          int
	PlannedCostUSD money.Amount // sum of the line items' USD planned costs at today's rates
	// PlannedCostPlan is the sum of the line items' planned costs in Currency, the plan currency (016 FR-015).
	PlannedCostPlan money.Amount
	Currency        string
	AutoSpread      int          // spread Evenly, First or Last month: follows the activity's dates
	CustomSpread    int          // custom spread: kept as typed
	Incomplete      []Incomplete // the line items whose missing list is not empty, in creation order
}

// Incomplete is a line item that still stops its activity from being submitted, with what it misses (data
// model §3.6).
type Incomplete struct {
	LineItemID, LineItemName string
	Missing                  []settings.MissingField
}

// Summaries returns the summary of each activity in activityIDs that has line items; an activity without
// line items, or one that is not stored, has no entry. It runs inside the caller's transaction (it joins
// it), so a change shows at once: nothing here is stored. A summary is worked out once per transaction
// until data changes: the plan levels and the activities ask for the same activities' summaries.
func (s *Service) Summaries(ctx context.Context, activityIDs []string) (map[string]Summary, error) {
	summaries := map[string]Summary{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		kept, err := memo.Many(ctx, memo.Derived+":lineItemSummary:", activityIDs, s.summariesOf)
		for id, summary := range kept {
			if summary != nil {
				summaries[id] = *summary
			}
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return summaries, nil
}

// summariesOf works out the summaries of the activities with line items among activityIDs.
func (s *Service) summariesOf(ctx context.Context, activityIDs []string) (map[string]*Summary, error) {
	summaries := map[string]*Summary{}
	items, err := s.repo.ListByActivities(ctx, activityIDs)
	if err != nil || len(items) == 0 {
		return summaries, err
	}
	if ctx, err = s.withRates(ctx); err != nil { // once for every activity (016: rates are dated records)
		return nil, err
	}
	byActivity := map[string][]LineItem{}
	for _, li := range items {
		byActivity[li.ActivityID] = append(byActivity[li.ActivityID], li)
	}
	for activityID, list := range byActivity {
		f, found, err := s.deps.Activities.Facts(ctx, activityID)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		views, err := s.views(ctx, f, list)
		if err != nil {
			return nil, err
		}
		summaries[activityID] = summaryOf(f, views)
	}
	return summaries, nil
}

func summaryOf(f ActivityFacts, views []View) *Summary {
	summary := &Summary{Count: len(views), Incomplete: []Incomplete{}, Currency: f.planCurrency()}
	for _, v := range views {
		summary.PlannedCostUSD = summary.PlannedCostUSD.Add(v.PlannedCostUSD.Amount)
		summary.PlannedCostPlan = summary.PlannedCostPlan.Add(v.PlannedCostPlan.Amount)
		if v.Spread.Method == SpreadCustom {
			summary.CustomSpread++
		} else {
			summary.AutoSpread++
		}
		if len(v.Missing) > 0 {
			summary.Incomplete = append(summary.Incomplete, Incomplete{LineItemID: v.ID, LineItemName: v.Name, Missing: v.Missing})
		}
	}
	return summary
}

// CopyLineItems copies every line item of the activity fromActivityID into the activity toActivityID (data
// model §5, R13): every stored field and the same names, with new IDs and the actor as creator. With a
// yearShift of 1 — a copy into the next year's plan — the copies also move one year on (D-148, see
// shiftYears). It joins the caller's Write and writes no log entries: the feature that copies writes the one
// "copied" entry of the person's action (D-110). It returns how many line items it copied. It is the CopyAll
// of feature 006's activities.LineItems, for single activity copies and the copies of campaigns and
// initiatives.
func (s *Service) CopyLineItems(ctx context.Context, a actor.Actor, fromActivityID, toActivityID string, yearShift int) (int, error) {
	newIDs, err := s.CopyForActivities(ctx, a, map[string]string{fromActivityID: toActivityID}, yearShift)
	return len(newIDs), err
}

// CopyForActivities copies the line items of every activity of newActivityIDs (the id of a copied activity →
// the id of its copy) into that copy, as CopyLineItems does, and returns the old → new ID of every line item
// it copied. It is the Copy of the lineItem level of feature 004's plan copy (D-195), which runs after the
// activity level has copied the activities: each line item is copied once.
func (s *Service) CopyForActivities(ctx context.Context, a actor.Actor, newActivityIDs map[string]string, yearShift int) (map[string]string, error) {
	newIDs := map[string]string{}
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		items, err := s.repo.ListByActivities(ctx, slices.Sorted(maps.Keys(newActivityIDs)))
		if err != nil {
			return err
		}
		var budgetItems []BudgetItem // read only for a copy into another year
		if yearShift != 0 && len(items) > 0 {
			if budgetItems, err = s.deps.FinanceBudgets.Items(ctx); err != nil {
				return err
			}
		}
		for _, li := range items { // by id, so the copies get their IDs in the same order
			c := copyOf(li)
			c.ActivityID = newActivityIDs[li.ActivityID]
			if c.ID, err = s.ids.Next(ctx, ids.PrefixLineItem); err != nil {
				return err
			}
			c.Meta = record.NewMeta(a, s.now())
			if yearShift != 0 {
				shiftYears(&c, yearShift, budgetItems)
			}
			if err := s.repo.Put(ctx, c); err != nil {
				return err
			}
			newIDs[li.ID] = c.ID
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return newIDs, nil
}

// shiftYears moves the copy li of a line item years on (D-148, 004 FR-028): its finance budget year, and its
// custom months; its finance budget item becomes the active item of the new year with the same code, when
// the old item has a code and exactly one such item exists, and otherwise none — a cleared link shows as
// needed, never as silently wrong. items are every finance budget item of every year.
func shiftYears(li *LineItem, years int, items []BudgetItem) {
	if li.FinanceBudgetYear != nil {
		year := *li.FinanceBudgetYear + years
		li.FinanceBudgetYear = &year
	}
	if li.FinanceBudgetItemID != "" {
		li.FinanceBudgetItemID = sameCodeIn(li.FinanceBudgetItemID, years, items)
	}
	for i, m := range li.Spread.Months {
		li.Spread.Months[i].Month = shiftMonth(m.Month, years)
	}
}

// sameCodeIn returns the id of the only active finance budget item, years after the item itemID's year,
// with the item's code; "" when the item is unknown, has no code, or there is no such item or more than one.
func sameCodeIn(itemID string, years int, items []BudgetItem) string {
	i := slices.IndexFunc(items, func(item BudgetItem) bool { return item.ID == itemID })
	if i < 0 || items[i].Code == "" {
		return ""
	}
	old := items[i]
	var found []string
	for _, item := range items {
		if item.Active && item.Year == old.Year+years && item.Code == old.Code {
			found = append(found, item.ID)
		}
	}
	if len(found) != 1 {
		return ""
	}
	return found[0]
}

// shiftMonth returns the month (YYYY-MM) years later; a month it cannot read stays as it is.
func shiftMonth(month string, years int) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return t.AddDate(years, 0, 0).Format("2006-01")
}
