package dashboard

import (
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
)

// sums are a set of records' amounts in USD; a nil one has nothing to add up, or is financial data not counted.
type sums struct {
	budget, planned, committed, actual *money.Amount
}

func (s sums) plus(o sums) sums {
	return sums{budget: add(s.budget, o.budget), planned: add(s.planned, o.planned),
		committed: add(s.committed, o.committed), actual: add(s.actual, o.actual)}
}

func add(a, b *money.Amount) *money.Amount {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	}
	sum := a.Add(*b)
	return &sum
}

func usd(a *money.Amount) *money.Money {
	if a == nil {
		return nil
	}
	m := money.USD(*a)
	return &m
}

// figures converts the sums once to the display currency; the percents and differences are worked out in USD.
func (s sums) figures(d money.Display) Figures {
	budget, planned, actual := usd(s.budget), usd(s.planned), usd(s.actual)
	f := Figures{
		Budget: d.Optional(budget), PlannedCost: d.Optional(planned), Committed: d.Optional(usd(s.committed)),
		Actual: d.Optional(actual), BudgetUsedPercent: rollup.BudgetUsedPercent(planned, budget),
		LeftToPlan: d.Optional(rollup.LeftToPlan(budget, planned)), OverBudget: rollup.OverBudget(planned, budget),
	}
	if budget != nil && actual != nil {
		f.BudgetSpentPercent = rollup.BudgetUsedPercent(actual, budget)
		unspent := money.USD(budget.Amount.Sub(actual.Amount))
		f.UnspentBudget = d.Optional(&unspent)
	}
	return f
}

func shown(d money.Display, a money.Amount) money.Money {
	return d.Money(money.USD(a))
}

func optionalShown(d money.Display, a *money.Amount) *money.Money {
	return d.Optional(usd(a))
}

// percentOf is part × 100 ÷ whole, a whole percent rounded half up; nil when whole is 0.
func percentOf(part, whole int) *int {
	if whole <= 0 {
		return nil
	}
	percent := (part*200 + whole) / (2 * whole)
	return &percent
}
