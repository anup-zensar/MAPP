package bulk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
)

type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) AddAction(ctx context.Context, action Action) error {
	q := s.db.Queries(ctx)
	err := q.AddBulkAction(ctx, sqlcdb.AddBulkActionParams{
		ID: action.ID, Action: string(action.Kind), Note: action.Note, ActorID: action.Actor.ID,
		ActorName: action.Actor.Name, RanAt: action.RanAt,
	})
	if err != nil {
		return fmt.Errorf("add bulk action %s: %w", action.ID, err)
	}
	if len(action.BinEntryIDs) == 0 {
		return nil
	}
	rows := make([]sqlcdb.AddBulkActionBinEntriesParams, 0, len(action.BinEntryIDs))
	for i, id := range action.BinEntryIDs {
		rows = append(rows, sqlcdb.AddBulkActionBinEntriesParams{BulkActionID: action.ID, Position: i + 1, BinEntryID: id})
	}
	if _, err := q.AddBulkActionBinEntries(ctx, rows); err != nil {
		return fmt.Errorf("add the bin entries of bulk action %s: %w", action.ID, err)
	}
	return nil
}

func (s *PostgresStore) GetAction(ctx context.Context, id, actorID string) (Action, bool, error) {
	row, err := s.db.Queries(ctx).GetBulkActionOfActor(ctx, sqlcdb.GetBulkActionOfActorParams{ID: id, ActorID: actorID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Action{}, false, nil
	}
	if err != nil {
		return Action{}, false, fmt.Errorf("select bulk action %s: %w", id, err)
	}
	b := row.BulkAction
	action := Action{
		ID: b.ID, Kind: Kind(b.Action), Note: b.Note, Actor: actor.Actor{ID: b.ActorID, Name: b.ActorName},
		RanAt: b.RanAt, BinEntryIDs: row.BinEntryIds, UndoneAt: b.UndoneAt,
	}
	if b.UndoneByID != nil && b.UndoneByName != nil {
		action.UndoneBy = &actor.Actor{ID: *b.UndoneByID, Name: *b.UndoneByName}
	}
	return action, true, nil
}

func (s *PostgresStore) MarkUndone(ctx context.Context, id string, by actor.Actor, at time.Time) error {
	err := s.db.Queries(ctx).MarkBulkActionUndone(ctx, sqlcdb.MarkBulkActionUndoneParams{
		ID: id, UndoneAt: &at, UndoneByID: &by.ID, UndoneByName: &by.Name,
	})
	if err != nil {
		return fmt.Errorf("mark bulk action %s undone: %w", id, err)
	}
	return nil
}
