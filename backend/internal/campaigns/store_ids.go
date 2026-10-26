package campaigns

import (
	"context"
	"slices"
)

// CampaignsByIDs returns the stored campaigns among ids, ordered by id.
func (r jsonRepository) CampaignsByIDs(ctx context.Context, ids []string) ([]Campaign, error) {
	all, err := r.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	return where(all, func(c Campaign) bool { return slices.Contains(ids, c.ID) }), nil
}

// InitiativesByIDs returns the stored initiatives among ids, ordered by id.
func (r jsonRepository) InitiativesByIDs(ctx context.Context, ids []string) ([]Initiative, error) {
	all, err := r.ListInitiatives(ctx)
	if err != nil {
		return nil, err
	}
	return where(all, func(i Initiative) bool { return slices.Contains(ids, i.ID) }), nil
}

func (s *PostgresStore) CampaignsByIDs(ctx context.Context, ids []string) ([]Campaign, error) {
	return campaignsFromRows(s.campaignRecords().ByID(ctx, ids))
}

func (s *PostgresStore) InitiativesByIDs(ctx context.Context, ids []string) ([]Initiative, error) {
	return initiativesFromRows(s.initiativeRecords().ByID(ctx, ids))
}
