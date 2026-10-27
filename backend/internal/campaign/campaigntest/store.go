// Package campaigntest is the in-memory fake of the campaigns and initiatives store, for tests that build
// services or the whole API without a database.
package campaigntest

import (
	"context"
	"slices"

	"mapp/internal/campaign"
	"mapp/internal/platform/fake"
	"mapp/internal/settings"
)

type Store struct {
	campaigns   *fake.Table[campaign.Campaign]
	initiatives *fake.Table[campaign.Initiative]
}

func NewStore(db *fake.DB) Store {
	return Store{
		campaigns:   fake.TableOf(db, "campaigns", func(c campaign.Campaign) string { return c.ID }),
		initiatives: fake.TableOf(db, "initiatives", func(i campaign.Initiative) string { return i.ID }),
	}
}

func (s Store) GetCampaign(_ context.Context, id string) (campaign.Campaign, bool, error) {
	return s.campaigns.Get(id)
}

func (s Store) ListCampaigns(context.Context) ([]campaign.Campaign, error) {
	return s.campaigns.List()
}

func (s Store) CampaignsOfPlan(ctx context.Context, planID string) ([]campaign.Campaign, error) {
	return s.CampaignsOfPlans(ctx, []string{planID})
}

func (s Store) CampaignsOfPlans(_ context.Context, planIDs []string) ([]campaign.Campaign, error) {
	return s.campaigns.Where(func(c campaign.Campaign) bool { return slices.Contains(planIDs, c.PlanID) })
}

func (s Store) CampaignsByIDs(_ context.Context, ids []string) ([]campaign.Campaign, error) {
	return s.campaigns.Where(func(c campaign.Campaign) bool { return slices.Contains(ids, c.ID) })
}

func (s Store) PutCampaign(ctx context.Context, c campaign.Campaign) error {
	return s.campaigns.Put(ctx, normalisedCampaign(c))
}

func (s Store) AddCampaigns(ctx context.Context, list []campaign.Campaign) error {
	stored := make([]campaign.Campaign, 0, len(list))
	for _, c := range list {
		stored = append(stored, normalisedCampaign(c))
	}
	return s.campaigns.Put(ctx, stored...)
}

func (s Store) DeleteCampaign(ctx context.Context, id string) error {
	return s.campaigns.Delete(ctx, id)
}

func (s Store) DeleteCampaigns(ctx context.Context, ids []string) error {
	return s.campaigns.Delete(ctx, ids...)
}

func (s Store) CountCampaignsOwnedBy(_ context.Context, personID string) (int, error) {
	owned, err := s.campaigns.Where(func(c campaign.Campaign) bool { return isOwner(c.OwnerID, personID) })
	return len(owned), err
}

func (s Store) GetInitiative(_ context.Context, id string) (campaign.Initiative, bool, error) {
	return s.initiatives.Get(id)
}

func (s Store) ListInitiatives(context.Context) ([]campaign.Initiative, error) {
	return s.initiatives.List()
}

func (s Store) InitiativesOfCampaign(ctx context.Context, campaignID string) ([]campaign.Initiative, error) {
	return s.InitiativesOfCampaigns(ctx, []string{campaignID})
}

func (s Store) InitiativesOfCampaigns(_ context.Context, campaignIDs []string) ([]campaign.Initiative, error) {
	return s.initiatives.Where(func(i campaign.Initiative) bool { return slices.Contains(campaignIDs, i.CampaignID) })
}

func (s Store) InitiativesByIDs(_ context.Context, ids []string) ([]campaign.Initiative, error) {
	return s.initiatives.Where(func(i campaign.Initiative) bool { return slices.Contains(ids, i.ID) })
}

func (s Store) PutInitiative(ctx context.Context, i campaign.Initiative) error {
	return s.initiatives.Put(ctx, normalisedInitiative(i))
}

func (s Store) AddInitiatives(ctx context.Context, list []campaign.Initiative) error {
	stored := make([]campaign.Initiative, 0, len(list))
	for _, i := range list {
		stored = append(stored, normalisedInitiative(i))
	}
	return s.initiatives.Put(ctx, stored...)
}

func (s Store) DeleteInitiative(ctx context.Context, id string) error {
	return s.initiatives.Delete(ctx, id)
}

func (s Store) DeleteInitiatives(ctx context.Context, ids []string) error {
	return s.initiatives.Delete(ctx, ids...)
}

func (s Store) CountInitiativesOwnedBy(_ context.Context, personID string) (int, error) {
	owned, err := s.initiatives.Where(func(i campaign.Initiative) bool { return isOwner(i.OwnerID, personID) })
	return len(owned), err
}

// normalisedCampaign and normalisedInitiative are what the PostgreSQL store reads back: an owner "" is none,
// the attributes an object and the phasing normalised.
func normalisedCampaign(c campaign.Campaign) campaign.Campaign {
	c.OwnerID = ownerColumn(c.OwnerID)
	c.Attributes = attributesColumn(c.Attributes)
	c.BudgetPhasing = c.BudgetPhasing.Normalised()
	return c
}

func normalisedInitiative(i campaign.Initiative) campaign.Initiative {
	i.OwnerID = ownerColumn(i.OwnerID)
	i.Attributes = attributesColumn(i.Attributes)
	i.BudgetPhasing = i.BudgetPhasing.Normalised()
	return i
}

func ownerColumn(ownerID *string) *string {
	if ownerID == nil || *ownerID == "" {
		return nil
	}
	return ownerID
}

func attributesColumn(values settings.AttributeValues) settings.AttributeValues {
	if values == nil {
		return settings.AttributeValues{}
	}
	return values
}

func isOwner(ownerID *string, personID string) bool {
	return ownerID != nil && *ownerID == personID
}
