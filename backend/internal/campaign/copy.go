package campaign

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

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
		target, err := s.campaignCopyTargetOf(ctx, plan, in.PlanID)
		if err != nil {
			return err
		}
		var problems errs.Problems
		name := checkName(&problems, in.Name, func(name string) string {
			if target.problem != "" {
				return "" // the name is checked in a plan it can go to
			}
			return campaignNameTaken(target.siblings, target.plan, name, "")
		})
		if target.problem != "" {
			problems.Add("planId", target.problem)
		}
		if err = problems.Err(); err != nil {
			return err
		}

		copied, err := s.saveCampaignCopy(ctx, a, source, copyPlace{parentID: target.plan.ID, name: name})
		if err != nil {
			return err
		}
		view, err = s.campaignView(ctx, copied.ID)
		return err
	})
	if err != nil {
		return CampaignView{}, err
	}
	return s.shownCampaign(ctx, a, view)
}

// campaignCopyTarget is the plan a campaign copy goes to, with its campaigns; problem says why the copy
// cannot go there, "" when it can.
type campaignCopyTarget struct {
	plan     PlanFacts
	siblings []Campaign
	problem  string
}

func (s *Service) campaignCopyTargetOf(ctx context.Context, plan PlanFacts, planID string) (campaignCopyTarget, error) {
	target, found, err := s.deps.Plans.Plan(ctx, strings.TrimSpace(planID))
	if err != nil {
		return campaignCopyTarget{}, err
	}
	if found { // the destination needs "Add, edit, move and copy campaigns" (014 FR-014)
		if err = authz.RequireIn(ctx, authz.CampaignsEdit, target.Access); err != nil {
			return campaignCopyTarget{}, err
		}
	}
	t := campaignCopyTarget{plan: target}
	switch {
	case !found:
		t.problem = planGoneChooseMessage
	case target.Year != plan.Year:
		t.problem = fmt.Sprintf(campaignCopyOtherYear, plan.Year)
	case target.Currency != plan.Currency: // 016 FR-017
		t.problem = fmt.Sprintf(otherCurrencyReason, target.Currency, plan.Currency)
	default:
		if t.siblings, err = s.repo.CampaignsOfPlan(ctx, target.ID); err != nil {
			return campaignCopyTarget{}, err
		}
	}
	return t, nil
}

// CopyInitiative copies the initiative id, with its activities and line items, into its own campaign or
// another campaign of the same plan whose dates contain it (FR-026, FR-027, D-096, R13), completely or not
// at all. The name is checked as a new initiative's in the destination campaign. One "copied" entry, on the
// new initiative, says where it came from and what came with it (D-097).
func (s *Service) CopyInitiative(ctx context.Context, a actor.Actor, id string, in InitiativeCopy) (InitiativeView, error) {
	var view InitiativeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		source, c, err := s.initiativeAndCampaign(ctx, id)
		if err != nil {
			return err
		}
		target, err := s.initiativeCopyTargetOf(ctx, source, c, in.CampaignID)
		if err != nil {
			return err
		}
		var problems errs.Problems
		name := checkName(&problems, in.Name, func(name string) string {
			if !target.inPlan {
				return "" // the name is checked in a campaign it can go to
			}
			return initiativeNameTaken(target.siblings, target.campaign, name, "")
		})
		if target.problem != "" {
			problems.Add("campaignId", target.problem)
		}
		if err = problems.Err(); err != nil {
			return err
		}

		copied, err := s.saveInitiativeCopy(ctx, a, source, copyPlace{parentID: target.campaign.ID, name: name})
		if err != nil {
			return err
		}
		view, err = s.initiativeView(ctx, copied.ID)
		return err
	})
	if err != nil {
		return InitiativeView{}, err
	}
	return s.shownInitiative(ctx, a, view)
}

// initiativeCopyTarget is the campaign an initiative copy goes to, with its initiatives; problem says why the
// copy cannot go there, "" when it can.
type initiativeCopyTarget struct {
	campaign Campaign
	siblings []Initiative
	inPlan   bool
	problem  string
}

func (s *Service) initiativeCopyTargetOf(ctx context.Context, source Initiative, c Campaign, campaignID string) (initiativeCopyTarget, error) {
	target, found, err := s.repo.GetCampaign(ctx, strings.TrimSpace(campaignID))
	if err != nil {
		return initiativeCopyTarget{}, err
	}
	t := initiativeCopyTarget{campaign: target, inPlan: found && target.PlanID == c.PlanID}
	switch {
	case !found:
		t.problem = campaignGoneChooseMessage
	case !t.inPlan:
		t.problem = initiativeCopyOtherPlan
	default:
		if !fitsInside(source, target) {
			t.problem = fmt.Sprintf(datesDontFitMessage, date.FormatRange(*target.StartDate, *target.EndDate))
		}
		if t.siblings, err = s.repo.InitiativesOfCampaign(ctx, target.ID); err != nil {
			return initiativeCopyTarget{}, err
		}
	}
	return t, nil
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
