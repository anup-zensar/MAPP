package dashboard

import (
	"fmt"

	"mapp/internal/allwork"
	"mapp/internal/platform/money"
)

const monthsInYear = 12

// months is the planned cost by month as All work's Budget & cost view spreads it, and the payments by posting
// month that countSpend counted, with what was posted in other years apart (FR-014).
func (b *board) months(planned map[string]allwork.Months, year int, d money.Display) Months {
	plannedBy := map[string]money.Amount{}
	var noDates, otherYears *money.Amount
	for _, a := range b.byLevel[allwork.LevelActivity] {
		m := planned[a.ID]
		for month, amount := range m.ByMonth {
			plannedBy[month] = plannedBy[month].Add(amount)
		}
		noDates, otherYears = addNonZero(noDates, m.NoDates), addNonZero(otherYears, m.OtherYears)
	}
	out := Months{Items: make([]Month, 0, monthsInYear), NoDates: optionalShown(d, noDates),
		OtherYears: optionalShown(d, otherYears), CommittedOutsideYear: optionalShown(d, b.outside.committed),
		PaidOutsideYear: optionalShown(d, b.outside.actual)}
	showsActual := b.financeCount() > 0
	var plannedSoFar, actualSoFar money.Amount
	for i := 1; i <= monthsInYear; i++ {
		key := fmt.Sprintf("%04d-%02d", year, i)
		plannedSoFar, actualSoFar = plannedSoFar.Add(plannedBy[key]), actualSoFar.Add(b.paidBy[key])
		month := Month{Month: key, Planned: shown(d, plannedBy[key]), PlannedToDate: shown(d, plannedSoFar)}
		if showsActual {
			actual, toDate := shown(d, b.paidBy[key]), shown(d, actualSoFar)
			month.Actual, month.ActualToDate = &actual, &toDate
		}
		out.Items = append(out.Items, month)
	}
	return out
}

func addNonZero(total *money.Amount, a money.Amount) *money.Amount {
	if a.IsZero() {
		return total
	}
	return add(total, &a)
}
