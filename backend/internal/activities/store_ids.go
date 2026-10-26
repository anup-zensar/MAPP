package activities

import (
	"context"
	"slices"
)

// ListByIDs returns the stored activities among ids, ordered by id.
func (r jsonRepository) ListByIDs(ctx context.Context, ids []string) ([]Activity, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	kept := []Activity{}
	for _, a := range all {
		if slices.Contains(ids, a.ID) {
			kept = append(kept, a)
		}
	}
	return kept, nil
}

func (s *PostgresStore) ListByIDs(ctx context.Context, ids []string) ([]Activity, error) {
	return activitiesFromRows(s.records().ByID(ctx, ids))
}
