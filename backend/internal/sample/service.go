package sample

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

const (
	folderPrefix = "SF"
	notePrefix   = "SN"
)

// The longest folder name and note title, in characters.
const (
	maxNameLength  = 80
	maxTitleLength = 120
)

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

type Service struct {
	tx      Transactor
	repo    Store
	ids     *ids.Generator
	history *history.Service
	now     clock.Clock
}

func NewService(tools history.Tools, repo Store) *Service {
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now}
}

// ListFolders sorts by name ignoring case. Every signed-in person may see the samples, so a is not used.
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

func newFolderView(folder Folder, notes []Note) FolderView {
	if notes == nil {
		notes = []Note{}
	}
	return FolderView{Folder: folder, Notes: notes}
}

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

// UpdateFolder reports a missing version with the name's problems; a name that changes nothing makes no
// new version and no log entry.
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

// rename saves and logs only a real change: "Ideas" to "ideas" is one.
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

// DeleteFolder moves the folder and its notes to the recycle bin; the entry lets the person undo it.
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

func checkName(problems *errs.Problems, name string) {
	if name == "" {
		problems.Add("name", enterNameMessage)
	} else if utf8.RuneCountInString(name) > maxNameLength {
		problems.Add("name", nameTooLongMessage)
	}
}

// checkNameFree ignores letter case and names the other folder as it is stored.
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

func folderNamed(folders []Folder, name, exceptID string) (Folder, bool) {
	for _, f := range folders {
		if f.ID != exceptID && strings.EqualFold(f.Name, name) {
			return f, true
		}
	}
	return Folder{}, false
}
