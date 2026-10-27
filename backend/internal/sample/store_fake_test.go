package sample

import (
	"context"

	"mapp/internal/platform/fake"
)

// fakeStore is sampletest.Store for this package's own tests, which cannot import sampletest.
type fakeStore struct {
	folders *fake.Table[Folder]
	notes   *fake.Table[Note]
}

func newFakeStore(db *fake.DB) fakeStore {
	return fakeStore{
		folders: fake.TableOf(db, "sample_folders", func(f Folder) string { return f.ID }),
		notes:   fake.TableOf(db, "sample_notes", func(n Note) string { return n.ID }),
	}
}

func (s fakeStore) GetFolder(_ context.Context, id string) (Folder, bool, error) {
	return s.folders.Get(id)
}

func (s fakeStore) ListFolders(context.Context) ([]Folder, error) {
	return s.folders.List()
}

func (s fakeStore) PutFolder(ctx context.Context, f Folder) error {
	return s.folders.Put(ctx, f)
}

func (s fakeStore) DeleteFolder(ctx context.Context, id string) error {
	return s.folders.Delete(ctx, id)
}

func (s fakeStore) GetNote(_ context.Context, id string) (Note, bool, error) {
	return s.notes.Get(id)
}

func (s fakeStore) ListNotes(context.Context) ([]Note, error) {
	return s.notes.List()
}

func (s fakeStore) NotesInFolder(_ context.Context, folderID string) ([]Note, error) {
	return s.notes.Where(func(n Note) bool { return n.FolderID == folderID })
}

func (s fakeStore) PutNote(ctx context.Context, n Note) error {
	return s.notes.Put(ctx, n)
}

func (s fakeStore) DeleteNote(ctx context.Context, id string) error {
	return s.notes.Delete(ctx, id)
}
