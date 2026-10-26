package campaigns

import (
	"context"

	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/memo"
)

// The store keeps the campaigns and initiatives a transaction reads, by id and by parent (memo): pages read
// the campaign and initiative of each record they show. A change forgets both.
const (
	campaignArea   = "campaigns"
	initiativeArea = "initiatives"
)

func (s *PostgresStore) change(ctx context.Context) *sqlcdb.Queries {
	memo.Forget(ctx, campaignArea)
	memo.Forget(ctx, initiativeArea)
	return s.db.Queries(ctx)
}

func (s *PostgresStore) campaignRecords() memo.Records[sqlcdb.ListCampaignsRow] {
	return memo.Records[sqlcdb.ListCampaignsRow]{
		Area:   campaignArea,
		ID:     func(r sqlcdb.ListCampaignsRow) string { return r.Campaign.ID },
		Parent: func(r sqlcdb.ListCampaignsRow) string { return r.Campaign.PlanID },
		Read: func(ctx context.Context, ids, planIDs []string) ([]sqlcdb.ListCampaignsRow, error) {
			return s.db.Queries(ctx).ListCampaigns(ctx, sqlcdb.ListCampaignsParams{Ids: ids, PlanIds: planIDs})
		},
	}
}

func (s *PostgresStore) initiativeRecords() memo.Records[sqlcdb.ListInitiativesRow] {
	return memo.Records[sqlcdb.ListInitiativesRow]{
		Area:   initiativeArea,
		ID:     func(r sqlcdb.ListInitiativesRow) string { return r.Initiative.ID },
		Parent: func(r sqlcdb.ListInitiativesRow) string { return r.Initiative.CampaignID },
		Read: func(ctx context.Context, ids, campaignIDs []string) ([]sqlcdb.ListInitiativesRow, error) {
			return s.db.Queries(ctx).ListInitiatives(ctx, sqlcdb.ListInitiativesParams{Ids: ids, CampaignIds: campaignIDs})
		},
	}
}
