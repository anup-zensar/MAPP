package people

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists people, teams and memberships ordered by id, and never deletes a person.
type Store interface {
	Get(ctx context.Context, id string) (Person, bool, error)
	List(ctx context.Context) ([]Person, error)
	Put(ctx context.Context, p Person) error
	GetPreferences(ctx context.Context, personID string) (Preferences, bool, error)
	PutPreferences(ctx context.Context, p Preferences) error

	GetTeam(ctx context.Context, id string) (Team, bool, error)
	ListTeams(ctx context.Context) ([]Team, error)
	PutTeam(ctx context.Context, t Team) error
	// DeleteTeam leaves the team's memberships to the caller.
	DeleteTeam(ctx context.Context, id string) error

	GetMembership(ctx context.Context, id string) (Membership, bool, error)
	ListMemberships(ctx context.Context) ([]Membership, error)
	MembershipsOfTeam(ctx context.Context, teamID string) ([]Membership, error)
	MembershipsOfPerson(ctx context.Context, personID string) ([]Membership, error)
	PutMembership(ctx context.Context, m Membership) error
	DeleteMembership(ctx context.Context, id string) error
}
