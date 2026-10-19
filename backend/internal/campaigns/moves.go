package campaigns

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/rollup"
	"northstar/internal/settings"
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
	initiativeCopyOtherPlan    = "Initiatives can be copied only into campaigns of the same plan."
	copiedFromDetails          = "Copied from '%s' (%s)" // + ", with 2 initiatives, 7 activities and …" when records came along
	copySuffix                 = " (copy)"
	numberedCopySuffix         = " (copy %d)"
	attributeGroupName         = "%s attributes (%s)" // "Lubricants attributes (activities)"
	parentField                = "parent"             // the change of a move (data model §8)
	activityKindName           = "activity"           // feature 006's kind: the counts of a copy's details
	lineItemKindName           = "lineItem"           // feature 007's kind
)

// levelsBelowPlan are the levels whose Shell attributes move with a campaign, top-down, with their names in
// the attribute groups of the move consequences (FR-024).
var levelsBelowPlan = []struct {
	level  settings.Level
	plural string
}{
	{settings.LevelCampaign, "campaigns"}, {settings.LevelInitiative, "initiatives"},
	{settings.LevelActivity, "activities"}, {settings.LevelLineItem, "line items"},
}

// CampaignDestinations returns the plans the campaign id can be moved or copied to (FR-023, FR-024, FR-026,
// D-100): the plans of its plan's year only (D-017). For a move, every other plan, unavailable — with the
// reason and the fix — when it already has a campaign with this name; for a copy, its own plan first, then
// the others, all available, each with a suggested name. Each says what the campaign's budget would leave
// there and, for a plan with another model or business unit, what would change below.
func (s *Service) CampaignDestinations(ctx context.Context, a actor.Actor, id, purpose string) (DestinationList, error) {
	if err := checkPurpose(purpose); err != nil {
		return DestinationList{}, err
	}
	var list DestinationList
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, plan, err := s.campaignAndPlan(ctx, id)
		if err != nil {
			return err
		}
		plans, err := s.deps.Plans.PlansOfYear(ctx, plan.Year)
		if err != nil {
			return err
		}
		below, err := s.activitiesBelow(ctx, c.ID)
		if err != nil {
			return err
		}
		list = DestinationList{Items: make([]Destination, 0, len(plans)), Rule: campaignRuleMove}
		if purpose == PurposeCopy {
			list.Rule = campaignRuleCopy
		}
		for _, p := range plans {
			current := p.ID == plan.ID
			if current && purpose == PurposeMove {
				continue
			}
			siblings, err := s.repo.CampaignsOfPlan(ctx, p.ID)
			if err != nil {
				return err
			}
			status := p.Status
			d := Destination{ID: p.ID, Name: p.Name, Current: current, Available: true, PlanStatus: &status}
			d.RemainingAfter, d.OverAllocatedBy = remainingAfter(p.Budget, campaignBudgets(siblings), c.Budget)
			if purpose == PurposeMove {
				if other, taken := campaignNamed(siblings, c.Name, c.ID); taken {
					unavailable(&d, fmt.Sprintf(campaignNameTakenReason, other.Name))
				}
			} else {
				name := suggestCopyName(c.Name, func(n string) bool { _, taken := campaignNamed(siblings, n, ""); return taken })
				d.SuggestedName = &name
			}
			if !current {
				if d.Consequences, err = s.consequences(ctx, plan, p, below); err != nil {
					return err
				}
			}
			list.Items = append(list.Items, d)
		}
		slices.SortStableFunc(list.Items, func(x, y Destination) int {
			return cmp.Or(compareCurrent(x, y), strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), cmp.Compare(x.ID, y.ID))
		})
		list.Total = len(list.Items)
		return nil
	})
	if err != nil {
		return DestinationList{}, err
	}
	return list, nil
}

// InitiativeDestinations returns the campaigns of its plan the initiative id can be moved or copied to
// (FR-023, FR-026): for a move, the other campaigns, unavailable when one already has an initiative with
// this name or its dates do not contain the initiative's (both reasons when both apply); for a copy, its own
// campaign first, then the others, where only the dates make one unavailable, each with a suggested name.
// Each says what the initiative's budget would leave in that campaign.
func (s *Service) InitiativeDestinations(ctx context.Context, a actor.Actor, id, purpose string) (DestinationList, error) {
	if err := checkPurpose(purpose); err != nil {
		return DestinationList{}, err
	}
	var list DestinationList
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		i, found, err := s.repo.GetInitiative(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		c, found, err := s.repo.GetCampaign(ctx, i.CampaignID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		campaigns, err := s.repo.CampaignsOfPlan(ctx, c.PlanID)
		if err != nil {
			return err
		}
		campaignIDs := make([]string, 0, len(campaigns))
		for _, t := range campaigns {
			campaignIDs = append(campaignIDs, t.ID)
		}
		initiatives, err := s.repo.InitiativesOfCampaigns(ctx, campaignIDs)
		if err != nil {
			return err
		}
		initiativesOf := map[string][]Initiative{}
		for _, x := range initiatives {
			initiativesOf[x.CampaignID] = append(initiativesOf[x.CampaignID], x)
		}

		list = DestinationList{Items: make([]Destination, 0, len(campaigns)), Rule: initiativeRuleMove}
		if purpose == PurposeCopy {
			list.Rule = initiativeRuleCopy
		}
		for _, t := range campaigns {
			current := t.ID == c.ID
			if current && purpose == PurposeMove {
				continue
			}
			siblings := initiativesOf[t.ID]
			d := Destination{ID: t.ID, Name: t.Name, Current: current, Available: true, StartDate: t.StartDate, EndDate: t.EndDate}
			d.RemainingAfter, d.OverAllocatedBy = remainingAfter(t.Budget, initiativeBudgets(siblings), i.Budget)
			var reasons []string
			if purpose == PurposeMove {
				if other, taken := initiativeNamed(siblings, i.Name, i.ID); taken {
					reasons = append(reasons, fmt.Sprintf(initiativeNameTakenReason, other.Name))
				}
			} else {
				name := suggestCopyName(i.Name, func(n string) bool { _, taken := initiativeNamed(siblings, n, ""); return taken })
				d.SuggestedName = &name
			}
			if !fitsInside(i, t) {
				reasons = append(reasons, fmt.Sprintf(datesDontFitReason, date.FormatRange(*t.StartDate, *t.EndDate)))
			}
			if len(reasons) > 0 {
				unavailable(&d, strings.Join(reasons, reasonSeparator))
			}
			list.Items = append(list.Items, d)
		}
		slices.SortStableFunc(list.Items, func(x, y Destination) int {
			return cmp.Or(compareCurrent(x, y), compareInTree(x.StartDate, x.Name, x.ID, y.StartDate, y.Name, y.ID))
		})
		list.Total = len(list.Items)
		return nil
	})
	if err != nil {
		return DestinationList{}, err
	}
	return list, nil
}

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
		switch {
		case !found:
			problems.Add("planId", planGoneChooseMessage)
		case target.Year != plan.Year:
			problems.Add("planId", fmt.Sprintf(campaignOtherYearMessage, plan.Year))
		case target.ID == plan.ID:
			problems.Add("planId", fmt.Sprintf(alreadyInMessage, c.Name, plan.Name))
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
	return view, nil
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
	return view, nil
}

// CopyCampaign copies the campaign id, with its initiatives and everything below them, into its own plan or
// another plan of the same year (FR-026, FR-027, D-096, R13), completely or not at all: a new campaign with
// every stored field and the chosen name, its initiatives with their names, and their activities and line
// items through feature 006 — new IDs, activities Created, no approvals, comments or history. The name is
// checked as a new campaign's in the destination plan. One "copied" entry, on the new campaign, says where
// it came from and what came with it (D-097).
func (s *Service) CopyCampaign(ctx context.Context, a actor.Actor, id string, in CampaignCopy) (CampaignView, error) {
	var view CampaignView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		source, plan, err := s.campaignAndPlan(ctx, id)
		if err != nil {
			return err
		}
		target, found, err := s.deps.Plans.Plan(ctx, strings.TrimSpace(in.PlanID))
		if err != nil {
			return err
		}
		var targetProblem string
		var siblings []Campaign
		switch {
		case !found:
			targetProblem = planGoneChooseMessage
		case target.Year != plan.Year:
			targetProblem = fmt.Sprintf(campaignCopyOtherYear, plan.Year)
		default:
			if siblings, err = s.repo.CampaignsOfPlan(ctx, target.ID); err != nil {
				return err
			}
		}
		var problems errs.Problems
		name := checkName(&problems, in.Name, func(name string) string {
			if targetProblem != "" {
				return "" // the name is checked in a plan it can go to
			}
			return campaignNameTaken(siblings, target, name, "")
		})
		if targetProblem != "" {
			problems.Add("planId", targetProblem)
		}
		if err := problems.Err(); err != nil {
			return err
		}

		dup, err := s.putCampaignCopy(ctx, a, source, target.ID, name, 0)
		if err != nil {
			return err
		}
		initiatives, err := s.repo.InitiativesOfCampaign(ctx, source.ID)
		if err != nil {
			return err
		}
		counts := map[string]int{initiativeKind: len(initiatives)}
		for _, i := range initiatives {
			copied, err := s.putInitiativeCopy(ctx, a, i, dup.ID, i.Name, 0)
			if err != nil {
				return err
			}
			if err := s.copyActivities(ctx, a, i.ID, copied.ID, counts); err != nil {
				return err
			}
		}
		if err := s.logCopy(ctx, a, campaignKind, dup.ID, dup.Name, source.Name, source.ID, counts); err != nil {
			return err
		}
		view, err = s.campaignView(ctx, dup.ID)
		return err
	})
	if err != nil {
		return CampaignView{}, err
	}
	return view, nil
}

// CopyInitiative copies the initiative id, with its activities and line items, into its own campaign or
// another campaign of the same plan whose dates contain it (FR-026, FR-027, D-096, R13), completely or not
// at all. The name is checked as a new initiative's in the destination campaign. One "copied" entry, on the
// new initiative, says where it came from and what came with it (D-097).
func (s *Service) CopyInitiative(ctx context.Context, a actor.Actor, id string, in InitiativeCopy) (InitiativeView, error) {
	var view InitiativeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		source, found, err := s.repo.GetInitiative(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		c, found, err := s.repo.GetCampaign(ctx, source.CampaignID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		target, found, err := s.repo.GetCampaign(ctx, strings.TrimSpace(in.CampaignID))
		if err != nil {
			return err
		}
		inPlan := found && target.PlanID == c.PlanID
		var targetProblem string
		var siblings []Initiative
		switch {
		case !found:
			targetProblem = campaignGoneChooseMessage
		case !inPlan:
			targetProblem = initiativeCopyOtherPlan
		default:
			if !fitsInside(source, target) {
				targetProblem = fmt.Sprintf(datesDontFitMessage, date.FormatRange(*target.StartDate, *target.EndDate))
			}
			if siblings, err = s.repo.InitiativesOfCampaign(ctx, target.ID); err != nil {
				return err
			}
		}
		var problems errs.Problems
		name := checkName(&problems, in.Name, func(name string) string {
			if !inPlan {
				return "" // the name is checked in a campaign it can go to
			}
			return initiativeNameTaken(siblings, target, name, "")
		})
		if targetProblem != "" {
			problems.Add("campaignId", targetProblem)
		}
		if err := problems.Err(); err != nil {
			return err
		}

		dup, err := s.putInitiativeCopy(ctx, a, source, target.ID, name, 0)
		if err != nil {
			return err
		}
		counts := map[string]int{}
		if err := s.copyActivities(ctx, a, source.ID, dup.ID, counts); err != nil {
			return err
		}
		if err := s.logCopy(ctx, a, initiativeKind, dup.ID, dup.Name, source.Name, source.ID, counts); err != nil {
			return err
		}
		view, err = s.initiativeView(ctx, dup.ID)
		return err
	})
	if err != nil {
		return InitiativeView{}, err
	}
	return view, nil
}

// CopyCounts is how many campaigns and initiatives a plan copy made.
type CopyCounts struct {
	Campaigns, Initiatives int
}

// CopyPlanContents copies every campaign of the plan fromPlanID, with its initiatives, into the plan
// toPlanID for feature 004's plan copy (004 FR-027, FR-028; R13): the same names and values, every date
// shiftYears years later (29 February becomes 28 February), new IDs and the actor as creator. It joins the
// caller's Write and writes no log entries: the plan copy writes the one "copied" entry. It returns the
// counts and the old → new ID of every campaign and initiative, for 004's CopyJob.NewIDs; the activities and
// line items below are copied by their own levels of the plan copy, not here (D-195).
func (s *Service) CopyPlanContents(ctx context.Context, a actor.Actor, fromPlanID, toPlanID string, shiftYears int) (CopyCounts, map[string]string, error) {
	var counts CopyCounts
	newIDs := map[string]string{}
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		campaigns, err := s.repo.CampaignsOfPlan(ctx, fromPlanID)
		if err != nil {
			return err
		}
		for _, c := range campaigns { // by id, so the copies get their IDs in the same order
			dup, err := s.putCampaignCopy(ctx, a, c, toPlanID, c.Name, shiftYears)
			if err != nil {
				return err
			}
			newIDs[c.ID] = dup.ID
			initiatives, err := s.repo.InitiativesOfCampaign(ctx, c.ID)
			if err != nil {
				return err
			}
			for _, i := range initiatives {
				dupInitiative, err := s.putInitiativeCopy(ctx, a, i, dup.ID, i.Name, shiftYears)
				if err != nil {
					return err
				}
				newIDs[i.ID] = dupInitiative.ID
			}
			counts.Campaigns++
			counts.Initiatives += len(initiatives)
		}
		return nil
	})
	if err != nil {
		return CopyCounts{}, nil, err
	}
	return counts, newIDs, nil
}

// putCampaignCopy stores a copy of the campaign c in the plan planID, named name, with a new ID, the actor
// as creator and its dates shiftYears years later, inside the caller's Write.
func (s *Service) putCampaignCopy(ctx context.Context, a actor.Actor, c Campaign, planID, name string, shiftYears int) (Campaign, error) {
	dup := c
	dup.PlanID, dup.Name = planID, name
	dup.OwnerID, dup.Budget = clonePtr(c.OwnerID), clonePtr(c.Budget)
	dup.StartDate, dup.EndDate = shiftDate(c.StartDate, shiftYears), shiftDate(c.EndDate, shiftYears)
	dup.Attributes = maps.Clone(orEmpty(c.Attributes))
	dup.Meta = record.NewMeta(a, s.now())
	var err error
	if dup.ID, err = s.ids.Next(ctx, ids.PrefixCampaign); err != nil {
		return Campaign{}, err
	}
	return dup, s.repo.PutCampaign(ctx, dup)
}

// putInitiativeCopy stores a copy of the initiative i in the campaign campaignID, named name, with a new ID,
// the actor as creator and its dates shiftYears years later, inside the caller's Write.
func (s *Service) putInitiativeCopy(ctx context.Context, a actor.Actor, i Initiative, campaignID, name string, shiftYears int) (Initiative, error) {
	dup := i
	dup.CampaignID, dup.Name = campaignID, name
	dup.OwnerID, dup.Budget = clonePtr(i.OwnerID), clonePtr(i.Budget)
	dup.StartDate, dup.EndDate = shiftDate(i.StartDate, shiftYears), shiftDate(i.EndDate, shiftYears)
	dup.Attributes = maps.Clone(orEmpty(i.Attributes))
	dup.Meta = record.NewMeta(a, s.now())
	var err error
	if dup.ID, err = s.ids.Next(ctx, ids.PrefixInitiative); err != nil {
		return Initiative{}, err
	}
	return dup, s.repo.PutInitiative(ctx, dup)
}

// copyActivities has feature 006 copy the activities of the initiative fromID, with their line items, into
// the initiative toID within the year, and adds what it copied to counts, by history kind.
func (s *Service) copyActivities(ctx context.Context, a actor.Actor, fromID, toID string, counts map[string]int) error {
	copied, err := s.deps.Activities.CopyAll(ctx, a, fromID, toID, 0)
	if err != nil {
		return err
	}
	counts[activityKindName] += copied.Activities
	counts[lineItemKindName] += copied.LineItems
	return nil
}

// logCopy writes the one "copied" entry of a campaign or initiative copy, on the new record id: "Copied
// from '<source name>' (<source id>)", with what came along (D-097).
func (s *Service) logCopy(ctx context.Context, a actor.Actor, kind, id, name, sourceName, sourceID string, counts map[string]int) error {
	details := fmt.Sprintf(copiedFromDetails, sourceName, sourceID)
	if with := s.history.CountsText(counts); with != "" {
		details += ", " + with
	}
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: kind, RecordID: id, RecordName: name, Action: history.ActionCopied, Details: details,
	})
	return err
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

// activitiesBelow returns the activities of the campaign id's initiatives, in tree order: initiative by
// initiative, then by name.
func (s *Service) activitiesBelow(ctx context.Context, id string) ([]ActivitySummary, error) {
	initiatives, err := s.repo.InitiativesOfCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	sortInTree(initiatives, func(i Initiative) (*date.Date, string, string) { return i.StartDate, i.Name, i.ID })
	initiativeIDs := make([]string, 0, len(initiatives))
	for _, i := range initiatives {
		initiativeIDs = append(initiativeIDs, i.ID)
	}
	summaries, err := s.deps.Activities.Summaries(ctx, initiativeIDs)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(summaries, func(x, y ActivitySummary) int {
		return cmp.Or(cmp.Compare(slices.Index(initiativeIDs, x.InitiativeID), slices.Index(initiativeIDs, y.InitiativeID)),
			strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), cmp.Compare(x.ID, y.ID))
	})
	return summaries, nil
}

// consequences says what changes below a campaign of the plan from that lands in the plan to (FR-024, R7,
// R9): nil when both plans have the same model and business unit. Otherwise the destination's model, the
// activities below whose type that model does not allow (they keep it), and the attribute groups that would
// be hidden there, with their values kept, or shown again.
func (s *Service) consequences(ctx context.Context, from, to PlanFacts, below []ActivitySummary) (*MoveConsequences, error) {
	fromUnit, err := s.deps.Settings.BusinessUnit(ctx, from.Attributes)
	if err != nil {
		return nil, err
	}
	toUnit, err := s.deps.Settings.BusinessUnit(ctx, to.Attributes)
	if err != nil {
		return nil, err
	}
	modelChanges := from.ModelID != to.ModelID
	unitChanges := !sameOptions(fromUnit, toUnit)
	if !modelChanges && !unitChanges {
		return nil, nil
	}
	model, found, err := s.deps.Settings.Model(ctx, to.ModelID)
	if err != nil {
		return nil, err
	}
	c := &MoveConsequences{
		ModelName: cmp.Or(model.Name, to.ModelID), ModelChanges: modelChanges, BusinessUnitChanges: unitChanges,
		TypesNotAllowed: []TypeNotAllowed{}, GroupsHidden: []string{}, GroupsShown: []string{},
	}
	if modelChanges {
		for _, a := range below {
			if a.TypeID == "" || (found && model.Allows(a.TypeID)) {
				continue
			}
			t, _, err := s.deps.Settings.ActivityType(ctx, a.TypeID)
			if err != nil {
				return nil, err
			}
			c.TypesNotAllowed = append(c.TypesNotAllowed, TypeNotAllowed{ID: a.ID, Name: a.Name, TypeName: cmp.Or(t.Name, a.TypeID)})
		}
	}
	if unitChanges {
		if c.GroupsHidden, err = s.groupNames(ctx, fromUnit, toUnit); err != nil {
			return nil, err
		}
		if c.GroupsShown, err = s.groupNames(ctx, toUnit, fromUnit); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// groupNames names the attribute groups below a plan that are shown with the business unit from and hidden
// with the business unit to (R9): "Lubricants attributes (activities)", one per business-unit option and
// level with at least one such active attribute, by the options' display order, then top-down. Never nil.
func (s *Service) groupNames(ctx context.Context, from, to []string) ([]string, error) {
	type group struct {
		position, level int
		name            string
	}
	var groups []group
	for levelIndex, l := range levelsBelowPlan {
		attributes, err := s.deps.Settings.Attributes(ctx, l.level)
		if err != nil {
			return nil, err
		}
		for _, attribute := range attributes {
			if !attribute.Active || !settings.Visible(attribute, from) || settings.Visible(attribute, to) {
				continue
			}
			// Shown with from but not with to: one of its condition's options is in from, not in to.
			unit := attribute.Condition[slices.IndexFunc(attribute.Condition, func(id string) bool { return slices.Contains(from, id) })]
			o, found, err := s.deps.Settings.Option(ctx, unit)
			if err != nil {
				return nil, err
			}
			if !found {
				o.Label = unit // an option that no longer exists is named by its id
			}
			name := fmt.Sprintf(attributeGroupName, o.Label, l.plural)
			if !slices.ContainsFunc(groups, func(g group) bool { return g.name == name }) {
				groups = append(groups, group{position: o.Position, level: levelIndex, name: name})
			}
		}
	}
	slices.SortStableFunc(groups, func(x, y group) int {
		return cmp.Or(cmp.Compare(x.position, y.position), cmp.Compare(x.level, y.level))
	})
	names := make([]string, 0, len(groups))
	for _, g := range groups {
		names = append(names, g.name)
	}
	return names, nil
}

// suggestCopyName returns the name a copy suggests (R13, FR-026): "<name> (copy)", then "<name> (copy 2)",
// "<name> (copy 3)", … — the first that taken says the destination does not have yet. When the name and the
// suffix together pass 120 characters, the name is shortened, without trailing spaces, so the suffix fits
// (edge case).
func suggestCopyName(name string, taken func(string) bool) string {
	for n := 1; ; n++ {
		suffix := copySuffix
		if n > 1 {
			suffix = fmt.Sprintf(numberedCopySuffix, n)
		}
		base := name
		if room := maxNameLength - utf8.RuneCountInString(suffix); utf8.RuneCountInString(base) > room {
			base = strings.TrimRight(string([]rune(base)[:room]), " ")
		}
		if candidate := base + suffix; !taken(candidate) {
			return candidate
		}
	}
}

// checkPurpose refuses a destinations query for anything but a move or a copy.
func checkPurpose(purpose string) error {
	if purpose != PurposeMove && purpose != PurposeCopy {
		return errs.Validation(errs.FieldError{Field: "purpose", Message: choosePurposeMessage})
	}
	return nil
}

// unavailable marks the destination d as one that cannot be chosen, with the reason and the fix.
func unavailable(d *Destination, reason string) {
	d.Available, d.UnavailableReason = false, &reason
}

// compareCurrent puts the record's own parent first.
func compareCurrent(x, y Destination) int {
	switch {
	case x.Current == y.Current:
		return 0
	case x.Current:
		return -1
	default:
		return 1
	}
}

// remainingAfter is what a destination with budget, whose children already have the budgets allocated,
// would have left once a record with the budget added comes in, and how much that over-allocates it
// (FR-026: never blocking, D-015). Both are nil when the destination has no budget.
func remainingAfter(budget *money.Money, allocated []*money.Money, added *money.Money) (*money.Money, *money.Money) {
	remaining := rollup.Remaining(budget, rollup.Total(append(slices.Clone(allocated), added)))
	return remaining, rollup.OverBy(remaining)
}

// campaignBudgets returns the budgets of campaigns.
func campaignBudgets(campaigns []Campaign) []*money.Money {
	budgets := make([]*money.Money, 0, len(campaigns))
	for _, c := range campaigns {
		budgets = append(budgets, c.Budget)
	}
	return budgets
}

// initiativeBudgets returns the budgets of initiatives.
func initiativeBudgets(initiatives []Initiative) []*money.Money {
	budgets := make([]*money.Money, 0, len(initiatives))
	for _, i := range initiatives {
		budgets = append(budgets, i.Budget)
	}
	return budgets
}

// fitsInside reports whether the initiative i may be in the campaign c by its dates (R3): always, unless
// both have dates and i's are not inside c's.
func fitsInside(i Initiative, c Campaign) bool {
	if c.StartDate == nil || c.EndDate == nil || i.StartDate == nil || i.EndDate == nil {
		return true
	}
	return !i.StartDate.Before(*c.StartDate) && !i.EndDate.After(*c.EndDate)
}

// sameOptions reports whether two lists of option ids hold the same options, in any order.
func sameOptions(x, y []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(x)), slices.Sorted(slices.Values(y)))
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

// shiftDate returns the date d years later (29 February → 28 February), or nil for no date.
func shiftDate(d *date.Date, years int) *date.Date {
	if d == nil {
		return nil
	}
	shifted := d.AddYears(years)
	return &shifted
}

// clonePtr returns a pointer to a copy of what p points at, or nil.
func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
