package allwork

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
)

// fieldCell returns the cell of a level-specific value (FR-020): "—" when empty; a choice by its label,
// several joined with ", ", a % split with the shares ("Germany 60 %, France 40 %"); inactive values
// labelled; Yes or No; an amount in USD in the display currency, a local amount as it is.
func fieldCell(v Value, cur Currency) *Cell {
	switch {
	case v.USD != nil:
		return moneyCell(cur, v.USD)
	case v.Money != nil:
		m := *v.Money
		cell := &Cell{Kind: KindMoney, Money: &m}
		if m.Amount.Sign() < 0 {
			cell.Tone = ToneProblem
		}
		return cell
	case v.Number != nil:
		return numberCell(v.Number)
	case v.Yes != nil:
		return textCell(yesNo(*v.Yes))
	case len(v.Choices) == 1 && v.Choices[0].Share == nil:
		return &Cell{Kind: KindText, Text: v.Choices[0].Label, Inactive: v.Choices[0].Inactive}
	case len(v.Choices) > 0:
		parts := make([]string, 0, len(v.Choices))
		for _, c := range v.Choices {
			part := c.Label
			if c.Inactive {
				part += " (inactive)"
			}
			if c.Share != nil {
				part += " " + shareText(*c.Share) + " %"
			}
			parts = append(parts, part)
		}
		return textCell(strings.Join(parts, ", "))
	}
	cell := textCell(v.Text)
	cell.Inactive = v.Inactive && v.Text != ""
	return cell
}

// yesNo is a yes/no value as people read it.
func yesNo(yes bool) string {
	if yes {
		return "Yes"
	}
	return "No"
}

// shareText writes a share without trailing zeros: 60.00 → "60", 33.50 → "33.5".
func shareText(p money.Percent) string {
	return strings.TrimSuffix(strings.TrimRight(p.String(), "0"), ".")
}

// rolledCell returns the value of a roll-up column from figures f of the record t.
func rolledCell(key string, t *tnode, f figures, cur Currency) *Cell {
	switch key {
	case "budgetUsed":
		return budgetUsedCell(f.planned, t.n.Budget)
	case "plannedCost":
		return moneyCell(cur, f.planned)
	case "lineItems":
		return moneyCell(cur, f.lineItems)
	case "unassigned":
		return moneyCell(cur, f.unassigned)
	case "activities":
		count := int64(f.activities)
		return numberCell(&count)
	case "inq":
		return numberCell(f.inq)
	case "mql":
		return numberCell(f.mql)
	case "revenue":
		return moneyCell(cur, f.revenue)
	}
	if slices.Contains(financialKeys, key) {
		return financeCell(key, f, cur)
	}
	return emptyCell()
}

// ongoingWord is the End date of an ongoing activity.
const ongoingWord = "Ongoing"

// remainingCell is a plan's or campaign's own budget minus the budgets of its children; red when negative
// (D-117). "—" without a budget.
func remainingCell(t *tnode, cur Currency) *Cell {
	if t.n.Budget == nil {
		return emptyCell()
	}
	children := make([]*money.Money, 0, len(t.children))
	for _, c := range t.children {
		if c.n.Budget != nil {
			b := money.USD(*c.n.Budget)
			children = append(children, &b)
		}
	}
	budget := money.USD(*t.n.Budget)
	remaining := rollup.Remaining(&budget, rollup.Total(children))
	return moneyCell(cur, &remaining.Amount)
}

// budgetUsedCell is planned cost ÷ own budget as a whole per cent, half up (005's rule); red above 100.
// "—" without a budget, with a budget of 0.00 or without a planned cost.
func budgetUsedCell(planned, budget *money.Amount) *Cell {
	if planned == nil || budget == nil {
		return emptyCell()
	}
	p, b := money.USD(*planned), money.USD(*budget)
	percent := rollup.BudgetUsedPercent(&p, &b)
	if percent == nil {
		return emptyCell()
	}
	cell := &Cell{Kind: KindPercent, Percent: strconv.Itoa(*percent)}
	if *percent > 100 {
		cell.Tone = ToneProblem
	}
	return cell
}

// emptyCell is "—": the column's own level without a value.
func emptyCell() *Cell {
	return &Cell{Kind: KindEmpty}
}

// textCell is a text, "—" when empty.
func textCell(text string) *Cell {
	if text == "" {
		return emptyCell()
	}
	return &Cell{Kind: KindText, Text: text}
}

// moneyCell is an amount in USD shown in the display currency; red when negative. "—" for no amount.
func moneyCell(cur Currency, usd *money.Amount) *Cell {
	if usd == nil {
		return emptyCell()
	}
	m := cur.money(*usd)
	cell := &Cell{Kind: KindMoney, Money: &m}
	if m.Amount.Sign() < 0 {
		cell.Tone = ToneProblem
	}
	return cell
}

// planMoneyCell is an amount in its plan's currency, as it is (016 FR-037); red when negative, "—" for none.
func planMoneyCell(m *money.Money) *Cell {
	if m == nil {
		return emptyCell()
	}
	shown := *m
	cell := &Cell{Kind: KindMoney, Money: &shown}
	if shown.Amount.Sign() < 0 {
		cell.Tone = ToneProblem
	}
	return cell
}

// numberCell is a whole number, "—" for none.
func numberCell(n *int64) *Cell {
	if n == nil {
		return emptyCell()
	}
	v := *n
	return &Cell{Kind: KindNumber, Number: &v}
}

// dateCell is a date.
func dateCell(d date.Date) *Cell {
	return &Cell{Kind: KindDate, Date: d}
}

// optionalDateCell is a date, "—" for none.
func optionalDateCell(d *date.Date) *Cell {
	if d == nil {
		return emptyCell()
	}
	return dateCell(*d)
}

// dateTimeCell is a moment in UTC, "—" for none.
func dateTimeCell(t time.Time) *Cell {
	if t.IsZero() {
		return emptyCell()
	}
	utc := t.UTC()
	return &Cell{Kind: KindDateTime, DateTime: &utc}
}

// statusCell is a status chip, "—" for none.
func statusCell(s *Status) *Cell {
	if s == nil {
		return emptyCell()
	}
	return &Cell{Kind: KindStatus, Text: s.Label, Tone: s.Tone}
}
