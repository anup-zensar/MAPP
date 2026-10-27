package campaigns

import (
	"context"
	"maps"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
)

// copyInitiativesOf copies the initiatives of each campaign of pairs into its copy; the copies get their IDs
// campaign by campaign in the order of pairs, as one copy per campaign gave them.
func (s *Service) copyInitiativesOf(ctx context.Context, a actor.Actor, pairs []CopyPair, shiftYears int) ([]CopyPair, error) {
	fromIDs := make([]string, 0, len(pairs))
	for _, p := range pairs {
		fromIDs = append(fromIDs, p.FromID)
	}
	initiatives, err := s.repo.InitiativesOfCampaigns(ctx, fromIDs)
	if err != nil {
		return nil, err
	}
	byCampaign := map[string][]Initiative{}
	for _, i := range initiatives {
		byCampaign[i.CampaignID] = append(byCampaign[i.CampaignID], i)
	}
	var sources, copies []Initiative
	for _, p := range pairs {
		for _, i := range byCampaign[p.FromID] {
			sources = append(sources, i)
			copies = append(copies, s.initiativeCopy(a, i, copyPlace{parentID: p.ToID, name: i.Name, years: shiftYears}))
		}
	}
	if err := s.addInitiativeCopies(ctx, copies); err != nil {
		return nil, err
	}
	copied := make([]CopyPair, 0, len(copies))
	for n, i := range sources {
		copied = append(copied, CopyPair{FromID: i.ID, ToID: copies[n].ID})
	}
	return copied, nil
}

func (s *Service) copyBelowCampaign(ctx context.Context, a actor.Actor, pair CopyPair) (map[string]int, error) {
	initiatives, err := s.copyInitiativesOf(ctx, a, []CopyPair{pair}, 0)
	if err != nil {
		return nil, err
	}
	copied, err := s.deps.Activities.CopyMany(ctx, a, initiatives, 0)
	if err != nil {
		return nil, err
	}
	return map[string]int{initiativeKind: len(initiatives), activityKindName: copied.Activities,
		lineItemKindName: copied.LineItems}, nil
}

func (s *Service) addCampaignCopies(ctx context.Context, copies []Campaign) error {
	fresh, err := s.ids.NextIDs(ctx, ids.PrefixCampaign, len(copies))
	if err != nil {
		return err
	}
	for n := range copies {
		copies[n].ID = fresh[n]
	}
	return s.repo.AddCampaigns(ctx, copies)
}

func (s *Service) addInitiativeCopies(ctx context.Context, copies []Initiative) error {
	fresh, err := s.ids.NextIDs(ctx, ids.PrefixInitiative, len(copies))
	if err != nil {
		return err
	}
	for n := range copies {
		copies[n].ID = fresh[n]
	}
	return s.repo.AddInitiatives(ctx, copies)
}

// copyPlace is where a copy goes: its new parent, its name, and the years its dates move on.
type copyPlace struct {
	parentID, name string
	years          int
}

func (s *Service) campaignCopy(a actor.Actor, c Campaign, to copyPlace) Campaign {
	dup := c
	dup.PlanID, dup.Name = to.parentID, to.name
	dup.OwnerID, dup.Budget = clonePtr(c.OwnerID), clonePtr(c.Budget)
	dup.StartDate, dup.EndDate = shiftDate(c.StartDate, to.years), shiftDate(c.EndDate, to.years)
	dup.Attributes = maps.Clone(orEmpty(c.Attributes))
	dup.BudgetPhasing = rollup.ShiftPhasing(c.BudgetPhasing, to.years)
	dup.Meta = record.NewMeta(a, s.now())
	return dup
}

func (s *Service) initiativeCopy(a actor.Actor, i Initiative, to copyPlace) Initiative {
	dup := i
	dup.CampaignID, dup.Name = to.parentID, to.name
	dup.OwnerID, dup.Budget = clonePtr(i.OwnerID), clonePtr(i.Budget)
	dup.StartDate, dup.EndDate = shiftDate(i.StartDate, to.years), shiftDate(i.EndDate, to.years)
	dup.Attributes = maps.Clone(orEmpty(i.Attributes))
	dup.BudgetPhasing = rollup.ShiftPhasing(i.BudgetPhasing, to.years)
	dup.Meta = record.NewMeta(a, s.now())
	return dup
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
