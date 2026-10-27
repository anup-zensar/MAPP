package plan

import (
	"context"
	"fmt"
)

func (s *PostgresStore) ListByIDs(ctx context.Context, ids []string) ([]Plan, error) {
	rows, err := s.records().ByID(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("select plans: %w", err)
	}
	return plansFromRows(rows)
}
