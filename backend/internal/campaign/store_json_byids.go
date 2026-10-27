package campaign

import (
	"context"
	"slices"
)

func (r JSONStore) CampaignsByIDs(ctx context.Context, ids []string) ([]Campaign, error) {
	all, err := r.ListCampaigns(ctx)
	return where(all, func(c Campaign) bool { return slices.Contains(ids, c.ID) }), err
}

func (r JSONStore) InitiativesByIDs(ctx context.Context, ids []string) ([]Initiative, error) {
	all, err := r.ListInitiatives(ctx)
	return where(all, func(i Initiative) bool { return slices.Contains(ids, i.ID) }), err
}
