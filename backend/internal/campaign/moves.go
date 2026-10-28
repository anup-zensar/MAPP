package campaign

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

// Moving and copying campaigns and initiatives (US6, FR-023 – FR-027): the destinations the dialogs offer,
// the moves, the copies and the copy of a plan's contents for feature 004.

const (
	choosePurposeMessage       = "Choose move or copy."
	campaignRuleMove           = "Campaigns move only between plans of the same year"
	initiativeRuleMove         = "Initiatives move only between campaigns of the same plan"
	campaignRuleCopy           = "A copy can go into this plan or another plan of the same year"
	initiativeRuleCopy         = "A copy can go into this campaign or another campaign of the same plan"
	campaignNameTakenReason    = "Already has a campaign called '%s' — rename it first"
	initiativeNameTakenReason  = "Already has an initiative called '%s' — rename it first"
	datesDontFitReason         = "Dates don't fit (%s) — change the initiative's dates first"
	reasonSeparator            = " · "
	planGoneChooseMessage      = "That plan no longer exists — choose another one."
	campaignGoneChooseMessage  = "That campaign no longer exists — choose another one."
	campaignOtherYearMessage   = "Campaigns move only between plans of the same year (%d)."
	initiativeOtherPlanMessage = "Initiatives move only between campaigns of the same plan."
	alreadyInMessage           = "'%s' is already in '%s'."
	campaignMoveNameTaken      = "'%s' already has a campaign called '%s' — rename it first."
	initiativeMoveNameTaken    = "'%s' already has an initiative called '%s' — rename it first."
	datesDontFitMessage        = "Dates don't fit (%s) — change the initiative's dates first."
	campaignCopyOtherYear      = "Campaigns can be copied only into plans of the same year (%d)."
	otherCurrencyReason        = "Planned in %s — this campaign's amounts are in %s."
	initiativeCopyOtherPlan    = "Initiatives can be copied only into campaigns of the same plan."
	copiedFromDetails          = "Copied from '%s' (%s)" // + ", with 2 initiatives, 7 activities and …" when records came along
	copySuffix                 = " (copy)"
	numberedCopySuffix         = " (copy %d)"
	attributeGroupName         = "%s attributes (%s)" // "Lubricants attributes (activities)"
	parentField                = "parent"
	activityKindName           = "activity"
	lineItemKindName           = "lineItem"
)

const (
	planIDField     = "planId"
	campaignIDField = "campaignId"
)

// MoveCampaign moves the campaign id, with everything below it, to another plan of the same year (FR-023,
// FR-025, D-017): IDs, dates, values and the activities' statuses and approvals stay as they are, and the
// roll-ups of both plans follow because they are derived. The version must be the one the person saw. A
// gone plan, a plan of another year, the campaign's own plan, or one that already has a campaign with this
// name is refused on planId. One "moved" entry records the old and the new plan (FR-030).
func (s *Service) MoveCampaign(ctx context.Context, a actor.Actor, id string, in CampaignMove) (CampaignView, error) {
	var view CampaignView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		c, plan, err := s.campaignAndPlan(ctx, id)
		if err != nil {
			return err
		}
		if err = checkVersionIfSent(c.Meta, in.Version, campaignSingular); err != nil {
			return err
		}
		target, problems, err := s.campaignMoveTarget(ctx, c, plan, in.PlanID)
		if err != nil {
			return err
		}
		checkVersionSent(&problems, in.Version)
		if err = problems.Err(); err != nil {
			return err
		}
		if err = s.refuseDrawnOn(ctx, c.ID, target.ID != c.PlanID); err != nil {
			return err
		}
		c.PlanID = target.ID
		if err = s.saveCampaignMove(ctx, a, c, plan.ID); err != nil {
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

func (s *Service) campaignMoveTarget(ctx context.Context, c Campaign, plan PlanFacts, planID string) (PlanFacts, errs.Problems, error) {
	target, found, err := s.deps.Plans.Plan(ctx, strings.TrimSpace(planID))
	if err != nil {
		return PlanFacts{}, nil, err
	}
	var problems errs.Problems
	if !found {
		problems.Add(planIDField, planGoneChooseMessage)
		return target, problems, nil
	}
	if err = authz.RequireIn(ctx, authz.CampaignsEdit, target.Access); err != nil { // the destination too needs "Add, edit, move and copy campaigns" (014 FR-014)
		return PlanFacts{}, nil, err
	}
	if rule := campaignMoveRule(c, plan, target); rule != "" {
		problems.Add(planIDField, rule)
		return target, problems, nil
	}
	siblings, err := s.repo.CampaignsOfPlan(ctx, target.ID)
	if err != nil {
		return PlanFacts{}, nil, err
	}
	if other, taken := campaignNamed(siblings, c.Name, c.ID); taken {
		problems.Add(planIDField, fmt.Sprintf(campaignMoveNameTaken, target.Name, other.Name))
	}
	return target, problems, nil
}

func campaignMoveRule(c Campaign, plan, target PlanFacts) string {
	switch {
	case target.Year != plan.Year:
		return fmt.Sprintf(campaignOtherYearMessage, plan.Year)
	case target.ID == plan.ID:
		return fmt.Sprintf(alreadyInMessage, c.Name, plan.Name)
	case target.Currency != plan.Currency:
		return fmt.Sprintf(otherCurrencyReason, target.Currency, plan.Currency)
	}
	return ""
}

func (s *Service) saveCampaignMove(ctx context.Context, a actor.Actor, c Campaign, fromPlanID string) error {
	c.Touch(a, s.now())
	if err := s.repo.PutCampaign(ctx, c); err != nil {
		return err
	}
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: campaignKind, RecordID: c.ID, RecordName: c.Name, Action: history.ActionMoved,
		Changes: parentChange(fromPlanID, c.PlanID),
	})
	return err
}

// MoveInitiative moves the initiative id, with its activities and line items, to another campaign of the
// same plan (FR-023, FR-025): nothing below changes. A gone campaign, a campaign of another plan, or the
// initiative's own campaign is refused on campaignId; so is — together, when both apply — a campaign that
// already has an initiative with this name or whose dates do not contain the initiative's. One "moved"
// entry records the old and the new campaign (FR-030).
func (s *Service) MoveInitiative(ctx context.Context, a actor.Actor, id string, in InitiativeMove) (InitiativeView, error) {
	var view InitiativeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		i, c, err := s.initiativeAndCampaign(ctx, id)
		if err != nil {
			return err
		}
		if err = checkVersionIfSent(i.Meta, in.Version, initiativeSingular); err != nil {
			return err
		}
		target, problems, err := s.initiativeMoveTarget(ctx, i, c, in.CampaignID)
		if err != nil {
			return err
		}
		checkVersionSent(&problems, in.Version)
		if err = problems.Err(); err != nil {
			return err
		}
		i.CampaignID = target.ID
		if err = s.saveInitiativeMove(ctx, a, i, c.ID); err != nil {
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

func (s *Service) initiativeMoveTarget(ctx context.Context, i Initiative, c Campaign, campaignID string) (Campaign, errs.Problems, error) {
	target, found, err := s.repo.GetCampaign(ctx, strings.TrimSpace(campaignID))
	if err != nil {
		return Campaign{}, nil, err
	}
	var problems errs.Problems
	if !found {
		problems.Add(campaignIDField, campaignGoneChooseMessage)
		return target, problems, nil
	}
	if rule := initiativeMoveRule(i, c, target); rule != "" {
		problems.Add(campaignIDField, rule)
		return target, problems, nil
	}
	siblings, err := s.repo.InitiativesOfCampaign(ctx, target.ID)
	if err != nil {
		return Campaign{}, nil, err
	}
	if other, taken := initiativeNamed(siblings, i.Name, i.ID); taken {
		problems.Add(campaignIDField, fmt.Sprintf(initiativeMoveNameTaken, target.Name, other.Name))
	}
	if !fitsInside(i, target) {
		problems.Add(campaignIDField, fmt.Sprintf(datesDontFitMessage, date.FormatRange(*target.StartDate, *target.EndDate)))
	}
	return target, problems, nil
}

func initiativeMoveRule(i Initiative, c, target Campaign) string {
	switch {
	case target.PlanID != c.PlanID:
		return initiativeOtherPlanMessage
	case target.ID == c.ID:
		return fmt.Sprintf(alreadyInMessage, i.Name, c.Name)
	}
	return ""
}

func (s *Service) saveInitiativeMove(ctx context.Context, a actor.Actor, i Initiative, fromCampaignID string) error {
	i.Touch(a, s.now())
	if err := s.repo.PutInitiative(ctx, i); err != nil {
		return err
	}
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: initiativeKind, RecordID: i.ID, RecordName: i.Name, Action: history.ActionMoved,
		Changes: parentChange(fromCampaignID, i.CampaignID),
	})
	return err
}

func fitsInside(i Initiative, c Campaign) bool {
	if c.StartDate == nil || c.EndDate == nil || i.StartDate == nil || i.EndDate == nil {
		return true
	}
	return !i.StartDate.Before(*c.StartDate) && !i.EndDate.After(*c.EndDate)
}

func parentChange(oldID, newID string) []history.Change {
	return []history.Change{{Field: parentField, Old: jsonText(oldID), New: jsonText(newID)}}
}

func jsonText(s string) json.RawMessage {
	encoded, _ := json.Marshal(s)
	return encoded
}
