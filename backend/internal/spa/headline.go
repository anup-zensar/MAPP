package spa

import "mapp/internal/platform/money"

const msgNoPlannedCost = "No planned cost"

func plannedOf(a *activity) money.Amount   { return a.planned }
func committedOf(a *activity) money.Amount { return a.spend.Committed }
func actualOf(a *activity) money.Amount    { return a.spend.Actual }
func invoicedOf(a *activity) money.Amount  { return a.spend.Invoiced }

// figures are the headline figures of a set of activities, in USD (FR-015).
type figures struct {
	planned, actual money.Amount
}

func (f figures) remaining() money.Amount {
	return f.planned.Sub(f.actual)
}

func (f figures) utilisation() *string {
	return percentOf(f.actual, f.planned, 1)
}

func (f figures) variance() *string {
	return percentOf(f.actual.Sub(f.planned), f.planned, 1)
}

// overspend is FR-016: remaining below zero, which is utilisation above 100 % and variance above 0 %.
func (f figures) overspend() bool {
	return f.remaining().Sign() < 0
}

func headlineOf(acts []*activity, sees bool, s shower) Headline {
	f := figures{planned: sumOf(acts, plannedOf), actual: sumOf(acts, actualOf)}
	h := Headline{
		PlannedCost: s.money(f.planned), PlannedCostTrace: s.trace(acts, plannedOf),
		CommittedTrace: []Contribution{}, ActualSpendTrace: []Contribution{},
	}
	if f.planned.IsZero() {
		h.NoPlannedCostReason = text(msgNoPlannedCost)
	}
	if !sees {
		return h
	}
	h.Committed, h.ActualSpend = s.financial(sees, sumOf(acts, committedOf)), s.financial(sees, f.actual)
	h.Invoiced, h.Remaining = s.financial(sees, sumOf(acts, invoicedOf)), s.financial(sees, f.remaining())
	h.UtilisationPercent, h.VariancePercent, h.Overspend = f.utilisation(), f.variance(), f.overspend()
	h.CommittedTrace, h.ActualSpendTrace = s.trace(acts, committedOf), s.trace(acts, actualOf)
	return h
}
