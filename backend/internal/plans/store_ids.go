package plans

import (
	"context"
	"fmt"
	"slices"
)

// ListByIDs returns the stored plans among ids, ordered by id.
func (r jsonRepository) ListByIDs(ctx context.Context, ids []string) ([]Plan, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	kept := []Plan{}
	for _, p := range all {
		if slices.Contains(ids, p.ID) {
			kept = append(kept, p)
		}
	}
	return kept, nil
}

func (s *PostgresStore) ListByIDs(ctx context.Context, ids []string) ([]Plan, error) {
	rows, err := s.records().ByID(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("select plans: %w", err)
	}
	return plansFromRows(rows)
}
