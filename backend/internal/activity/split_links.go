package activity

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"mapp/internal/history"
)

// splitLinks returns the split links of the activity a of the plan planID (FR-031): the activity it was split
// from, and the activities split off it — live ones and those in the recycle bin, "deleted".
func (s *Service) splitLinks(ctx context.Context, a Activity, planID string) (*SplitLink, []SplitLink, error) {
	into := []SplitLink{}
	binned, err := s.history.SnapshotsInBin(ctx, kindName)
	if err != nil {
		return nil, nil, err
	}
	from, err := s.splitFrom(ctx, a, planID, binned)
	if err != nil {
		return nil, nil, err
	}
	all, err := s.splitOff(ctx, a.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, other := range all {
		into = append(into, SplitLink{ID: other.ID, Name: other.Name, PlanID: planID})
	}
	for _, snapshot := range binned {
		var stored Activity
		if err = json.Unmarshal(snapshot.Data, &stored); err != nil {
			return nil, nil, err
		}
		if stored.SplitFromID == a.ID {
			into = append(into, SplitLink{ID: stored.ID, Name: stored.Name, PlanID: planID, Deleted: true})
		}
	}
	slices.SortFunc(into, func(x, y SplitLink) int { return compareIDs(x.ID, y.ID) })
	return from, into, nil
}

// splitFrom is the link to the activity a was split from, named as it is live or in the recycle bin (binned);
// nil when a was not split from another.
func (s *Service) splitFrom(ctx context.Context, a Activity, planID string, binned []history.Snapshot) (*SplitLink, error) {
	if a.SplitFromID == "" {
		return nil, nil
	}
	link := SplitLink{ID: a.SplitFromID, Name: a.SplitFromID, PlanID: planID, Deleted: true}
	other, found, err := s.repo.Get(ctx, a.SplitFromID)
	if err != nil {
		return nil, err
	}
	if found {
		link.Name, link.Deleted = other.Name, false
	} else if snapshot, inBin := snapshotOf(binned, a.SplitFromID); inBin {
		link.Name = snapshot.Name
	}
	return &link, nil
}

// splitOff returns the stored activities split off the activity id.
func (s *Service) splitOff(ctx context.Context, id string) ([]Activity, error) {
	return s.repo.ListSplitFrom(ctx, id)
}

// snapshotOf returns the snapshot of the record id among snapshots.
func snapshotOf(snapshots []history.Snapshot, id string) (history.Snapshot, bool) {
	for _, snapshot := range snapshots {
		if snapshot.ID == id {
			return snapshot, true
		}
	}
	return history.Snapshot{}, false
}

// compareIDs orders readable IDs as the store does: shorter first, then alphabetically.
func compareIDs(x, y string) int {
	if len(x) != len(y) {
		return len(x) - len(y)
	}
	return strings.Compare(x, y)
}

// splitFromID is an activity's split link as the plan tree answers it: nil when it was not split from another.
func splitFromID(a Activity) *string {
	if a.SplitFromID == "" {
		return nil
	}
	id := a.SplitFromID
	return &id
}
