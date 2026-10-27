package lineitem

import (
	"context"
	"fmt"
)

func (s *PostgresStore) CountOwnedBy(ctx context.Context, personID string) (int, error) {
	n, err := s.db.Queries(ctx).CountLineItemsOwnedBy(ctx, personID)
	if err != nil {
		return 0, fmt.Errorf("count line items of %s: %w", personID, err)
	}
	return n, nil
}
