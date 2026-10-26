package campaigns

import (
	"context"
	"fmt"
)

func (s *PostgresStore) CountCampaignsOwnedBy(ctx context.Context, personID string) (int, error) {
	n, err := s.db.Queries(ctx).CountCampaignsOwnedBy(ctx, personID)
	if err != nil {
		return 0, fmt.Errorf("count campaigns of %s: %w", personID, err)
	}
	return n, nil
}

func (s *PostgresStore) CountInitiativesOwnedBy(ctx context.Context, personID string) (int, error) {
	n, err := s.db.Queries(ctx).CountInitiativesOwnedBy(ctx, personID)
	if err != nil {
		return 0, fmt.Errorf("count initiatives of %s: %w", personID, err)
	}
	return n, nil
}

func (r jsonRepository) CountCampaignsOwnedBy(ctx context.Context, personID string) (int, error) {
	list, err := r.ListCampaigns(ctx)
	n := 0
	for _, c := range list {
		if c.OwnerID != nil && *c.OwnerID == personID {
			n++
		}
	}
	return n, err
}

func (r jsonRepository) CountInitiativesOwnedBy(ctx context.Context, personID string) (int, error) {
	list, err := r.ListInitiatives(ctx)
	n := 0
	for _, i := range list {
		if i.OwnerID != nil && *i.OwnerID == personID {
			n++
		}
	}
	return n, err
}
