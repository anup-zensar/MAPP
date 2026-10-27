package history

import (
	"context"
	"slices"

	"mapp/internal/platform/errs"
)

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
	read    *readSnapshots // the stored records the restore rules and the log look up
}

// binIndex reads the recycle bin and indexes it. Call it inside a transaction.
func (s *Service) binIndex(ctx context.Context) (binIndex, error) {
	entries, err := s.repo.BinEntries(ctx)
	if err != nil {
		return binIndex{}, err
	}
	index := binIndex{entries: entries, held: map[recordKey]heldRecord{}, read: s.newReadSnapshots()}
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
	return s.newReadSnapshots().isStored(ctx, ref)
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
		stored, err := index.read.isStored(ctx, *parent)
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
