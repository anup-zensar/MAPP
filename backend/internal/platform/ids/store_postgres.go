package ids

import (
	"context"
	"fmt"

	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
)

// PostgresCounters keeps the counters in PostgreSQL. Raising a counter is one statement, and a Write that
// races another for the same prefix runs again, so no number is ever given out twice.
type PostgresCounters struct {
	db *database.DB
}

func NewPostgresCounters(db *database.DB) *PostgresCounters {
	return &PostgresCounters{db: db}
}

func (c *PostgresCounters) Increment(ctx context.Context, prefix string) (int, error) {
	last, err := c.db.Queries(ctx).IncrementCounter(ctx, sqlcdb.IncrementCounterParams{Prefix: prefix, FirstNumber: firstNumber})
	if err != nil {
		return 0, fmt.Errorf("increment counter %s: %w", prefix, err)
	}
	return last, nil
}
