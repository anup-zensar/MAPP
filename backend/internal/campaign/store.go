package campaign

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists records ordered by id, never nil; a record in the recycle bin is not stored.
type Store interface {
	GetCampaign(ctx context.Context, id string) (Campaign, bool, error)
	CampaignsOfPlan(ctx context.Context, planID string) ([]Campaign, error)
	CampaignsOfPlans(ctx context.Context, planIDs []string) ([]Campaign, error)
	PutCampaign(ctx context.Context, c Campaign) error
	DeleteCampaign(ctx context.Context, id string) error

	CampaignsByIDs(ctx context.Context, ids []string) ([]Campaign, error)
	InitiativesByIDs(ctx context.Context, ids []string) ([]Initiative, error)
	GetInitiative(ctx context.Context, id string) (Initiative, bool, error)
	InitiativesOfCampaign(ctx context.Context, campaignID string) ([]Initiative, error)
	InitiativesOfCampaigns(ctx context.Context, campaignIDs []string) ([]Initiative, error)
	PutInitiative(ctx context.Context, i Initiative) error
	DeleteInitiative(ctx context.Context, id string) error
	// AddCampaigns and AddInitiatives save records whose ids are not stored yet, all at once.
	AddCampaigns(ctx context.Context, list []Campaign) error
	AddInitiatives(ctx context.Context, list []Initiative) error
	DeleteCampaigns(ctx context.Context, ids []string) error
	DeleteInitiatives(ctx context.Context, ids []string) error
}

var (
	_ Store = (*PostgresStore)(nil)
	_ Store = JSONStore{}
)
