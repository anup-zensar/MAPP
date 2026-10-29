package dashboard

import (
	"fmt"

	"mapp/internal/allwork"
	"mapp/internal/platform/money"
)

const monthsInYear = 12

// months is the planned cost by month as All work's Budget & cost view spreads it, and the payments by posting
// month of the plans whose financial figures count (FR-014).
func (b *board) months(planned map[string]allwork.Months, paid map[string]map[string]money.Amount, year int,
	d money.Display) Months {
	plannedBy, actualBy := map[string]money.Amount{}, map[string]money.Amount{}
	var noDates, otherYears, outside *money.Amount
	for _, a := range b.byLevel[allwork.LevelActivity] {
		m := planned[a.ID]
		for month, amount := range m.ByMonth {
			plannedBy[month] = plannedBy[month].Add(amount)
		}
		noDates, otherYears = addNonZero(noDates, m.NoDates), addNonZero(otherYears, m.OtherYears)
	}
	for _, li := range b.byLevel[allwork.LevelLineItem] {
		if !b.finance[li.PlanID] {
			continue
		}
		for month, amount := range paid[li.ID] {
			if month[:4] == fmt.Sprint(year) {
				actualBy[month] = actualBy[month].Add(amount)
			} else {
				outside = addNonZero(outside, amount)
			}
		}
	}
	out := Months{Items: make([]Month, 0, monthsInYear), NoDates: optionalShown(d, noDates),
		OtherYears: optionalShown(d, otherYears), PaidOutsideYear: optionalShown(d, outside)}
	showsActual := b.financeCount() > 0
	var plannedSoFar, actualSoFar money.Amount
	for i := 1; i <= monthsInYear; i++ {
		key := fmt.Sprintf("%04d-%02d", year, i)
		plannedSoFar, actualSoFar = plannedSoFar.Add(plannedBy[key]), actualSoFar.Add(actualBy[key])
		month := Month{Month: key, Planned: shown(d, plannedBy[key]), PlannedToDate: shown(d, plannedSoFar)}
		if showsActual {
			actual, toDate := shown(d, actualBy[key]), shown(d, actualSoFar)
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
