package history

import (
	"context"
	"errors"
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

const (
	verbDeletedPermanently = "permanently deleted"
	verbEmptiedRecycleBin  = "emptied the recycle bin"
)

// errDryRun ends the Write of a restore preview: the store keeps nothing of what the dry run did (D-177).
var errDryRun = errors.New("history: dry run of a restore")

// ListBinItems returns the unresolved entries, newest first, each with its restore state, dependants and what
// the person may do; entries of unknown kinds (D-186) or out of their sight are left out (014 FR-007).
func (s *Service) ListBinItems(ctx context.Context, a actor.Actor) ([]BinListItem, error) {
	var items []BinListItem
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		index, err := s.binIndex(ctx)
		if err != nil {
			return err
		}
		items, err = s.binItems(ctx, index)
		return err
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) binItems(ctx context.Context, index binIndex) ([]BinListItem, error) {
	items := []BinListItem{}
	var parents []recordKey
	for _, entry := range index.entries {
		if entry.Resolution == nil && entry.Parent != nil {
			parents = append(parents, recordKey{entry.Parent.Type, entry.Parent.ID})
		}
	}
	if err := index.read.loadWithAncestors(ctx, parents); err != nil {
		return nil, err
	}
	for _, entry := range slices.Backward(index.entries) { // the repository lists them oldest first
		if entry.Resolution != nil || !s.knowsKindsOf(entry) || !s.canSeeBinEntry(ctx, entry) {
			continue
		}
		state, err := s.restoreStateOf(ctx, entry, index)
		if err != nil {
			return nil, err
		}
		item := s.listItem(entry, state)
		if item.Access, err = s.binEntryAccess(ctx, entry); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	// An entry waiting for this one can no longer be restored once this one is deleted permanently (FR-028).
	for i := range items {
		for _, other := range items {
			if other.RestoreState == ParentInRecycleBin && slices.Contains(other.ParentEntryIDs, items[i].ID) {
				items[i].InsideEntryIDs = append(items[i].InsideEntryIDs, other.ID)
			}
		}
	}
	return items, nil
}

// unresolvedEntry refuses an entry unknown, out of sight, of an unknown kind or already left the bin (FR-032).
func (s *Service) unresolvedEntry(ctx context.Context, entryID string) (BinEntry, error) {
	entry, found, err := s.repo.GetBinEntry(ctx, entryID)
	if err != nil {
		return BinEntry{}, err
	}
	if !found || !s.canSeeBinEntry(ctx, entry) {
		return BinEntry{}, errs.NotFound(entryUnknownMessage)
	}
	if !s.knowsKindsOf(entry) {
		return BinEntry{}, errs.NotFound(kindUnknownMessage)
	}
	if entry.Resolution != nil {
		return BinEntry{}, errs.AlreadyResolved(entry.RecordName, *entry.Resolution)
	}
	return entry, nil
}

// PreviewRestore runs RestoreWithParents in one Write that it throws away (D-177): nothing is saved, and the
// log, the bin and the ID counters stay as they were. With a parent deleted permanently the lists are empty.
func (s *Service) PreviewRestore(ctx context.Context, a actor.Actor, entryID string) (RestorePreview, error) {
	var preview RestorePreview
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		entry, err := s.unresolvedEntry(ctx, entryID)
		if err != nil {
			return err
		}
		index, err := s.binIndex(ctx)
		if err != nil {
			return err
		}
		items, err := s.binItems(ctx, index)
		if err != nil {
			return err
		}
		state, err := s.restoreStateOf(ctx, entry, index)
		if err != nil {
			return err
		}
		if err := s.canRestoreAll(ctx, append(state.chain, entry)); err != nil {
			return err
		}
		preview = RestorePreview{
			Entry:         itemOf(items, entry.ID),
			Destination:   []RecordRef{},
			ParentEntries: []BinListItem{},
			Changes:       []RecordChange{},
			Notes:         []string{},
		}
		if state.state == ParentDeletedPermanently {
			return errDryRun
		}
		for _, parentEntry := range state.chain {
			preview.ParentEntries = append(preview.ParentEntries, itemOf(items, parentEntry.ID))
		}

		dryRun, collector := withLogCollector(ctx)
		done, err := s.restoreChain(dryRun, a, append(state.chain, entry), s.now())
		if err != nil {
			return err
		}
		if err := s.describePreview(dryRun, &preview, done, collector.entries); err != nil {
			return err
		}
		k, err := s.registeredKind(entry.RecordType)
		if err != nil {
			return err
		}
		if entry.Parent != nil {
			preview.Destination, err = s.ancestors(dryRun, k.Parent, entry.Parent.ID)
			if err != nil {
				return err
			}
		}
		return errDryRun
	})
	if err != nil && !errors.Is(err, errDryRun) {
		return RestorePreview{}, err
	}
	return preview, nil
}

// describePreview adds every change the dry run made that the notes do not state, as the hooks' entries say.
func (s *Service) describePreview(ctx context.Context, preview *RestorePreview, done []restored, logged []Entry) error {
	came := map[recordKey]bool{}
	for _, r := range done {
		for _, pair := range r.pairs {
			came[recordKey{pair.kind.Name, pair.after.ID}] = true
			changes, err := Diff(pair.before.Data, pair.after.Data)
			if err != nil {
				return err
			}
			described, err := describeChanges(ctx, pair.kind, changes)
			if err != nil {
				return err
			}
			for _, c := range unstated(described, r.notes) {
				preview.Changes = append(preview.Changes, RecordChange{
					RecordType: pair.kind.Name, RecordTypeLabel: pair.kind.Singular,
					RecordID: pair.before.ID, RecordName: pair.before.Name, Restored: true,
					Field: c.Field, Label: c.Label, OldText: c.OldText, NewText: c.NewText,
				})
			}
		}
		preview.Notes = append(preview.Notes, r.notes...)
	}
	for _, e := range logged {
		if e.Action != ActionChanged && e.Action != ActionStatusChanged || came[recordKey{e.RecordType, e.RecordID}] {
			continue
		}
		for _, c := range e.Changes {
			preview.Changes = append(preview.Changes, RecordChange{
				RecordType: e.RecordType, RecordTypeLabel: e.RecordTypeLabel,
				RecordID: e.RecordID, RecordName: e.RecordName, Restored: false,
				Field: c.Field, Label: c.Label, OldText: c.OldText, NewText: c.NewText,
			})
		}
	}
	return nil
}

// unstated drops the rename and status reset that the restore notes already state (038 FR-049).
func unstated(changes []Change, notes []string) []Change {
	if len(notes) == 0 {
		return changes
	}
	return slices.DeleteFunc(changes, func(c Change) bool { return c.Field == "name" || c.Field == "status" })
}

func (s *Service) restoreChain(ctx context.Context, a actor.Actor, chain []BinEntry, now time.Time) ([]restored, error) {
	done := make([]restored, 0, len(chain))
	for _, entry := range chain {
		r, err := s.restoreEntry(ctx, a, entry, false, now)
		if err != nil {
			return nil, err
		}
		done = append(done, r)
	}
	return done, nil
}

// RestoreWithParents is Restore both / Restore all N: parents first, each with its own log entry, in the
// caller's Write. A parent, or a record above it, deleted permanently refuses it (FR-024).
func (s *Service) RestoreWithParents(ctx context.Context, a actor.Actor, entryID string) (RestoreResult, error) {
	var result RestoreResult
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		entry, err := s.unresolvedEntry(ctx, entryID)
		if err != nil {
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
		for _, parentEntry := range state.chain {
			if !s.knowsKindsOf(parentEntry) {
				return errs.NotFound(kindUnknownMessage)
			}
		}
		if err := s.canRestoreAll(ctx, append(state.chain, entry)); err != nil {
			return err
		}
		done, err := s.restoreChain(ctx, a, append(state.chain, entry), s.now())
		if err != nil {
			return err
		}
		result = done[len(done)-1].result
		return nil
	})
	if err != nil {
		return RestoreResult{}, err
	}
	return result, nil
}
