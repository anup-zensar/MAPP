package samples

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

// PostgresStore is the Store on PostgreSQL. Sample records belong to no plan, so no method takes a
// scope.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) GetFolder(ctx context.Context, id string) (Folder, bool, error) {
	row, err := s.db.Queries(ctx).GetSampleFolder(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Folder{}, false, nil
	}
	if err != nil {
		return Folder{}, false, fmt.Errorf("select sample folder %s: %w", id, err)
	}
	return folderFromRow(row), true, nil
}

func (s *PostgresStore) ListFolders(ctx context.Context) ([]Folder, error) {
	rows, err := s.db.Queries(ctx).ListSampleFolders(ctx)
	if err != nil {
		return nil, fmt.Errorf("select sample folders: %w", err)
	}
	folders := make([]Folder, 0, len(rows))
	for _, row := range rows {
		folders = append(folders, folderFromRow(row))
	}
	return folders, nil
}

func (s *PostgresStore) PutFolder(ctx context.Context, f Folder) error {
	if err := s.db.Queries(ctx).PutSampleFolder(ctx, sqlcdb.PutSampleFolderParams(folderRow(f))); err != nil {
		return fmt.Errorf("save sample folder %s: %w", f.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteFolder(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteSampleFolder(ctx, id); err != nil {
		return fmt.Errorf("delete sample folder %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) GetNote(ctx context.Context, id string) (Note, bool, error) {
	row, err := s.db.Queries(ctx).GetSampleNote(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Note{}, false, nil
	}
	if err != nil {
		return Note{}, false, fmt.Errorf("select sample note %s: %w", id, err)
	}
	return noteFromRow(row), true, nil
}

func (s *PostgresStore) ListNotes(ctx context.Context) ([]Note, error) {
	rows, err := s.db.Queries(ctx).ListSampleNotes(ctx)
	if err != nil {
		return nil, fmt.Errorf("select sample notes: %w", err)
	}
	return notesFromRows(rows), nil
}

func (s *PostgresStore) NotesInFolder(ctx context.Context, folderID string) ([]Note, error) {
	rows, err := s.db.Queries(ctx).ListSampleNotesInFolder(ctx, folderID)
	if err != nil {
		return nil, fmt.Errorf("select notes of sample folder %s: %w", folderID, err)
	}
	return notesFromRows(rows), nil
}

func (s *PostgresStore) PutNote(ctx context.Context, n Note) error {
	if err := s.db.Queries(ctx).PutSampleNote(ctx, sqlcdb.PutSampleNoteParams(noteRow(n))); err != nil {
		return fmt.Errorf("save sample note %s: %w", n.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteNote(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteSampleNote(ctx, id); err != nil {
		return fmt.Errorf("delete sample note %s: %w", id, err)
	}
	return nil
}

func folderFromRow(row sqlcdb.SampleFolder) Folder {
	return Folder{
		ID:   row.ID,
		Name: row.Name,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}

func folderRow(f Folder) sqlcdb.SampleFolder {
	return sqlcdb.SampleFolder{
		ID:            f.ID,
		Name:          f.Name,
		Version:       f.Version,
		CreatedAt:     f.CreatedAt,
		CreatedByID:   f.CreatedBy.ID,
		CreatedByName: f.CreatedBy.Name,
		UpdatedAt:     f.UpdatedAt,
		UpdatedByID:   f.UpdatedBy.ID,
		UpdatedByName: f.UpdatedBy.Name,
	}
}

func notesFromRows(rows []sqlcdb.SampleNote) []Note {
	notes := make([]Note, 0, len(rows))
	for _, row := range rows {
		notes = append(notes, noteFromRow(row))
	}
	return notes
}

func noteFromRow(row sqlcdb.SampleNote) Note {
	return Note{
		ID:       row.ID,
		FolderID: row.FolderID,
		Title:    row.Title,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}

func noteRow(n Note) sqlcdb.SampleNote {
	return sqlcdb.SampleNote{
		ID:            n.ID,
		FolderID:      n.FolderID,
		Title:         n.Title,
		Version:       n.Version,
		CreatedAt:     n.CreatedAt,
		CreatedByID:   n.CreatedBy.ID,
		CreatedByName: n.CreatedBy.Name,
		UpdatedAt:     n.UpdatedAt,
		UpdatedByID:   n.UpdatedBy.ID,
		UpdatedByName: n.UpdatedBy.Name,
	}
}
