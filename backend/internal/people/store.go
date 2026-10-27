package people

import "context"

// Store stores people, their preferences, teams and team memberships. The JSON store implements it now (store_json.go) and
// PostgreSQL will later, without any change to the service. Every method runs inside the caller's
// transaction, which the context carries.
type Store interface {
	// Get returns the person with id, and false when there is none.
	Get(ctx context.Context, id string) (Person, bool, error)
	// FindByEmail returns the person whose stored e-mail is email (people's e-mails are stored in lower
	// case), and false when there is none.
	FindByEmail(ctx context.Context, email string) (Person, bool, error)
	// List returns every person, active or not, ordered by id.
	List(ctx context.Context) ([]Person, error)
	// Put saves p, replacing the person with the same id.
	Put(ctx context.Context, p Person) error
	// GetPreferences returns the preferences of a person, and false when they have none saved.
	GetPreferences(ctx context.Context, personID string) (Preferences, bool, error)
	// PutPreferences saves the preferences of p.PersonID.
	PutPreferences(ctx context.Context, p Preferences) error
	// ListPreferences returns the saved preferences of everyone who changed one, ordered by person id.
	ListPreferences(ctx context.Context) ([]Preferences, error)

	// There is no DeletePerson: people are never deleted (003 FR-008).

	// GetTeam returns the team with id, and false when there is none.
	GetTeam(ctx context.Context, id string) (Team, bool, error)
	// ListTeams returns every team, active or not, ordered by id.
	ListTeams(ctx context.Context) ([]Team, error)
	// PutTeam saves t, replacing the team with the same id.
	PutTeam(ctx context.Context, t Team) error
	// DeleteTeam removes the team with id (its memberships are removed separately).
	DeleteTeam(ctx context.Context, id string) error

	// GetMembership returns the team membership with id, and false when there is none.
	GetMembership(ctx context.Context, id string) (Membership, bool, error)
	// ListMemberships returns every team membership, ordered by id.
	ListMemberships(ctx context.Context) ([]Membership, error)
	// MembershipsOfTeam returns the memberships of the team teamID, ordered by id.
	MembershipsOfTeam(ctx context.Context, teamID string) ([]Membership, error)
	// MembershipsOfPerson returns the memberships of the person personID, ordered by id.
	MembershipsOfPerson(ctx context.Context, personID string) ([]Membership, error)
	// PutMembership saves m, replacing the membership with the same id.
	PutMembership(ctx context.Context, m Membership) error
	// DeleteMembership removes the membership with id.
	DeleteMembership(ctx context.Context, id string) error
}
