package history

import (
	"context"
	"errors"
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// The verbs of the two actions of feature 011 (FR-006).
const (
	verbDeletedPermanently = "permanently deleted"
	verbEmptiedRecycleBin  = "emptied the recycle bin"
)

// errDryRun ends the Write of a restore preview: the store keeps nothing of what the dry run did (D-177).
var errDryRun = errors.New("history: dry run of a restore")

// ListBinItems returns what is in the recycle bin as the recycle bin page lists it (FR-014 – FR-017,
// FR-028): the unresolved entries, newest first, each with its restore state, the entries Restore both /
// all N brings back first, the entries that depend on it and the records it holds. Entries holding a kind
// this API does not know are left out (D-186), and so are the entries the person acting may not see; each
// item says what they may do with it (014 FR-007, FR-011, D-238).
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

// binItems lists the unresolved entries of index whose kinds are all registered and that the person acting
// may see, newest first, as ListBinItems describes them. The list is never nil.
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
	// An entry waiting for this one to come back first can no longer be restored once this one is
	// deleted permanently (FR-028).
	for i := range items {
		for _, other := range items {
			if other.RestoreState == ParentInRecycleBin && slices.Contains(other.ParentEntryIDs, items[i].ID) {
				items[i].InsideEntryIDs = append(items[i].InsideEntryIDs, other.ID)
			}
		}
	}
	return items, nil
}

// unresolvedEntry reads the recycle-bin entry entryID for an action on it and refuses what the action
// cannot work on (FR-032): an entry the bin never had or the person cannot see, one holding a kind this API
// does not know, and one that already left the bin, with how, by whom and when. Call it inside a
// transaction.
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

// PreviewRestore says what restoring the recycle-bin entry entryID would do, without doing it (FR-018 –
// FR-020, FR-022, FR-023, D-177): it restores the entries holding its missing parents and then the entry,
// exactly as RestoreWithParents does, in one Write that it throws away. The preview names where the record
// returns (its parent's place after the parents came back), the entries that come back first, every field
// the kinds' restore rules change in the records that come back and in stored records (what their hooks
// log), and the rules' notes. Nothing is saved, and the log, the bin and the ID counters stay as they were.
// For an entry whose parent was deleted permanently there is nothing to try: the lists are empty (FR-024).
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
		if err := s.canRestoreAll(ctx, append(state.chain, entry)); err != nil { // the person's roles (014 D-238)
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
			return errDryRun // the restore is not possible: the dialog explains why
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
		// Where it returns: the parent's place now, after the parents in the bin came back.
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

// describePreview adds to preview what the dry run did: every changed field of every record that came back
// (Restored), then every field the restore changed in a stored record, as the entries that its hooks
// logged say (FR-020), and the notes of the kinds' restore rules, in order.
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
			for _, c := range described {
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

// restoreChain restores the entries in order, the entries holding the missing parents first, by a at now,
// each with its own "restored" log entry, and returns what each brought back. Call it inside a Write after
// checking the entries.
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

// RestoreWithParents restores the recycle-bin entry entryID together with the entries holding its missing
// parents, parents first, in one step (FR-023): Restore both, or Restore all N. Each entry gets its own
// "restored" log entry with the actor. With no parent in the bin it is Restore. A parent, or a record above
// it, deleted permanently refuses it, and nothing changes (FR-024). It joins the caller's Write.
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
		if err := s.canRestoreAll(ctx, append(state.chain, entry)); err != nil { // the person's roles (014 D-238)
			return err
		}
		done, err := s.restoreChain(ctx, a, append(state.chain, entry), s.now())
		if err != nil {
			return err
		}
		result = done[len(done)-1].result // the chosen entry comes back last
		return nil
	})
	if err != nil {
		return RestoreResult{}, err
	}
	return result, nil
}
