package lineitem

import (
	"slices"
	"testing"

	"mapp/internal/platform/money"
)

// viewOf is the part of a line item's view the grid reads: its USD planned cost and its spread.
func viewOf(id, name string, li LineItem, f ActivityFacts, r money.Rate) View {
	usd := money.USD(PlannedCostUSD(li, r))
	return View{ID: id, Name: name, PlannedCostUSD: usd, PlannedCostPlan: usd, Spread: spreadView(li, f, r, money.One)} // a USD plan
}

// lineItem returns a line item of planned cost in its currency, spread with method; custom months as
// "YYYY-MM", "amount" pairs.
func lineItem(cost string, method SpreadMethod, custom ...string) LineItem {
	li := LineItem{PlannedCost: amount(cost), Currency: "USD", Spread: Spread{Method: method, Months: []MonthAmount{}}}
	for i := 0; i+1 < len(custom); i += 2 {
		li.Spread.Months = append(li.Spread.Months, MonthAmount{Month: custom[i], Amount: amount(custom[i+1])})
	}
	return li
}

// columnKeys lists the keys of the grid's columns, with "!" after those outside the activity's dates.
func columnKeys(g SpreadGrid) []string {
	var keys []string
	for _, c := range g.Columns {
		key := c.Key
		if c.OutsideActivityDates {
			key += "!"
		}
		keys = append(keys, key)
	}
	return keys
}

// cellTexts writes cells as amounts, "—" for null.
func cellTexts(cells []*money.Money) []string {
	var texts []string
	for _, c := range cells {
		if c == nil {
			texts = append(texts, "—")
		} else {
			texts = append(texts, c.Amount.String())
		}
	}
	return texts
}

// The grid has a column per month of the activity and per month a custom spread uses, sorted, without
// filling gaps; months outside the activity's dates are marked (FR-003, FR-018).
func TestGrid_Columns_FR003(t *testing.T) {
	t.Parallel()
	launch := newFakeActivities().facts[launchFilm] // Mar–May 2026
	ongoing := newFakeActivities().facts[partnerFund]
	tests := []struct {
		name  string
		f     ActivityFacts
		items []LineItem
		want  []string
	}{
		{"no line items", launch, nil, []string{"2026-03", "2026-04", "2026-05"}},
		{"a deposit and a late payment", launch, []LineItem{
			lineItem("100.00", SpreadEvenly),
			lineItem("300.00", SpreadCustom, "2026-02", "100.00", "2026-04", "100.00", "2026-08", "100.00"),
		}, []string{"2026-02!", "2026-03", "2026-04", "2026-05", "2026-08!"}},
		{"ongoing: the start month and later custom months", ongoing, []LineItem{
			lineItem("100.00", SpreadEvenly),
			lineItem("200.00", SpreadCustom, "2026-01", "100.00", "2026-12", "100.00"),
		}, []string{"2026-01!", "2026-03", "2026-12"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var views []View
			for i, li := range tt.items {
				views = append(views, viewOf(string(rune('A'+i)), "item", li, tt.f, money.One))
			}

			grid := Grid(tt.f, views)

			if got := columnKeys(grid); !slices.Equal(got, tt.want) {
				t.Errorf("columns = %v, want %v", got, tt.want)
			}
			if len(grid.Totals) != len(grid.Columns) || len(grid.Rows) != len(views) {
				t.Errorf("%d totals and %d rows for %d columns and %d line items", len(grid.Totals), len(grid.Rows), len(grid.Columns), len(views))
			}
		})
	}
}

// Without activity dates, Evenly, first and last are not spread: their USD planned cost is in a "No dates
// yet" column, after the months of custom spreads (D-032, FR-017).
func TestGrid_NoDatesColumn_D032(t *testing.T) {
	t.Parallel()
	f := newFakeActivities().facts[lunarPromo]
	views := []View{
		viewOf("L-1", "Evenly", lineItem("920.00", SpreadEvenly), f, rate("0.92")),
		viewOf("L-2", "Custom", lineItem("50.00", SpreadCustom, "2026-02", "50.00"), f, money.One),
	}

	grid := Grid(f, views)

	if got := columnKeys(grid); !slices.Equal(got, []string{"2026-02", NoDatesColumn}) || grid.Columns[1].Month != nil {
		t.Fatalf("columns = %v, want 2026-02 (not outside: there are no dates) and noDates without a month", got)
	}
	if got := cellTexts(grid.Rows[0].Cells); !slices.Equal(got, []string{"—", "1000.00"}) {
		t.Errorf("evenly row = %v, want — and 1000.00 in No dates yet", got)
	}
	if got := cellTexts(grid.Totals); !slices.Equal(got, []string{"50.00", "1000.00"}) || grid.Total.Amount.String() != "1050.00" {
		t.Errorf("totals = %v, total %s", got, grid.Total.Amount)
	}
}

// Each row holds the line item's USD months, which add up exactly to its USD planned cost; the totals row
// adds up each column and the grand total every row (FR-003, FR-020): Agency fee 50,000.00 USD and Film
// production 46,000.00 EUR, both evenly over Mar–May 2026.
func TestGrid_Totals_FR003_FR020(t *testing.T) {
	t.Parallel()
	f := newFakeActivities().facts[launchFilm]
	film := lineItem("46000.00", SpreadEvenly)
	film.Currency = "EUR"
	views := []View{
		viewOf("L-1001", "Agency fee", lineItem("50000.00", SpreadEvenly), f, money.One),
		viewOf("L-1002", "Film production", film, f, rate("0.92")),
		viewOf("L-1003", "Deposit", lineItem("10.00", SpreadCustom, "2026-04", "10.00"), f, money.One),
	}

	grid := Grid(f, views)

	rows := [][]string{
		{"16666.66", "16666.66", "16666.68"},
		{"16666.66", "16666.66", "16666.68"}, // 15,333.33 · 15,333.33 · 15,333.34 EUR
		{"—", "10.00", "—"},
	}
	for i, want := range rows {
		if got := cellTexts(grid.Rows[i].Cells); !slices.Equal(got, want) {
			t.Errorf("row %s = %v, want %v", grid.Rows[i].Name, got, want)
		}
	}
	if grid.Rows[1].Total.Amount.String() != "50000.00" || grid.Rows[1].LineItemID != "L-1002" {
		t.Errorf("Film production total = %s, want exactly its 50,000.00 USD", grid.Rows[1].Total.Amount)
	}
	if got := cellTexts(grid.Totals); !slices.Equal(got, []string{"33333.32", "33343.32", "33333.36"}) {
		t.Errorf("totals = %v", got)
	}
	if grid.Total != money.USD(amount("100010.00")) {
		t.Errorf("grand total = %+v, want 100,010.00 USD", grid.Total)
	}
}
