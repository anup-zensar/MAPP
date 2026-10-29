package spa

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"mapp/internal/platform/money"
)

// activity is one activity of the scope with its figures in USD, counted only in the plan's year (D-336).
type activity struct {
	rec          *Record
	campaign     *Record
	initiative   *Record
	lineItems    []*lineItem
	planned      money.Amount
	months       [12]money.Amount
	notScheduled money.Amount // planned cost not placed in a month, never below 0 (038 FR-023)
	above        money.Amount // months above the planned cost: line items exceeding it
	outside      money.Amount
	spend        Spend
	actual       [12]money.Amount
}

type lineItem struct {
	rec     *Record
	planned money.Amount
	spend   Spend
}

type tree struct {
	byID     map[string]*Record
	children map[string][]*Record
}

func newTree(records []Record) tree {
	t := tree{byID: make(map[string]*Record, len(records)), children: map[string][]*Record{}}
	for i := range records {
		r := &records[i]
		t.byID[r.ID] = r
		t.children[r.ParentID] = append(t.children[r.ParentID], r)
	}
	for _, list := range t.children {
		slices.SortFunc(list, func(x, y *Record) int { return cmp.Or(cmp.Compare(x.Order, y.Order), compareIDs(x.ID, y.ID)) })
	}
	return t
}

func compareIDs(x, y string) int {
	return cmp.Or(cmp.Compare(len(x), len(y)), strings.Compare(x, y))
}

func (t tree) kids(id string, level Level) []*Record {
	var list []*Record
	for _, r := range t.children[id] {
		if r.Level == level {
			list = append(list, r)
		}
	}
	return list
}

func (t tree) activitiesUnder(campaigns []*Record) []*activity {
	var list []*activity
	for _, c := range campaigns {
		for _, i := range t.kids(c.ID, LevelInitiative) {
			for _, a := range t.kids(i.ID, LevelActivity) {
				act := &activity{rec: a, campaign: c, initiative: i}
				for _, li := range t.kids(a.ID, LevelLineItem) {
					act.lineItems = append(act.lineItems, &lineItem{rec: li})
				}
				list = append(list, act)
			}
		}
	}
	return list
}

// placePlanned spreads the activity's planned cost over the months of year: each line item by its spread, a
// positive unassigned balance evenly over the activity's months (All work's Budget & cost rule); what stays
// unplaced is not scheduled, what the months hold above the planned cost is apart (038 FR-023).
func (a *activity) placePlanned(year int) {
	typed := amountOf(a.rec.PlannedCost)
	var onLineItems money.Amount
	for _, li := range a.lineItems {
		li.planned = amountOf(li.rec.PlannedCost)
		onLineItems = onLineItems.Add(li.planned)
		for _, m := range li.rec.Spread {
			if !a.place(year, m.Month, m.USD) {
				li.planned = li.planned.Sub(m.USD)
			}
		}
	}
	if balance := typed.Sub(onLineItems); balance.Sign() > 0 && len(a.rec.Months) > 0 {
		for i, part := range balance.SplitEvenly(len(a.rec.Months)) {
			a.place(year, a.rec.Months[i], part)
		}
	}
	a.planned = typed.Sub(a.outside)
	if left := a.planned.Sub(money.Sum(a.months[:]...)); left.Sign() > 0 {
		a.notScheduled = left
	} else {
		a.above = left.Neg()
	}
}

// place adds amount to its month when the month is in year and reports true; otherwise it counts as outside.
func (a *activity) place(year int, month string, amount money.Amount) bool {
	i, in := monthIndex(year, month)
	if !in {
		a.outside = a.outside.Add(amount)
		return false
	}
	a.months[i] = a.months[i].Add(amount)
	return true
}

// placeSpend adds the transactions of the activity's line items posted in year, and returns those posted in
// other years.
func (a *activity) placeSpend(year int, byLineItem map[string]map[string]Spend) Spend {
	var outside Spend
	for _, li := range a.lineItems {
		for month, s := range byLineItem[li.rec.ID] {
			i, in := monthIndex(year, month)
			if !in {
				outside = outside.plus(s)
				continue
			}
			li.spend, a.spend = li.spend.plus(s), a.spend.plus(s)
			a.actual[i] = a.actual[i].Add(s.Actual)
		}
	}
	return outside
}

// monthIndex is 0–11 for a month "2026-03" of year, and false for any other month.
func monthIndex(year int, month string) (int, bool) {
	y, m, found := strings.Cut(month, "-")
	number, err := strconv.Atoi(m)
	if !found || y != strconv.Itoa(year) || err != nil || number < 1 || number > 12 {
		return 0, false
	}
	return number - 1, true
}

func monthKey(year, index int) string {
	return fmt.Sprintf("%04d-%02d", year, index+1)
}

func amountOf(a *money.Amount) money.Amount {
	if a == nil {
		return money.Amount{}
	}
	return *a
}
