package campaigns

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

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
		if found { // the destination needs "Add, edit, move and copy campaigns" (014 FR-014)
			if err := authz.RequireIn(ctx, authz.CampaignsEdit, target.Access); err != nil {
				return err
			}
		}
		var targetProblem string
		var siblings []Campaign
		switch {
		case !found:
			targetProblem = planGoneChooseMessage
		case target.Year != plan.Year:
			targetProblem = fmt.Sprintf(campaignCopyOtherYear, plan.Year)
		case target.Currency != plan.Currency: // 016 FR-017
			targetProblem = fmt.Sprintf(otherCurrencyReason, target.Currency, plan.Currency)
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

		dup := []Campaign{s.campaignCopy(a, source, copyPlace{parentID: target.ID, name: name})}
		if err = s.addCampaignCopies(ctx, dup); err != nil {
			return err
		}
		counts, err := s.copyBelowCampaign(ctx, a, CopyPair{FromID: source.ID, ToID: dup[0].ID})
		if err != nil {
			return err
		}
		if err = s.logCopy(ctx, a, campaignKind, dup[0].ID, dup[0].Name, source.Name, source.ID, counts); err != nil {
			return err
		}
		view, err = s.campaignView(ctx, dup[0].ID)
		return err
	})
	if err != nil {
		return CampaignView{}, err
	}
	return s.shownCampaign(ctx, a, view)
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

		dup := []Initiative{s.initiativeCopy(a, source, copyPlace{parentID: target.ID, name: name})}
		if err = s.addInitiativeCopies(ctx, dup); err != nil {
			return err
		}
		counts := map[string]int{}
		if err = s.copyActivities(ctx, a, source.ID, dup[0].ID, counts); err != nil {
			return err
		}
		if err = s.logCopy(ctx, a, initiativeKind, dup[0].ID, dup[0].Name, source.Name, source.ID, counts); err != nil {
			return err
		}
		view, err = s.initiativeView(ctx, dup[0].ID)
		return err
	})
	if err != nil {
		return InitiativeView{}, err
	}
	return s.shownInitiative(ctx, a, view)
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
