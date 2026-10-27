package campaign

import "context"

func (s *PostgresStore) CampaignsByIDs(ctx context.Context, ids []string) ([]Campaign, error) {
	return campaignsFromRows(s.campaignRecords().ByID(ctx, ids))
}

func (s *PostgresStore) InitiativesByIDs(ctx context.Context, ids []string) ([]Initiative, error) {
	return initiativesFromRows(s.initiativeRecords().ByID(ctx, ids))
}
