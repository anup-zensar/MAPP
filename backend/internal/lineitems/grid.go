package lineitems

import (
	"slices"

	"northstar/internal/platform/money"
)

// Grid builds the spread grid of the activity f from the views of its line items, in creation order (data
// model §3.8, FR-003, FR-020), in USD:
//
//   - columns: every activity month (an ongoing activity: its start month) and every month a custom spread
//     uses, sorted, without filling gaps, each marked when it lies outside the activity's dates; then a "No
//     dates yet" column when the activity has no dates and a line item is spread Evenly, first or last (D-032);
//   - rows: one per line item, a cell per column with its USD month amount, or null; the No dates cell holds
//     the USD planned cost of a line item that is not spread; the row total is the sum of its cells;
//   - totals: the sum of each column (null when every cell is null) and the grand total.
//
// Quarterly is the web app's display choice; the unassigned balance is not part of the grid.
func Grid(f ActivityFacts, items []View) SpreadGrid {
	months, _ := ActivityMonths(f)
	notSpread := false
	for _, item := range items {
		for _, m := range item.Spread.Months {
			if !slices.Contains(months, m.Month) {
				months = append(months, m.Month)
			}
		}
		notSpread = notSpread || item.Spread.State == StateNotSpread
	}
	slices.Sort(months)

	columns := make([]SpreadGridColumn, 0, len(months)+1)
	for _, m := range months {
		month := m
		columns = append(columns, SpreadGridColumn{Key: m, Month: &month, OutsideActivityDates: OutsideActivityDates(m, f)})
	}
	if notSpread {
		columns = append(columns, SpreadGridColumn{Key: NoDatesColumn})
	}

	grid := SpreadGrid{Columns: columns, Rows: []SpreadGridRow{}, Totals: make([]*money.Money, len(columns))}
	var total money.Amount
	for _, item := range items {
		row := SpreadGridRow{LineItemID: item.ID, Name: item.Name, Cells: make([]*money.Money, len(columns))}
		var rowTotal money.Amount
		put := func(column int, cell money.Money) {
			row.Cells[column] = &cell
			rowTotal = rowTotal.Add(cell.Amount)
			if grid.Totals[column] == nil {
				grid.Totals[column] = &money.Money{Currency: money.BaseCurrency}
			}
			grid.Totals[column].Amount = grid.Totals[column].Amount.Add(cell.Amount)
		}
		for _, m := range item.Spread.Months {
			put(slices.Index(months, m.Month), m.AmountUSD)
		}
		if item.Spread.State == StateNotSpread {
			put(len(columns)-1, item.PlannedCostUSD)
		}
		row.Total = money.USD(rowTotal)
		total = total.Add(rowTotal)
		grid.Rows = append(grid.Rows, row)
	}
	grid.Total = money.USD(total)
	return grid
}
