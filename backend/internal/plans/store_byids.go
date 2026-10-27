package plans

import (
	"context"
	"fmt"
	"slices"
)

func (s *PostgresStore) ListByIDs(ctx context.Context, ids []string) ([]Plan, error) {
	rows, err := s.records().ByID(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("select plans: %w", err)
	}
	return plansFromRows(rows)
}

func (r JSONStore) ListByIDs(ctx context.Context, ids []string) ([]Plan, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	found := []Plan{}
	for _, p := range all {
		if slices.Contains(ids, p.ID) {
			found = append(found, p)
		}
	}
	return found, nil
}
