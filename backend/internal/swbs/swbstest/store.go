// Package swbstest is the in-memory fake of the SWBS runs store, for tests that build services or the whole
// API without a database.
package swbstest

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/platform/fake"
	"mapp/internal/swbs"
)

type Store struct {
	runs *fake.Table[swbs.Run]
}

func NewStore(db *fake.DB) Store {
	return Store{runs: fake.TableOf(db, "swbs_runs", func(r swbs.Run) string { return r.ID })}
}

func (s Store) AddRun(ctx context.Context, run swbs.Run) error {
	if run.Codes == nil {
		run.Codes = []swbs.Code{}
	}
	return s.runs.Put(ctx, run)
}

func (s Store) GetRun(_ context.Context, id string) (swbs.Run, bool, error) {
	return s.runs.Get(id)
}

func (s Store) DeleteRun(ctx context.Context, id string) error {
	return s.runs.Delete(ctx, id)
}

func (s Store) ListRuns(_ context.Context, limit int) ([]swbs.Run, int, error) {
	all, err := s.runs.List()
	if err != nil {
		return nil, 0, err
	}
	slices.Reverse(all)
	return all[:min(limit, len(all))], len(all), nil
}

func (s Store) CountCodes(_ context.Context, prefix string) (int, error) {
	all, err := s.runs.List()
	count := 0
	for _, run := range all {
		for _, c := range run.Codes {
			if strings.HasPrefix(c.SwbsCode, prefix) {
				count++
			}
		}
	}
	return count, err
}
