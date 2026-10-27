package people

import (
	"context"

	"mapp/internal/platform/fake"
)

// fakeStore is peopletest.Store for this package's own tests, which cannot import peopletest.
type fakeStore struct {
	people      *fake.Table[Person]
	preferences *fake.Table[Preferences]
	teams       *fake.Table[Team]
	memberships *fake.Table[Membership]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{
		people:      fake.TableOf(db, "people", func(p Person) string { return p.ID }),
		preferences: fake.TableOf(db, "preferences", func(p Preferences) string { return p.PersonID }),
		teams:       fake.TableOf(db, "teams", func(t Team) string { return t.ID }),
		memberships: fake.TableOf(db, "team_memberships", func(m Membership) string { return m.ID }),
	}
}

func (s fakeStore) Get(_ context.Context, id string) (Person, bool, error) {
	return s.people.Get(id)
}

func (s fakeStore) FindByEmail(_ context.Context, email string) (Person, bool, error) {
	found, err := s.people.Where(func(p Person) bool { return p.Email == email })
	if err != nil || len(found) == 0 {
		return Person{}, false, err
	}
	return found[0], true, nil
}

func (s fakeStore) List(context.Context) ([]Person, error) {
	return s.people.List()
}

func (s fakeStore) Put(ctx context.Context, p Person) error {
	return s.people.Put(ctx, p)
}

func (s fakeStore) GetPreferences(_ context.Context, personID string) (Preferences, bool, error) {
	return s.preferences.Get(personID)
}

func (s fakeStore) PutPreferences(ctx context.Context, p Preferences) error {
	return s.preferences.Put(ctx, p)
}

func (s fakeStore) GetTeam(_ context.Context, id string) (Team, bool, error) {
	return s.teams.Get(id)
}

func (s fakeStore) ListTeams(context.Context) ([]Team, error) {
	return s.teams.List()
}

func (s fakeStore) PutTeam(ctx context.Context, t Team) error {
	return s.teams.Put(ctx, t)
}

func (s fakeStore) DeleteTeam(ctx context.Context, id string) error {
	return s.teams.Delete(ctx, id)
}

func (s fakeStore) GetMembership(_ context.Context, id string) (Membership, bool, error) {
	return s.memberships.Get(id)
}

func (s fakeStore) ListMemberships(context.Context) ([]Membership, error) {
	return s.memberships.List()
}

func (s fakeStore) MembershipsOfTeam(_ context.Context, teamID string) ([]Membership, error) {
	return s.memberships.Where(func(m Membership) bool { return m.TeamID == teamID })
}

func (s fakeStore) MembershipsOfPerson(_ context.Context, personID string) ([]Membership, error) {
	return s.memberships.Where(func(m Membership) bool { return m.PersonID == personID })
}

func (s fakeStore) PutMembership(ctx context.Context, m Membership) error {
	return s.memberships.Put(ctx, m)
}

func (s fakeStore) DeleteMembership(ctx context.Context, id string) error {
	return s.memberships.Delete(ctx, id)
}
