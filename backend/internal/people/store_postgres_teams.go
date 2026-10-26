package people

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/record"
)

func (s *PostgresStore) GetTeam(ctx context.Context, id string) (Team, bool, error) {
	row, err := s.db.Queries(ctx).GetTeam(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Team{}, false, nil
	}
	if err != nil {
		return Team{}, false, fmt.Errorf("select team %s: %w", id, err)
	}
	return teamFromRow(row), true, nil
}

func (s *PostgresStore) ListTeams(ctx context.Context) ([]Team, error) {
	rows, err := s.db.Queries(ctx).ListTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("select teams: %w", err)
	}
	teams := make([]Team, 0, len(rows))
	for _, row := range rows {
		teams = append(teams, teamFromRow(row))
	}
	return teams, nil
}

func (s *PostgresStore) PutTeam(ctx context.Context, t Team) error {
	err := s.db.Queries(ctx).PutTeam(ctx, sqlcdb.PutTeamParams{
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
	if err := s.db.Queries(ctx).DeleteTeam(ctx, id); err != nil {
		return fmt.Errorf("delete team %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) GetMembership(ctx context.Context, id string) (Membership, bool, error) {
	row, err := s.db.Queries(ctx).GetMembership(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, false, nil
	}
	if err != nil {
		return Membership{}, false, fmt.Errorf("select team membership %s: %w", id, err)
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
	err := s.db.Queries(ctx).PutMembership(ctx, sqlcdb.PutMembershipParams{
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
	if err := s.db.Queries(ctx).DeleteMembership(ctx, id); err != nil {
		return fmt.Errorf("delete team membership %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) memberships(ctx context.Context, filter sqlcdb.ListMembershipsParams) ([]Membership, error) {
	rows, err := s.db.Queries(ctx).ListMemberships(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("select team memberships: %w", err)
	}
	memberships := make([]Membership, 0, len(rows))
	for _, row := range rows {
		memberships = append(memberships, membershipFromRow(row))
	}
	return memberships, nil
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
