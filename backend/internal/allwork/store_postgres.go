package allwork

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
)

// PostgresStore is the Store on PostgreSQL. A saved view's state is a json document: it is screen
// state, which no query looks inside.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) GetPreferences(ctx context.Context, personID string) (PreferencesRecord, bool, error) {
	row, err := s.db.Queries(ctx).GetAllWorkPreferences(ctx, personID)
	if errors.Is(err, pgx.ErrNoRows) {
		return PreferencesRecord{}, false, nil
	}
	if err != nil {
		return PreferencesRecord{}, false, fmt.Errorf("select All work preferences of %s: %w", personID, err)
	}
	p := PreferencesRecord{PersonID: row.PersonID, Columns: row.Columns, UpdatedAt: row.UpdatedAt}
	if row.SortColumn != nil && row.SortDirection != nil {
		p.Sort = &Sort{Column: *row.SortColumn, Direction: *row.SortDirection}
	}
	return p, true, nil
}

func (s *PostgresStore) PutPreferences(ctx context.Context, p PreferencesRecord) error {
	params := sqlcdb.PutAllWorkPreferencesParams{PersonID: p.PersonID, Columns: p.Columns, UpdatedAt: p.UpdatedAt}
	if p.Sort != nil {
		params.SortColumn, params.SortDirection = &p.Sort.Column, &p.Sort.Direction
	}
	if err := s.db.Queries(ctx).PutAllWorkPreferences(ctx, params); err != nil {
		return fmt.Errorf("save All work preferences of %s: %w", p.PersonID, err)
	}
	return nil
}

func (s *PostgresStore) GetView(ctx context.Context, id string) (StoredView, bool, error) {
	row, err := s.db.Queries(ctx).GetSavedView(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredView{}, false, nil
	}
	if err != nil {
		return StoredView{}, false, fmt.Errorf("select saved view %s: %w", id, err)
	}
	v, err := viewFromRow(row)
	return v, err == nil, err
}

func (s *PostgresStore) ListViews(ctx context.Context, personID string) ([]StoredView, error) {
	rows, err := s.db.Queries(ctx).ListSavedViews(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("select saved views of %s: %w", personID, err)
	}
	views := make([]StoredView, 0, len(rows))
	for _, row := range rows {
		v, err := viewFromRow(row)
		if err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	return views, nil
}

func (s *PostgresStore) PutView(ctx context.Context, v StoredView) error {
	state, err := json.Marshal(v.State)
	if err != nil {
		return fmt.Errorf("encode saved view %s: %w", v.ID, err)
	}
	err = s.db.Queries(ctx).PutSavedView(ctx, sqlcdb.PutSavedViewParams{
		ID: v.ID, PersonID: v.PersonID, Name: v.Name, IsDefault: v.IsDefault, State: state,
		CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("save saved view %s: %w", v.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteView(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteSavedView(ctx, id); err != nil {
		return fmt.Errorf("delete saved view %s: %w", id, err)
	}
	return nil
}

func viewFromRow(row sqlcdb.SavedView) (StoredView, error) {
	v := StoredView{ID: row.ID, PersonID: row.PersonID, Name: row.Name, IsDefault: row.IsDefault,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if err := json.Unmarshal(row.State, &v.State); err != nil {
		return StoredView{}, fmt.Errorf("decode saved view %s: %w", row.ID, err)
	}
	return v, nil
}
