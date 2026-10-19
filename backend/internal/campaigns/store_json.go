package campaigns

import (
	"context"
	"slices"

	"northstar/internal/platform/store"
)

// The JSON store tables of this package.
const (
	campaignsTable   = "campaigns"
	initiativesTable = "initiatives"
)

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// GetCampaign reads one campaign.
func (jsonRepository) GetCampaign(ctx context.Context, id string) (Campaign, bool, error) {
	return store.Get[Campaign](ctx, campaignsTable, id)
}

// ListCampaigns reads the whole campaigns table.
func (jsonRepository) ListCampaigns(ctx context.Context) ([]Campaign, error) {
	return store.List[Campaign](ctx, campaignsTable)
}

// CampaignsOfPlan reads every campaign to find those of one plan: quick for the records of a demo.
func (r jsonRepository) CampaignsOfPlan(ctx context.Context, planID string) ([]Campaign, error) {
	return r.CampaignsOfPlans(ctx, []string{planID})
}

// CampaignsOfPlans reads every campaign to find those of the plans.
func (r jsonRepository) CampaignsOfPlans(ctx context.Context, planIDs []string) ([]Campaign, error) {
	all, err := r.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	return where(all, func(c Campaign) bool { return slices.Contains(planIDs, c.PlanID) }), nil
}

// PutCampaign saves a campaign under its id.
func (jsonRepository) PutCampaign(ctx context.Context, c Campaign) error {
	return store.Put(ctx, campaignsTable, c.ID, c)
}

// DeleteCampaign removes one campaign.
func (jsonRepository) DeleteCampaign(ctx context.Context, id string) error {
	return store.Delete(ctx, campaignsTable, id)
}

// GetInitiative reads one initiative.
func (jsonRepository) GetInitiative(ctx context.Context, id string) (Initiative, bool, error) {
	return store.Get[Initiative](ctx, initiativesTable, id)
}

// ListInitiatives reads the whole initiatives table.
func (jsonRepository) ListInitiatives(ctx context.Context) ([]Initiative, error) {
	return store.List[Initiative](ctx, initiativesTable)
}

// InitiativesOfCampaign reads every initiative to find those of one campaign.
func (r jsonRepository) InitiativesOfCampaign(ctx context.Context, campaignID string) ([]Initiative, error) {
	return r.InitiativesOfCampaigns(ctx, []string{campaignID})
}

// InitiativesOfCampaigns reads every initiative to find those of the campaigns.
func (r jsonRepository) InitiativesOfCampaigns(ctx context.Context, campaignIDs []string) ([]Initiative, error) {
	all, err := r.ListInitiatives(ctx)
	if err != nil {
		return nil, err
	}
	return where(all, func(i Initiative) bool { return slices.Contains(campaignIDs, i.CampaignID) }), nil
}

// PutInitiative saves an initiative under its id.
func (jsonRepository) PutInitiative(ctx context.Context, i Initiative) error {
	return store.Put(ctx, initiativesTable, i.ID, i)
}

// DeleteInitiative removes one initiative.
func (jsonRepository) DeleteInitiative(ctx context.Context, id string) error {
	return store.Delete(ctx, initiativesTable, id)
}

// where returns the records that keep accepts, in order; never nil.
func where[T any](records []T, keep func(T) bool) []T {
	kept := []T{}
	for _, r := range records {
		if keep(r) {
			kept = append(kept, r)
		}
	}
	return kept
}
