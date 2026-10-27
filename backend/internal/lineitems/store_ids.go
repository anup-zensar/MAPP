package lineitems

import (
	"context"
	"slices"
)

// ListByIDs returns the stored line items among ids, ordered by id.
func (r JSONStore) ListByIDs(ctx context.Context, ids []string) ([]LineItem, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	kept := []LineItem{}
	for _, li := range all {
		if slices.Contains(ids, li.ID) {
			kept = append(kept, li)
		}
	}
	return kept, nil
}

func (s *PostgresStore) ListByIDs(ctx context.Context, ids []string) ([]LineItem, error) {
	return lineItemsFromRows(s.records().ByID(ctx, ids))
}
