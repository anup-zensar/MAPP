package campaigns

import (
	"context"
	"slices"

	"mapp/internal/platform/store"
)

// The JSON store tables of this package.
const (
	campaignsTable   = "campaigns"
	initiativesTable = "initiatives"
)

// JSONStore is the Store of the JSON store.
type JSONStore struct{}

// NewJSONStore returns the Store of the JSON store.
func NewJSONStore() JSONStore {
	return JSONStore{}
}

// GetCampaign reads one campaign, its phasing read with the defaults of older data (016 data-model §4.5).
func (JSONStore) GetCampaign(ctx context.Context, id string) (Campaign, bool, error) {
	c, found, err := store.Get[Campaign](ctx, campaignsTable, id)
	c.BudgetPhasing = c.BudgetPhasing.Normalised()
	return c, found, err
}

// ListCampaigns reads the whole campaigns table, each phasing with the defaults of older data.
func (JSONStore) ListCampaigns(ctx context.Context) ([]Campaign, error) {
	campaigns, err := store.List[Campaign](ctx, campaignsTable)
	for i := range campaigns {
		campaigns[i].BudgetPhasing = campaigns[i].BudgetPhasing.Normalised()
	}
	return campaigns, err
}

// CampaignsOfPlan reads every campaign to find those of one plan: quick for the records of a demo.
func (r JSONStore) CampaignsOfPlan(ctx context.Context, planID string) ([]Campaign, error) {
	return r.CampaignsOfPlans(ctx, []string{planID})
}

// CampaignsOfPlans reads every campaign to find those of the plans.
func (r JSONStore) CampaignsOfPlans(ctx context.Context, planIDs []string) ([]Campaign, error) {
	all, err := r.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	return where(all, func(c Campaign) bool { return slices.Contains(planIDs, c.PlanID) }), nil
}

// PutCampaign saves a campaign under its id.
func (JSONStore) PutCampaign(ctx context.Context, c Campaign) error {
	return store.Put(ctx, campaignsTable, c.ID, c)
}

// DeleteCampaign removes one campaign.
func (JSONStore) DeleteCampaign(ctx context.Context, id string) error {
	return store.Delete(ctx, campaignsTable, id)
}

// GetInitiative reads one initiative, its phasing read with the defaults of older data (016 data-model §4.5).
func (JSONStore) GetInitiative(ctx context.Context, id string) (Initiative, bool, error) {
	i, found, err := store.Get[Initiative](ctx, initiativesTable, id)
	i.BudgetPhasing = i.BudgetPhasing.Normalised()
	return i, found, err
}

// ListInitiatives reads the whole initiatives table, each phasing with the defaults of older data.
func (JSONStore) ListInitiatives(ctx context.Context) ([]Initiative, error) {
	initiatives, err := store.List[Initiative](ctx, initiativesTable)
	for i := range initiatives {
		initiatives[i].BudgetPhasing = initiatives[i].BudgetPhasing.Normalised()
	}
	return initiatives, err
}

// InitiativesOfCampaign reads every initiative to find those of one campaign.
func (r JSONStore) InitiativesOfCampaign(ctx context.Context, campaignID string) ([]Initiative, error) {
	return r.InitiativesOfCampaigns(ctx, []string{campaignID})
}

// InitiativesOfCampaigns reads every initiative to find those of the campaigns.
func (r JSONStore) InitiativesOfCampaigns(ctx context.Context, campaignIDs []string) ([]Initiative, error) {
	all, err := r.ListInitiatives(ctx)
	if err != nil {
		return nil, err
	}
	return where(all, func(i Initiative) bool { return slices.Contains(campaignIDs, i.CampaignID) }), nil
}

// PutInitiative saves an initiative under its id.
func (JSONStore) PutInitiative(ctx context.Context, i Initiative) error {
	return store.Put(ctx, initiativesTable, i.ID, i)
}

// DeleteInitiative removes one initiative.
func (JSONStore) DeleteInitiative(ctx context.Context, id string) error {
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
