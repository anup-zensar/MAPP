package performance

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
)

// Only the data source reaches the recycle bin: a fact is replaced or undone with its import (FR-042).
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{{
		Name: sourceKind, Singular: sourceKindSingular, Plural: sourceKindPlural,
		Snapshot: s.snapshotSource, Remove: s.removeSource, Put: s.restoreSource,
	}}
}

func (s *Service) snapshotSource(ctx context.Context, id string) (history.Snapshot, bool, error) {
	found, err := s.repo.ListSources(ctx, id)
	if err != nil || len(found) == 0 {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(found[0].Source)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: found[0].ID, Name: found[0].Name, Data: data}, true, nil
}

func (s *Service) removeSource(ctx context.Context, id string) error {
	_, err := s.repo.DeleteSource(ctx, id)
	return err
}

func (s *Service) restoreSource(ctx context.Context, snapshot history.Snapshot) error {
	var src Source
	if err := json.Unmarshal(snapshot.Data, &src); err != nil {
		return err
	}
	return s.repo.PutSource(ctx, src)
}
