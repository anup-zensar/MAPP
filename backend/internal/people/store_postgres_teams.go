package people

import (
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/record"
)

func (s *PostgresStore) GetTeam(ctx context.Context, id string) (Team, bool, error) {
	teams, err := s.teams(ctx)
	row, found := teams.Find(id)
	if err != nil || !found {
		return Team{}, false, err
	}
	return teamFromRow(row), true, nil
}

func (s *PostgresStore) ListTeams(ctx context.Context) ([]Team, error) {
	rows, err := s.teams(ctx)
	if err != nil {
		return nil, err
	}
	teams := make([]Team, 0, len(rows.Rows))
	for _, row := range rows.Rows {
		teams = append(teams, teamFromRow(row))
	}
	return teams, nil
}

func (s *PostgresStore) PutTeam(ctx context.Context, t Team) error {
	err := s.change(ctx).PutTeam(ctx, sqlcdb.PutTeamParams{
		ID: t.ID, Name: t.Name, Description: t.Description, Colour: t.Colour, Active: t.Active,
		Version: t.Version, CreatedAt: t.CreatedAt, CreatedByID: t.CreatedBy.ID, CreatedByName: t.CreatedBy.Name,
		UpdatedAt: t.UpdatedAt, UpdatedByID: t.UpdatedBy.ID, UpdatedByName: t.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save team %s: %w", t.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteTeam(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteTeam(ctx, id); err != nil {
		return fmt.Errorf("delete team %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) GetMembership(ctx context.Context, id string) (Membership, bool, error) {
	memberships, err := s.membershipRows(ctx)
	row, found := memberships.Find(id)
	if err != nil || !found {
		return Membership{}, false, err
	}
	return membershipFromRow(row), true, nil
}

func (s *PostgresStore) ListMemberships(ctx context.Context) ([]Membership, error) {
	return s.memberships(ctx, sqlcdb.ListMembershipsParams{})
}

func (s *PostgresStore) MembershipsOfTeam(ctx context.Context, teamID string) ([]Membership, error) {
	return s.memberships(ctx, sqlcdb.ListMembershipsParams{TeamID: &teamID})
}

func (s *PostgresStore) MembershipsOfPerson(ctx context.Context, personID string) ([]Membership, error) {
	return s.memberships(ctx, sqlcdb.ListMembershipsParams{PersonID: &personID})
}

func (s *PostgresStore) PutMembership(ctx context.Context, m Membership) error {
	err := s.change(ctx).PutMembership(ctx, sqlcdb.PutMembershipParams{
		ID: m.ID, TeamID: m.TeamID, PersonID: m.PersonID, Version: m.Version, CreatedAt: m.CreatedAt,
		CreatedByID: m.CreatedBy.ID, CreatedByName: m.CreatedBy.Name, UpdatedAt: m.UpdatedAt,
		UpdatedByID: m.UpdatedBy.ID, UpdatedByName: m.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save team membership %s: %w", m.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteMembership(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteMembership(ctx, id); err != nil {
		return fmt.Errorf("delete team membership %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) memberships(ctx context.Context, filter sqlcdb.ListMembershipsParams) ([]Membership, error) {
	rows, err := s.membershipRows(ctx)
	if err != nil {
		return nil, err
	}
	memberships := make([]Membership, 0, len(rows.Rows))
	for _, row := range rows.Rows {
		if matches(filter.TeamID, row.TeamID) && matches(filter.PersonID, row.PersonID) {
			memberships = append(memberships, membershipFromRow(row))
		}
	}
	return memberships, nil
}

// matches reports whether value is the one a filter asks for; a nil filter asks for any.
func matches(filter *string, value string) bool {
	return filter == nil || *filter == value
}

func teamFromRow(row sqlcdb.Team) Team {
	return Team{
		ID: row.ID, Name: row.Name, Description: row.Description, Colour: row.Colour, Active: row.Active,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}

func membershipFromRow(row sqlcdb.TeamMembership) Membership {
	return Membership{
		ID: row.ID, TeamID: row.TeamID, PersonID: row.PersonID,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}
