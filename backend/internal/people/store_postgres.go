package people

import (
	"context"
	"errors"
	"fmt"
	"slices"

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
	everyone, err := s.people(ctx)
	row, found := everyone.Find(id)
	if err != nil || !found {
		return Person{}, false, err
	}
	return personFromRow(row), true, nil
}

func (s *PostgresStore) FindByEmail(ctx context.Context, email string) (Person, bool, error) {
	row, err := s.db.Queries(ctx).FindPersonByEmail(ctx, email)
	return onePerson(row, err, email)
}

func (s *PostgresStore) List(ctx context.Context) ([]Person, error) {
	rows, err := s.people(ctx)
	if err != nil {
		return nil, err
	}
	everyone := make([]Person, 0, len(rows.Rows))
	for _, row := range rows.Rows {
		everyone = append(everyone, personFromRow(row))
	}
	return everyone, nil
}

func (s *PostgresStore) Put(ctx context.Context, p Person) error {
	err := s.change(ctx).PutPerson(ctx, sqlcdb.PutPersonParams{
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
	all, err := s.preferences(ctx)
	row, found := all.Find(personID)
	if err != nil || !found {
		return Preferences{}, false, err
	}
	return preferencesFromRow(row), true, nil
}

func (s *PostgresStore) PutPreferences(ctx context.Context, p Preferences) error {
	pinned := p.PinnedPlanIDs
	if pinned == nil {
		pinned = []string{}
	}
	err := s.change(ctx).PutPreferences(ctx, sqlcdb.PutPreferencesParams{
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
	rows, err := s.preferences(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]Preferences, 0, len(rows.Rows))
	for _, row := range rows.Rows {
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
		HistoryIncludeInside: row.HistoryIncludeInside, PinnedPlanIDs: slices.Clone(row.PinnedPlanIds),
	}
	if len(p.PinnedPlanIDs) == 0 {
		p.PinnedPlanIDs = nil // the JSON store leaves an empty list out, so it reads back as nil
	}
	return p
}
