package campaigns

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

// Messages of the plan's "API validation and error messages" table about moves and copies.
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
	otherCurrencyReason        = "Planned in %s — this campaign's amounts are in %s." // 016 FR-017
	initiativeCopyOtherPlan    = "Initiatives can be copied only into campaigns of the same plan."
	copiedFromDetails          = "Copied from '%s' (%s)" // + ", with 2 initiatives, 7 activities and …" when records came along
	copySuffix                 = " (copy)"
	numberedCopySuffix         = " (copy %d)"
	attributeGroupName         = "%s attributes (%s)" // "Lubricants attributes (activities)"
	parentField                = "parent"             // the change of a move (data model §8)
	activityKindName           = "activity"           // feature 006's kind: the counts of a copy's details
	lineItemKindName           = "lineItem"           // feature 007's kind
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
		if in.Version != nil && *in.Version >= 1 {
			if err := c.CheckVersion(*in.Version, campaignSingular); err != nil {
				return err
			}
		}
		var problems errs.Problems
		target, found, err := s.deps.Plans.Plan(ctx, strings.TrimSpace(in.PlanID))
		if err != nil {
			return err
		}
		if found { // the destination too needs "Add, edit, move and copy campaigns" (014 FR-014)
			if err := authz.RequireIn(ctx, authz.CampaignsEdit, target.Access); err != nil {
				return err
			}
		}
		switch {
		case !found:
			problems.Add("planId", planGoneChooseMessage)
		case target.Year != plan.Year:
			problems.Add("planId", fmt.Sprintf(campaignOtherYearMessage, plan.Year))
		case target.ID == plan.ID:
			problems.Add("planId", fmt.Sprintf(alreadyInMessage, c.Name, plan.Name))
		case target.Currency != plan.Currency: // 016 FR-017
			problems.Add("planId", fmt.Sprintf(otherCurrencyReason, target.Currency, plan.Currency))
		default:
			siblings, err := s.repo.CampaignsOfPlan(ctx, target.ID)
			if err != nil {
				return err
			}
			if other, taken := campaignNamed(siblings, c.Name, c.ID); taken {
				problems.Add("planId", fmt.Sprintf(campaignMoveNameTaken, target.Name, other.Name))
			}
		}
		checkVersionSent(&problems, in.Version)
		if err := problems.Err(); err != nil {
			return err
		}

		c.PlanID = target.ID
		c.Touch(a, s.now())
		if err := s.repo.PutCampaign(ctx, c); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: campaignKind, RecordID: c.ID, RecordName: c.Name, Action: history.ActionMoved,
			Changes: parentChange(plan.ID, target.ID),
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

// MoveInitiative moves the initiative id, with its activities and line items, to another campaign of the
// same plan (FR-023, FR-025): nothing below changes. A gone campaign, a campaign of another plan, or the
// initiative's own campaign is refused on campaignId; so is — together, when both apply — a campaign that
// already has an initiative with this name or whose dates do not contain the initiative's. One "moved"
// entry records the old and the new campaign (FR-030).
func (s *Service) MoveInitiative(ctx context.Context, a actor.Actor, id string, in InitiativeMove) (InitiativeView, error) {
	var view InitiativeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		i, found, err := s.repo.GetInitiative(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		if in.Version != nil && *in.Version >= 1 {
			if err := i.CheckVersion(*in.Version, initiativeSingular); err != nil {
				return err
			}
		}
		c, found, err := s.repo.GetCampaign(ctx, i.CampaignID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		var problems errs.Problems
		target, found, err := s.repo.GetCampaign(ctx, strings.TrimSpace(in.CampaignID))
		if err != nil {
			return err
		}
		switch {
		case !found:
			problems.Add("campaignId", campaignGoneChooseMessage)
		case target.PlanID != c.PlanID:
			problems.Add("campaignId", initiativeOtherPlanMessage)
		case target.ID == c.ID:
			problems.Add("campaignId", fmt.Sprintf(alreadyInMessage, i.Name, c.Name))
		default:
			siblings, err := s.repo.InitiativesOfCampaign(ctx, target.ID)
			if err != nil {
				return err
			}
			if other, taken := initiativeNamed(siblings, i.Name, i.ID); taken {
				problems.Add("campaignId", fmt.Sprintf(initiativeMoveNameTaken, target.Name, other.Name))
			}
			if !fitsInside(i, target) {
				problems.Add("campaignId", fmt.Sprintf(datesDontFitMessage, date.FormatRange(*target.StartDate, *target.EndDate)))
			}
		}
		checkVersionSent(&problems, in.Version)
		if err := problems.Err(); err != nil {
			return err
		}

		i.CampaignID = target.ID
		i.Touch(a, s.now())
		if err := s.repo.PutInitiative(ctx, i); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: initiativeKind, RecordID: i.ID, RecordName: i.Name, Action: history.ActionMoved,
			Changes: parentChange(c.ID, target.ID),
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

// fitsInside reports whether the initiative i may be in the campaign c by its dates (R3): always, unless
// both have dates and i's are not inside c's.
func fitsInside(i Initiative, c Campaign) bool {
	if c.StartDate == nil || c.EndDate == nil || i.StartDate == nil || i.EndDate == nil {
		return true
	}
	return !i.StartDate.Before(*c.StartDate) && !i.EndDate.After(*c.EndDate)
}

// parentChange is the change of a move (data model §8): the parent's id before and after; the log shows
// the parents' names.
func parentChange(oldID, newID string) []history.Change {
	return []history.Change{{Field: parentField, Old: jsonText(oldID), New: jsonText(newID)}}
}

// jsonText returns the text s as a JSON value.
func jsonText(s string) json.RawMessage {
	encoded, _ := json.Marshal(s) // a string always encodes
	return encoded
}
