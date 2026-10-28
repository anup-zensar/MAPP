package app

import (
	"context"
	"errors"
	"fmt"

	"mapp/internal/activity"
	"mapp/internal/bulk"
	"mapp/internal/campaign"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

const (
	maxCopyNames     = 20
	alreadyThereText = "Already in '%s'"
	pathSeparator    = " › "
)

func (p bulkPlanning) Copy(ctx context.Context, a actor.Actor, r bulk.Record, targetID string) (string, error) {
	if err := p.permitChange(ctx, r, r.Level != bulk.LevelCampaign); err != nil {
		return "", err
	}
	switch r.Level {
	case bulk.LevelCampaign:
		return copyNamed(r.Name, func(name string) (string, error) {
			v, err := p.campaigns.CopyCampaign(ctx, a, r.ID, campaign.CampaignCopy{PlanID: targetID, Name: name})
			return v.ID, err
		})
	case bulk.LevelInitiative:
		return copyNamed(r.Name, func(name string) (string, error) {
			v, err := p.campaigns.CopyInitiative(ctx, a, r.ID, campaign.InitiativeCopy{CampaignID: targetID, Name: name})
			return v.ID, err
		})
	case bulk.LevelActivity:
	}
	v, err := p.activities.Copy(ctx, a, r.ID, activity.CopyInput{InitiativeID: targetID})
	return v.ID, err
}

// copyNamed copies under the record's name, or "<name> (copy)", "<name> (copy 2)", … when the name is taken
// (R13, D-131): the single copy refuses a taken name on its field "name".
func copyNamed(name string, copyAs func(name string) (string, error)) (string, error) {
	var err error
	for n := 0; n < maxCopyNames; n++ {
		candidate := name
		switch {
		case n == 1:
			candidate = name + " (copy)"
		case n > 1:
			candidate = fmt.Sprintf("%s (copy %d)", name, n)
		}
		var id string
		if id, err = copyAs(candidate); !isNameProblem(err) {
			return id, err
		}
	}
	return "", err
}

func isNameProblem(err error) bool {
	apiErr, ok := errors.AsType[*errs.Error](err)
	if !ok || apiErr.Code != errs.CodeValidationFailed {
		return false
	}
	for _, f := range apiErr.Fields {
		if f.Field == "name" {
			return true
		}
	}
	return false
}

func (p bulkPlanning) Move(ctx context.Context, a actor.Actor, r bulk.Record, targetID string) error {
	if err := p.permitChange(ctx, r, true); err != nil {
		return err
	}
	if len(r.Ancestors) > 0 && r.Ancestors[0] == targetID {
		return errs.Validation(errs.FieldError{Field: "targetId", Message: fmt.Sprintf(alreadyThereText, p.nameOf(ctx, r.Level, targetID))})
	}
	switch r.Level {
	case bulk.LevelCampaign:
		return p.moveCampaign(ctx, a, r.ID, targetID)
	case bulk.LevelInitiative:
		return p.moveInitiative(ctx, a, r.ID, targetID)
	case bulk.LevelActivity:
	}
	stored, found, err := p.activitiesRepo.Get(ctx, r.ID)
	if err != nil || !found {
		return readProblem(err, found)
	}
	_, err = p.activities.Move(ctx, a, r.ID, activity.MoveInput{InitiativeID: targetID, Version: &stored.Version})
	return err
}

func (p bulkPlanning) moveCampaign(ctx context.Context, a actor.Actor, id, planID string) error {
	stored, found, err := p.campaignsRepo.GetCampaign(ctx, id)
	if err != nil || !found {
		return readProblem(err, found)
	}
	_, err = p.campaigns.MoveCampaign(ctx, a, id, campaign.CampaignMove{PlanID: planID, Version: &stored.Version})
	return err
}

func (p bulkPlanning) moveInitiative(ctx context.Context, a actor.Actor, id, campaignID string) error {
	stored, found, err := p.campaignsRepo.GetInitiative(ctx, id)
	if err != nil || !found {
		return readProblem(err, found)
	}
	_, err = p.campaigns.MoveInitiative(ctx, a, id, campaign.InitiativeMove{CampaignID: campaignID, Version: &stored.Version})
	return err
}

// permitChange checks the edit permission a move or copy needs: in the record's plan, or for a campaign copy
// anywhere, the copy's service checking it at the destination (route table, 014).
func (p bulkPlanning) permitChange(ctx context.Context, r bulk.Record, inPlan bool) error {
	permission := authz.CampaignsEdit
	if r.Level == bulk.LevelActivity {
		permission = authz.ActivitiesEdit
	}
	if inPlan {
		return p.permit(ctx, r.ID, permission)
	}
	return authz.Require(ctx, permission)
}

// nameOf names the parent a record of level sits in: a plan, a campaign or an initiative.
func (p bulkPlanning) nameOf(ctx context.Context, level bulk.Level, id string) string {
	switch level {
	case bulk.LevelCampaign:
		plan, _, _ := p.plans.Get(ctx, id)
		return plan.Name
	case bulk.LevelInitiative:
		c, _, _ := p.campaignsRepo.GetCampaign(ctx, id)
		return c.Name
	case bulk.LevelActivity:
	}
	i, _, _ := p.campaignsRepo.GetInitiative(ctx, id)
	return i.Name
}

func (p bulkPlanning) Destinations(ctx context.Context, a actor.Actor, action bulk.Kind, r bulk.Record) ([]bulk.Target, error) {
	purpose := campaign.PurposeMove
	if action == bulk.ActionCopy {
		purpose = campaign.PurposeCopy
	}
	switch r.Level {
	case bulk.LevelCampaign:
		list, err := p.campaigns.CampaignDestinations(ctx, a, r.ID, purpose)
		targets := make([]bulk.Target, 0, len(list.Items))
		for _, d := range list.Items {
			targets = append(targets, bulk.Target{ID: d.ID, Level: planKind, Name: d.Name, Path: d.Name, PlanID: d.ID, Year: r.Year})
		}
		return targets, err
	case bulk.LevelInitiative:
		list, err := p.campaigns.InitiativeDestinations(ctx, a, r.ID, purpose)
		targets := make([]bulk.Target, 0, len(list.Items))
		for _, d := range list.Items {
			targets = append(targets, bulk.Target{ID: d.ID, Level: campaignLevel, Name: d.Name, Path: r.PlanName + pathSeparator + d.Name,
				PlanID: r.PlanID, Year: r.Year, StartDate: d.StartDate, EndDate: d.EndDate})
		}
		return targets, err
	case bulk.LevelActivity:
	}
	list, err := p.activities.Destinations(ctx, a, r.ID)
	targets := make([]bulk.Target, 0, len(list))
	for _, d := range list {
		targets = append(targets, bulk.Target{ID: d.Initiative.ID, Level: initiativeLevel, Name: d.Initiative.Name,
			Path: r.PlanName + pathSeparator + d.Campaign.Name + pathSeparator + d.Initiative.Name, PlanID: r.PlanID, Year: r.Year,
			StartDate: d.StartDate, EndDate: d.EndDate})
	}
	return targets, err
}

// Consequences reads what the single move or copy of a campaign says changes below it in plan targetID.
func (p bulkPlanning) Consequences(ctx context.Context, a actor.Actor, r bulk.Record, targetID string) (bulk.Consequences, error) {
	list, err := p.campaigns.CampaignDestinations(ctx, a, r.ID, campaign.PurposeCopy)
	if err != nil {
		return bulk.Consequences{}, err
	}
	for _, d := range list.Items {
		if d.ID != targetID {
			continue
		}
		c := bulk.Consequences{OtherPlanName: d.Name}
		if d.Consequences != nil {
			c.ModelName, c.TypesNotAllowed = d.Consequences.ModelName, len(d.Consequences.TypesNotAllowed)
			c.GroupsHidden, c.GroupsShown = d.Consequences.GroupsHidden, d.Consequences.GroupsShown
		}
		return c, nil
	}
	return bulk.Consequences{}, nil
}
