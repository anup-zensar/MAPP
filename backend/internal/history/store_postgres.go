package history

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
)

// PostgresStore is the Repository on PostgreSQL. The documents of an entry (its path, changes and
// snapshots) are json columns, which keep the text exactly as written.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) AppendEntry(ctx context.Context, e Entry) error {
	items := e.Items
	if len(items) == 0 {
		items = nil // the JSON store leaves an empty list out, so it reads back as nil
	}
	path, pathErr := json.Marshal(e.Path)
	changes, changesErr := json.Marshal(e.Changes)
	itemsJSON, itemsErr := json.Marshal(items)
	if err := errors.Join(pathErr, changesErr, itemsErr); err != nil {
		return fmt.Errorf("encode log entry %s: %w", e.ID, err)
	}
	err := s.db.Queries(ctx).AppendLogEntry(ctx, sqlcdb.AppendLogEntryParams{
		ID: e.ID, Seq: e.Seq, At: e.At, ActorID: e.Actor.ID, ActorName: e.Actor.Name, Action: string(e.Action),
		Verb: e.Verb, RecordType: e.RecordType, RecordTypeLabel: e.RecordTypeLabel, RecordID: e.RecordID,
		RecordName: e.RecordName, Path: path, Changes: changes, Comment: e.Comment, Details: e.Details, Items: itemsJSON,
	})
	if err != nil {
		return fmt.Errorf("append log entry %s: %w", e.ID, err)
	}
	return nil
}

func (s *PostgresStore) Entries(ctx context.Context) ([]Entry, error) {
	rows, err := s.db.Queries(ctx).ListLogEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("select log entries: %w", err)
	}
	entries := make([]Entry, 0, len(rows))
	for _, row := range rows {
		e := Entry{
			ID: row.ID, Seq: row.Seq, At: row.At, Actor: actor.Actor{ID: row.ActorID, Name: row.ActorName},
			Action: Action(row.Action), Verb: row.Verb, RecordType: row.RecordType,
			RecordTypeLabel: row.RecordTypeLabel, RecordID: row.RecordID, RecordName: row.RecordName,
			Comment: row.Comment, Details: row.Details,
		}
		err = errors.Join(json.Unmarshal(row.Path, &e.Path), json.Unmarshal(row.Changes, &e.Changes),
			json.Unmarshal(row.Items, &e.Items))
		if err != nil {
			return nil, fmt.Errorf("decode log entry %s: %w", row.ID, err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *PostgresStore) PutBinEntry(ctx context.Context, e BinEntry) error {
	params, err := binParams(e)
	if err != nil {
		return fmt.Errorf("encode bin entry %s: %w", e.ID, err)
	}
	if err = s.db.Queries(ctx).PutBinEntry(ctx, params); err != nil {
		return fmt.Errorf("save bin entry %s: %w", e.ID, err)
	}
	return nil
}

func (s *PostgresStore) GetBinEntry(ctx context.Context, id string) (BinEntry, bool, error) {
	row, err := s.db.Queries(ctx).GetBinEntry(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return BinEntry{}, false, nil
	}
	if err != nil {
		return BinEntry{}, false, fmt.Errorf("select bin entry %s: %w", id, err)
	}
	e, err := binEntryFromRow(row)
	return e, err == nil, err
}

func (s *PostgresStore) BinEntries(ctx context.Context) ([]BinEntry, error) {
	rows, err := s.db.Queries(ctx).ListBinEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("select bin entries: %w", err)
	}
	entries := make([]BinEntry, 0, len(rows))
	for _, row := range rows {
		e, err := binEntryFromRow(row)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func binParams(e BinEntry) (sqlcdb.PutBinEntryParams, error) {
	path, pathErr := json.Marshal(e.Path)
	parent, parentErr := json.Marshal(e.Parent)
	contents, contentsErr := json.Marshal(e.Contents)
	snapshots, snapshotsErr := json.Marshal(e.Snapshots)
	params := sqlcdb.PutBinEntryParams{
		ID: e.ID, RecordType: e.RecordType, RecordTypeLabel: e.RecordTypeLabel, RecordID: e.RecordID,
		RecordName: e.RecordName, Path: path, Parent: parent, DeletedAt: e.DeletedAt, DeletedByID: e.DeletedBy.ID,
		DeletedByName: e.DeletedBy.Name, Contents: contents, Snapshots: snapshots,
	}
	if r := e.Resolution; r != nil {
		params.ResolutionHow, params.ResolutionByID, params.ResolutionByName = &r.How, &r.By.ID, &r.By.Name
		params.ResolutionAt = &r.At
	}
	return params, errors.Join(pathErr, parentErr, contentsErr, snapshotsErr)
}

func binEntryFromRow(row sqlcdb.RecycleBin) (BinEntry, error) {
	e := BinEntry{
		ID: row.ID, RecordType: row.RecordType, RecordTypeLabel: row.RecordTypeLabel, RecordID: row.RecordID,
		RecordName: row.RecordName, DeletedAt: row.DeletedAt,
		DeletedBy: actor.Actor{ID: row.DeletedByID, Name: row.DeletedByName},
	}
	if row.ResolutionHow != nil {
		e.Resolution = &Resolution{How: *row.ResolutionHow, At: *row.ResolutionAt,
			By: actor.Actor{ID: *row.ResolutionByID, Name: *row.ResolutionByName}}
	}
	err := errors.Join(json.Unmarshal(row.Path, &e.Path), json.Unmarshal(row.Parent, &e.Parent),
		json.Unmarshal(row.Contents, &e.Contents), json.Unmarshal(row.Snapshots, &e.Snapshots))
	if err != nil {
		return BinEntry{}, fmt.Errorf("decode bin entry %s: %w", row.ID, err)
	}
	return e, nil
}
