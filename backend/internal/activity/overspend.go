package activity

import (
	"context"

	"mapp/internal/overspend"
)

// Overspend gives the budgets set to Refuse above an activity and what is left to plan in each (037 FR-014).
type Overspend interface {
	Limits(ctx context.Context, path overspend.Path) ([]overspend.Limit, error)
}

func (s *Service) UseOverspend(o Overspend) {
	s.overspend = o
}

func (s *Service) overspendLimits(ctx context.Context, pc planContext) ([]overspend.Limit, error) {
	if s.overspend == nil {
		return []overspend.Limit{}, nil
	}
	return s.overspend.Limits(ctx, overspend.Path{PlanID: pc.plan.ID, CampaignID: pc.campaign.ID,
		InitiativeID: pc.initiative.ID, PlanCurrency: pc.plan.Normalised().Currency})
}
