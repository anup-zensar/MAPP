package lineitem

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
)

// The history functions of line items for many records at once (history.Kind).

func (s *Service) allLineItemIDs(ctx context.Context, activityIDs []string) ([]string, error) {
	items, err := s.repo.ListByActivities(ctx, activityIDs)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(items))
	for _, li := range items {
		ids = append(ids, li.ID)
	}
	return ids, nil
}

func (s *Service) snapshotsOf(ctx context.Context, ids []string) ([]history.Snapshot, error) {
	items, err := s.repo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	snapshots := make([]history.Snapshot, 0, len(items))
	for _, li := range items {
		data, err := json.Marshal(li)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, history.Snapshot{ID: li.ID, Name: li.Name, ParentID: li.ActivityID, Data: data})
	}
	return snapshots, nil
}

func (s *Service) putAll(ctx context.Context, snapshots []history.Snapshot) error {
	items := make([]LineItem, 0, len(snapshots))
	for _, snapshot := range snapshots {
		var li LineItem
		if err := json.Unmarshal(snapshot.Data, &li); err != nil {
			return err
		}
		items = append(items, li)
	}
	return s.repo.AddMany(ctx, items)
}
