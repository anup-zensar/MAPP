package campaigns

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/date"
	"mapp/internal/settings"
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

// sameOptions reports whether two lists of option ids hold the same options, in any order.
func sameOptions(x, y []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(x)), slices.Sorted(slices.Values(y)))
}
