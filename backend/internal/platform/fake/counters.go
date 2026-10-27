package fake

import "context"

const firstNumber = 1001

type counter struct {
	Prefix string `json:"prefix"`
	Last   int    `json:"last"`
}

// Counters are the readable-ID counters (ids.Counters) of a DB; a failed Write takes its numbers back.
type Counters struct {
	table *Table[counter]
}

func NewCounters(db *DB) Counters {
	return Counters{table: TableOf(db, "counters", func(c counter) string { return c.Prefix })}
}

func (c Counters) Advance(ctx context.Context, prefix string, count int) (int, error) {
	last, found, err := c.table.Get(prefix)
	if err != nil {
		return 0, err
	}
	if !found {
		last = counter{Prefix: prefix, Last: firstNumber - 1}
	}
	last.Last += count
	if err = c.table.Put(ctx, last); err != nil {
		return 0, err
	}
	return last.Last, nil
}

// Used reports whether a number was given out: the seed loads only a DB that has given out none.
func (c Counters) Used() bool {
	c.table.mu.Lock()
	defer c.table.mu.Unlock()
	return len(c.table.rows) > 0
}
