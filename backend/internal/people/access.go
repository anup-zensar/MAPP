package people

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

// The hooks of feature 014 (access control), set by internal/app over the access service before the API
// serves requests. Unset, people have no roles: CurrentUser carries no access, new people get no role, and
// nothing stops a deactivation or a team delete.

// AccessHooks is what people asks of access control (014 plan "Changes to existing packages").
type AccessHooks struct {
	// Summary returns the signed-in person's access for CurrentUser.access (FR-024).
	Summary func(ctx context.Context, personID string) (authz.MyAccess, error)
	// NewPerson gives a person just created the role for new people, in the same Write, and logs it
	// (FR-023).
	NewPerson func(ctx context.Context, a actor.Actor, personID string) error
	// CanDeactivate refuses to deactivate the last active administrator (FR-021).
	CanDeactivate func(ctx context.Context, personID string) error
	// TeamDeleted takes a deleted team out of every Chosen teams scope, in the same Write (data model §1.2).
	TeamDeleted func(ctx context.Context, a actor.Actor, teamID string) error
}

// UseAccess tells people about access control (feature 014).
func (s *Service) UseAccess(hooks AccessHooks) {
	s.access = hooks
}

// PersonTeams is a person with the ids of their teams, active or not, ordered by id: what access control
// reads to build a person's Grants (feature 014).
type PersonTeams struct {
	Person  Person
	TeamIDs []string
}

// EveryoneWithTeams returns every person, active or not, with their teams, inside the caller's transaction,
// ordered by id.
func (s *Service) EveryoneWithTeams(ctx context.Context) ([]PersonTeams, error) {
	var list []PersonTeams
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		d, err := s.readDirectory(ctx)
		if err != nil {
			return err
		}
		everyone, err := s.repo.List(ctx)
		if err != nil {
			return err
		}
		list = make([]PersonTeams, 0, len(everyone))
		for _, p := range everyone {
			list = append(list, PersonTeams{Person: p, TeamIDs: d.teamIDsOf(p.ID)})
		}
		return nil
	})
	return list, err
}

// AllTeams returns every team, active or not, inside the caller's transaction, ordered by id.
func (s *Service) AllTeams(ctx context.Context) ([]Team, error) {
	var teams []Team
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		teams, err = s.repo.ListTeams(ctx)
		return err
	})
	return teams, err
}

// CheckPersonVersion refuses, with version_conflict, a change of the person id's roles made from an
// out-of-date copy (014 D-233); an unknown person is not_found. It runs inside the caller's transaction.
func (s *Service) CheckPersonVersion(ctx context.Context, id string, version int) error {
	return s.tx.Read(ctx, func(ctx context.Context) error {
		p, err := s.storedPerson(ctx, id)
		if err != nil {
			return err
		}
		return p.CheckVersion(version, personSingular)
	})
}

// TouchPerson records that a changed the person id now — their roles, which access control keeps (014
// D-233) — and returns the person's new version. It runs inside the caller's Write, which also logs the
// change on the person.
func (s *Service) TouchPerson(ctx context.Context, a actor.Actor, id string) (int, error) {
	var version int
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		p, err := s.storedPerson(ctx, id)
		if err != nil {
			return err
		}
		p.Touch(a, s.now())
		version = p.Version
		return s.repo.Put(ctx, p)
	})
	return version, err
}

// accessOf returns the access of the person personID for CurrentUser, or nil while access control is not
// wired.
func (s *Service) accessOf(ctx context.Context, personID string) (*authz.MyAccess, error) {
	if s.access.Summary == nil {
		return nil, nil
	}
	summary, err := s.access.Summary(ctx, personID)
	if err != nil {
		return nil, err
	}
	return &summary, nil
}
