package campaigns

import (
	"context"
	"slices"
)

func (s *PostgresStore) CampaignsByIDs(ctx context.Context, ids []string) ([]Campaign, error) {
	return campaignsFromRows(s.campaignRecords().ByID(ctx, ids))
}

func (s *PostgresStore) InitiativesByIDs(ctx context.Context, ids []string) ([]Initiative, error) {
	return initiativesFromRows(s.initiativeRecords().ByID(ctx, ids))
}

func (r JSONStore) CampaignsByIDs(ctx context.Context, ids []string) ([]Campaign, error) {
	all, err := r.ListCampaigns(ctx)
	return where(all, func(c Campaign) bool { return slices.Contains(ids, c.ID) }), err
}

func (r JSONStore) InitiativesByIDs(ctx context.Context, ids []string) ([]Initiative, error) {
	all, err := r.ListInitiatives(ctx)
	return where(all, func(i Initiative) bool { return slices.Contains(ids, i.ID) }), err
}
