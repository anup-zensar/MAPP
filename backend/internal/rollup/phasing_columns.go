package rollup

import "mapp/internal/platform/money"

// MonthColumns returns p's typed months as the two parallel lists a store saves them in.
func (p Phasing) MonthColumns() (months []string, amounts []int64) {
	months, amounts = make([]string, 0, len(p.Months)), make([]int64, 0, len(p.Months))
	for _, m := range p.Months {
		months, amounts = append(months, m.Month), append(amounts, m.Amount.Cents())
	}
	return months, amounts
}

func PhasingOf(method string, months []string, amounts []int64) Phasing {
	p := Phasing{Method: PhasingMethod(method), Months: make([]PhasingMonth, 0, len(months))}
	for i, month := range months {
		p.Months = append(p.Months, PhasingMonth{Month: month, Amount: money.FromCents(amounts[i])})
	}
	return p
}
