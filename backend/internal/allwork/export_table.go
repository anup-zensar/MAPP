package allwork

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"mapp/internal/platform/money"
	"mapp/internal/platform/sheet"
)

// table returns the heading row, the rows of the view and the totals row (FR-032, FR-033).
func (e exporter) table() [][]sheet.Cell {
	r := e.b.result
	rows, withContext := e.rows()
	heading := []sheet.Cell{sheet.Bold("Level"), sheet.Bold("Path"), sheet.Bold("Name")}
	if withContext {
		heading = append(heading, sheet.Bold("Shown for context"))
	}
	heading = append(heading, e.headings()...)
	table := [][]sheet.Cell{heading}
	for _, row := range rows {
		line := []sheet.Cell{sheet.Text(levelWords[row.Level]), sheet.Text(e.path(row)), sheet.Text(row.Name)}
		if withContext {
			yes := ""
			if row.Role == RoleContext {
				yes = labelYes
			}
			line = append(line, sheet.Text(yes))
		}
		table = append(table, append(line, e.values(row)...))
	}
	noun := nounActivities
	if r.Totals.Activities == 1 {
		noun = nounActivity
	}
	totals := []sheet.Cell{{}, {}, sheet.Bold(fmt.Sprintf("Total · %d %s", r.Totals.Activities, noun))}
	if withContext {
		totals = append(totals, sheet.Cell{})
	}
	return append(table, append(totals, e.totals()...))
}

// rows returns the rows of the file, the unassigned planned cost rows only when they are asked for, and
// whether any of them is shown for context.
func (e exporter) rows() ([]Row, bool) {
	r := e.b.result
	withContext := false
	rows := make([]Row, 0, len(r.Rows))
	for _, row := range r.Rows {
		if row.Level == LevelUnassigned && !e.unassigned {
			continue
		}
		rows = append(rows, row)
		withContext = withContext || row.Role == RoleContext
	}
	return rows, withContext
}

// headings returns the headings of the view's columns (FR-033): money with the currency, "Planned cost
// (USD)"; a local amount followed by "Currency".
func (e exporter) headings() []sheet.Cell {
	cur := e.b.result.DisplayCurrency
	withCurrency := func(label string) sheet.Cell { return sheet.Bold(label + " (" + cur + ")") }
	var headings []sheet.Cell
	switch e.b.result.View {
	case ViewCalendar:
		headings = []sheet.Cell{sheet.Bold("ID"), sheet.Bold("Status"), sheet.Bold("Start date"), sheet.Bold("End date"),
			sheet.Bold("Owner"), withCurrency("Planned cost")}
	case ViewBudget:
		headings = []sheet.Cell{sheet.Bold("ID"), withCurrency("Plan budget")}
		for _, p := range e.b.result.Periods {
			headings = append(headings, withCurrency(p.Label))
		}
		headings = append(headings, withCurrency("Total"))
	default:
		for _, c := range e.b.result.Columns {
			switch {
			case isLocal(c.Key):
				headings = append(headings, sheet.Bold(c.Label), sheet.Bold("Currency"))
			case c.Kind == KindMoney:
				headings = append(headings, withCurrency(c.Label))
			default:
				headings = append(headings, sheet.Bold(c.Label))
			}
		}
	}
	return headings
}

// values returns the values of a row in the view's columns.
func (e exporter) values(row Row) []sheet.Cell {
	id := sheet.Text(row.RecordID)
	if row.Level == LevelUnassigned {
		id = sheet.Cell{}
	}
	switch e.b.result.View {
	case ViewCalendar:
		return calendarValues(id, row)
	case ViewBudget:
		values := []sheet.Cell{id, moneyValue(row.Budget)}
		for _, a := range row.Periods {
			values = append(values, moneyValue(a))
		}
		return append(values, moneyValue(row.Total))
	}
	var values []sheet.Cell
	for i, c := range e.b.result.Columns {
		values = append(values, sheetCell(row.Cells[i]))
		if isLocal(c.Key) {
			currency := sheet.Cell{}
			if cell := row.Cells[i]; cell != nil && cell.Money != nil {
				currency = sheet.Text(cell.Money.Currency)
			}
			values = append(values, currency)
		}
	}
	return values
}

// calendarValues returns the values of a row in the Calendar's columns, after its id.
func calendarValues(id sheet.Cell, row Row) []sheet.Cell {
	end := sheet.Date(row.EndDate)
	if row.Ongoing {
		end = sheet.Text(ongoingWord)
	}
	owner := sheet.Cell{}
	if row.Owner != nil {
		owner = sheet.Text(inactiveLabel(row.Owner.Name, row.Owner.Inactive))
	}
	status := sheet.Cell{}
	if row.Status != nil {
		status = sheet.Text(row.Status.Label)
	}
	return []sheet.Cell{id, status, sheet.Date(row.StartDate), end, owner, moneyValue(row.PlannedCost)}
}

// totals returns the totals row's values in the view's columns (FR-014, FR-024).
func (e exporter) totals() []sheet.Cell {
	t := e.b.result.Totals
	switch e.b.result.View {
	case ViewCalendar:
		return []sheet.Cell{{}, {}, {}, {}, {}, moneyValue(t.PlannedCost)}
	case ViewBudget:
		values := []sheet.Cell{{}, moneyValue(t.PlanBudget)}
		for _, a := range t.Periods {
			values = append(values, moneyValue(a))
		}
		return append(values, moneyValue(t.Total))
	}
	var values []sheet.Cell
	for i, c := range e.b.result.Columns {
		values = append(values, sheetCell(t.Cells[i]))
		if isLocal(c.Key) {
			values = append(values, sheet.Cell{})
		}
	}
	return values
}

// path returns the names of a row's ancestors, top-down: "FY2026 Lubricants Europe › Helix launch" (FR-032).
func (e exporter) path(row Row) string {
	var names []string
	n := e.b.tree.byID[row.RecordID]
	if n != nil && row.Level != LevelUnassigned {
		n = n.parent
	}
	for ; n != nil; n = n.parent {
		names = append([]string{n.n.Name}, names...)
	}
	return strings.Join(names, " › ")
}

// sheetCell returns a grid cell as a cell of the file (FR-033): money and numbers as numbers, dates as dates,
// statuses as words, inactive values labelled; nil and "—" as empty cells.
func sheetCell(c *Cell) sheet.Cell {
	if c == nil {
		return sheet.Cell{}
	}
	switch c.Kind {
	case KindMoney:
		return sheet.Money(c.Money.Amount)
	case KindNumber:
		return sheet.Number(*c.Number)
	case KindPercent:
		n, _ := strconv.ParseInt(c.Percent, 10, 64)
		return sheet.Percent(n)
	case KindDate:
		return sheet.Date(c.Date)
	case KindDateTime:
		return sheet.DateTime(*c.DateTime)
	case KindText, KindStatus:
		return sheet.Text(inactiveLabel(c.Text, c.Inactive))
	}
	return sheet.Cell{}
}

// moneyValue returns an optional amount as a cell.
func moneyValue(m *money.Money) sheet.Cell {
	if m == nil {
		return sheet.Cell{}
	}
	return sheet.Money(m.Amount)
}

// inactiveLabel adds "(inactive)" to the text of an inactive value.
func inactiveLabel(text string, inactive bool) string {
	if inactive {
		return text + " (inactive)"
	}
	return text
}

// isLocal reports whether the column holds amounts in a line item's own currency.
func isLocal(key string) bool {
	return slices.Contains(localColumns, key)
}
