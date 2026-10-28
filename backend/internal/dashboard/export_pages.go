package dashboard

import (
	"time"

	"mapp/internal/platform/deck"
	"mapp/internal/platform/money"
)

const (
	labelBudget    = "Budget"
	labelPlanned   = "Planned cost"
	labelCommitted = "Committed"
	labelActual    = "Actual"
	labelCampaign  = "Campaign"
	labelPlan      = "Plan"
	momentLayout   = "2 Jan 2006, 15:04 UTC"
)

func figureCards(f Figures) []deck.Card {
	planned := deck.Card{Label: labelPlanned, Value: moneyText(f.PlannedCost), Note: "Budget used " + percentText(f.BudgetUsedPercent)}
	if f.OverBudget {
		planned.Note, planned.Problem = "Over budget", true
	}
	left := deck.Card{Label: wordLeftToPlan, Value: moneyText(f.LeftToPlan)}
	left.Problem = f.LeftToPlan != nil && f.LeftToPlan.Amount.Sign() < 0
	actual := deck.Card{Label: labelActual, Value: moneyText(f.Actual), Note: "Budget spent " + percentText(f.BudgetSpentPercent)}
	if f.Actual == nil {
		actual.Note = financeNote
	}
	return []deck.Card{
		{Label: labelBudget, Value: moneyText(f.Budget)}, planned, left,
		{Label: labelCommitted, Value: moneyText(f.Committed)}, actual,
		{Label: "Unspent budget", Value: moneyText(f.UnspentBudget)},
	}
}

func figureCells(f Figures) []string {
	return []string{moneyText(f.Budget), moneyText(f.PlannedCost), moneyText(f.Committed), moneyText(f.Actual),
		moneyText(f.LeftToPlan)}
}

func figureColumns(first ...deck.Column) []deck.Column {
	return append(first,
		deck.Column{Title: labelBudget, Weight: 2, Align: deck.Right}, deck.Column{Title: labelPlanned, Weight: 2, Align: deck.Right},
		deck.Column{Title: labelCommitted, Weight: 2, Align: deck.Right}, deck.Column{Title: labelActual, Weight: 2, Align: deck.Right},
		deck.Column{Title: wordLeftToPlan, Weight: 2, Align: deck.Right})
}

func monthsPage(title string, m Months, currency string) deck.Page {
	chart := &deck.Chart{Series: []deck.Series{{Name: labelPlanned, Colour: deck.ChartPlanned}, {Name: labelActual, Colour: deck.ChartActual}}}
	table := &deck.Table{Columns: []deck.Column{{Title: "Month", Weight: 3}, {Title: labelPlanned, Weight: 3, Align: deck.Right},
		{Title: labelActual, Weight: 3, Align: deck.Right}, {Title: "Planned to date", Weight: 3, Align: deck.Right}}}
	var largest money.Amount
	for _, month := range m.Items {
		label := monthLabel(month.Month)
		actual := money.Amount{}
		if month.Actual != nil {
			actual = month.Actual.Amount
		}
		chart.Categories = append(chart.Categories, label[:3])
		chart.Series[0].Values = append(chart.Series[0].Values, month.Planned.Amount.Cents())
		chart.Series[1].Values = append(chart.Series[1].Values, actual.Cents())
		largest = maxAmount(largest, month.Planned.Amount, actual)
		table.Rows = append(table.Rows, deck.Row{Cells: []string{label, month.Planned.Format(), moneyText(month.Actual),
			month.PlannedToDate.Format()}})
	}
	chart.MaxLabel = "Scale up to " + money.Money{Amount: largest, Currency: currency}.Format()
	return deck.Page{Title: title, Subtitle: "Planned cost spread by month and actual paid by posting month", Chart: chart,
		Table: table, Notes: monthNotes(m)}
}

func monthNotes(m Months) []string {
	var notes []string
	if m.NoDates != nil {
		notes = append(notes, "Not yet spread (activities without dates): "+m.NoDates.Format())
	}
	if m.OtherYears != nil {
		notes = append(notes, "Spread into other years: "+m.OtherYears.Format())
	}
	if m.PaidOutsideYear != nil {
		notes = append(notes, "Paid outside the year: "+m.PaidOutsideYear.Format())
	}
	return notes
}

func monthLabel(month string) string {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return month
	}
	return t.Format("Jan 2006")
}

func maxAmount(first money.Amount, others ...money.Amount) money.Amount {
	top := first
	for _, a := range others {
		if a.Cmp(top) > 0 {
			top = a
		}
	}
	return top
}

func (in *input) madeNotes() []string {
	notes := []string{"Amounts in " + in.query.Currency, "Made on " + in.now.UTC().Format(momentLayout)}
	if in.data.FinanceAsOf != nil {
		notes = append(notes, "GSAP data as of "+in.data.FinanceAsOf.UTC().Format(momentLayout))
	}
	return notes
}
