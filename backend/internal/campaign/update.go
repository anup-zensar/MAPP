package campaign

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// UpdateCampaign saves the campaign form of the campaign id (FR-018), whatever the plan's status. The
// version must be the one the person started from. New dates must still contain every initiative's dates
// (FR-007; checked only when the dates change, D-095). A save that changes nothing answers the campaign as
// it is, with no new version and no log entry; otherwise one "changed" entry lists every change (FR-030).
func (s *Service) UpdateCampaign(ctx context.Context, a actor.Actor, id string, in CampaignUpdate) (CampaignView, error) {
	var view CampaignView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, plan, err := s.campaignToUpdate(ctx, id, in.Version)
		if err != nil {
			return err
		}
		updated, err := s.checkCampaignUpdate(ctx, stored, plan, in)
		if err != nil {
			return err
		}
		changes, err := campaignChanges(stored, updated, plan)
		if err != nil {
			return err
		}
		if err = s.saveCampaignChanges(ctx, a, updated, changes); err != nil {
			return err
		}
		view, err = s.campaignView(ctx, id)
		return err
	})
	if err != nil {
		return CampaignView{}, err
	}
	return s.shownCampaign(ctx, a, view)
}

func (s *Service) campaignToUpdate(ctx context.Context, id string, version *int) (Campaign, PlanFacts, error) {
	stored, found, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return Campaign{}, PlanFacts{}, err
	}
	if !found {
		return Campaign{}, PlanFacts{}, errs.NotFound(campaignGoneMessage)
	}
	if err = checkVersionIfSent(stored.Meta, version, campaignSingular); err != nil {
		return Campaign{}, PlanFacts{}, err
	}
	plan, found, err := s.deps.Plans.Plan(ctx, stored.PlanID)
	if err != nil {
		return Campaign{}, PlanFacts{}, err
	}
	if !found {
		return Campaign{}, PlanFacts{}, errs.NotFound(planGoneMessage)
	}
	return stored, plan, nil
}

func (s *Service) checkCampaignUpdate(ctx context.Context, stored Campaign, plan PlanFacts, in CampaignUpdate) (Campaign, error) {
	siblings, err := s.repo.CampaignsOfPlan(ctx, stored.PlanID)
	if err != nil {
		return Campaign{}, err
	}
	var problems errs.Problems
	f := checkForm(&problems, form{in.Name, in.Description, in.OwnerID, in.StartDate, in.EndDate, in.Budget, in.Attributes},
		plan.Currency, func(name string) string { return campaignNameTaken(siblings, plan, name, stored.ID) })
	if err = s.checkOwner(ctx, &problems, f.OwnerID, stored.OwnerID); err != nil {
		return Campaign{}, err
	}
	if err = s.checkInitiativesInside(ctx, &problems, stored, f); err != nil {
		return Campaign{}, err
	}
	mode, err := changedOverspend(ctx, &problems, in.Overspend, ownedSetting{stored.Overspend, stored.OwnerID, plan})
	if err != nil {
		return Campaign{}, err
	}
	values, err := s.checkAttributes(ctx, &problems, attributesSent{settings.LevelCampaign, plan, in.Attributes, stored.Attributes})
	if err != nil {
		return Campaign{}, err
	}
	checkVersionSent(&problems, in.Version)
	if err = problems.Err(); err != nil {
		return Campaign{}, err
	}
	updated := updatedCampaign(stored, f, values)
	updated.Overspend = mode
	return updated, nil
}

// checkInitiativesInside runs only when the form sets both dates to new ones: unchanged dates are not checked again.
func (s *Service) checkInitiativesInside(ctx context.Context, problems *errs.Problems, stored Campaign, f checked) error {
	if !datesChanged(stored.StartDate, stored.EndDate, f.StartDate, f.EndDate) || f.StartDate == nil || f.EndDate == nil {
		return nil
	}
	initiatives, err := s.repo.InitiativesOfCampaign(ctx, stored.ID)
	if err != nil {
		return err
	}
	children := make([]dated, 0, len(initiatives))
	for _, i := range initiatives {
		children = append(children, dated{name: i.Name, start: i.StartDate, end: i.EndDate})
	}
	checkChildrenInside(problems, initiativePlural, children, *f.StartDate, *f.EndDate)
	return nil
}

func updatedCampaign(stored Campaign, f checked, values settings.AttributeValues) Campaign {
	updated := stored
	updated.Name, updated.Description, updated.OwnerID = f.Name, f.Description, f.OwnerID
	updated.StartDate, updated.EndDate, updated.Budget, updated.Attributes = f.StartDate, f.EndDate, f.Budget, values
	if updated.Budget == nil { // a removed budget removes its phasing (016 FR-022)
		updated.BudgetPhasing = rollup.NotPhased
	}
	return updated
}

func campaignChanges(stored, updated Campaign, plan PlanFacts) ([]history.Change, error) {
	withoutValues := func(c Campaign) Campaign { c.Attributes, c.BudgetPhasing = nil, rollup.Phasing{}; return c }
	changes, err := recordChanges(withoutValues(stored), withoutValues(updated), stored.Attributes, updated.Attributes)
	if err != nil {
		return nil, err
	}
	return append(changes, phasingChanges(stored.BudgetPhasing, updated.BudgetPhasing, stored.Budget, updated.Budget, plan)...), nil
}

// saveCampaignChanges writes nothing for a save without changes: no new version and no log entry.
func (s *Service) saveCampaignChanges(ctx context.Context, a actor.Actor, c Campaign, changes []history.Change) error {
	if len(changes) == 0 {
		return nil
	}
	c.Touch(a, s.now())
	if err := s.repo.PutCampaign(ctx, c); err != nil {
		return err
	}
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: campaignKind, RecordID: c.ID, RecordName: c.Name, Action: history.ActionChanged, Changes: changes,
	})
	return err
}

// UpdateInitiative saves the initiative form of the initiative id (FR-018). When the dates change (D-095)
// they must lie inside the campaign's (R3, FR-006) and still contain every activity's dates — only the
// start of an ongoing activity (FR-007).
func (s *Service) UpdateInitiative(ctx context.Context, a actor.Actor, id string, in InitiativeUpdate) (InitiativeView, error) {
	var view InitiativeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		e, err := s.initiativeToUpdate(ctx, id, in.Version)
		if err != nil {
			return err
		}
		updated, err := s.checkInitiativeUpdate(ctx, e, in)
		if err != nil {
			return err
		}
		changes, err := initiativeChanges(e.stored, updated, e.plan)
		if err != nil {
			return err
		}
		if err = s.saveInitiativeChanges(ctx, a, updated, changes); err != nil {
			return err
		}
		view, err = s.initiativeView(ctx, id)
		return err
	})
	if err != nil {
		return InitiativeView{}, err
	}
	return s.shownInitiative(ctx, a, view)
}

type initiativeEdit struct {
	stored   Initiative
	campaign Campaign
	plan     PlanFacts
}

func (s *Service) initiativeToUpdate(ctx context.Context, id string, version *int) (initiativeEdit, error) {
	stored, found, err := s.repo.GetInitiative(ctx, id)
	if err != nil {
		return initiativeEdit{}, err
	}
	if !found {
		return initiativeEdit{}, errs.NotFound(initiativeGoneMessage)
	}
	if err = checkVersionIfSent(stored.Meta, version, initiativeSingular); err != nil {
		return initiativeEdit{}, err
	}
	c, found, err := s.repo.GetCampaign(ctx, stored.CampaignID)
	if err != nil {
		return initiativeEdit{}, err
	}
	if !found {
		return initiativeEdit{}, errs.NotFound(campaignGoneMessage)
	}
	plan, found, err := s.deps.Plans.Plan(ctx, c.PlanID)
	if err != nil {
		return initiativeEdit{}, err
	}
	if !found {
		return initiativeEdit{}, errs.NotFound(planGoneMessage)
	}
	return initiativeEdit{stored: stored, campaign: c, plan: plan}, nil
}

func (s *Service) checkInitiativeUpdate(ctx context.Context, e initiativeEdit, in InitiativeUpdate) (Initiative, error) {
	siblings, err := s.repo.InitiativesOfCampaign(ctx, e.campaign.ID)
	if err != nil {
		return Initiative{}, err
	}
	var problems errs.Problems
	f := checkForm(&problems, form{in.Name, in.Description, in.OwnerID, in.StartDate, in.EndDate, in.Budget, in.Attributes},
		e.plan.Currency, func(name string) string { return initiativeNameTaken(siblings, e.campaign, name, e.stored.ID) })
	if err = s.checkOwner(ctx, &problems, f.OwnerID, e.stored.OwnerID); err != nil {
		return Initiative{}, err
	}
	if err = s.checkInitiativeDates(ctx, &problems, e, f); err != nil {
		return Initiative{}, err
	}
	mode, err := changedOverspend(ctx, &problems, in.Overspend, ownedSetting{e.stored.Overspend, e.stored.OwnerID, e.plan})
	if err != nil {
		return Initiative{}, err
	}
	values, err := s.checkAttributes(ctx, &problems, attributesSent{settings.LevelInitiative, e.plan, in.Attributes, e.stored.Attributes})
	if err != nil {
		return Initiative{}, err
	}
	checkVersionSent(&problems, in.Version)
	if err = problems.Err(); err != nil {
		return Initiative{}, err
	}
	updated := updatedInitiative(e.stored, f, values)
	updated.Overspend = mode
	return updated, nil
}

// checkInitiativeDates, like checkInitiativesInside, runs only when the form changes the dates.
func (s *Service) checkInitiativeDates(ctx context.Context, problems *errs.Problems, e initiativeEdit, f checked) error {
	if !datesChanged(e.stored.StartDate, e.stored.EndDate, f.StartDate, f.EndDate) {
		return nil
	}
	checkInsideCampaign(problems, e.campaign, f.StartDate, f.EndDate)
	if f.StartDate == nil || f.EndDate == nil {
		return nil
	}
	activities, err := s.deps.Activities.Summaries(ctx, []string{e.stored.ID})
	if err != nil {
		return err
	}
	children := make([]dated, 0, len(activities))
	for _, a := range activities {
		children = append(children, dated{name: a.Name, start: a.StartDate, end: a.EndDate, ongoing: a.Ongoing})
	}
	checkChildrenInside(problems, activityPlural, children, *f.StartDate, *f.EndDate)
	return nil
}

func updatedInitiative(stored Initiative, f checked, values settings.AttributeValues) Initiative {
	updated := stored
	updated.Name, updated.Description, updated.OwnerID = f.Name, f.Description, f.OwnerID
	updated.StartDate, updated.EndDate, updated.Budget, updated.Attributes = f.StartDate, f.EndDate, f.Budget, values
	if updated.Budget == nil { // a removed budget removes its phasing (016 FR-022)
		updated.BudgetPhasing = rollup.NotPhased
	}
	return updated
}

func initiativeChanges(stored, updated Initiative, plan PlanFacts) ([]history.Change, error) {
	withoutValues := func(i Initiative) Initiative { i.Attributes, i.BudgetPhasing = nil, rollup.Phasing{}; return i }
	changes, err := recordChanges(withoutValues(stored), withoutValues(updated), stored.Attributes, updated.Attributes)
	if err != nil {
		return nil, err
	}
	return append(changes, phasingChanges(stored.BudgetPhasing, updated.BudgetPhasing, stored.Budget, updated.Budget, plan)...), nil
}

func (s *Service) saveInitiativeChanges(ctx context.Context, a actor.Actor, i Initiative, changes []history.Change) error {
	if len(changes) == 0 {
		return nil
	}
	i.Touch(a, s.now())
	if err := s.repo.PutInitiative(ctx, i); err != nil {
		return err
	}
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: initiativeKind, RecordID: i.ID, RecordName: i.Name, Action: history.ActionChanged, Changes: changes,
	})
	return err
}
