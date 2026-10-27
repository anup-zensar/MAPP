package allwork

import (
	"slices"

	"mapp/internal/platform/money"
	"mapp/internal/rollup"
)

// budgetOf returns the budgets of the records at the grouping's level whose value is in the group g (several
// values → each), counting only the records the filters show; within a "then by" group, only those of the
// parent group's activities' branches. nil when no such record has a budget.
func (w *breakdownWork) budgetOf(g *accum, by Grouping, parent *accum) *money.Amount {
	var budget *money.Amount
	for _, r := range w.recordsAt(by.level()) {
		if !r.shown || !slices.ContainsFunc(w.recordValues(r, by), func(v value) bool { return v.key == g.key }) {
			continue
		}
		if parent != nil && !w.inBranch(r, parent) {
			continue
		}
		budget = addAmount(budget, w.budgetAmount(r.n))
	}
	return budget
}

// inBranch reports whether one of the activities of the group is under the record r, or r is under one of
// them.
func (w *breakdownWork) inBranch(r *tnode, group *accum) bool {
	for a := range group.activities {
		for t := a; t != nil; t = t.parent {
			if t == r {
				return true
			}
		}
	}
	return false
}

// recordsAt returns every record of the level.
func (w *breakdownWork) recordsAt(level Level) []*tnode {
	var list []*tnode
	for _, t := range w.t.byID {
		if t.n.Level == level {
			list = append(list, t)
		}
	}
	return list
}

// recordValues returns the groups of the record r (a plan, campaign or initiative) by the grouping.
func (w *breakdownWork) recordValues(r *tnode, by Grouping) []value {
	if by.attribute != nil {
		return attributeValues(*by.attribute, r.n, w.units[r.n.PlanID])
	}
	return w.values(piece{activity: r}, by)
}

// budgetAmount is a record's budget in the Breakdown's currency: USD, or the plan currency.
func (w *breakdownWork) budgetAmount(n *Node) *money.Amount {
	if w.planCurrency != "" {
		if n.BudgetPlan == nil {
			return nil
		}
		return &n.BudgetPlan.Amount
	}
	return n.Budget
}

// totalGroup is the totals row (FR-040): every matching activity once, their line items, the total planned
// cost — equal to the Planned cost card — and, for a grouping with budgets, the budgets of the shown records of
// its level once each.
func (w *breakdownWork) totalGroup(total *accum, activities []*tnode) BreakdownGroup {
	result := BreakdownGroup{
		Key: keyTotal, Label: labelTotal, Activities: len(activities), PlannedCost: w.money(total.planned),
		Share: shareText2(total.planned, total.planned), Children: []BreakdownSubgroup{},
	}
	for _, a := range activities {
		result.LineItems += len(a.children)
	}
	if w.by.HasBudget {
		var budget *money.Amount
		for _, r := range w.recordsAt(w.by.level()) {
			if r.shown {
				budget = addAmount(budget, w.budgetAmount(r.n))
			}
		}
		if budget != nil {
			shown := w.money(*budget)
			result.Budget = &shown
			p, b := money.USD(total.planned), money.USD(*budget)
			result.BudgetUsedPercent = rollup.BudgetUsedPercent(&p, &b)
			result.OverBudget = total.planned.Cmp(*budget) > 0
		}
	}
	return result
}
