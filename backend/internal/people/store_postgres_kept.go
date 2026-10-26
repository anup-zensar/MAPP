package people

import (
	"context"
	"fmt"

	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/memo"
)

// The store reads people, teams, memberships and preferences once per transaction and keeps the rows (memo):
// pages name an owner or a team for every record they show, and only this store changes them.
const keptArea = "people"

func (s *PostgresStore) change(ctx context.Context) *sqlcdb.Queries {
	memo.Forget(ctx, keptArea)
	return s.db.Queries(ctx)
}

func keptTable[R any](ctx context.Context, name string, read func(ctx context.Context) ([]R, error),
	idOf func(R) string) (memo.Table[R], error) {
	table, err := memo.GetTable(ctx, keptArea+":"+name, read, idOf)
	if err != nil {
		return memo.Table[R]{}, fmt.Errorf("select %s: %w", name, err)
	}
	return table, nil
}

func (s *PostgresStore) people(ctx context.Context) (memo.Table[sqlcdb.Person], error) {
	return keptTable(ctx, "people", func(ctx context.Context) ([]sqlcdb.Person, error) {
		return s.db.Queries(ctx).ListPeople(ctx)
	}, func(r sqlcdb.Person) string { return r.ID })
}

func (s *PostgresStore) preferences(ctx context.Context) (memo.Table[sqlcdb.Preference], error) {
	return keptTable(ctx, "preferences", func(ctx context.Context) ([]sqlcdb.Preference, error) {
		return s.db.Queries(ctx).ListPreferences(ctx)
	}, func(r sqlcdb.Preference) string { return r.PersonID })
}

func (s *PostgresStore) teams(ctx context.Context) (memo.Table[sqlcdb.Team], error) {
	return keptTable(ctx, "teams", func(ctx context.Context) ([]sqlcdb.Team, error) {
		return s.db.Queries(ctx).ListTeams(ctx)
	}, func(r sqlcdb.Team) string { return r.ID })
}

func (s *PostgresStore) membershipRows(ctx context.Context) (memo.Table[sqlcdb.TeamMembership], error) {
	return keptTable(ctx, "memberships", func(ctx context.Context) ([]sqlcdb.TeamMembership, error) {
		return s.db.Queries(ctx).ListMemberships(ctx, sqlcdb.ListMembershipsParams{})
	}, func(r sqlcdb.TeamMembership) string { return r.ID })
}
