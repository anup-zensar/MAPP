package lineitems

import (
	"context"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/money"
)

// The amounts the line-item section and page show — a line item's planned cost in USD, the unassigned
// balance and the spread grid — are answered in the reader's display currency (007 FR-005, D-061, D-213):
// worked out first (the balance and the grid in the plan currency, 016 FR-015), then each amount converted
// once at the current rates, half up to the cent. The line-item form keeps the plan currency (its "After
// saving" figure is calculated from what is typed, and the web app never converts), so FormOptions answers
// its balance unconverted.

// display converts amounts of any currency with a current rate into one display currency.
type display struct {
	money.Display
}

// displayOf returns the display currency of the person a, with the current rates of every currency; an
// unusable rate shows USD.
func (s *Service) displayOf(ctx context.Context, a actor.Actor) (display, error) {
	d, err := s.deps.People.DisplayRateOf(ctx, a.ID)
	if err != nil {
		return display{}, err
	}
	rates, err := s.deps.Currencies.Rates(ctx)
	if err != nil {
		return display{}, err
	}
	if _, err := money.ParseRate(d.Rate); err != nil || d.Code == "" {
		return display{money.DisplayUSD.WithRates(rates)}, nil
	}
	return display{money.DisplayIn(d.Code, d.Rate).WithRates(rates)}, nil
}

// money converts m.
func (d display) money(m money.Money) money.Money {
	return d.Money(m)
}

// optional converts m, or gives nil when there is no amount.
func (d display) optional(m *money.Money) *money.Money {
	return d.Optional(m)
}

// view returns v with its USD planned cost in the display currency (View.PlannedCostDisplay).
func (d display) view(v View) View {
	v.PlannedCostDisplay = d.money(v.PlannedCostUSD)
	return v
}

// views converts every view of views; never nil.
func (d display) views(views []View) []View {
	converted := make([]View, 0, len(views))
	for _, v := range views {
		converted = append(converted, d.view(v))
	}
	return converted
}

// balance returns the amounts of b, worked out in the plan currency, in the display currency. Its state and
// rounding hint stay as worked out in the plan currency.
func (d display) balance(b UnassignedBalance) UnassignedBalance {
	b.PlannedCost = d.optional(b.PlannedCost)
	b.Assigned = d.money(b.Assigned)
	b.Balance = d.optional(b.Balance)
	return b
}

// grid returns g, built in the plan currency, with every cell and total converted on its own: in the plan
// currency the cells add up exactly; converted, a total can differ by a cent from the sum of its cells.
func (d display) grid(g SpreadGrid) SpreadGrid {
	converted := SpreadGrid{Columns: g.Columns, Rows: make([]SpreadGridRow, 0, len(g.Rows)), Total: d.money(g.Total)}
	converted.Totals = d.cells(g.Totals)
	for _, row := range g.Rows {
		converted.Rows = append(converted.Rows, SpreadGridRow{
			LineItemID: row.LineItemID, Name: row.Name, Cells: d.cells(row.Cells), Total: d.money(row.Total),
		})
	}
	return converted
}

// cells converts each amount of cells, keeping the empty ones empty.
func (d display) cells(cells []*money.Money) []*money.Money {
	converted := make([]*money.Money, len(cells))
	for i, cell := range cells {
		converted[i] = d.optional(cell)
	}
	return converted
}
