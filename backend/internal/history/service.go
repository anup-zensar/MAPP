package history

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/store"
)

// The prefixes of the readable IDs history gives out: LOG-1001 for log entries, BIN-1001 for recycle-bin
// entries (research R4).
const (
	logPrefix = "LOG"
	binPrefix = "BIN"
)

// undoWindow is how long after a delete the server accepts Undo (FR-026, D-037). The message offering Undo
// shows for 10 seconds and pauses while the pointer or the keyboard is on it, so the server leaves a margin.
const undoWindow = 2 * time.Minute

// maxLimit is the largest number of log entries one request can ask for.
const maxLimit = 200

// Messages of the plan's "User-visible messages" table.
const (
	entryUnknownMessage = "This item is not in the recycle bin."
	kindUnknownMessage  = "This item can't be restored here."
	limitMessage        = "Choose a number from 1 to 200."
)

// Service holds the rules of the activity log and the recycle bin.
type Service struct {
	tx    store.Transactor
	repo  Repository
	ids   *ids.Generator
	now   clock.Clock
	kinds []Kind // the registered kinds, parents before children
}

// NewService returns the history service. It keeps the log and the bin in repo, gives out readable IDs with
// generator, works in transactions of tx and reads the time from now. It knows no kinds until they are
// registered with Register.
func NewService(tx store.Transactor, repo Repository, generator *ids.Generator, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, ids: generator, now: now}
}

// Log writes one activity-log entry about a change of a record (FR-022) and returns it. Call it inside the
// Write that makes the change: Log joins that Write, so the change and its entry are saved together, or
// neither is. History adds the id and sequence number, the time, the actor, the kind's label, the record's
// path and the readable label and texts of every change.
//
// A mistake in the calling code is an error, and nothing is written: a kind that is not registered, an
// unknown action, a status change without its verb, or no ParentID for a record that has a parent but is
// no longer stored.
func (s *Service) Log(ctx context.Context, a actor.Actor, in LogInput) (Entry, error) {
	var entry Entry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		k, err := s.registeredKind(in.Kind)
		if err != nil {
			return err
		}
		verb, err := verbOf(in.Action, in.Verb)
		if err != nil {
			return err
		}
		path, err := s.pathOf(ctx, k, in.RecordID, in.ParentID)
		if err != nil {
			return err
		}
		changes, err := describeChanges(ctx, k, in.Changes)
		if err != nil {
			return err
		}
		// One number gives both the id and the sequence number, which orders entries saved in the same
		// millisecond (spec edge case).
		seq, err := s.ids.NextNumber(ctx, logPrefix)
		if err != nil {
			return err
		}

		entry = Entry{
			ID:              fmt.Sprintf("%s-%d", logPrefix, seq), // the form of ids.Generator.Next: LOG-1001
			Seq:             seq,
			At:              s.now(),
			Actor:           a,
			Action:          in.Action,
			Verb:            verb,
			RecordType:      k.Name,
			RecordTypeLabel: k.Singular,
			RecordID:        in.RecordID,
			RecordName:      in.RecordName,
			Path:            path,
			Changes:         changes,
			Comment:         in.Comment,
			Details:         in.Details,
		}
		return s.repo.AppendEntry(ctx, entry)
	})
	if err != nil {
		return Entry{}, err
	}
	return entry, nil
}

// verbOf returns the verb of an entry's sentence "<person> <verb> <record type> '<name>'". Each action has a
// verb of the same name ("created", "deleted", …), used when the caller names no other. A status change has
// none: it must name its own, such as "published".
func verbOf(action Action, verb string) (string, error) {
	switch action {
	case ActionCreated, ActionChanged, ActionMoved, ActionCopied, ActionDeleted, ActionRestored:
		return cmp.Or(verb, string(action)), nil // cmp.Or returns the first value that is not empty
	case ActionStatusChanged:
		if verb == "" {
			return "", errors.New(`history: a status change needs its verb, such as "published"`)
		}
		return verb, nil
	default:
		return "", fmt.Errorf("history: %q is not an action", action)
	}
}

// pathOf returns where the record recordID of kind k is: its ancestors as they are stored now, top level
// first, or an empty list for a top-level record. The path starts at the parent the record's snapshot
// names; for a record that is no longer stored, it starts at parentID.
func (s *Service) pathOf(ctx context.Context, k Kind, recordID, parentID string) ([]RecordRef, error) {
	snapshot, found, err := k.Snapshot(ctx, recordID)
	if err != nil {
		return nil, err
	}
	if found {
		parentID = snapshot.ParentID
	} else if k.Parent != "" && parentID == "" {
		return nil, fmt.Errorf("history: the %s %s is not stored, so the log needs its ParentID to say where it was",
			k.Singular, recordID)
	}
	return s.ancestors(ctx, k.Parent, parentID)
}

// ancestors returns the record id of the kind called kindName followed upwards by its own ancestors, as a
// path: top level first. An empty kindName (the parent of a top-level record) gives an empty path.
func (s *Service) ancestors(ctx context.Context, kindName, id string) ([]RecordRef, error) {
	path := []RecordRef{}
	for kindName != "" {
		k, err := s.registeredKind(kindName)
		if err != nil {
			return nil, err
		}
		snapshot, found, err := k.Snapshot(ctx, id)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("history: the %s %s is not stored", k.Singular, id)
		}
		path = append(path, RecordRef{Type: k.Name, ID: id, Name: snapshot.Name})
		kindName, id = k.Parent, snapshot.ParentID // one level up
	}
	slices.Reverse(path) // collected from the bottom up; a path starts at the top level
	return path, nil
}

// describeChanges returns changes with the label and texts people read, fixed at the time of the change
// (data model §5). The list is never nil.
func describeChanges(ctx context.Context, k Kind, changes []Change) ([]Change, error) {
	described := make([]Change, 0, len(changes))
	for _, c := range changes {
		c.Old, c.New = orNull(c.Old), orNull(c.New) // a missing value is stored as null
		label, oldText, err := describe(ctx, k, c.Field, c.Old)
		if err != nil {
			return nil, err
		}
		_, newText, err := describe(ctx, k, c.Field, c.New)
		if err != nil {
			return nil, err
		}
		c.Label, c.OldText, c.NewText = label, oldText, newText
		described = append(described, c)
	}
	return described, nil
}

// describe returns the label of field and the text of value: what the kind's Describe gives, or else the
// defaults, the field name in words and the value as people read it (data model §5).
func describe(ctx context.Context, k Kind, field string, value json.RawMessage) (label, text string, err error) {
	decoded, err := decode(value)
	if err != nil {
		return "", "", err
	}
	if k.Describe != nil {
		label, text, err = k.Describe(ctx, field, decoded)
		if err != nil {
			return "", "", err
		}
	}
	return cmp.Or(label, fieldLabel(field)), cmp.Or(text, defaultText(decoded)), nil
}

// MoveToBin deletes the record id of the kind called kindName together with everything under it (FR-024):
// it keeps them all in one recycle-bin entry, removes them from storage and writes one "deleted" log entry,
// in one Write that joins the caller's. It returns the new recycle-bin entry; its View is what the API
// answers, so the web app can offer Undo. The record must exist: the calling service checks that first,
// so it can answer its own "no longer exists" message.
func (s *Service) MoveToBin(ctx context.Context, a actor.Actor, kindName, id string) (BinEntry, error) {
	var entry BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		k, err := s.registeredKind(kindName)
		if err != nil {
			return err
		}
		snapshots, err := s.collect(ctx, k, id)
		if err != nil {
			return err
		}
		top := snapshots[0] // the deleted record itself
		path, err := s.ancestors(ctx, k.Parent, top.ParentID)
		if err != nil {
			return err
		}
		entryID, err := s.ids.Next(ctx, binPrefix)
		if err != nil {
			return err
		}

		entry = BinEntry{
			ID:              entryID,
			RecordType:      k.Name,
			RecordTypeLabel: k.Singular,
			RecordID:        top.ID,
			RecordName:      top.Name,
			Path:            path,
			Parent:          lastOf(path),
			DeletedAt:       s.now(),
			DeletedBy:       a,
			Contents:        s.contentsOf(snapshots[1:]),
			Snapshots:       snapshots,
		}
		if err := s.remove(ctx, snapshots); err != nil {
			return err
		}
		if err := s.repo.PutBinEntry(ctx, entry); err != nil {
			return err
		}
		// One entry for the whole delete (SC-004). The record is gone now, so its path starts at its parent.
		_, err = s.Log(ctx, a, LogInput{
			Kind:       k.Name,
			RecordID:   top.ID,
			RecordName: top.Name,
			ParentID:   top.ParentID,
			Action:     ActionDeleted,
			Details:    contentsText(entry.Contents),
		})
		return err
	})
	if err != nil {
		return BinEntry{}, err
	}
	return entry, nil
}

// collect returns the snapshots of the record id of kind k and of everything under it: the record first,
// then each child followed by everything under that child, so parents always come before their children.
// Child kinds are visited in the order they were registered and the records of each kind in the order of
// their ids, so the result never depends on the order in which a repository lists records.
func (s *Service) collect(ctx context.Context, k Kind, id string) ([]Snapshot, error) {
	snapshot, found, err := k.Snapshot(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("history: there is no %s %s", k.Singular, id)
	}
	snapshot.Kind = k.Name
	snapshots := []Snapshot{snapshot}

	for _, childKind := range s.childKinds(k.Name) {
		childIDs, err := childKind.ChildIDs(ctx, id)
		if err != nil {
			return nil, err
		}
		slices.Sort(childIDs)
		for _, childID := range childIDs {
			// The same steps one level down: the child, then everything under it.
			below, err := s.collect(ctx, childKind, childID)
			if err != nil {
				return nil, err
			}
			snapshots = append(snapshots, below...)
		}
	}
	return snapshots, nil
}

// remove deletes the records of snapshots from storage, children first (the list backwards), so that no
// record is ever left without its parent.
func (s *Service) remove(ctx context.Context, snapshots []Snapshot) error {
	for _, snapshot := range slices.Backward(snapshots) {
		k, err := s.registeredKind(snapshot.Kind)
		if err != nil {
			return err
		}
		if err := k.Remove(ctx, snapshot.ID); err != nil {
			return err
		}
	}
	return nil
}

// contentsOf counts the records under a deleted record by kind, in the order the kinds were registered,
// and leaves out the kinds it has none of (data model §6). The label is the kind's singular for a count of
// 1. The list is never nil.
func (s *Service) contentsOf(below []Snapshot) []ContentCount {
	contents := []ContentCount{}
	for _, k := range s.kinds {
		count := 0
		for _, snapshot := range below {
			if snapshot.Kind == k.Name {
				count++
			}
		}
		if count == 0 {
			continue
		}
		label := k.Plural
		if count == 1 {
			label = k.Singular
		}
		contents = append(contents, ContentCount{RecordType: k.Name, Label: label, Count: count})
	}
	return contents
}

// contentsText says what was under a deleted record, for the details of its log entries: "with 2 sample
// notes", or "with 2 campaigns and 5 initiatives". It is "" when nothing was under it.
func contentsText(contents []ContentCount) string {
	if len(contents) == 0 {
		return ""
	}
	counts := make([]string, 0, len(contents))
	for _, c := range contents {
		counts = append(counts, fmt.Sprintf("%d %s", c.Count, c.Label))
	}
	last := counts[len(counts)-1]
	if len(counts) == 1 {
		return "with " + last
	}
	return "with " + strings.Join(counts[:len(counts)-1], ", ") + " and " + last
}

// lastOf returns a copy of the last element of path, or nil for an empty path: the parent of a record.
func lastOf(path []RecordRef) *RecordRef {
	if len(path) == 0 {
		return nil
	}
	parent := path[len(path)-1]
	return &parent
}

// Undo undoes the delete of the recycle-bin entry entryID (FR-026): it puts every record back exactly as it
// was, with the same IDs, data and versions (domain model R14). The server accepts it for 2 minutes after
// the delete. A kind's PrepareRestore may refuse it, for example when someone has meanwhile taken the
// name; the item then stays in the bin. It joins the caller's Write.
func (s *Service) Undo(ctx context.Context, a actor.Actor, entryID string) (RestoreResult, error) {
	return s.putBack(ctx, a, entryID, true)
}

// Restore brings back the records of the recycle-bin entry entryID at any time (FR-025), applying each
// kind's restore rules in its PrepareRestore, such as a new name when the old one is taken. It joins the
// caller's Write.
func (s *Service) Restore(ctx context.Context, a actor.Actor, entryID string) (RestoreResult, error) {
	return s.putBack(ctx, a, entryID, false)
}

// putBack does the work of Undo (undo true) and Restore, in one Write: it checks that the entry can come
// back, puts its records back, marks the entry as resolved and writes one "restored" log entry. When
// anything refuses, nothing changes.
func (s *Service) putBack(ctx context.Context, a actor.Actor, entryID string, undo bool) (RestoreResult, error) {
	var result RestoreResult
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		entry, found, err := s.repo.GetBinEntry(ctx, entryID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(entryUnknownMessage)
		}
		now := s.now()
		if err := s.checkCanComeBack(ctx, entry, undo, now); err != nil {
			return err
		}

		before := entry.Snapshots[0] // an entry in the bin always holds its record first
		after, notes, err := s.putSnapshots(ctx, RestoreContext{Actor: a, Now: now, Undo: undo}, entry.Snapshots)
		if err != nil {
			return err
		}
		changes, err := Diff(before.Data, after.Data) // what PrepareRestore changed, such as the name
		if err != nil {
			return err
		}

		how, details := "restored", "Restored from the recycle bin"
		if undo {
			how, details = "undone", "Undo"
		}
		entry.Resolution = &Resolution{How: how, By: a, At: now}
		entry.Snapshots = []Snapshot{} // the records are stored again; the entry keeps only who and when
		if err := s.repo.PutBinEntry(ctx, entry); err != nil {
			return err
		}
		_, err = s.Log(ctx, a, LogInput{
			Kind:       entry.RecordType,
			RecordID:   entry.RecordID,
			RecordName: after.Name,
			Action:     ActionRestored,
			Changes:    changes,
			Details:    restoreDetails(details, entry.Contents, notes),
		})
		if err != nil {
			return err
		}

		k, err := s.registeredKind(entry.RecordType)
		if err != nil {
			return err
		}
		result = RestoreResult{
			RecordType:      k.Name,
			RecordTypeLabel: k.Singular,
			RecordID:        entry.RecordID,
			RecordName:      after.Name,
			Details:         strings.Join(notes, " "),
		}
		return nil
	})
	if err != nil {
		return RestoreResult{}, err
	}
	return result, nil
}

// checkCanComeBack refuses an entry that cannot come back (FR-025, FR-026): one holding a kind that is not
// registered here (the developer sample records outside demo mode), one that already left the bin, an
// Undo asked for more than 2 minutes after the delete, and one whose parent is no longer stored.
func (s *Service) checkCanComeBack(ctx context.Context, entry BinEntry, undo bool, now time.Time) error {
	if !s.knowsKindsOf(entry) {
		return errs.NotFound(kindUnknownMessage)
	}
	if entry.Resolution != nil {
		return errs.AlreadyResolved(entry.RecordName, *entry.Resolution)
	}
	if undo && now.Sub(entry.DeletedAt) > undoWindow {
		return errs.UndoExpired()
	}
	if entry.Parent != nil {
		parentKind, err := s.registeredKind(entry.Parent.Type)
		if err != nil {
			return err
		}
		_, parentStored, err := parentKind.Snapshot(ctx, entry.Parent.ID)
		if err != nil {
			return err
		}
		if !parentStored {
			return errs.RestoreParentFirst(*entry.Parent)
		}
	}
	return nil
}

// knowsKindsOf reports whether the kind of the entry and of every record it holds are registered here.
func (s *Service) knowsKindsOf(entry BinEntry) bool {
	if _, found := s.findKind(entry.RecordType); !found {
		return false
	}
	for _, snapshot := range entry.Snapshots {
		if _, found := s.findKind(snapshot.Kind); !found {
			return false
		}
	}
	return true
}

// putSnapshots puts the records of snapshots back in storage, parents before children, each after its
// kind's PrepareRestore when it has one. It returns the first record as it was put back, and the notes of
// the PrepareRestore hooks, in order.
func (s *Service) putSnapshots(ctx context.Context, rc RestoreContext, snapshots []Snapshot) (Snapshot, []string, error) {
	var first Snapshot
	var notes []string
	for i, snapshot := range snapshots {
		k, err := s.registeredKind(snapshot.Kind)
		if err != nil {
			return Snapshot{}, nil, err
		}
		if k.PrepareRestore != nil {
			kindNotes, err := k.PrepareRestore(ctx, rc, &snapshot) // it may change this copy of the snapshot
			if err != nil {
				return Snapshot{}, nil, err
			}
			notes = append(notes, kindNotes...)
		}
		if err := k.Put(ctx, snapshot); err != nil {
			return Snapshot{}, nil, err
		}
		if i == 0 {
			first = snapshot
		}
	}
	return first, notes, nil
}

// restoreDetails writes the details of a "restored" log entry: how it came back ("Undo" or "Restored from
// the recycle bin"), then what came with it (", with 2 sample notes"), then each note of the restore as a
// sentence of its own.
func restoreDetails(how string, contents []ContentCount, notes []string) string {
	details := how
	if with := contentsText(contents); with != "" {
		details += ", " + with
	}
	if len(notes) > 0 {
		details += ". " + strings.Join(notes, " ")
	}
	return details
}

// ListLog returns the newest activity-log entries that match filter, at most filter.Limit of them, and how
// many entries match in all. The limit must be from 1 to 200. Every signed-in person may read the whole log
// (FR-008), so a is not used yet; it is where a permission check will go when roles arrive.
func (s *Service) ListLog(ctx context.Context, a actor.Actor, filter LogFilter) ([]Entry, int, error) {
	if filter.Limit < 1 || filter.Limit > maxLimit {
		return nil, 0, invalidLimit()
	}
	var entries []Entry
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		entries, err = s.repo.Entries(ctx)
		return err
	})
	if err != nil {
		return nil, 0, err
	}

	matching := []Entry{}
	for _, e := range entries {
		if (filter.RecordType == "" || e.RecordType == filter.RecordType) &&
			(filter.RecordID == "" || e.RecordID == filter.RecordID) {
			matching = append(matching, e)
		}
	}
	// Newest first. The sequence number orders entries, also those saved in the same millisecond.
	slices.SortFunc(matching, func(x, y Entry) int { return cmp.Compare(y.Seq, x.Seq) })
	return matching[:min(filter.Limit, len(matching))], len(matching), nil
}

// invalidLimit is the answer to a limit that is not a number from 1 to 200.
func invalidLimit() error {
	return errs.Validation(errs.FieldError{Field: "limit", Message: limitMessage})
}

// ListBin returns what is in the recycle bin, newest first (data model §6): the entries that were not yet
// undone or restored. Every signed-in person may see it (FR-008), so a is not used yet; it is where a
// permission check will go when roles arrive.
func (s *Service) ListBin(ctx context.Context, a actor.Actor) ([]BinEntry, error) {
	var all []BinEntry
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		all, err = s.repo.BinEntries(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	inBin := []BinEntry{}
	for _, e := range slices.Backward(all) { // the repository lists them oldest first
		if e.Resolution == nil {
			inBin = append(inBin, e)
		}
	}
	return inBin, nil
}
