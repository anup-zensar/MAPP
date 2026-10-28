package swbs

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
)

// Kinds returns the history kind of runs: top level, so nothing ever moves a run to the recycle bin, but
// history needs every kind to read, remove and put back its records.
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{{
		Name: runKind, Singular: runKindSingular, Plural: runKindPlural,
		Snapshot: s.snapshot, Remove: s.repo.DeleteRun, Put: s.put,
	}}
}

func (s *Service) snapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	run, found, err := s.repo.GetRun(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(run)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: run.ID, Name: run.Source.text(), Data: data}, true, nil
}

func (s *Service) put(ctx context.Context, snapshot history.Snapshot) error {
	var run Run
	if err := json.Unmarshal(snapshot.Data, &run); err != nil {
		return err
	}
	return s.repo.AddRun(ctx, run)
}
