// Package sampletest is the in-memory fake of the developer sample folders and notes store, for tests that
// build services or the whole API without a database.
package sampletest

import (
	"context"

	"mapp/internal/platform/fake"
	"mapp/internal/sample"
)

type Store struct {
	folders *fake.Table[sample.Folder]
	notes   *fake.Table[sample.Note]
}

func NewStore(db *fake.DB) Store {
	return Store{
		folders: fake.TableOf(db, "sample_folders", func(f sample.Folder) string { return f.ID }),
		notes:   fake.TableOf(db, "sample_notes", func(n sample.Note) string { return n.ID }),
	}
}

func (s Store) GetFolder(_ context.Context, id string) (sample.Folder, bool, error) {
	return s.folders.Get(id)
}

func (s Store) ListFolders(context.Context) ([]sample.Folder, error) {
	return s.folders.List()
}

func (s Store) PutFolder(ctx context.Context, f sample.Folder) error {
	return s.folders.Put(ctx, f)
}

func (s Store) DeleteFolder(ctx context.Context, id string) error {
	return s.folders.Delete(ctx, id)
}

func (s Store) GetNote(_ context.Context, id string) (sample.Note, bool, error) {
	return s.notes.Get(id)
}

func (s Store) ListNotes(context.Context) ([]sample.Note, error) {
	return s.notes.List()
}

func (s Store) NotesInFolder(_ context.Context, folderID string) ([]sample.Note, error) {
	return s.notes.Where(func(n sample.Note) bool { return n.FolderID == folderID })
}

func (s Store) PutNote(ctx context.Context, n sample.Note) error {
	return s.notes.Put(ctx, n)
}

func (s Store) DeleteNote(ctx context.Context, id string) error {
	return s.notes.Delete(ctx, id)
}
