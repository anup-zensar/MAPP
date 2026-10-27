package history

import (
	"context"
	"slices"
	"strings"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
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
	tx     store.Transactor
	repo   Store
	ids    *ids.Generator
	now    clock.Clock
	kinds  []Kind // the registered kinds, parents before children
	access Access // feature 014 (access.go); nil: no restriction
}

// NewService returns the history service. It keeps the log and the bin in repo, gives out readable IDs with
// generator, works in transactions of tx and reads the time from now. It knows no kinds until they are
// registered with Register.
func NewService(tx store.Transactor, repo Store, generator *ids.Generator, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, ids: generator, now: now}
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
// back, puts its records back, runs the AfterRestore of the deleted record's kind, marks the entry as
// resolved and writes one "restored" log entry. When anything refuses, nothing changes.
func (s *Service) putBack(ctx context.Context, a actor.Actor, entryID string, undo bool) (RestoreResult, error) {
	var result RestoreResult
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		entry, found, err := s.repo.GetBinEntry(ctx, entryID)
		if err != nil {
			return err
		}
		if !found || !s.canSeeBinEntry(ctx, entry) {
			return errs.NotFound(entryUnknownMessage)
		}
		if err := s.canRestore(ctx, entry); err != nil { // the person's roles (014 D-238)
			return err
		}
		now := s.now()
		if err := s.checkCanComeBack(ctx, entry, undo, now); err != nil {
			return err
		}
		restored, err := s.restoreEntry(ctx, a, entry, undo, now)
		result = restored.result
		return err
	})
	if err != nil {
		return RestoreResult{}, err
	}
	return result, nil
}

// restored is what restoreEntry brought back.
type restored struct {
	result RestoreResult  // the answer of the restore
	pairs  []restoredPair // every record of the entry, parents before children
	notes  []string       // the notes of the kinds' PrepareRestore and AfterRestore, in order
}

// restoredPair is one record a restore put back: as the recycle bin kept it, and as it came back.
type restoredPair struct {
	kind          Kind
	before, after Snapshot
}

// restoreEntry puts the records of the recycle-bin entry back, runs the AfterRestore of the deleted record's
// kind, marks the entry as resolved (undone or restored, by a at now) and writes one "restored" log entry
// with what PrepareRestore changed in the record and the notes of both hooks. The caller has checked that
// the entry can come back (checkCanComeBack) and runs it inside a Write. Undo, Restore, RestoreWithParents
// and the restore preview use it.
func (s *Service) restoreEntry(ctx context.Context, a actor.Actor, entry BinEntry, undo bool, now time.Time) (restored, error) {
	rc := RestoreContext{Actor: a, Now: now, Undo: undo, DeletedAt: entry.DeletedAt, DeletedBy: entry.DeletedBy}
	afters, notes, err := s.putSnapshots(ctx, rc, entry.Snapshots)
	if err != nil {
		return restored{}, err
	}
	pairs := make([]restoredPair, 0, len(afters))
	for i, after := range afters {
		k, err := s.registeredKind(after.Kind)
		if err != nil {
			return restored{}, err
		}
		pairs = append(pairs, restoredPair{kind: k, before: entry.Snapshots[i], after: after})
	}
	if top := pairs[0]; top.kind.AfterRestore != nil { // once, for the record the person deleted, now that everything is back
		afterNotes, err := top.kind.AfterRestore(ctx, rc, top.after)
		if err != nil {
			return restored{}, err
		}
		notes = append(notes, afterNotes...)
	}
	before, after := pairs[0].before, pairs[0].after // an entry in the bin always holds its record first
	changes, err := Diff(before.Data, after.Data)    // what PrepareRestore changed, such as the name
	if err != nil {
		return restored{}, err
	}

	how, details := "restored", "Restored from the recycle bin"
	if undo {
		how, details = "undone", "Undo"
	}
	entry.Resolution = &Resolution{How: how, By: a, At: now}
	entry.Snapshots = []Snapshot{} // the records are stored again; the entry keeps only who and when
	if err := s.repo.PutBinEntry(ctx, entry); err != nil {
		return restored{}, err
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
		return restored{}, err
	}

	k := pairs[0].kind
	result := RestoreResult{
		RecordType:      k.Name,
		RecordTypeLabel: k.Singular,
		RecordID:        entry.RecordID,
		RecordName:      after.Name,
		Details:         strings.Join(notes, " "),
	}
	return restored{result: result, pairs: pairs, notes: notes}, nil
}

// checkCanComeBack refuses an entry that cannot come back (FR-025, FR-026): one holding a kind that is not
// registered here (the developer sample records outside demo mode), one that already left the bin, an
// Undo asked for more than 2 minutes after the delete, and one whose parent is no longer stored: while the
// parent is in the recycle bin it must be restored first, and when it (or a record above it) was deleted
// permanently, the entry can never come back (011 FR-023, FR-024).
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
	if entry.Parent == nil {
		return nil
	}
	parentStored, err := s.isStored(ctx, *entry.Parent)
	if err != nil || parentStored {
		return err
	}
	index, err := s.binIndex(ctx)
	if err != nil {
		return err
	}
	state, err := s.restoreStateOf(ctx, entry, index)
	if err != nil {
		return err
	}
	if state.state == ParentDeletedPermanently {
		return s.parentDeletedPermanently(entry, *state.missingParent)
	}
	return errs.RestoreParentFirst(*entry.Parent)
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

// putSnapshots puts the records of snapshots back in storage, a kind at a time, parents' kinds first, each
// record after its kind's PrepareRestore when it has one. It returns every record as it was put back, in the
// same order, and the notes of the PrepareRestore hooks, in that order too.
func (s *Service) putSnapshots(ctx context.Context, rc RestoreContext, snapshots []Snapshot) ([]Snapshot, []string, error) {
	r := &restoring{rc: rc, snapshots: slices.Clone(snapshots), notes: make([][]string, len(snapshots))}
	for _, g := range groupsByKind(snapshots) {
		k, err := s.registeredKind(g.kind)
		if err != nil {
			return nil, nil, err
		}
		if err := r.prepare(ctx, k, g.positions); err != nil {
			return nil, nil, err
		}
		if err := r.put(ctx, k, g.positions); err != nil {
			return nil, nil, err
		}
	}
	return r.snapshots, slices.Concat(r.notes...), nil
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

// invalidLimit is the answer to a limit that is not a number from 1 to 200.
func invalidLimit() error {
	return errs.Validation(errs.FieldError{Field: "limit", Message: limitMessage})
}

// ListBin returns what is in the recycle bin that a may see, newest first (data model §6, 014 D-238): the
// entries that were not yet undone or restored. Features that must count every record in the bin, whoever
// asks, call it with a context without a person (authz.Unrestricted).
func (s *Service) ListBin(ctx context.Context, a actor.Actor) ([]BinEntry, error) {
	inBin := []BinEntry{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		all, err := s.repo.BinEntries(ctx)
		if err != nil {
			return err
		}
		for _, e := range slices.Backward(all) { // the repository lists them oldest first
			if e.Resolution == nil && s.canSeeBinEntry(ctx, e) {
				inBin = append(inBin, e)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return inBin, nil
}
