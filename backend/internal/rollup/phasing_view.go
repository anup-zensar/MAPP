package rollup

import "mapp/internal/platform/money"

// PhasingMonthView is one typed month as the API answers it (contract PhasingMonth).
type PhasingMonthView struct {
	Month  string      `json:"month"`
	Amount money.Money `json:"amount"`
}

// PhasingView is a phasing as the API answers it (contract BudgetPhasing), in the plan currency.
type PhasingView struct {
	Method  PhasingMethod      `json:"method"`
	Months  []PhasingMonthView `json:"months"`  // Custom only: the stored months
	Monthly []money.Money      `json:"monthly"` // the 12 monthly amounts; empty when not phased
	Total   *money.Money       `json:"total"`   // the sum of Monthly; null when not phased
	Problem *string            `json:"problem"` // the FR-022 problem; null when fine
}

// ViewOf returns the phasing p of a record with budget in a plan of year, in the plan currency.
func ViewOf(p Phasing, budget *money.Amount, year int, currency string) PhasingView {
	p = p.Normalised()
	view := PhasingView{Method: p.Method, Months: []PhasingMonthView{}, Monthly: []money.Money{}}
	for _, m := range p.Months {
		view.Months = append(view.Months, PhasingMonthView{Month: m.Month, Amount: money.Money{Amount: m.Amount, Currency: currency}})
	}
	if amounts := p.MonthAmounts(budget, year); amounts != nil {
		for _, a := range amounts {
			view.Monthly = append(view.Monthly, money.Money{Amount: a, Currency: currency})
		}
		total := money.Money{Amount: money.Sum(amounts...), Currency: currency}
		view.Total = &total
	}
	if problem := PhasingProblem(p, budget, year, currency); problem != nil {
		view.Problem = &problem.Reason
	}
	return view
}
