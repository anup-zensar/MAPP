package history

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/authz"
	"northstar/internal/platform/errs"
)

// The verbs of the two actions of feature 011 (FR-006).
const (
	verbDeletedPermanently = "permanently deleted"
	verbEmptiedRecycleBin  = "emptied the recycle bin"
)

// errDryRun ends the Write of a restore preview: the store keeps nothing of what the dry run did (D-177).
var errDryRun = errors.New("history: dry run of a restore")

// recordKey names one record of one kind.
type recordKey struct {
	kind, id string
}

// heldRecord is a record that an unresolved recycle-bin entry holds.
type heldRecord struct {
	entry BinEntry // the entry holding it
	name  string   // its name in the bin
}

// binIndex is the recycle bin as the restore rules read it: every entry, oldest first, and every record
// the unresolved entries hold, whether at the top or inside.
type binIndex struct {
	entries []BinEntry
	held    map[recordKey]heldRecord
}

// binIndex reads the recycle bin and indexes it. Call it inside a transaction.
func (s *Service) binIndex(ctx context.Context) (binIndex, error) {
	entries, err := s.repo.BinEntries(ctx)
	if err != nil {
		return binIndex{}, err
	}
	index := binIndex{entries: entries, held: map[recordKey]heldRecord{}}
	for _, e := range entries {
		if e.Resolution != nil {
			continue // a resolved entry holds nothing any more
		}
		for _, snapshot := range e.Snapshots {
			index.held[recordKey{snapshot.Kind, snapshot.ID}] = heldRecord{entry: e, name: snapshot.Name}
		}
	}
	return index, nil
}

// isStored reports whether the record ref is stored: its kind is registered here and finds it.
func (s *Service) isStored(ctx context.Context, ref RecordRef) (bool, error) {
	k, found := s.findKind(ref.Type)
	if !found {
		return false, nil
	}
	_, stored, err := k.Snapshot(ctx, ref.ID)
	return stored, err
}

// restoreState is where the parent of an unresolved recycle-bin entry is (011 data model §5).
type restoreState struct {
	state         string     // Restorable, ParentInRecycleBin or ParentDeletedPermanently
	missingParent *RecordRef // the entry's parent in the bin, or the record gone for good; nil when restorable
	chain         []BinEntry // the entries to restore first, top-most first; empty unless ParentInRecycleBin
}

// restoreStateOf follows the parents of the unresolved entry upwards (FR-017, FR-023, FR-024): while a
// parent is not stored, the entry of the bin that holds it must come back first, and so on up. It stops at
// a stored parent or the top level (restorable, or parent in the recycle bin with the chain of entries),
// or at a parent that is neither stored nor in the bin: gone for good.
func (s *Service) restoreStateOf(ctx context.Context, entry BinEntry, index binIndex) (restoreState, error) {
	var chain []BinEntry
	visited := map[string]bool{entry.ID: true} // the entries met so far: a loop would never end
	for parent := entry.Parent; parent != nil; {
		stored, err := s.isStored(ctx, *parent)
		if err != nil {
			return restoreState{}, err
		}
		if stored {
			break
		}
		holder, inBin := index.held[recordKey{parent.Type, parent.ID}]
		if !inBin || visited[holder.entry.ID] {
			gone := *parent
			return restoreState{state: ParentDeletedPermanently, missingParent: &gone}, nil
		}
		visited[holder.entry.ID] = true
		chain = append(chain, holder.entry)
		parent = holder.entry.Parent // the holder comes back as a whole, so its own parent must be there
	}
	if len(chain) == 0 {
		return restoreState{state: Restorable}, nil
	}
	slices.Reverse(chain) // collected from the bottom up; restored from the top down
	parent := *entry.Parent
	return restoreState{state: ParentInRecycleBin, missingParent: &parent, chain: chain}, nil
}

// parentDeletedPermanently is the refusal to restore entry because the record gone, its parent or a record
// above it, was deleted permanently (FR-024).
func (s *Service) parentDeletedPermanently(entry BinEntry, gone RecordRef) *errs.Error {
	return errs.ParentDeletedPermanently(s.singular(entry.RecordType), gone, s.singular(gone.Type))
}

// singular returns the mid-sentence name of the kind called name, or the name itself for a kind that is
// not registered here.
func (s *Service) singular(name string) string {
	if k, found := s.findKind(name); found {
		return k.Singular
	}
	return name
}

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

// listItem returns the unresolved entry with its restore state as the recycle bin lists it. Its
// InsideEntryIDs are left empty: only the whole bin says which entries wait for it.
func (s *Service) listItem(entry BinEntry, state restoreState) BinListItem {
	k, _ := s.findKind(entry.RecordType)
	item := BinListItem{
		BinEntryView:     entry.View(),
		RecordTypePlural: k.Plural,
		RestoreState:     state.state,
		ParentEntryIDs:   []string{},
		InsideEntryIDs:   []string{},
		Records:          make([]BinRecord, 0, len(entry.Snapshots)),
	}
	if state.missingParent != nil {
		p := state.missingParent
		item.MissingParent = &LabelledRef{Type: p.Type, TypeLabel: s.singular(p.Type), ID: p.ID, Name: p.Name}
	}
	for _, parentEntry := range state.chain {
		item.ParentEntryIDs = append(item.ParentEntryIDs, parentEntry.ID)
	}
	for _, snapshot := range entry.Snapshots {
		item.Records = append(item.Records, BinRecord{
			RecordType:      snapshot.Kind,
			RecordTypeLabel: s.singular(snapshot.Kind),
			RecordID:        snapshot.ID,
			RecordName:      snapshot.Name,
			ParentID:        snapshot.ParentID,
		})
	}
	return item
}

// itemOf returns the item of the entry id among items.
func itemOf(items []BinListItem, id string) BinListItem {
	i := slices.IndexFunc(items, func(item BinListItem) bool { return item.ID == id })
	if i < 0 {
		return BinListItem{}
	}
	return items[i]
}

// unresolvedEntry reads the recycle-bin entry entryID for an action on it and refuses what the action
// cannot work on (FR-032): an entry the bin never had, one holding a kind this API does not know, and one
// that already left the bin, with how, by whom and when. Call it inside a transaction.
func (s *Service) unresolvedEntry(ctx context.Context, entryID string) (BinEntry, error) {
	entry, found, err := s.repo.GetBinEntry(ctx, entryID)
	if err != nil {
		return BinEntry{}, err
	}
	if !found {
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

// Purge deletes the recycle-bin entry entryID permanently (FR-028, FR-030): its records are removed for
// good, which they already are from their tables, so the entry keeps only who did it and when; their IDs
// are never given out again and the log keeps every entry about them. It writes one "permanently deleted"
// entry with the record's place at the time of the delete and what was inside it. Entries waiting for this
// one stay in the bin and can no longer be restored (D-181). An entry that already left the bin is refused
// with how, by whom and when (FR-032). It joins the caller's Write.
func (s *Service) Purge(ctx context.Context, a actor.Actor, entryID string) (PurgeResult, error) {
	var result PurgeResult
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		entry, err := s.unresolvedEntry(ctx, entryID)
		if err != nil {
			return err
		}
		if !s.canSeeBinEntry(ctx, entry) { // 014 D-238: an entry the person cannot see names nothing of it
			return errs.Forbidden(authz.NoAccessMessage(entry.RecordTypeLabel))
		}
		if err := s.purge(ctx, a, entry, s.now()); err != nil {
			return err
		}
		_, err = s.appendEntry(ctx, a, Entry{
			Action:          ActionDeletedPermanently,
			Verb:            verbDeletedPermanently,
			RecordType:      entry.RecordType,
			RecordTypeLabel: entry.RecordTypeLabel,
			RecordID:        entry.RecordID,
			RecordName:      entry.RecordName,
			Path:            entry.Path, // the record is gone: its place when it was deleted
			Details:         contentsText(entry.Contents),
		})
		result = PurgeResult{
			RecordType:      entry.RecordType,
			RecordTypeLabel: entry.RecordTypeLabel,
			RecordID:        entry.RecordID,
			RecordName:      entry.RecordName,
		}
		return err
	})
	if err != nil {
		return PurgeResult{}, err
	}
	return result, nil
}

// purge marks the entry as deleted permanently by a at now and drops its records. Call it inside a Write.
func (s *Service) purge(ctx context.Context, a actor.Actor, entry BinEntry, now time.Time) error {
	entry.Resolution = &Resolution{How: string(ActionDeletedPermanently), By: a, At: now}
	entry.Snapshots = []Snapshot{}
	return s.repo.PutBinEntry(ctx, entry)
}

// EmptyBin deletes permanently the recycle-bin entries entryIDs, the ones the person saw in the
// confirmation (FR-029, FR-030, D-180), in one Write. Entries that already left the bin, entries holding a
// kind this API does not know and entries not listed are left alone (FR-032), and an id listed twice
// counts once. When at least one entry was deleted, it writes one "emptied the recycle bin" entry that
// lists them. It returns how many were deleted; 0 when someone else emptied the bin first. The list must
// name at least one entry. It joins the caller's Write.
func (s *Service) EmptyBin(ctx context.Context, a actor.Actor, entryIDs []string) (EmptyResult, error) {
	if len(entryIDs) == 0 {
		return EmptyResult{}, errs.Validation(errs.FieldError{Field: "entryIds", Message: reloadMessage})
	}
	var purged []LogItem
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		purged = nil
		now := s.now()
		seen := map[string]bool{}
		for _, id := range entryIDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			entry, found, err := s.repo.GetBinEntry(ctx, id)
			if err != nil {
				return err
			}
			if !found || entry.Resolution != nil || !s.knowsKindsOf(entry) || !s.canSeeBinEntry(ctx, entry) {
				continue // 014: an entry the person cannot see was not in their confirmation either
			}
			if err := s.purge(ctx, a, entry, now); err != nil {
				return err
			}
			purged = append(purged, LogItem{
				RecordType:      entry.RecordType,
				RecordTypeLabel: entry.RecordTypeLabel,
				RecordID:        entry.RecordID,
				RecordName:      entry.RecordName,
				Details:         contentsText(entry.Contents),
			})
		}
		if len(purged) == 0 {
			return nil
		}
		details := "1 item"
		if len(purged) > 1 {
			details = strconv.Itoa(len(purged)) + " items"
		}
		_, err := s.appendEntry(ctx, a, Entry{
			Action:          ActionEmptiedRecycleBin,
			Verb:            verbEmptiedRecycleBin,
			RecordType:      emptiedBinType,
			RecordTypeLabel: emptiedBinLabel,
			RecordName:      emptiedBinName,
			Details:         details,
			Items:           purged,
		})
		return err
	})
	if err != nil {
		return EmptyResult{}, err
	}
	return EmptyResult{Count: len(purged)}, nil
}
