package ids

import (
	"context"
	"fmt"

	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
)

// PostgresCounters keeps the counters in PostgreSQL. Raising a counter is one statement inside a Write,
// and Writes run one at a time, so no number is ever given out twice.
type PostgresCounters struct {
	db *database.DB
}

func NewPostgresCounters(db *database.DB) *PostgresCounters {
	return &PostgresCounters{db: db}
}

func (c *PostgresCounters) Advance(ctx context.Context, prefix string, count int) (int, error) {
	last, err := c.db.Queries(ctx).AdvanceCounter(ctx, sqlcdb.AdvanceCounterParams{
		Prefix: prefix, FirstNumber: firstNumber, Count: count,
	})
	if err != nil {
		return 0, fmt.Errorf("advance counter %s: %w", prefix, err)
	}
	return last, nil
}

func (c *PostgresCounters) Used(ctx context.Context) (bool, error) {
	used, err := c.db.Queries(ctx).AnyCounter(ctx)
	if err != nil {
		return false, fmt.Errorf("look for counters: %w", err)
	}
	return used, nil
}
