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
		origin, err := s.campaignOriginOf(ctx, id)
		if err != nil {
			return err
		}
		list = newDestinationList(len(origin.plans), purpose, campaignRuleMove, campaignRuleCopy)
		for _, p := range origin.plans {
			current := p.ID == origin.plan.ID
			if (current && purpose == PurposeMove) || !authz.InScope(ctx, authz.CampaignsEdit, p.Access) { // 014 FR-014
				continue
			}
			var d Destination
			if d, err = s.campaignDestination(ctx, origin, p, purpose); err != nil {
				return err
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

// campaignOrigin is a campaign with what its destinations are worked out from.
type campaignOrigin struct {
	campaign    Campaign
	plan        PlanFacts
	plans       []PlanFacts // of the plan's year
	below       []ActivitySummary
	campaignsOf map[string][]Campaign
}

func (s *Service) campaignOriginOf(ctx context.Context, id string) (campaignOrigin, error) {
	c, plan, err := s.campaignAndPlan(ctx, id)
	if err != nil {
		return campaignOrigin{}, err
	}
	plans, err := s.deps.Plans.PlansOfYear(ctx, plan.Year)
	if err != nil {
		return campaignOrigin{}, err
	}
	below, err := s.activitiesBelow(ctx, c.ID)
	if err != nil {
		return campaignOrigin{}, err
	}
	campaignsOf, err := s.campaignsOfPlans(ctx, plans)
	if err != nil {
		return campaignOrigin{}, err
	}
	return campaignOrigin{campaign: c, plan: plan, plans: plans, below: below, campaignsOf: campaignsOf}, nil
}

func (s *Service) campaignDestination(ctx context.Context, origin campaignOrigin, p PlanFacts, purpose string) (Destination, error) {
	c, plan := origin.campaign, origin.plan
	current := p.ID == plan.ID
	siblings := origin.campaignsOf[p.ID]
	status := p.Status
	d := Destination{ID: p.ID, Name: p.Name, Current: current, Available: true, PlanStatus: &status}
	d.RemainingAfter, d.OverAllocatedBy = remainingAfter(p.Budget, campaignBudgets(siblings), c.Budget)
	switch {
	case p.Currency != plan.Currency: // 016 FR-017: amounts are never converted silently
		d.RemainingAfter, d.OverAllocatedBy = nil, nil
		unavailable(&d, fmt.Sprintf(otherCurrencyReason, p.Currency, plan.Currency))
	case purpose == PurposeMove:
		if other, taken := campaignNamed(siblings, c.Name, c.ID); taken {
			unavailable(&d, fmt.Sprintf(campaignNameTakenReason, other.Name))
		}
	default:
		name := suggestCopyName(c.Name, func(n string) bool { _, taken := campaignNamed(siblings, n, ""); return taken })
		d.SuggestedName = &name
	}
	if !current {
		var err error
		if d.Consequences, err = s.consequences(ctx, plan, p, origin.below); err != nil {
			return Destination{}, err
		}
	}
	return d, nil
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
		origin, err := s.initiativeOriginOf(ctx, id)
		if err != nil {
			return err
		}
		list = newDestinationList(len(origin.campaigns), purpose, initiativeRuleMove, initiativeRuleCopy)
		for _, t := range origin.campaigns {
			if t.ID == origin.campaign.ID && purpose == PurposeMove {
				continue
			}
			list.Items = append(list.Items, initiativeDestination(origin, t, purpose))
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

// initiativeOrigin is an initiative with what its destinations are worked out from.
type initiativeOrigin struct {
	initiative    Initiative
	campaign      Campaign
	campaigns     []Campaign // of the campaign's plan
	initiativesOf map[string][]Initiative
}

func (s *Service) initiativeOriginOf(ctx context.Context, id string) (initiativeOrigin, error) {
	i, c, err := s.initiativeAndCampaign(ctx, id)
	if err != nil {
		return initiativeOrigin{}, err
	}
	campaigns, err := s.repo.CampaignsOfPlan(ctx, c.PlanID)
	if err != nil {
		return initiativeOrigin{}, err
	}
	campaignIDs := make([]string, 0, len(campaigns))
	for _, t := range campaigns {
		campaignIDs = append(campaignIDs, t.ID)
	}
	initiatives, err := s.repo.InitiativesOfCampaigns(ctx, campaignIDs)
	if err != nil {
		return initiativeOrigin{}, err
	}
	initiativesOf := map[string][]Initiative{}
	for _, x := range initiatives {
		initiativesOf[x.CampaignID] = append(initiativesOf[x.CampaignID], x)
	}
	return initiativeOrigin{initiative: i, campaign: c, campaigns: campaigns, initiativesOf: initiativesOf}, nil
}

func initiativeDestination(origin initiativeOrigin, t Campaign, purpose string) Destination {
	i := origin.initiative
	siblings := origin.initiativesOf[t.ID]
	d := Destination{ID: t.ID, Name: t.Name, Current: t.ID == origin.campaign.ID, Available: true, StartDate: t.StartDate, EndDate: t.EndDate}
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
	return d
}

func newDestinationList(size int, purpose, moveRule, copyRule string) DestinationList {
	list := DestinationList{Items: make([]Destination, 0, size), Rule: moveRule}
	if purpose == PurposeCopy {
		list.Rule = copyRule
	}
	return list
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
