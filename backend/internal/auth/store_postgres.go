package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
)

type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Session, bool, error) {
	row, err := s.db.Queries(ctx).GetSession(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, fmt.Errorf("select session: %w", err)
	}
	return Session(row), true, nil
}

func (s *PostgresStore) Put(ctx context.Context, session Session) error {
	if err := s.db.Queries(ctx).PutSession(ctx, sqlcdb.PutSessionParams(session)); err != nil {
		return fmt.Errorf("save session of %s: %w", session.PersonID, err)
	}
	return nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteSession(ctx, id); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *PostgresStore) List(ctx context.Context) ([]Session, error) {
	rows, err := s.db.Queries(ctx).ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("select sessions: %w", err)
	}
	sessions := make([]Session, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, Session(row))
	}
	return sessions, nil
}
