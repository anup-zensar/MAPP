package campaigns

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// ListPlanCampaigns returns the campaigns of the plan planID, each with its initiatives, in tree order
// (FR-020), and the plan's totals (FR-022).
func (s *Service) ListPlanCampaigns(ctx context.Context, a actor.Actor, planID string) (PlanCampaigns, error) {
	var result PlanCampaigns
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		plan, found, err := s.deps.Plans.Plan(ctx, planID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(planGoneMessage)
		}
		result, err = s.planCampaigns(ctx, plan)
		return err
	})
	if err != nil {
		return PlanCampaigns{}, err
	}
	return s.shownTree(ctx, a, result)
}

// GetCampaign returns the campaign id with its initiatives and every derived value (FR-016).
func (s *Service) GetCampaign(ctx context.Context, a actor.Actor, id string) (CampaignView, error) {
	var view CampaignView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		view, err = s.campaignView(ctx, id)
		return err
	})
	if err != nil {
		return CampaignView{}, err
	}
	return s.shownCampaign(ctx, a, view)
}

// GetInitiative returns the initiative id with every derived value (FR-016).
func (s *Service) GetInitiative(ctx context.Context, a actor.Actor, id string) (InitiativeView, error) {
	var view InitiativeView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		view, err = s.initiativeView(ctx, id)
		return err
	})
	if err != nil {
		return InitiativeView{}, err
	}
	return s.shownInitiative(ctx, a, view)
}

// campaignView builds the view of the campaign id inside the caller's transaction: its plan's whole tree is
// built, because the campaign's remaining plan budget depends on every other campaign of the plan.
func (s *Service) campaignView(ctx context.Context, id string) (CampaignView, error) {
	c, found, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return CampaignView{}, err
	}
	if !found {
		return CampaignView{}, errs.NotFound(campaignGoneMessage)
	}
	tree, err := s.treeOf(ctx, c.PlanID, campaignGoneMessage)
	if err != nil {
		return CampaignView{}, err
	}
	for _, view := range tree.Items {
		if view.ID == id {
			return view, nil
		}
	}
	return CampaignView{}, errs.NotFound(campaignGoneMessage)
}

// initiativeView builds the view of the initiative id inside the caller's transaction.
func (s *Service) initiativeView(ctx context.Context, id string) (InitiativeView, error) {
	i, found, err := s.repo.GetInitiative(ctx, id)
	if err != nil {
		return InitiativeView{}, err
	}
	if !found {
		return InitiativeView{}, errs.NotFound(initiativeGoneMessage)
	}
	campaign, err := s.campaignView(ctx, i.CampaignID)
	if err != nil {
		return InitiativeView{}, err
	}
	for _, view := range campaign.Initiatives {
		if view.ID == id {
			return view, nil
		}
	}
	return InitiativeView{}, errs.NotFound(initiativeGoneMessage)
}

// treeOf builds the campaign tree of the plan planID; a plan that is gone answers notFound, the message of
// the record that was asked for.
func (s *Service) treeOf(ctx context.Context, planID, notFound string) (PlanCampaigns, error) {
	plan, found, err := s.deps.Plans.Plan(ctx, planID)
	if err != nil {
		return PlanCampaigns{}, err
	}
	if !found {
		return PlanCampaigns{}, errs.NotFound(notFound)
	}
	return s.planCampaigns(ctx, plan)
}

// campaignAndPlan reads the campaign id and its plan inside the caller's transaction; a campaign that is
// gone, or whose plan is gone, is not found.
func (s *Service) campaignAndPlan(ctx context.Context, id string) (Campaign, PlanFacts, error) {
	c, found, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return Campaign{}, PlanFacts{}, err
	}
	if !found {
		return Campaign{}, PlanFacts{}, errs.NotFound(campaignGoneMessage)
	}
	plan, found, err := s.deps.Plans.Plan(ctx, c.PlanID)
	if err != nil {
		return Campaign{}, PlanFacts{}, err
	}
	if !found {
		return Campaign{}, PlanFacts{}, errs.NotFound(campaignGoneMessage)
	}
	return c, plan, nil
}
