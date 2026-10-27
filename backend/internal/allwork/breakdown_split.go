package allwork

import (
	"fmt"
	"math/big"

	"mapp/internal/platform/money"
)

// amountOf returns the planned cost of a record in the Breakdown's currency, before conversion: USD, or the
// plan currency; nil when it has none.
func (w *breakdownWork) amountOf(n *Node) *money.Amount {
	if w.planCurrency != "" {
		if n.PlannedCostPlan == nil {
			return nil
		}
		return &n.PlannedCostPlan.Amount
	}
	return n.PlannedCost
}

// pieces returns the parts of the activity a's planned cost (data-model §3.5): the whole activity, or, for a
// line-item grouping, each line item and the unassigned planned cost ("Not on a line item"). Line items of an
// activity without a planned cost count too, and are named in a note.
func (w *breakdownWork) pieces(a *tnode, byLineItems bool) []piece {
	planned := w.amountOf(a.n)
	if !byLineItems {
		var amount money.Amount
		if planned != nil {
			amount = *planned
		}
		return []piece{{activity: a, amount: amount}}
	}
	var list []piece
	var assigned money.Amount
	for _, li := range a.children {
		var amount money.Amount
		if cost := w.amountOf(li.n); cost != nil {
			amount = *cost
		}
		assigned = assigned.Add(amount)
		list = append(list, piece{activity: a, lineItem: li, amount: amount})
	}
	switch {
	case planned == nil:
		w.withoutPC = w.withoutPC.Add(assigned)
		if len(list) == 0 {
			list = append(list, piece{activity: a, unassigned: true})
		}
	case !planned.Sub(assigned).IsZero() || len(list) == 0:
		list = append(list, piece{activity: a, unassigned: true, amount: planned.Sub(assigned)})
	}
	return list
}

// part is the amount of a piece that goes to one group.
type part struct {
	value
	amount money.Amount
}

// split divides amount among the values (data-model §3.5): one value gets it all; a % split gives each value
// amount × share ÷ 100, half up, and the last the rest; several values without shares each get it all, and
// the Breakdown says so.
func (w *breakdownWork) split(amount money.Amount, values []value) []part {
	if len(values) == 1 {
		return []part{{value: values[0], amount: amount}}
	}
	parts := make([]part, 0, len(values))
	if values[0].share == nil {
		w.countsInEach = true
		for _, v := range values {
			parts = append(parts, part{value: v, amount: amount})
		}
		return parts
	}
	var given money.Amount
	for i, v := range values {
		a := amount.Sub(given)
		if i < len(values)-1 {
			a = shareOfAmount(amount, *v.share)
			given = given.Add(a)
		}
		parts = append(parts, part{value: v, amount: a})
	}
	return parts
}

// shareOfAmount is a × p ÷ 100, rounded half up (away from zero) to the cent.
func shareOfAmount(a money.Amount, p money.Percent) money.Amount {
	num := new(big.Int).Mul(big.NewInt(a.Cents()), big.NewInt(p.Hundredths()))
	return money.FromCents(roundHalfAway(num, big.NewInt(10000)))
}

// roundHalfAway divides num by den, rounding half away from zero.
func roundHalfAway(num, den *big.Int) int64 {
	negative := num.Sign()*den.Sign() < 0
	n, d := new(big.Int).Abs(num), new(big.Int).Abs(den)
	q, r := new(big.Int).QuoRem(n, d, new(big.Int))
	if new(big.Int).Mul(r, big.NewInt(2)).Cmp(d) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if negative {
		q.Neg(q)
	}
	return q.Int64()
}

// shareText2 is part ÷ whole × 100, half up to two decimals ("37.50"); "0.00" when whole is 0.00.
func shareText2(part, whole money.Amount) string {
	if whole.IsZero() {
		return "0.00"
	}
	num := new(big.Int).Mul(big.NewInt(part.Cents()), big.NewInt(10000))
	hundredths := roundHalfAway(num, big.NewInt(whole.Cents()))
	sign := ""
	if hundredths < 0 {
		sign, hundredths = "-", -hundredths
	}
	return fmt.Sprintf("%s%d.%02d", sign, hundredths/100, hundredths%100)
}
