package history

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/actor"
)

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
		if err = s.guarded(ctx, snapshots); err != nil {
			return err
		}
		top := snapshots[0]
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
		if err = s.remove(ctx, snapshots); err != nil {
			return err
		}
		if err = s.repo.PutBinEntry(ctx, entry); err != nil {
			return err
		}
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

func lastOf(path []RecordRef) *RecordRef {
	if len(path) == 0 {
		return nil
	}
	parent := path[len(path)-1]
	return &parent
}
