package campaigns

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
)

// CampaignDestinations returns the plans the campaign id can be moved or copied to (FR-023, FR-024, FR-026,
// D-100): the plans of its plan's year only (D-017) where a may change campaigns (014 FR-014). For a move,
// every other plan, unavailable — with the reason and the fix — when it already has a campaign with this
// name; for a copy, its own plan first, then the others, all available, each with a suggested name. Each says what the campaign's budget would leave
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
		campaignsOf, err := s.campaignsOfPlans(ctx, plans)
		if err != nil {
			return err
		}
		list = DestinationList{Items: make([]Destination, 0, len(plans)), Rule: campaignRuleMove}
		if purpose == PurposeCopy {
			list.Rule = campaignRuleCopy
		}
		for _, p := range plans {
			current := p.ID == plan.ID
			if (current && purpose == PurposeMove) || !authz.InScope(ctx, authz.CampaignsEdit, p.Access) { // 014 FR-014
				continue
			}
			siblings := campaignsOf[p.ID]
			status := p.Status
			d := Destination{ID: p.ID, Name: p.Name, Current: current, Available: true, PlanStatus: &status}
			d.RemainingAfter, d.OverAllocatedBy = remainingAfter(p.Budget, campaignBudgets(siblings), c.Budget)
			if p.Currency != plan.Currency { // 016 FR-017: amounts are never converted silently
				d.RemainingAfter, d.OverAllocatedBy = nil, nil
				unavailable(&d, fmt.Sprintf(otherCurrencyReason, p.Currency, plan.Currency))
			} else if purpose == PurposeMove {
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
	return s.shownDestinations(ctx, a, list)
}

func (s *Service) campaignsOfPlans(ctx context.Context, plans []PlanFacts) (map[string][]Campaign, error) {
	planIDs := make([]string, 0, len(plans))
	for _, p := range plans {
		planIDs = append(planIDs, p.ID)
	}
	campaigns, err := s.repo.CampaignsOfPlans(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	campaignsOf := make(map[string][]Campaign, len(plans))
	for _, c := range campaigns {
		campaignsOf[c.PlanID] = append(campaignsOf[c.PlanID], c)
	}
	return campaignsOf, nil
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
	return s.shownDestinations(ctx, a, list)
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
