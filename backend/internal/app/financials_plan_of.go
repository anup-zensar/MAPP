package app

import (
	"context"
	"strings"

	"mapp/internal/platform/ids"
)

// PlanOf returns the plan of the live planning record recordID, walking up through its parents; a record
// that is not stored, or a readable ID of no planning level, is not live.
func (p financialPlanning) PlanOf(ctx context.Context, recordID string) (string, bool, error) {
	prefix, _, _ := strings.Cut(recordID, "-")
	switch prefix {
	case ids.PrefixLineItem:
		return p.planOfLineItem(ctx, recordID)
	case ids.PrefixActivity:
		return p.planOfActivity(ctx, recordID)
	case ids.PrefixInitiative:
		return p.planOfInitiative(ctx, recordID)
	case ids.PrefixCampaign:
		return p.planOfCampaign(ctx, recordID)
	case ids.PrefixPlan:
		_, found, err := p.plans.Get(ctx, recordID)
		return recordID, found, err
	}
	return "", false, nil
}

func (p financialPlanning) planOfLineItem(ctx context.Context, id string) (string, bool, error) {
	li, found, err := p.lineItems.Get(ctx, id)
	if err != nil || !found {
		return "", false, err
	}
	return p.planOfActivity(ctx, li.ActivityID)
}

func (p financialPlanning) planOfActivity(ctx context.Context, id string) (string, bool, error) {
	a, found, err := p.activities.Get(ctx, id)
	if err != nil || !found {
		return "", false, err
	}
	return p.planOfInitiative(ctx, a.InitiativeID)
}

func (p financialPlanning) planOfInitiative(ctx context.Context, id string) (string, bool, error) {
	i, found, err := p.campaignsRepo.GetInitiative(ctx, id)
	if err != nil || !found {
		return "", false, err
	}
	return p.planOfCampaign(ctx, i.CampaignID)
}

func (p financialPlanning) planOfCampaign(ctx context.Context, id string) (string, bool, error) {
	c, found, err := p.campaignsRepo.GetCampaign(ctx, id)
	if err != nil || !found {
		return "", false, err
	}
	_, found, err = p.plans.Get(ctx, c.PlanID)
	return c.PlanID, found, err
}
