package lineitem

import (
	"testing"

	"mapp/internal/platform/money"
)

// a1011Grid is A-1011's grid in GBP: Venue 15,000.00 GBP and 23,000.00 EUR = 19,750.00 GBP over March – May 2027.
func a1011Grid() SpreadGrid {
	gbp := func(s string) *money.Money { return &money.Money{Amount: amount(s), Currency: "GBP"} }
	return SpreadGrid{
		Rows: []SpreadGridRow{
			{LineItemID: "L-1009", Name: "Venue", Cells: []*money.Money{nil, gbp("5000.00"), gbp("5000.00"), gbp("5000.00")},
				Total: *gbp("15000.00")},
			{LineItemID: "L-1010", Name: "Agency fee (Ireland)",
				Cells: []*money.Money{nil, gbp("6583.33"), gbp("6583.33"), gbp("6583.34")}, Total: *gbp("19750.00")},
		},
		Totals: []*money.Money{nil, gbp("11583.33"), gbp("11583.33"), gbp("11583.34")},
		Total:  *gbp("34750.00"),
	}
}

// In any display currency rows, columns and totals add up; a row total is its planned cost converted once.
func TestGrid_AddsUp_038_FR022_M1(t *testing.T) {
	t.Parallel()
	rates := money.Rates{"GBP": rate("0.79"), "EUR": rate("0.92")}
	for _, d := range []money.Display{money.DisplayUSD.WithRates(rates), money.DisplayIn("EUR", "0.92").WithRates(rates),
		money.DisplayIn("GBP", "0.79").WithRates(rates)} {
		plan := a1011Grid()
		g := display{d}.grid(plan)
		rowTotals := checkGridRows(t, d, plan, g)
		checkGridColumns(t, d, g)
		if sumCells(g.Totals) != g.Total.Amount || money.Sum(amountsOf(rowTotals)...) != g.Total.Amount {
			t.Errorf("%s: grand total %s, columns add up to %s", d.Currency(), g.Total.Format(), sumCells(g.Totals))
		}
	}
}

// checkGridRows checks each row total is its planned cost converted once and its months add up to it.
func checkGridRows(t *testing.T, d money.Display, plan, g SpreadGrid) []money.Money {
	t.Helper()
	rowTotals := make([]money.Money, 0, len(g.Rows))
	for i, row := range g.Rows {
		want := d.Money(plan.Rows[i].Total)
		if sum := sumCells(row.Cells); row.Total != want || sum != want.Amount {
			t.Errorf("%s: row %s = %s with months adding up to %s, want %s", d.Currency(), row.Name,
				row.Total.Format(), sum, want.Format())
		}
		rowTotals = append(rowTotals, row.Total)
	}
	return rowTotals
}

func checkGridColumns(t *testing.T, d money.Display, g SpreadGrid) {
	t.Helper()
	for col := range g.Totals {
		column := make([]*money.Money, 0, len(g.Rows))
		for _, row := range g.Rows {
			column = append(column, row.Cells[col])
		}
		if got := sumCells(g.Totals[col : col+1]); got != sumCells(column) {
			t.Errorf("%s: column %d total = %s, its cells add up to %s", d.Currency(), col, got, sumCells(column))
		}
	}
}

func sumCells(cells []*money.Money) money.Amount {
	var total money.Amount
	for _, c := range cells {
		if c != nil {
			total = total.Add(c.Amount)
		}
	}
	return total
}

func amountsOf(ms []money.Money) []money.Amount {
	out := make([]money.Amount, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.Amount)
	}
	return out
}
