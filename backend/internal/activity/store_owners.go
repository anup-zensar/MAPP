package activity

import (
	"context"
	"fmt"
)

func (s *PostgresStore) CountOwnedBy(ctx context.Context, personID string) (int, error) {
	n, err := s.db.Queries(ctx).CountActivitiesOwnedBy(ctx, personID)
	if err != nil {
		return 0, fmt.Errorf("count activities of %s: %w", personID, err)
	}
	return n, nil
}
