package activities

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
)

// The history functions of activities for many records at once (history.Kind).

func (s *Service) allChildIDs(ctx context.Context, initiativeIDs []string) ([]string, error) {
	list, err := s.repo.ListByInitiatives(ctx, initiativeIDs)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ID)
	}
	return ids, nil
}

func (s *Service) snapshotsOf(ctx context.Context, ids []string) ([]history.Snapshot, error) {
	list, err := s.repo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	snapshots := make([]history.Snapshot, 0, len(list))
	for _, a := range list {
		data, err := json.Marshal(a)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, history.Snapshot{ID: a.ID, Name: a.Name, ParentID: a.InitiativeID, Data: data})
	}
	return snapshots, nil
}

func (s *Service) putSnapshots(ctx context.Context, snapshots []history.Snapshot) error {
	list := make([]Activity, 0, len(snapshots))
	for _, snapshot := range snapshots {
		var a Activity
		if err := json.Unmarshal(snapshot.Data, &a); err != nil {
			return err
		}
		list = append(list, a)
	}
	return s.repo.AddMany(ctx, list)
}
