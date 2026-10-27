package campaign

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// CreateCampaign adds a campaign to the plan planID (FR-001 – FR-005). Only the name is required; the API
// stores what it receives (D-093): the defaults of FR-003 are filled in by the form. Dates outside the
// plan's year are allowed and only flagged (FR-008); a budget larger than what remains is allowed (D-015).
// The campaign gets a readable ID (FR-004) and a "created" log entry (FR-030).
func (s *Service) CreateCampaign(ctx context.Context, a actor.Actor, planID string, in CampaignInput) (CampaignView, error) { //nolint:gocyclo // 11: one Write whose every step returns its error.
	var view CampaignView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		plan, found, err := s.deps.Plans.Plan(ctx, planID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(planGoneMessage)
		}
		siblings, err := s.repo.CampaignsOfPlan(ctx, planID)
		if err != nil {
			return err
		}

		var problems errs.Problems
		f := checkForm(&problems, form(in), plan.Currency, func(name string) string { return campaignNameTaken(siblings, plan, name, "") })
		if err = s.checkOwner(ctx, &problems, f.OwnerID, nil); err != nil {
			return err
		}
		values, err := s.checkAttributes(ctx, &problems, settings.LevelCampaign, plan, in.Attributes, nil)
		if err != nil {
			return err
		}
		if err = problems.Err(); err != nil {
			return err
		}

		id, err := s.ids.Next(ctx, ids.PrefixCampaign)
		if err != nil {
			return err
		}
		c := Campaign{
			ID: id, PlanID: planID, Name: f.Name, Description: f.Description, OwnerID: f.OwnerID,
			StartDate: f.StartDate, EndDate: f.EndDate, Budget: f.Budget, Attributes: values, BudgetPhasing: rollup.NotPhased,
			Meta: record.NewMeta(a, s.now()),
		}
		if err = s.repo.PutCampaign(ctx, c); err != nil {
			return err
		}
		if _, err = s.history.Log(ctx, a, history.LogInput{
			Kind: campaignKind, RecordID: c.ID, RecordName: c.Name, Action: history.ActionCreated,
		}); err != nil {
			return err
		}
		view, err = s.campaignView(ctx, c.ID)
		return err
	})
	if err != nil {
		return CampaignView{}, err
	}
	return s.shownCampaign(ctx, a, view)
}

// CreateInitiative adds an initiative to the campaign campaignID (FR-001 – FR-006). Only the name is
// required; its dates must lie inside the campaign's when the campaign has both (R3, FR-006).
func (s *Service) CreateInitiative(ctx context.Context, a actor.Actor, campaignID string, in InitiativeInput) (InitiativeView, error) { //nolint:gocyclo // 13: one Write whose every step returns its error.
	var view InitiativeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		campaign, found, err := s.repo.GetCampaign(ctx, campaignID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(campaignGoneMessage)
		}
		plan, found, err := s.deps.Plans.Plan(ctx, campaign.PlanID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(planGoneMessage)
		}
		siblings, err := s.repo.InitiativesOfCampaign(ctx, campaignID)
		if err != nil {
			return err
		}

		var problems errs.Problems
		f := checkForm(&problems, form(in), plan.Currency, func(name string) string { return initiativeNameTaken(siblings, campaign, name, "") })
		if err = s.checkOwner(ctx, &problems, f.OwnerID, nil); err != nil {
			return err
		}
		checkInsideCampaign(&problems, campaign, f.StartDate, f.EndDate)
		values, err := s.checkAttributes(ctx, &problems, settings.LevelInitiative, plan, in.Attributes, nil)
		if err != nil {
			return err
		}
		if err = problems.Err(); err != nil {
			return err
		}

		id, err := s.ids.Next(ctx, ids.PrefixInitiative)
		if err != nil {
			return err
		}
		i := Initiative{
			ID: id, CampaignID: campaignID, Name: f.Name, Description: f.Description, OwnerID: f.OwnerID,
			StartDate: f.StartDate, EndDate: f.EndDate, Budget: f.Budget, Attributes: values, BudgetPhasing: rollup.NotPhased,
			Meta: record.NewMeta(a, s.now()),
		}
		if err = s.repo.PutInitiative(ctx, i); err != nil {
			return err
		}
		if _, err = s.history.Log(ctx, a, history.LogInput{
			Kind: initiativeKind, RecordID: i.ID, RecordName: i.Name, Action: history.ActionCreated,
		}); err != nil {
			return err
		}
		view, err = s.initiativeView(ctx, i.ID)
		return err
	})
	if err != nil {
		return InitiativeView{}, err
	}
	return s.shownInitiative(ctx, a, view)
}
