package people

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"strings"
)

// directory is every person, team and team membership, read inside one transaction: what the views of
// people and teams are built from. The joins (a person's teams, a team's members) are done here in memory,
// which is quick at demo scale; PostgreSQL will add indexed queries behind the same Store.
type directory struct {
	people      map[string]Person // by id
	teams       map[string]Team   // by id
	memberships []Membership      // ordered by id
}

// readDirectory reads the directory inside the caller's transaction.
func (s *Service) readDirectory(ctx context.Context) (directory, error) {
	everyone, err := s.repo.List(ctx)
	if err != nil {
		return directory{}, err
	}
	teams, err := s.repo.ListTeams(ctx)
	if err != nil {
		return directory{}, err
	}
	memberships, err := s.repo.ListMemberships(ctx)
	if err != nil {
		return directory{}, err
	}
	d := directory{people: map[string]Person{}, teams: map[string]Team{}, memberships: memberships}
	for _, p := range everyone {
		d.people[p.ID] = p
	}
	for _, t := range teams {
		d.teams[t.ID] = t
	}
	return d, nil
}

// sortedPeople returns everyone, sorted by name ignoring letter case, then by e-mail.
func (d directory) sortedPeople() []Person {
	everyone := slices.Collect(maps.Values(d.people))
	slices.SortFunc(everyone, comparePeople)
	return everyone
}

// sortedTeams returns every team, sorted by name ignoring letter case.
func (d directory) sortedTeams() []Team {
	teams := slices.Collect(maps.Values(d.teams))
	slices.SortFunc(teams, compareTeams)
	return teams
}

// teamIDsOf returns the ids of the person's teams, active or not, ordered by id; never nil.
func (d directory) teamIDsOf(personID string) []string {
	list := []string{}
	for _, m := range d.memberships {
		if m.PersonID == personID {
			list = append(list, m.TeamID)
		}
	}
	slices.Sort(list)
	return list
}

// memberIDsOf returns the ids of the team's members, active or not, ordered by id; never nil.
func (d directory) memberIDsOf(teamID string) []string {
	list := []string{}
	for _, m := range d.memberships {
		if m.TeamID == teamID {
			list = append(list, m.PersonID)
		}
	}
	slices.Sort(list)
	return list
}

// teamRefs returns the known teams among teamIDs as references, sorted by name; never nil.
func (d directory) teamRefs(teamIDs []string) []TeamRef {
	teams := make([]Team, 0, len(teamIDs))
	for _, id := range teamIDs {
		if t, found := d.teams[id]; found {
			teams = append(teams, t)
		}
	}
	slices.SortFunc(teams, compareTeams)
	refs := make([]TeamRef, 0, len(teams))
	for _, t := range teams {
		refs = append(refs, t.Ref())
	}
	return refs
}

// personRefs returns the known people among personIDs as references, sorted by name; never nil.
func (d directory) personRefs(personIDs []string) []PersonRef {
	everyone := make([]Person, 0, len(personIDs))
	for _, id := range personIDs {
		if p, found := d.people[id]; found {
			everyone = append(everyone, p)
		}
	}
	slices.SortFunc(everyone, comparePeople)
	refs := make([]PersonRef, 0, len(everyone))
	for _, p := range everyone {
		refs = append(refs, p.Ref())
	}
	return refs
}

// personView returns p as the people list and the pickers show them, with their teams.
func (d directory) personView(p Person) PersonView {
	return PersonView{
		ID:       p.ID,
		Name:     p.Name,
		Email:    p.Email,
		JobTitle: p.JobTitle,
		Initials: p.Initials(),
		Active:   p.Active,
		Teams:    d.teamRefs(d.teamIDsOf(p.ID)),
		Meta:     p.Meta,
	}
}

// listItem returns p as the people list shows them: their view with their line manager and how many people,
// active or not, have them as line manager (017 FR-016).
func (d directory) listItem(p Person) PersonListItem {
	item := PersonListItem{PersonView: d.personView(p), ReportCount: d.reportCount(p.ID)}
	if manager, found := d.people[p.ManagerID]; found {
		ref := manager.Ref()
		item.Manager = &ref
	}
	return item
}

// reportCount is how many people, active or not, have the person personID as their line manager.
func (d directory) reportCount(personID string) int {
	n := 0
	for _, p := range d.people {
		if personID != "" && p.ManagerID == personID {
			n++
		}
	}
	return n
}

// teamView returns t with its members and the plans among plans that use it, in the recycle bin or not
// (FR-010, FR-014).
func (d directory) teamView(t Team, plans []PlanUse) TeamView {
	return TeamView{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Colour:      t.Colour,
		Active:      t.Active,
		Members:     d.personRefs(d.memberIDsOf(t.ID)),
		Plans:       plansUsing(plans, t.ID),
		Meta:        t.Meta,
	}
}

// plansUsing returns the plans among plans that use the team teamID, including those in the recycle bin,
// sorted by name; never nil.
func plansUsing(plans []PlanUse, teamID string) []TeamPlanUse {
	uses := []TeamPlanUse{}
	for _, p := range plans {
		if slices.Contains(p.TeamIDs, teamID) {
			uses = append(uses, TeamPlanUse{ID: p.ID, Name: p.Name, InRecycleBin: p.InRecycleBin})
		}
	}
	slices.SortFunc(uses, func(a, b TeamPlanUse) int {
		return cmp.Or(compareNames(a.Name, b.Name), strings.Compare(a.ID, b.ID))
	})
	return uses
}

// compareNames orders names the way people sort them: ignoring letter case.
func compareNames(a, b string) int {
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}

// comparePeople orders people by name ignoring letter case, then by e-mail (two people may share a name),
// then by id.
func comparePeople(a, b Person) int {
	return cmp.Or(compareNames(a.Name, b.Name), strings.Compare(a.Email, b.Email), strings.Compare(a.ID, b.ID))
}

// compareTeams orders teams by name ignoring letter case, then by id.
func compareTeams(a, b Team) int {
	return cmp.Or(compareNames(a.Name, b.Name), strings.Compare(a.ID, b.ID))
}
