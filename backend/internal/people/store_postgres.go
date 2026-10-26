package people

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/record"
)

// PostgresStore is the Repository on PostgreSQL; teams and memberships are in store_postgres_teams.go.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Person, bool, error) {
	row, err := s.db.Queries(ctx).GetPerson(ctx, id)
	return onePerson(row, err, id)
}

func (s *PostgresStore) FindByEmail(ctx context.Context, email string) (Person, bool, error) {
	row, err := s.db.Queries(ctx).FindPersonByEmail(ctx, email)
	return onePerson(row, err, email)
}

func (s *PostgresStore) List(ctx context.Context) ([]Person, error) {
	rows, err := s.db.Queries(ctx).ListPeople(ctx)
	if err != nil {
		return nil, fmt.Errorf("select people: %w", err)
	}
	everyone := make([]Person, 0, len(rows))
	for _, row := range rows {
		everyone = append(everyone, personFromRow(row))
	}
	return everyone, nil
}

func (s *PostgresStore) Put(ctx context.Context, p Person) error {
	err := s.db.Queries(ctx).PutPerson(ctx, sqlcdb.PutPersonParams{
		ID: p.ID, Name: p.Name, Email: p.Email, JobTitle: p.JobTitle, Active: p.Active,
		ManagerID: database.NullIfEmpty(p.ManagerID), Version: p.Version, CreatedAt: p.CreatedAt,
		CreatedByID: p.CreatedBy.ID, CreatedByName: p.CreatedBy.Name, UpdatedAt: p.UpdatedAt,
		UpdatedByID: p.UpdatedBy.ID, UpdatedByName: p.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save person %s: %w", p.ID, err)
	}
	return nil
}

func (s *PostgresStore) GetPreferences(ctx context.Context, personID string) (Preferences, bool, error) {
	row, err := s.db.Queries(ctx).GetPreferences(ctx, personID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, false, nil
	}
	if err != nil {
		return Preferences{}, false, fmt.Errorf("select preferences of %s: %w", personID, err)
	}
	return preferencesFromRow(row), true, nil
}

func (s *PostgresStore) PutPreferences(ctx context.Context, p Preferences) error {
	pinned := p.PinnedPlanIDs
	if pinned == nil {
		pinned = []string{}
	}
	err := s.db.Queries(ctx).PutPreferences(ctx, sqlcdb.PutPreferencesParams{
		PersonID: p.PersonID, Theme: p.Theme, SidebarCollapsed: p.SidebarCollapsed,
		DisplayCurrency: p.DisplayCurrency, DateFormat: p.DateFormat, HomeScope: p.HomeScope,
		HistoryIncludeInside: p.HistoryIncludeInside, PinnedPlanIds: pinned,
	})
	if err != nil {
		return fmt.Errorf("save preferences of %s: %w", p.PersonID, err)
	}
	return nil
}

func (s *PostgresStore) ListPreferences(ctx context.Context) ([]Preferences, error) {
	rows, err := s.db.Queries(ctx).ListPreferences(ctx)
	if err != nil {
		return nil, fmt.Errorf("select preferences: %w", err)
	}
	list := make([]Preferences, 0, len(rows))
	for _, row := range rows {
		list = append(list, preferencesFromRow(row))
	}
	return list, nil
}

func onePerson(row sqlcdb.Person, err error, key string) (Person, bool, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Person{}, false, nil
	}
	if err != nil {
		return Person{}, false, fmt.Errorf("select person %s: %w", key, err)
	}
	return personFromRow(row), true, nil
}

func personFromRow(row sqlcdb.Person) Person {
	return Person{
		ID: row.ID, Name: row.Name, Email: row.Email, JobTitle: row.JobTitle, Active: row.Active,
		ManagerID: database.EmptyIfNull(row.ManagerID),
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}

func preferencesFromRow(row sqlcdb.Preference) Preferences {
	p := Preferences{
		PersonID: row.PersonID, Theme: row.Theme, SidebarCollapsed: row.SidebarCollapsed,
		DisplayCurrency: row.DisplayCurrency, DateFormat: row.DateFormat, HomeScope: row.HomeScope,
		HistoryIncludeInside: row.HistoryIncludeInside, PinnedPlanIDs: row.PinnedPlanIds,
	}
	if len(p.PinnedPlanIDs) == 0 {
		p.PinnedPlanIDs = nil // the JSON store leaves an empty list out, so it reads back as nil
	}
	return p
}
