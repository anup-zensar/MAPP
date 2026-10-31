package performance

import (
	"strings"

	"mapp/internal/platform/money"
)

const stageRevenue = "revenue"

// Measure is what a column of a source's file holds. The mapping names it, never the cell (FR-006), so one
// row carrying an INQ column and an MQL column produces one fact for each, under the same reference.
type Measure struct {
	Stage     string    `json:"stage"`
	WebMetric WebMetric `json:"webMetric"`
}

func (m Measure) compare(other Measure) int {
	if by := strings.Compare(m.Stage, other.Stage); by != 0 {
		return by
	}
	return strings.Compare(string(m.WebMetric), string(other.WebMetric))
}

type Reading struct {
	Measure
	Cell
}

func (r Reading) money() bool {
	return r.Stage == stageRevenue
}

type Row struct {
	Raw       map[string]string
	Reference Cell
	Date      Cell
	Currency  Cell
	Keys      RowKeys
	Readings  []Reading
}

// filled treats an empty measure cell as an absent fact rather than a zero one.
func (r Row) filled() []Reading {
	var filled []Reading
	for _, reading := range r.Readings {
		if strings.TrimSpace(reading.Value) != "" {
			filled = append(filled, reading)
		}
	}
	return filled
}

type Measured struct {
	Measure
	Count  *int64
	Amount *money.Money
}
