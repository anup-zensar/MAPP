package ids

import (
	"context"

	"mapp/internal/platform/store"
)

// countersTable is the JSON store table that keeps the counters. The store saves it before every other
// table, so a counter never goes back behind an ID that was saved.
const countersTable = "counters"

// counter is one record of the counters table: the last number given for a prefix (data model §4).
type counter struct {
	Prefix string `json:"prefix"`
	Last   int    `json:"last"`
}

// jsonCounters is the Counters repository of the JSON store.
type jsonCounters struct{}

// NewJSONCounters returns the Counters repository of the JSON store.
func NewJSONCounters() Counters {
	return jsonCounters{}
}

// Advance raises the counter of prefix inside the caller's Write. A prefix without a counter yet starts
// at 1001.
func (jsonCounters) Advance(ctx context.Context, prefix string, count int) (int, error) {
	c, found, err := store.Get[counter](ctx, countersTable, prefix)
	if err != nil {
		return 0, err
	}
	if !found {
		c = counter{Prefix: prefix, Last: firstNumber - 1}
	}
	c.Last += count
	if err := store.Put(ctx, countersTable, prefix, c); err != nil {
		return 0, err
	}
	return c.Last, nil
}
