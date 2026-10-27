package history

import (
	"context"
)

// recordNow says where the record of the entry is now (FR-008, data model §4): stored, with its current
// name and place; in the recycle bin, with the entry holding it; or gone for good. An emptied recycle bin
// is about no record, and a kind this API does not know cannot be looked up.
func (s *Service) recordNow(ctx context.Context, e Entry, index binIndex) (RecordNow, error) {
	gone := RecordNow{State: RecordNone, Path: []RecordRef{}}
	k, found := s.findKind(e.RecordType)
	if e.Action == ActionEmptiedRecycleBin || !found {
		return gone, nil
	}
	snapshot, stored, err := index.read.get(ctx, k, e.RecordID)
	if err != nil {
		return RecordNow{}, err
	}
	if stored {
		path, err := index.read.ancestors(ctx, k.Parent, snapshot.ParentID)
		if err != nil {
			return RecordNow{}, err
		}
		return RecordNow{State: RecordExists, Name: snapshot.Name, Path: path}, nil
	}
	if held, inBin := index.held[recordKey{e.RecordType, e.RecordID}]; inBin {
		return RecordNow{State: RecordInRecycleBin, Name: held.name, Path: []RecordRef{}, BinEntryID: held.entry.ID}, nil
	}
	gone.State = RecordDeletedPermanently
	return gone, nil
}

// recordsOf returns the records of the entries at positions (every entry when positions is nil), so that
// where they are now is read for all of them at once.
func recordsOf(entries []Entry, positions []int) []recordKey {
	if positions == nil {
		positions = make([]int, len(entries))
		for i := range entries {
			positions[i] = i
		}
	}
	keys := make([]recordKey, 0, len(positions))
	for _, i := range positions {
		if entries[i].Action != ActionEmptiedRecycleBin {
			keys = append(keys, recordKey{entries[i].RecordType, entries[i].RecordID})
		}
	}
	return keys
}

// recordOf returns the record id for the Record filter's chip (FR-013): its kind and latest name in the
// log, as the record of an entry or in an entry's path, and its current name when it is stored. It is nil
// when the log does not name it.
func (s *Service) recordOf(ctx context.Context, id string, entries []Entry) (*RecordRef, error) {
	var ref *RecordRef
	for _, e := range entries { // oldest first: the latest name wins
		if e.RecordID == id {
			ref = &RecordRef{Type: e.RecordType, ID: id, Name: e.RecordName}
		}
		for _, p := range e.Path {
			if p.ID == id {
				ref = &RecordRef{Type: p.Type, ID: id, Name: p.Name}
			}
		}
	}
	if ref == nil {
		return nil, nil
	}
	if k, found := s.findKind(ref.Type); found {
		snapshot, stored, err := k.Snapshot(ctx, id)
		if err != nil {
			return nil, err
		}
		if stored {
			ref.Name = snapshot.Name
		}
	}
	return ref, nil
}

// EntryViews returns entries as the API returns them (ActivityLogEntry), each with where its record is now,
// in the same order: for other features that select entries of their own, such as the recent access changes
// of feature 014. It joins the caller's transaction.
func (s *Service) EntryViews(ctx context.Context, entries []Entry) ([]EntryView, error) {
	views := make([]EntryView, 0, len(entries))
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		index, err := s.binIndex(ctx)
		if err != nil {
			return err
		}
		if err := index.read.loadWithAncestors(ctx, recordsOf(entries, nil)); err != nil {
			return err
		}
		for _, e := range entries {
			now, err := s.recordNow(ctx, e, index)
			if err != nil {
				return err
			}
			views = append(views, EntryView{Entry: e, Items: orEmpty(e.Items), RecordNow: now})
		}
		return nil
	})
	return views, err
}
