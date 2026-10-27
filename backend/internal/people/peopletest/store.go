// Package peopletest is the in-memory fake of the people, teams and preferences store, for tests that build
// services or the whole API without a database.
package peopletest

import (
	"context"

	"mapp/internal/people"
	"mapp/internal/platform/fake"
)

type Store struct {
	people      *fake.Table[people.Person]
	preferences *fake.Table[people.Preferences]
	teams       *fake.Table[people.Team]
	memberships *fake.Table[people.Membership]
}

func NewStore(db *fake.DB) Store {
	return Store{
		people:      fake.TableOf(db, "people", func(p people.Person) string { return p.ID }),
		preferences: fake.TableOf(db, "preferences", func(p people.Preferences) string { return p.PersonID }),
		teams:       fake.TableOf(db, "teams", func(t people.Team) string { return t.ID }),
		memberships: fake.TableOf(db, "team_memberships", func(m people.Membership) string { return m.ID }),
	}
}

func (s Store) Get(_ context.Context, id string) (people.Person, bool, error) {
	return s.people.Get(id)
}

// FindByEmail matches the e-mail exactly: people's e-mails are stored in lower case.
func (s Store) FindByEmail(_ context.Context, email string) (people.Person, bool, error) {
	found, err := s.people.Where(func(p people.Person) bool { return p.Email == email })
	if err != nil || len(found) == 0 {
		return people.Person{}, false, err
	}
	return found[0], true, nil
}

func (s Store) List(context.Context) ([]people.Person, error) {
	return s.people.List()
}

func (s Store) Put(ctx context.Context, p people.Person) error {
	return s.people.Put(ctx, p)
}

func (s Store) GetPreferences(_ context.Context, personID string) (people.Preferences, bool, error) {
	return s.preferences.Get(personID)
}

func (s Store) PutPreferences(ctx context.Context, p people.Preferences) error {
	return s.preferences.Put(ctx, p)
}

func (s Store) ListPreferences(context.Context) ([]people.Preferences, error) {
	return s.preferences.List()
}

func (s Store) GetTeam(_ context.Context, id string) (people.Team, bool, error) {
	return s.teams.Get(id)
}

func (s Store) ListTeams(context.Context) ([]people.Team, error) {
	return s.teams.List()
}

func (s Store) PutTeam(ctx context.Context, t people.Team) error {
	return s.teams.Put(ctx, t)
}

func (s Store) DeleteTeam(ctx context.Context, id string) error {
	return s.teams.Delete(ctx, id)
}

func (s Store) GetMembership(_ context.Context, id string) (people.Membership, bool, error) {
	return s.memberships.Get(id)
}

func (s Store) ListMemberships(context.Context) ([]people.Membership, error) {
	return s.memberships.List()
}

func (s Store) MembershipsOfTeam(_ context.Context, teamID string) ([]people.Membership, error) {
	return s.memberships.Where(func(m people.Membership) bool { return m.TeamID == teamID })
}

func (s Store) MembershipsOfPerson(_ context.Context, personID string) ([]people.Membership, error) {
	return s.memberships.Where(func(m people.Membership) bool { return m.PersonID == personID })
}

func (s Store) PutMembership(ctx context.Context, m people.Membership) error {
	return s.memberships.Put(ctx, m)
}

func (s Store) DeleteMembership(ctx context.Context, id string) error {
	return s.memberships.Delete(ctx, id)
}
