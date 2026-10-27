package samples

import "context"

// Store stores sample folders and notes. The JSON store implements it now (store_json.go) and
// PostgreSQL could later, without any change to the service. Every method runs inside the caller's
// transaction, which the context carries.
type Store interface {
	// GetFolder returns the folder with id, and false when there is none.
	GetFolder(ctx context.Context, id string) (Folder, bool, error)
	// ListFolders returns every folder, ordered by id.
	ListFolders(ctx context.Context) ([]Folder, error)
	// PutFolder saves f, replacing the folder with the same id.
	PutFolder(ctx context.Context, f Folder) error
	// DeleteFolder removes the folder with id. Deleting a folder that does not exist changes nothing.
	DeleteFolder(ctx context.Context, id string) error

	// GetNote returns the note with id, and false when there is none.
	GetNote(ctx context.Context, id string) (Note, bool, error)
	// ListNotes returns every note of every folder, oldest first.
	ListNotes(ctx context.Context) ([]Note, error)
	// NotesInFolder returns the notes of the folder folderID, oldest first.
	NotesInFolder(ctx context.Context, folderID string) ([]Note, error)
	// PutNote saves n, replacing the note with the same id.
	PutNote(ctx context.Context, n Note) error
	// DeleteNote removes the note with id. Deleting a note that does not exist changes nothing.
	DeleteNote(ctx context.Context, id string) error
}
