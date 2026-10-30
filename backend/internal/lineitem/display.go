package lineitem

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// The section and page answer amounts in the reader's display currency, worked out in the plan currency and
// then converted once, half up to the cent (D-061, D-213). The form's balance stays in the plan currency:
// the web app never converts what is typed.

type display struct {
	money.Display
}

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

func (d display) money(m money.Money) money.Money {
	return d.Money(m)
}

func (d display) optional(m *money.Money) *money.Money {
	return d.Optional(m)
}

func (d display) view(v View) View {
	v.PlannedCostDisplay = d.money(v.PlannedCostUSD)
	return v
}

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

func (d display) grid(g SpreadGrid) SpreadGrid {
	converted := SpreadGrid{Columns: g.Columns, Rows: make([]SpreadGridRow, 0, len(g.Rows)),
		Totals: make([]*money.Money, len(g.Totals)), Total: d.money(money.Money{Currency: g.Total.Currency})}
	for _, row := range g.Rows {
		r := d.row(row)
		converted.Rows = append(converted.Rows, r)
		converted.Total.Amount = converted.Total.Amount.Add(r.Total.Amount)
		for i, cell := range r.Cells {
			if cell == nil {
				continue
			}
			if converted.Totals[i] == nil {
				converted.Totals[i] = &money.Money{Currency: cell.Currency}
			}
			converted.Totals[i].Amount = converted.Totals[i].Amount.Add(cell.Amount)
		}
	}
	return converted
}

func (d display) row(row SpreadGridRow) SpreadGridRow {
	columns := make([]int, 0, len(row.Cells))
	parts := make([]money.Money, 0, len(row.Cells))
	for i, cell := range row.Cells {
		if cell != nil {
			columns, parts = append(columns, i), append(parts, *cell)
		}
	}
	cells := make([]*money.Money, len(row.Cells))
	for j, part := range d.Parts(parts) {
		cells[columns[j]] = &part
	}
	return SpreadGridRow{LineItemID: row.LineItemID, Name: row.Name, Cells: cells, Total: d.money(row.Total)}
}
