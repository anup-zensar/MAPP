package samples

import (
	"context"
	"encoding/json"
	"fmt"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
)

// The history kinds of the sample records (data model §7).
const (
	folderKind = "sampleFolder"
	noteKind   = "sampleNote"
)

// The mid-sentence names of the sample records, as the log, the recycle bin and the messages show them.
const (
	folderSingular = "sample folder"
	noteSingular   = "sample note"
)

// Messages of the plan's "User-visible messages" table about bringing a folder back.
const (
	undoNameTakenMessage = "Can't undo — there is now another %s called '%s'. Restore it from the recycle bin instead."
	renamedNote          = "Renamed to '%s' because the name was taken."
)

// Kinds returns the history kinds of the sample records, the folder before the note, for internal/app to
// register (research R8). With them, history deletes a folder together with its notes, brings them back,
// and logs their changes with the folder as their path.
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{
		{
			Name:           folderKind,
			Singular:       folderSingular,
			Plural:         "sample folders",
			Snapshot:       s.folderSnapshot,
			Remove:         s.repo.DeleteFolder,
			Put:            s.putFolder,
			PrepareRestore: s.prepareFolderRestore,
		},
		{
			Name:     noteKind,
			Singular: noteSingular,
			Plural:   "sample notes",
			Parent:   folderKind,
			Snapshot: s.noteSnapshot,
			ChildIDs: s.noteIDs,
			Remove:   s.repo.DeleteNote,
			Put:      s.putNote,
		},
	}
}

// folderSnapshot reads the folder id as history keeps it: the stored folder and its name.
func (s *Service) folderSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	folder, found, err := s.repo.GetFolder(ctx, id)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	if !found {
		return history.Snapshot{}, false, nil
	}
	data, err := json.Marshal(folder)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: folder.ID, Name: folder.Name, Data: data}, true, nil
}

// noteSnapshot reads the note id as history keeps it: the stored note, its title as its name, and its
// folder as its parent.
func (s *Service) noteSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	note, found, err := s.repo.GetNote(ctx, id)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	if !found {
		return history.Snapshot{}, false, nil
	}
	data, err := json.Marshal(note)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: note.ID, Name: note.Title, ParentID: note.FolderID, Data: data}, true, nil
}

// noteIDs lists the ids of the notes in the folder folderID.
func (s *Service) noteIDs(ctx context.Context, folderID string) ([]string, error) {
	notes, err := s.repo.NotesInFolder(ctx, folderID)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, len(notes))
	for _, n := range notes {
		list = append(list, n.ID)
	}
	return list, nil
}

// putFolder stores the folder of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putFolder(ctx context.Context, snapshot history.Snapshot) error {
	var folder Folder
	if err := json.Unmarshal(snapshot.Data, &folder); err != nil {
		return err
	}
	return s.repo.PutFolder(ctx, folder)
}

// putNote stores the note of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putNote(ctx context.Context, snapshot history.Snapshot) error {
	var note Note
	if err := json.Unmarshal(snapshot.Data, &note); err != nil {
		return err
	}
	return s.repo.PutNote(ctx, note)
}

// prepareFolderRestore applies the name rule of sample folders just before a folder comes back (data model
// §8): no two folders may have the same name. When another folder has meanwhile taken the name, an Undo,
// which must put the folder back exactly, is refused with name_taken, and the folder stays in the recycle
// bin (spec edge case, D-037). A Restore renames the folder instead, to "<name> (restored)", "<name>
// (restored 2)" and so on (feature 011 FR-022), and says so in a note.
func (s *Service) prepareFolderRestore(ctx context.Context, rc history.RestoreContext, snapshot *history.Snapshot) ([]string, error) {
	var folder Folder
	if err := json.Unmarshal(snapshot.Data, &folder); err != nil {
		return nil, err
	}
	folders, err := s.repo.ListFolders(ctx)
	if err != nil {
		return nil, err
	}
	other, taken := folderNamed(folders, folder.Name, folder.ID)
	if !taken {
		return nil, nil
	}
	if rc.Undo {
		return nil, errs.NameTaken(fmt.Sprintf(undoNameTakenMessage, folderSingular, other.Name))
	}

	folder.Name = restoredName(folders, folder.Name, folder.ID)
	folder.Touch(rc.Actor, rc.Now) // a changed record gets a new version
	data, err := json.Marshal(folder)
	if err != nil {
		return nil, err
	}
	snapshot.Name, snapshot.Data = folder.Name, data
	return []string{fmt.Sprintf(renamedNote, folder.Name)}, nil
}

// restoredName returns the first of "<name> (restored)", "<name> (restored 2)", "<name> (restored 3)", …
// that no folder among folders other than id has. The suffix may make the name longer than 80
// characters: nothing but a missing parent may block a restore (feature 011 FR-025).
func restoredName(folders []Folder, name, id string) string {
	candidate := name + " (restored)"
	for n := 2; ; n++ {
		if _, taken := folderNamed(folders, candidate, id); !taken {
			return candidate
		}
		candidate = fmt.Sprintf("%s (restored %d)", name, n)
	}
}
