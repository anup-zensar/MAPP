package allwork

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/money"
	"mapp/internal/rollup"
)

// breakdownWork is one Breakdown being worked out.
type breakdownWork struct {
	in           input
	t            *tree
	by           Grouping
	thenBy       *Grouping
	planCurrency string // the plan currency of amountsIn=plan; "" for the display currency
	units        map[string][]string

	countsInEach bool
	withoutPC    money.Amount // line items of activities without a planned cost (line-item groupings)
}

// piece is a part of an activity's planned cost that goes to groups: the whole activity, one of its line items,
// or its unassigned planned cost (line-item groupings).
type piece struct {
	activity   *tnode
	lineItem   *tnode // nil for the whole activity and the unassigned part
	unassigned bool
	amount     money.Amount
}

// value is one group of a piece with the share it gets: nil for the full amount.
type value struct {
	key, label string
	order      int
	share      *money.Percent
}

// accum adds up one group.
type accum struct {
	value
	activities map[*tnode]bool
	lineItems  map[*tnode]bool
	planned    money.Amount
	children   map[string]*accum
}

// newAccum returns an empty group of v.
func newAccum(v value) *accum {
	return &accum{value: v, activities: map[*tnode]bool{}, lineItems: map[*tnode]bool{}, children: map[string]*accum{}}
}

// add counts the piece p with the amount a in the group.
func (g *accum) add(p piece, a money.Amount) {
	g.planned = g.planned.Add(a)
	g.activities[p.activity] = true
	if p.lineItem != nil {
		g.lineItems[p.lineItem] = true
	}
}

// result works out the groups, the totals and the notes.
func (w *breakdownWork) result(groupings []Grouping) Breakdown {
	byLineItems := w.by.level() == LevelLineItem || (w.thenBy != nil && w.thenBy.level() == LevelLineItem)
	groups := map[string]*accum{}
	total := newAccum(value{key: keyTotal, label: labelTotal})
	var activities []*tnode
	for _, a := range w.matchingActivities() {
		activities = append(activities, a)
		for _, p := range w.pieces(a, byLineItems) {
			total.add(p, p.amount)
			for _, part := range w.split(p.amount, w.values(p, w.by)) {
				w.addPart(groups, p, part)
			}
		}
	}
	result := Breakdown{
		By: w.by, ThenBy: w.thenBy, Currency: w.currency(), Groups: []BreakdownGroup{}, CountsInEach: w.countsInEach,
		Groupings: groupings,
	}
	for _, g := range sortedAccums(groups) {
		group := w.group(g, total.planned, w.by, nil)
		group.Children = []BreakdownSubgroup{}
		for _, c := range sortedAccums(g.children) {
			sub := w.group(c, total.planned, *w.thenBy, g)
			group.Children = append(group.Children, BreakdownSubgroup{
				Key: sub.Key, Label: sub.Label, Activities: sub.Activities, LineItems: sub.LineItems, PlannedCost: sub.PlannedCost,
				Share: sub.Share, Budget: sub.Budget, BudgetUsedPercent: sub.BudgetUsedPercent, OverBudget: sub.OverBudget,
			})
		}
		result.Groups = append(result.Groups, group)
	}
	result.Total = w.totalGroup(total, activities)
	result.Notes = w.notes()
	return result
}

// addPart counts the part of the piece p in its group and, with a "then by" grouping, in that group's "then
// by" groups.
func (w *breakdownWork) addPart(groups map[string]*accum, p piece, part part) {
	g := groups[part.key]
	if g == nil {
		g = newAccum(part.value)
		groups[part.key] = g
	}
	g.add(p, part.amount)
	if w.thenBy == nil {
		return
	}
	for _, sub := range w.split(part.amount, w.values(p, *w.thenBy)) {
		c := g.children[sub.key]
		if c == nil {
			c = newAccum(sub.value)
			g.children[sub.key] = c
		}
		c.add(p, sub.amount)
	}
}

// notes returns the notes under the Breakdown.
func (w *breakdownWork) notes() []string {
	notes := []string{}
	if w.countsInEach {
		notes = append(notes, noteCountsInEach)
	}
	if !w.withoutPC.IsZero() {
		notes = append(notes, fmt.Sprintf(noteLineItemsNoPC, w.money(w.withoutPC).Format()))
	}
	return notes
}

// matchingActivities returns the shown matching activities of the tree, in tree order (010 FR-005).
func (w *breakdownWork) matchingActivities() []*tnode {
	var list []*tnode
	var walk func(n *tnode)
	walk = func(n *tnode) {
		if n.n.Level == LevelActivity {
			if n.shown && n.match {
				list = append(list, n)
			}
			return
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	for _, p := range w.t.plans {
		walk(p)
	}
	return list
}

// sortedAccums returns the groups in order (data-model §3.5): by option order, then by name; Not filled in
// and Not on a line item last.
func sortedAccums(groups map[string]*accum) []*accum {
	list := make([]*accum, 0, len(groups))
	for _, g := range groups {
		list = append(list, g)
	}
	slices.SortFunc(list, func(x, y *accum) int {
		return cmp.Or(cmp.Compare(x.order, y.order), strings.Compare(strings.ToLower(x.label), strings.ToLower(y.label)),
			strings.Compare(x.key, y.key))
	})
	return list
}

// currency is the currency of the Breakdown's amounts.
func (w *breakdownWork) currency() string {
	if w.planCurrency != "" {
		return w.planCurrency
	}
	return w.in.currency.Code
}

// money returns an amount, summed in USD or the plan currency, as shown: converted once to the display
// currency, or as it is in the plan currency.
func (w *breakdownWork) money(a money.Amount) money.Money {
	if w.planCurrency != "" {
		return money.Money{Amount: a, Currency: w.planCurrency}
	}
	return w.in.currency.money(a)
}

// group returns the group g as answered (FR-040): its counts, planned cost, share of the total and, for a
// grouping with budgets, the budget of its records and the budget used. parent is the group of a "then by"
// group; nil for a group.
func (w *breakdownWork) group(g *accum, total money.Amount, by Grouping, parent *accum) BreakdownGroup {
	result := BreakdownGroup{
		Key: g.key, Label: g.label, Activities: len(g.activities), PlannedCost: w.money(g.planned), Share: shareText2(g.planned, total),
		Children: []BreakdownSubgroup{},
	}
	if by.level() == LevelLineItem {
		result.LineItems = len(g.lineItems)
	} else {
		for a := range g.activities {
			result.LineItems += len(a.children)
		}
	}
	if by.HasBudget && g.key != keyNotOnLineItem {
		if budget := w.budgetOf(g, by, parent); budget != nil {
			shown := w.money(*budget)
			result.Budget = &shown
			p, b := money.USD(g.planned), money.USD(*budget)
			result.BudgetUsedPercent = rollup.BudgetUsedPercent(&p, &b)
			result.OverBudget = g.planned.Cmp(*budget) > 0
		}
	}
	return result
}
