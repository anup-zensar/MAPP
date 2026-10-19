package campaigns

import "context"

// Repository stores campaigns and initiatives. The JSON store implements it now (store_json.go) and
// PostgreSQL can later (indexes on campaigns(plan_id) and initiatives(campaign_id)). Every method runs inside
// the caller's transaction, which the context carries. Lists are ordered by id and never nil; records in the
// recycle bin are not in the tables.
type Repository interface {
	// GetCampaign returns the campaign with id, and false when there is none.
	GetCampaign(ctx context.Context, id string) (Campaign, bool, error)
	// ListCampaigns returns every campaign of every plan.
	ListCampaigns(ctx context.Context) ([]Campaign, error)
	// CampaignsOfPlan returns the campaigns of the plan planID.
	CampaignsOfPlan(ctx context.Context, planID string) ([]Campaign, error)
	// CampaignsOfPlans returns the campaigns of every plan in planIDs (plan lists and roll-ups).
	CampaignsOfPlans(ctx context.Context, planIDs []string) ([]Campaign, error)
	// PutCampaign saves c, replacing the campaign with the same id.
	PutCampaign(ctx context.Context, c Campaign) error
	// DeleteCampaign removes the campaign with id; removing one that does not exist changes nothing.
	DeleteCampaign(ctx context.Context, id string) error

	// GetInitiative returns the initiative with id, and false when there is none.
	GetInitiative(ctx context.Context, id string) (Initiative, bool, error)
	// ListInitiatives returns every initiative of every campaign.
	ListInitiatives(ctx context.Context) ([]Initiative, error)
	// InitiativesOfCampaign returns the initiatives of the campaign campaignID.
	InitiativesOfCampaign(ctx context.Context, campaignID string) ([]Initiative, error)
	// InitiativesOfCampaigns returns the initiatives of every campaign in campaignIDs.
	InitiativesOfCampaigns(ctx context.Context, campaignIDs []string) ([]Initiative, error)
	// PutInitiative saves i, replacing the initiative with the same id.
	PutInitiative(ctx context.Context, i Initiative) error
	// DeleteInitiative removes the initiative with id.
	DeleteInitiative(ctx context.Context, id string) error
}
