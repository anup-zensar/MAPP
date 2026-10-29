package allwork

import "mapp/internal/platform/money"

// Months is an activity's planned cost in USD by month of one year ("2026-03"), with what falls in other years
// and what cannot be spread yet, as the Budget & cost view works it out (FR-023); feature 036's dashboards read it.
type Months struct {
	ByMonth    map[string]money.Amount
	OtherYears money.Amount
	NoDates    money.Amount
}

func PlannedByActivity(nodes []Node, year int) map[string]Months {
	p := periodsOf(buildTree(nodes), year, year)
	byActivity := map[string]Months{}
	for n, b := range p.of {
		if n.n.Level != LevelActivity {
			continue
		}
		m := Months{ByMonth: map[string]money.Amount{}}
		for key, amount := range b {
			switch key {
			case periodOther:
				m.OtherYears = amount
			case periodNoDates:
				m.NoDates = amount
			default:
				m.ByMonth[key] = amount
			}
		}
		byActivity[n.n.ID] = m
	}
	return byActivity
}
