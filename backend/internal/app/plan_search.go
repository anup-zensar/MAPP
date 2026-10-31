package app

import (
	"context"
	"strings"

	"mapp/internal/plan"
	"mapp/internal/stores"
)

// planSearcher finds the records below plans in four reads whatever their number: their campaigns, then the
// initiatives, activities and line items below those.
type planSearcher struct {
	campaigns  stores.CampaignStore
	activities stores.ActivityStore
	lineItems  stores.LineItemStore
}

func (p planSearcher) Matching(ctx context.Context, planIDs []string, lowerText string) ([]plan.SearchMatch, error) {
	found := newSearchFound(planIDs, lowerText)
	campaigns, err := p.campaigns.CampaignsOfPlans(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	for _, c := range campaigns {
		found.add(plan.LevelCampaign, c.ID, c.Name, c.PlanID)
	}
	initiatives, err := p.campaigns.InitiativesOfCampaigns(ctx, found.idsOf(plan.LevelCampaign))
	if err != nil {
		return nil, err
	}
	for _, i := range initiatives {
		found.add(plan.LevelInitiative, i.ID, i.Name, i.CampaignID)
	}
	return p.belowInitiatives(ctx, found)
}

func (p planSearcher) belowInitiatives(ctx context.Context, found *searchFound) ([]plan.SearchMatch, error) {
	activities, err := p.activities.ListByInitiatives(ctx, found.idsOf(plan.LevelInitiative))
	if err != nil {
		return nil, err
	}
	for _, a := range activities {
		found.add(plan.LevelActivity, a.ID, a.Name, a.InitiativeID)
	}
	lineItems, err := p.lineItems.ListByActivities(ctx, found.idsOf(plan.LevelActivity))
	if err != nil {
		return nil, err
	}
	for _, li := range lineItems {
		found.add(plan.LevelLineItem, li.ID, li.Name, li.ActivityID)
	}
	return found.matches, nil
}

type searchFound struct {
	text    string
	planOf  map[string]string
	ids     map[string][]string
	matches []plan.SearchMatch
}

func newSearchFound(planIDs []string, lowerText string) *searchFound {
	f := &searchFound{text: lowerText, planOf: map[string]string{}, ids: map[string][]string{},
		matches: []plan.SearchMatch{}}
	for _, id := range planIDs {
		f.planOf[id] = id
	}
	return f
}

// add notes the record's plan, its parent's, and keeps the record when its name or ID holds the text.
func (f *searchFound) add(level, id, name, parentID string) {
	planID := f.planOf[parentID]
	f.planOf[id] = planID
	f.ids[level] = append(f.ids[level], id)
	if strings.Contains(strings.ToLower(name), f.text) || strings.Contains(strings.ToLower(id), f.text) {
		f.matches = append(f.matches, plan.SearchMatch{Level: level, ID: id, Name: name, ParentID: parentID, PlanID: planID})
	}
}

func (f *searchFound) idsOf(level string) []string {
	return f.ids[level]
}
