package performance

import (
	"slices"
	"time"
)

type Breakdown struct {
	Measure
	Counts Counts `json:"counts"`
}

type Summary struct {
	Counts      Counts      `json:"counts"`
	Breakdown   []Breakdown `json:"breakdown"`
	DataThrough *time.Time  `json:"dataThrough,omitempty"`
}

// add is the only way a count rises, and Read rises only beside one of the five, so a Counts that fails
// Reconciles cannot be built (FR-022).
func (c *Counts) add(outcome Outcome) {
	switch outcome {
	case OutcomeMatched:
		c.Matched++
	case OutcomeHeld:
		c.Held++
	case OutcomeUnattributed:
		c.Unattributed++
	case OutcomeRejected:
		c.Rejected++
	case OutcomeUnmatched:
		c.Unmatched++
	default:
		return
	}
	c.Read++
}

// Summarise counts a row once overall and once per measure it fills, so groups do not add up to the totals.
func Summarise(results []RowResult) Summary {
	summary := Summary{Breakdown: []Breakdown{}}
	groups := map[Measure]*Counts{}
	for _, result := range results {
		summary.Counts.add(result.Outcome)
		for _, reading := range result.Row.filled() {
			counts, found := groups[reading.Measure]
			if !found {
				counts = &Counts{}
				groups[reading.Measure] = counts
			}
			counts.add(result.Outcome)
		}
		summary.DataThrough = laterOf(summary.DataThrough, result)
	}
	for measure, counts := range groups {
		summary.Breakdown = append(summary.Breakdown, Breakdown{Measure: measure, Counts: *counts})
	}
	slices.SortFunc(summary.Breakdown, func(a, b Breakdown) int { return a.compare(b.Measure) })
	return summary
}

// laterOf advances the data-through date only for a row that became a fact.
func laterOf(through *time.Time, result RowResult) *time.Time {
	if result.FactDate.IsZero() || result.Outcome == OutcomeRejected || result.Outcome == OutcomeUnmatched {
		return through
	}
	if through != nil && !result.FactDate.After(*through) {
		return through
	}
	return &result.FactDate
}
