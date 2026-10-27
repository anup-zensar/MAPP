package history

import (
	"context"
	"strconv"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

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
		if err = s.purge(ctx, a, entry, s.now()); err != nil {
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

// purgeable returns the entries among entryIDs that Empty recycle bin deletes, each once, and their items of
// the log entry: those still in the bin, of kinds this API knows, that the person acting may see.
func (s *Service) purgeable(ctx context.Context, entryIDs []string) ([]string, []LogItem, error) {
	all, err := s.repo.BinEntries(ctx)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]BinEntry, len(all))
	for _, e := range all {
		byID[e.ID] = e
	}
	var ids []string
	var items []LogItem
	for _, id := range entryIDs {
		entry, found := byID[id]
		delete(byID, id) // an id listed twice counts once
		if !found || entry.Resolution != nil || !s.knowsKindsOf(entry) || !s.canSeeBinEntry(ctx, entry) {
			continue // 014: an entry the person cannot see was not in their confirmation either
		}
		ids = append(ids, id)
		items = append(items, LogItem{
			RecordType: entry.RecordType, RecordTypeLabel: entry.RecordTypeLabel, RecordID: entry.RecordID,
			RecordName: entry.RecordName, Details: contentsText(entry.Contents),
		})
	}
	return ids, items, nil
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
		var ids []string
		var err error
		if ids, purged, err = s.purgeable(ctx, entryIDs); err != nil || len(ids) == 0 {
			return err
		}
		resolution := Resolution{How: string(ActionDeletedPermanently), By: a, At: s.now()}
		if err = s.repo.PurgeBinEntries(ctx, ids, resolution); err != nil {
			return err
		}
		details := "1 item"
		if len(purged) > 1 {
			details = strconv.Itoa(len(purged)) + " items"
		}
		_, err = s.appendEntry(ctx, a, Entry{
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
