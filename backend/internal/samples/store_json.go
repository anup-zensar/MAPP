package samples

import (
	"context"

	"mapp/internal/platform/store"
)

// The JSON store tables of this package.
const (
	foldersTable = "sample_folders"
	notesTable   = "sample_notes"
)

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// GetFolder reads one folder from the sample_folders table.
func (jsonRepository) GetFolder(ctx context.Context, id string) (Folder, bool, error) {
	return store.Get[Folder](ctx, foldersTable, id)
}

// ListFolders reads the whole sample_folders table.
func (jsonRepository) ListFolders(ctx context.Context) ([]Folder, error) {
	return store.List[Folder](ctx, foldersTable)
}

// PutFolder saves a folder in the sample_folders table under its id.
func (jsonRepository) PutFolder(ctx context.Context, f Folder) error {
	return store.Put(ctx, foldersTable, f.ID, f)
}

// DeleteFolder removes one folder from the sample_folders table.
func (jsonRepository) DeleteFolder(ctx context.Context, id string) error {
	return store.Delete(ctx, foldersTable, id)
}

// GetNote reads one note from the sample_notes table.
func (jsonRepository) GetNote(ctx context.Context, id string) (Note, bool, error) {
	return store.Get[Note](ctx, notesTable, id)
}

// ListNotes reads the whole sample_notes table. The store lists records by id (SN-999 before SN-1000), and
// ids are given out in order, so this is the order the notes were created in.
func (jsonRepository) ListNotes(ctx context.Context) ([]Note, error) {
	return store.List[Note](ctx, notesTable)
}

// NotesInFolder reads every note to find those of one folder, which is quick for the few notes of a demo.
// PostgreSQL would use an index instead.
func (r jsonRepository) NotesInFolder(ctx context.Context, folderID string) ([]Note, error) {
	notes, err := r.ListNotes(ctx)
	if err != nil {
		return nil, err
	}
	inFolder := []Note{}
	for _, n := range notes {
		if n.FolderID == folderID {
			inFolder = append(inFolder, n)
		}
	}
	return inFolder, nil
}

// PutNote saves a note in the sample_notes table under its id.
func (jsonRepository) PutNote(ctx context.Context, n Note) error {
	return store.Put(ctx, notesTable, n.ID, n)
}

// DeleteNote removes one note from the sample_notes table.
func (jsonRepository) DeleteNote(ctx context.Context, id string) error {
	return store.Delete(ctx, notesTable, id)
}
