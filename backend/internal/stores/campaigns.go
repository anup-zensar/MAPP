package stores

import (
	"context"

	"mapp/internal/campaigns"
)

// CampaignStore is the campaigns and initiatives store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type CampaignStore interface {
	campaigns.Store
	// ListCampaigns returns every campaign of every plan.
	ListCampaigns(ctx context.Context) ([]campaigns.Campaign, error)
	// ListInitiatives returns every initiative of every campaign.
	ListInitiatives(ctx context.Context) ([]campaigns.Initiative, error)
	// CountCampaignsOwnedBy and CountInitiativesOwnedBy count the stored records the person owns.
	CountCampaignsOwnedBy(ctx context.Context, personID string) (int, error)
	CountInitiativesOwnedBy(ctx context.Context, personID string) (int, error)
}
