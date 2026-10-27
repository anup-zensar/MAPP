package allwork

import (
	"fmt"
	"time"

	"mapp/internal/platform/money"
)

// view works out the view of q over the year of b on 18 Oct 2026, in US dollars unless cur says otherwise.
func view(b *yearBuilder, q Query, cur ...Currency) Result {
	return viewOf(b.data(), q, cur...).result
}

// viewOf is view over prepared data, with the tree.
func viewOf(data YearData, q Query, cur ...Currency) built {
	c := usd
	if len(cur) > 0 {
		c = cur[0]
	}
	if q.View == "" {
		q.View = ViewGrid
	}
	if q.Strategic == "" {
		q.Strategic = StrategicInclude
	}
	if q.ShowLevels == "" {
		q.ShowLevels = ShowEverything
	}
	if q.Period == "" {
		q.Period = PeriodMonth
	}
	return build(input{
		query: q, year: 2026, today: fixtureToday, now: fixtureNow, currency: c, data: data,
		choices: fixtureChoices(priya.ID), catalogue: catalogue(fixtureAttributes), attributes: fixtureAttributes,
	})
}

// matchEverything marks every record a shown match, as the view without filters does.
func matchEverything(t *tree) {
	for _, n := range t.byID {
		n.match, n.shown = true, true
	}
}

// rowKeys returns the keys of rows, in order.
func rowKeys(rows []Row) []string {
	keys := make([]string, 0, len(rows))
	for _, r := range rows {
		keys = append(keys, r.Key)
	}
	return keys
}

// rowOf returns the row with key, and fails the test when there is none.
func rowOf(rows []Row, key string) Row {
	for _, r := range rows {
		if r.Key == key {
			return r
		}
	}
	panic("no row " + key)
}

// hasRow reports whether rows has the key.
func hasRow(rows []Row, key string) bool {
	for _, r := range rows {
		if r.Key == key {
			return true
		}
	}
	return false
}

// cellText writes a cell as a short text, for comparisons: "—", "50000.00 USD", "Draft/positive", nil as "".
func cellText(c *Cell) string {
	if c == nil {
		return ""
	}
	var text string
	switch c.Kind {
	case KindEmpty:
		text = "—"
	case KindMoney:
		text = c.Money.Amount.String() + " " + c.Money.Currency
	case KindNumber:
		text = fmt.Sprint(*c.Number)
	case KindPercent:
		text = c.Percent + " %"
	case KindDate:
		text = c.Date.String()
	case KindDateTime:
		text = c.DateTime.Format(time.RFC3339)
	case KindStatus:
		text = c.Text + "/" + string(c.Tone)
	default:
		text = c.Text
	}
	if c.Inactive {
		text += " (inactive)"
	}
	if c.Unfiltered != nil {
		text += " [all: " + cellText(c.Unfiltered) + "]"
	}
	return text
}

// money writes an optional amount as "1234.50 USD", "—" for none.
func moneyText(m *money.Money) string {
	if m == nil {
		return "—"
	}
	return m.Amount.String() + " " + m.Currency
}

// columnIndex returns the index of the column key in result's columns.
func columnIndex(r Result, key string) int {
	for i, c := range r.Columns {
		if c.Key == key {
			return i
		}
	}
	panic("no column " + key)
}

// cell returns the cell of column key in the row with rowKey.
func cell(r Result, rowKey, key string) string {
	return cellText(rowOf(r.Rows, rowKey).Cells[columnIndex(r, key)])
}
