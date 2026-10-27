package stores

import (
	"context"

	"mapp/internal/campaign"
)

// CampaignStore is the campaigns and initiatives store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type CampaignStore interface {
	campaign.Store
	// ListCampaigns returns every campaign of every plan.
	ListCampaigns(ctx context.Context) ([]campaign.Campaign, error)
	// ListInitiatives returns every initiative of every campaign.
	ListInitiatives(ctx context.Context) ([]campaign.Initiative, error)
	// CountCampaignsOwnedBy and CountInitiativesOwnedBy count the stored records the person owns.
	CountCampaignsOwnedBy(ctx context.Context, personID string) (int, error)
	CountInitiativesOwnedBy(ctx context.Context, personID string) (int, error)
}
