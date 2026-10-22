// Package samples holds the developer sample records: sample folders and the sample notes inside them
// (data model §8–§9, decision D-036). They are the "simple demonstration record type provided for testing"
// of User Story 5: a test bench for the shared create, edit, delete, Undo, restore and activity-log
// behaviour, used exactly as plans and activities will use it. They exist only in the local and
// development environments.
package samples

import "mapp/internal/platform/record"

// Folder is a sample folder, stored in the table "sample_folders".
type Folder struct {
	ID   string `json:"id"`   // "SF-1001", …
	Name string `json:"name"` // required, trimmed, at most 80 characters, unique among sample folders ignoring case
	record.Meta
}

// Note is a sample note inside a sample folder, stored in the table "sample_notes". The API returns notes
// exactly as they are stored (SampleNote), so there is no separate API type.
type Note struct {
	ID       string `json:"id"`       // "SN-1001", …
	FolderID string `json:"folderId"` // the folder the note is in; never changes
	Title    string `json:"title"`    // required, trimmed, at most 120 characters
	record.Meta
}

// The API types below are what the API returns or receives. They are kept apart from the stored records
// above, so the stored data can change without changing the API (research R10).

// FolderView is a sample folder as the API returns it (SampleFolder): the folder with its notes, oldest
// first. Notes is never nil, so a folder without notes has an empty list.
type FolderView struct {
	Folder
	Notes []Note `json:"notes"`
}

// FolderPage is the answer of GET /api/sample-folders (SampleFolderPage): every sample folder, sorted by
// name.
type FolderPage struct {
	Items []FolderView `json:"items"`
	Total int          `json:"total"`
}

// FolderInput is the body of POST /api/sample-folders.
type FolderInput struct {
	Name string `json:"name"`
}

// FolderUpdate is the body of PATCH /api/sample-folders/{folderId}: the new name and the version the
// person started editing from (FR-029). Version is a pointer, so that a missing version can be told apart
// from a wrong one.
type FolderUpdate struct {
	Name    string `json:"name"`
	Version *int   `json:"version"`
}

// NoteInput is the body of POST /api/sample-folders/{folderId}/notes.
type NoteInput struct {
	Title string `json:"title"`
}
