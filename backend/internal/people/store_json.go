package people

import (
	"context"

	"mapp/internal/platform/store"
)

// The JSON store tables of this package.
const (
	peopleTable      = "people"
	preferencesTable = "preferences"
	teamsTable       = "teams"
	membershipsTable = "team_memberships"
)

// JSONStore is the Store of the JSON store.
type JSONStore struct{}

// NewJSONStore returns the Store of the JSON store.
func NewJSONStore() JSONStore {
	return JSONStore{}
}

// Get reads one person from the people table.
func (JSONStore) Get(ctx context.Context, id string) (Person, bool, error) {
	return store.Get[Person](ctx, peopleTable, id)
}

// FindByEmail reads every person to find the e-mail, which is quick for the few people of a demo.
// PostgreSQL will use an index instead.
func (JSONStore) FindByEmail(ctx context.Context, email string) (Person, bool, error) {
	everyone, err := store.List[Person](ctx, peopleTable)
	if err != nil {
		return Person{}, false, err
	}
	for _, p := range everyone {
		if p.Email == email {
			return p, true, nil
		}
	}
	return Person{}, false, nil
}

// List reads the whole people table.
func (JSONStore) List(ctx context.Context) ([]Person, error) {
	return store.List[Person](ctx, peopleTable)
}

// Put saves a person in the people table under their id.
func (JSONStore) Put(ctx context.Context, p Person) error {
	return store.Put(ctx, peopleTable, p.ID, p)
}

// GetPreferences reads one person's preferences from the preferences table.
func (JSONStore) GetPreferences(ctx context.Context, personID string) (Preferences, bool, error) {
	return store.Get[Preferences](ctx, preferencesTable, personID)
}

// PutPreferences saves preferences in the preferences table under the person's id.
func (JSONStore) PutPreferences(ctx context.Context, p Preferences) error {
	return store.Put(ctx, preferencesTable, p.PersonID, p)
}

// ListPreferences reads the whole preferences table.
func (JSONStore) ListPreferences(ctx context.Context) ([]Preferences, error) {
	return store.List[Preferences](ctx, preferencesTable)
}

// GetTeam reads one team from the teams table.
func (JSONStore) GetTeam(ctx context.Context, id string) (Team, bool, error) {
	return store.Get[Team](ctx, teamsTable, id)
}

// ListTeams reads the whole teams table.
func (JSONStore) ListTeams(ctx context.Context) ([]Team, error) {
	return store.List[Team](ctx, teamsTable)
}

// PutTeam saves a team in the teams table under its id.
func (JSONStore) PutTeam(ctx context.Context, t Team) error {
	return store.Put(ctx, teamsTable, t.ID, t)
}

// DeleteTeam removes one team from the teams table.
func (JSONStore) DeleteTeam(ctx context.Context, id string) error {
	return store.Delete(ctx, teamsTable, id)
}

// GetMembership reads one membership from the team_memberships table.
func (JSONStore) GetMembership(ctx context.Context, id string) (Membership, bool, error) {
	return store.Get[Membership](ctx, membershipsTable, id)
}

// ListMemberships reads the whole team_memberships table.
func (JSONStore) ListMemberships(ctx context.Context) ([]Membership, error) {
	return store.List[Membership](ctx, membershipsTable)
}

// MembershipsOfTeam reads every membership to find those of one team (250 members at most, edge case).
func (r JSONStore) MembershipsOfTeam(ctx context.Context, teamID string) ([]Membership, error) {
	return r.membershipsWhere(ctx, func(m Membership) bool { return m.TeamID == teamID })
}

// MembershipsOfPerson reads every membership to find those of one person.
func (r JSONStore) MembershipsOfPerson(ctx context.Context, personID string) ([]Membership, error) {
	return r.membershipsWhere(ctx, func(m Membership) bool { return m.PersonID == personID })
}

// membershipsWhere returns the memberships that keep accepts, ordered by id; never nil.
func (r JSONStore) membershipsWhere(ctx context.Context, keep func(Membership) bool) ([]Membership, error) {
	all, err := r.ListMemberships(ctx)
	if err != nil {
		return nil, err
	}
	kept := []Membership{}
	for _, m := range all {
		if keep(m) {
			kept = append(kept, m)
		}
	}
	return kept, nil
}

// PutMembership saves a membership in the team_memberships table under its id.
func (JSONStore) PutMembership(ctx context.Context, m Membership) error {
	return store.Put(ctx, membershipsTable, m.ID, m)
}

// DeleteMembership removes one membership from the team_memberships table.
func (JSONStore) DeleteMembership(ctx context.Context, id string) error {
	return store.Delete(ctx, membershipsTable, id)
}
