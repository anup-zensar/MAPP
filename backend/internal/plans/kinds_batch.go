package plans

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
)

// snapshotsOf reads the plans ids that are stored as history keeps them, all at once.
func (s *Service) snapshotsOf(ctx context.Context, ids []string) ([]history.Snapshot, error) {
	list, err := s.repo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	snapshots := make([]history.Snapshot, 0, len(list))
	for _, p := range list {
		data, err := json.Marshal(p)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, history.Snapshot{ID: p.ID, Name: p.Name, Data: data})
	}
	return snapshots, nil
}
