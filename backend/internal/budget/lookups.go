package budget

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"mapp/internal/people"
)

// lookups are the people and teams the rows and options of one answer refer to, read once for all of them.
type lookups struct {
	people map[string]people.PersonRef
	teams  map[string]people.TeamRef
}

// lookupsFor reads the owners and teams of plans, and the people personIDs.
func (s *Service) lookupsFor(ctx context.Context, plans []PlanInfo, personIDs ...string) (lookups, error) {
	var teamIDs []string
	for _, p := range plans {
		personIDs = append(personIDs, p.OwnerID)
		teamIDs = append(teamIDs, p.TeamIDs...)
	}
	var l lookups
	var err error
	if l.people, err = s.directory.PersonRefs(ctx, compact(personIDs)); err != nil {
		return lookups{}, err
	}
	if l.teams, err = s.directory.TeamRefs(ctx, compact(teamIDs)); err != nil {
		return lookups{}, err
	}
	return l, nil
}

// compact returns ids sorted, without duplicates and without "".
func compact(ids []string) []string {
	ids = slices.DeleteFunc(slices.Clone(ids), func(id string) bool { return id == "" })
	slices.Sort(ids)
	return slices.Compact(ids)
}

// person returns the person id as records show them; nil for "" and the id itself for one who no longer
// resolves.
func (l lookups) person(id string) *people.PersonRef {
	if id == "" {
		return nil
	}
	ref, found := l.people[id]
	if !found {
		ref = people.PersonRef{ID: id, Name: id}
	}
	return &ref
}

// teamsOf returns the teams of p by name; a team that no longer resolves is shown by its id. Never nil.
func (l lookups) teamsOf(p PlanInfo) []people.TeamRef {
	teams := make([]people.TeamRef, 0, len(p.TeamIDs))
	for _, id := range p.TeamIDs {
		team, found := l.teams[id]
		if !found {
			team = people.TeamRef{ID: id, Name: id, Colour: fallbackColour}
		}
		teams = append(teams, team)
	}
	slices.SortFunc(teams, func(x, y people.TeamRef) int {
		return cmp.Or(strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), strings.Compare(x.ID, y.ID))
	})
	return teams
}

// personRef reads one person as records show them; nil for "".
func (s *Service) personRef(ctx context.Context, id string) (*people.PersonRef, error) {
	l, err := s.lookupsFor(ctx, nil, id)
	if err != nil {
		return nil, err
	}
	return l.person(id), nil
}
