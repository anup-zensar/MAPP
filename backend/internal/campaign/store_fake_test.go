package campaign

import (
	"context"
	"slices"

	"mapp/internal/platform/fake"
)

// fakeStore is campaigntest.Store for this package's own tests, which cannot import campaigntest.
type fakeStore struct {
	campaigns   *fake.Table[Campaign]
	initiatives *fake.Table[Initiative]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{
		campaigns:   fake.TableOf(db, "campaigns", func(c Campaign) string { return c.ID }),
		initiatives: fake.TableOf(db, "initiatives", func(i Initiative) string { return i.ID }),
	}
}

func (s fakeStore) GetCampaign(_ context.Context, id string) (Campaign, bool, error) {
	return s.campaigns.Get(id)
}

func (s fakeStore) CampaignsOfPlan(ctx context.Context, planID string) ([]Campaign, error) {
	return s.CampaignsOfPlans(ctx, []string{planID})
}

func (s fakeStore) CampaignsOfPlans(_ context.Context, planIDs []string) ([]Campaign, error) {
	return s.campaigns.Where(func(c Campaign) bool { return slices.Contains(planIDs, c.PlanID) })
}

func (s fakeStore) CampaignsByIDs(_ context.Context, ids []string) ([]Campaign, error) {
	return s.campaigns.Where(func(c Campaign) bool { return slices.Contains(ids, c.ID) })
}

func (s fakeStore) PutCampaign(ctx context.Context, c Campaign) error {
	return s.campaigns.Put(ctx, normalisedCampaign(c))
}

func (s fakeStore) AddCampaigns(ctx context.Context, list []Campaign) error {
	stored := make([]Campaign, 0, len(list))
	for _, c := range list {
		stored = append(stored, normalisedCampaign(c))
	}
	return s.campaigns.Put(ctx, stored...)
}

func (s fakeStore) DeleteCampaign(ctx context.Context, id string) error {
	return s.campaigns.Delete(ctx, id)
}

func (s fakeStore) DeleteCampaigns(ctx context.Context, ids []string) error {
	return s.campaigns.Delete(ctx, ids...)
}

func (s fakeStore) GetInitiative(_ context.Context, id string) (Initiative, bool, error) {
	return s.initiatives.Get(id)
}

func (s fakeStore) InitiativesOfCampaign(ctx context.Context, campaignID string) ([]Initiative, error) {
	return s.InitiativesOfCampaigns(ctx, []string{campaignID})
}

func (s fakeStore) InitiativesOfCampaigns(_ context.Context, campaignIDs []string) ([]Initiative, error) {
	return s.initiatives.Where(func(i Initiative) bool { return slices.Contains(campaignIDs, i.CampaignID) })
}

func (s fakeStore) InitiativesByIDs(_ context.Context, ids []string) ([]Initiative, error) {
	return s.initiatives.Where(func(i Initiative) bool { return slices.Contains(ids, i.ID) })
}

func (s fakeStore) PutInitiative(ctx context.Context, i Initiative) error {
	return s.initiatives.Put(ctx, normalisedInitiative(i))
}

func (s fakeStore) AddInitiatives(ctx context.Context, list []Initiative) error {
	stored := make([]Initiative, 0, len(list))
	for _, i := range list {
		stored = append(stored, normalisedInitiative(i))
	}
	return s.initiatives.Put(ctx, stored...)
}

func (s fakeStore) DeleteInitiative(ctx context.Context, id string) error {
	return s.initiatives.Delete(ctx, id)
}

func (s fakeStore) DeleteInitiatives(ctx context.Context, ids []string) error {
	return s.initiatives.Delete(ctx, ids...)
}

func normalisedCampaign(c Campaign) Campaign {
	c.OwnerID = ownerColumn(c.OwnerID)
	c.Attributes = orEmpty(c.Attributes)
	c.BudgetPhasing = c.BudgetPhasing.Normalised()
	return c
}

func normalisedInitiative(i Initiative) Initiative {
	i.OwnerID = ownerColumn(i.OwnerID)
	i.Attributes = orEmpty(i.Attributes)
	i.BudgetPhasing = i.BudgetPhasing.Normalised()
	return i
}
