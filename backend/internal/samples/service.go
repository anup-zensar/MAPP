package samples

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
)

// The prefixes of the readable IDs of the sample records: SF-1001 for folders, SN-1001 for notes.
const (
	folderPrefix = "SF"
	notePrefix   = "SN"
)

// The longest folder name and note title, in characters (data model §8–§9).
const (
	maxNameLength  = 80
	maxTitleLength = 120
)

// Messages of the plan's "User-visible messages" table.
const (
	folderGoneMessage   = "This sample folder no longer exists — someone may have deleted it."
	noteGoneMessage     = "This sample note no longer exists — someone may have deleted it."
	enterNameMessage    = "Enter a name."
	nameTooLongMessage  = "Use 80 characters or fewer."
	nameTakenMessage    = "There is already a sample folder called '%s'. Choose another name."
	enterTitleMessage   = "Enter a title."
	titleTooLongMessage = "Use 120 characters or fewer."
	reloadMessage       = "Reload the page and try again."
)

// Service holds the rules of the sample records.
type Service struct {
	tx      store.Transactor
	repo    Repository
	ids     *ids.Generator
	history *history.Service
	now     clock.Clock
}

// NewService returns the samples service. It keeps the records in repo, gives them readable IDs with
// generator, logs their changes and deletes them through hist, works in transactions of tx and reads the
// time from now.
func NewService(tx store.Transactor, repo Repository, generator *ids.Generator, hist *history.Service, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, ids: generator, history: hist, now: now}
}

// ListFolders returns every sample folder with its notes, sorted by name the way people sort names:
// ignoring letter case. Every signed-in person may see them (FR-008), so a is not used yet; it is where a
// permission check will go when roles arrive.
func (s *Service) ListFolders(ctx context.Context, a actor.Actor) ([]FolderView, error) {
	var folders []Folder
	var notes []Note
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		folders, err = s.repo.ListFolders(ctx)
		if err != nil {
			return err
		}
		notes, err = s.repo.ListNotes(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	notesByFolder := map[string][]Note{} // each folder's notes, oldest first, as ListNotes gives them
	for _, n := range notes {
		notesByFolder[n.FolderID] = append(notesByFolder[n.FolderID], n)
	}
	views := make([]FolderView, 0, len(folders))
	for _, f := range folders {
		views = append(views, newFolderView(f, notesByFolder[f.ID]))
	}
	slices.SortStableFunc(views, func(x, y FolderView) int {
		return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
	})
	return views, nil
}

// newFolderView returns folder with its notes, as the API returns it. A folder without notes gets an empty
// list, never nil.
func newFolderView(folder Folder, notes []Note) FolderView {
	if notes == nil {
		notes = []Note{}
	}
	return FolderView{Folder: folder, Notes: notes}
}

// CreateFolder creates a sample folder named input.Name without its surrounding spaces, and writes a
// "created" log entry with it, in one Write.
func (s *Service) CreateFolder(ctx context.Context, a actor.Actor, input FolderInput) (FolderView, error) {
	name := strings.TrimSpace(input.Name)
	var problems errs.Problems
	checkName(&problems, name)
	if err := problems.Err(); err != nil {
		return FolderView{}, err
	}

	var folder Folder
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if err := s.checkNameFree(ctx, name, ""); err != nil {
			return err
		}
		id, err := s.ids.Next(ctx, folderPrefix)
		if err != nil {
			return err
		}
		folder = Folder{ID: id, Name: name, Meta: record.NewMeta(a, s.now())}
		if err := s.repo.PutFolder(ctx, folder); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind:       folderKind,
			RecordID:   folder.ID,
			RecordName: folder.Name,
			Action:     history.ActionCreated,
		})
		return err
	})
	if err != nil {
		return FolderView{}, err
	}
	return newFolderView(folder, nil), nil
}

// UpdateFolder renames the sample folder id to update.Name without its surrounding spaces (FR-029).
// update.Version must be the version the person started editing from: a missing one is reported together
// with the problems of the name, and an out-of-date one is refused with version_conflict, naming who changed
// the folder and when. A name that changes nothing is not a change: there is no new version and no log
// entry (research R5). Otherwise the folder gets a new version and a "changed" log entry lists the change.
func (s *Service) UpdateFolder(ctx context.Context, a actor.Actor, id string, update FolderUpdate) (FolderView, error) {
	name := strings.TrimSpace(update.Name)
	var problems errs.Problems
	checkName(&problems, name)
	if update.Version == nil || *update.Version < 1 {
		problems.Add("version", reloadMessage)
	}
	if err := problems.Err(); err != nil {
		return FolderView{}, err
	}

	var view FolderView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		folder, found, err := s.repo.GetFolder(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(folderGoneMessage)
		}
		if err := folder.CheckVersion(*update.Version, folderSingular); err != nil {
			return err
		}
		if err := s.checkNameFree(ctx, name, folder.ID); err != nil {
			return err
		}
		folder, err = s.rename(ctx, a, folder, name)
		if err != nil {
			return err
		}
		notes, err := s.repo.NotesInFolder(ctx, folder.ID)
		if err != nil {
			return err
		}
		view = newFolderView(folder, notes)
		return nil
	})
	if err != nil {
		return FolderView{}, err
	}
	return view, nil
}

// rename gives folder the name name inside the caller's Write and returns the folder as it is stored now.
// It saves and logs only when the name really changes: "Ideas" to "ideas" is a change, "Ideas" to "Ideas"
// is not.
func (s *Service) rename(ctx context.Context, a actor.Actor, folder Folder, name string) (Folder, error) {
	renamed := folder
	renamed.Name = name
	changes, err := history.Diff(folder, renamed)
	if err != nil {
		return Folder{}, err
	}
	if len(changes) == 0 {
		return folder, nil
	}

	renamed.Touch(a, s.now())
	if err := s.repo.PutFolder(ctx, renamed); err != nil {
		return Folder{}, err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{
		Kind:       folderKind,
		RecordID:   renamed.ID,
		RecordName: renamed.Name,
		Action:     history.ActionChanged,
		Changes:    changes,
	})
	if err != nil {
		return Folder{}, err
	}
	return renamed, nil
}

// DeleteFolder moves the sample folder id, together with its notes, to the recycle bin (FR-024), and
// returns the recycle-bin entry so that the person can undo the delete.
func (s *Service) DeleteFolder(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, found, err := s.repo.GetFolder(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(folderGoneMessage)
		}
		entry, err = s.history.MoveToBin(ctx, a, folderKind, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// CreateNote adds a note titled input.Title, without its surrounding spaces, to the sample folder folderID,
// and writes a "created" log entry with it, in one Write.
func (s *Service) CreateNote(ctx context.Context, a actor.Actor, folderID string, input NoteInput) (Note, error) {
	title := strings.TrimSpace(input.Title)
	var problems errs.Problems
	checkTitle(&problems, title)
	if err := problems.Err(); err != nil {
		return Note{}, err
	}

	var note Note
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, found, err := s.repo.GetFolder(ctx, folderID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(folderGoneMessage)
		}
		id, err := s.ids.Next(ctx, notePrefix)
		if err != nil {
			return err
		}
		note = Note{ID: id, FolderID: folderID, Title: title, Meta: record.NewMeta(a, s.now())}
		if err := s.repo.PutNote(ctx, note); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind:       noteKind,
			RecordID:   note.ID,
			RecordName: note.Title,
			Action:     history.ActionCreated,
		})
		return err
	})
	if err != nil {
		return Note{}, err
	}
	return note, nil
}

// DeleteNote moves the sample note id to the recycle bin (FR-024), and returns the recycle-bin entry so
// that the person can undo the delete.
func (s *Service) DeleteNote(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, found, err := s.repo.GetNote(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(noteGoneMessage)
		}
		entry, err = s.history.MoveToBin(ctx, a, noteKind, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// checkName adds the problem with a folder name, already trimmed, to problems: the name is required and
// has at most 80 characters (characters, not bytes: "é" counts as one).
func checkName(problems *errs.Problems, name string) {
	if name == "" {
		problems.Add("name", enterNameMessage)
	} else if utf8.RuneCountInString(name) > maxNameLength {
		problems.Add("name", nameTooLongMessage)
	}
}

// checkTitle adds the problem with a note title, already trimmed, to problems: the title is required and
// has at most 120 characters.
func checkTitle(problems *errs.Problems, title string) {
	if title == "" {
		problems.Add("title", enterTitleMessage)
	} else if utf8.RuneCountInString(title) > maxTitleLength {
		problems.Add("title", titleTooLongMessage)
	}
}

// checkNameFree refuses name when a sample folder other than exceptID has it, ignoring letter case (data
// model §8). The message names that folder as it is stored.
func (s *Service) checkNameFree(ctx context.Context, name, exceptID string) error {
	folders, err := s.repo.ListFolders(ctx)
	if err != nil {
		return err
	}
	if other, taken := folderNamed(folders, name, exceptID); taken {
		return errs.Validation(errs.FieldError{Field: "name", Message: fmt.Sprintf(nameTakenMessage, other.Name)})
	}
	return nil
}

// folderNamed returns the folder among folders, other than exceptID, whose name is name ignoring letter
// case, and false when there is none. Names are stored without surrounding spaces, so these are ignored
// too.
func folderNamed(folders []Folder, name, exceptID string) (Folder, bool) {
	for _, f := range folders {
		if f.ID != exceptID && strings.EqualFold(f.Name, name) {
			return f, true
		}
	}
	return Folder{}, false
}
