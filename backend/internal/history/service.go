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
)

const (
	logPrefix = "LOG"
	binPrefix = "BIN"
)

// undoWindow leaves a margin: the Undo message shows for 10 seconds and pauses while it has the pointer or focus.
const undoWindow = 2 * time.Minute

const maxLimit = 200

const (
	entryUnknownMessage = "This item is not in the recycle bin."
	kindUnknownMessage  = "This item can't be restored here."
	limitMessage        = "Choose a number from 1 to 200."
)

type Service struct {
	tx     Transactor
	repo   Store
	ids    *ids.Generator
	now    clock.Clock
	kinds  []Kind // parents before children
	access Access // nil: no restriction
}

func NewService(tx Transactor, repo Store, generator *ids.Generator, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, ids: generator, now: now}
}

// Undo puts every record back exactly as it was, same IDs and versions, until undoWindow after the delete.
func (s *Service) Undo(ctx context.Context, a actor.Actor, entryID string) (RestoreResult, error) {
	return s.putBack(ctx, a, entryID, true)
}

// Restore brings the records back at any time through each kind's PrepareRestore, such as a new name.
func (s *Service) Restore(ctx context.Context, a actor.Actor, entryID string) (RestoreResult, error) {
	return s.putBack(ctx, a, entryID, false)
}

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
		if err = s.canRestore(ctx, entry); err != nil {
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

type restored struct {
	result RestoreResult
	pairs  []restoredPair
	notes  []string
}

type restoredPair struct {
	kind          Kind
	before, after Snapshot
}

// restoreEntry runs inside a Write, once checkCanComeBack has let the entry back.
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
	if top := pairs[0]; top.kind.AfterRestore != nil { // once, for the deleted record, now that everything is back
		afterNotes, err := top.kind.AfterRestore(ctx, rc, top.after)
		if err != nil {
			return restored{}, err
		}
		notes = append(notes, afterNotes...)
	}
	before, after := pairs[0].before, pairs[0].after // an entry in the bin always holds its record first
	changes, err := Diff(before.Data, after.Data)
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

// checkCanComeBack: a parent still in the bin comes back first; one deleted permanently keeps the entry out.
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

// putSnapshots puts back parents' kinds first, each record after its kind's PrepareRestore.
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

func invalidLimit() error {
	return errs.Validation(errs.FieldError{Field: "limit", Message: limitMessage})
}

// ListBin lists newest first; a context without a person (authz.Unrestricted) sees every entry.
func (s *Service) ListBin(ctx context.Context, a actor.Actor) ([]BinEntry, error) {
	inBin := []BinEntry{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		all, err := s.repo.BinEntries(ctx)
		if err != nil {
			return err
		}
		for _, e := range slices.Backward(all) {
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
