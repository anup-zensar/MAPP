package sample

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists by id, the order of adding, never nil; deleting a missing record changes nothing.
type Store interface {
	GetFolder(ctx context.Context, id string) (Folder, bool, error)
	ListFolders(ctx context.Context) ([]Folder, error)
	PutFolder(ctx context.Context, f Folder) error
	DeleteFolder(ctx context.Context, id string) error

	GetNote(ctx context.Context, id string) (Note, bool, error)
	ListNotes(ctx context.Context) ([]Note, error)
	NotesInFolder(ctx context.Context, folderID string) ([]Note, error)
	PutNote(ctx context.Context, n Note) error
	DeleteNote(ctx context.Context, id string) error
}
