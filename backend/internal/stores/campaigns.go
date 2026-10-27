package stores

import "mapp/internal/campaigns"

// CampaignStore is the campaigns and initiatives store as the wiring and the seed use it: the service's Store and what other areas' adapters call.
type CampaignStore interface {
	campaigns.Store
}
