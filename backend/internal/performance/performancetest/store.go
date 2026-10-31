// Package performancetest is the in-memory fake of the performance store, for tests that build services or
// the whole API without a database.
package performancetest

import (
	"context"
	"slices"
	"strings"
	"time"

	"mapp/internal/performance"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/fake"
)

var _ performance.Store = (*Store)(nil)

type Store struct {
	sources  *fake.Table[performance.Source]
	mappings *fake.Table[performance.Mapping]
	facts    *fake.Table[performance.Fact]
	imports  *fake.Table[performance.Import]
	reviews  *fake.Table[performance.ReviewRow]
	codes    *fake.Table[performance.MediaCode]
	// Keys stands in for the matching tables: a key, lower-cased, to the activities it reaches.
	Keys map[performance.MatchedOn]map[string][]string
	// NextCode is the media-code counter the configuration holds in PostgreSQL.
	NextCode     int64
	aliases      map[string]performance.Alias
	integrations map[string]performance.IntegrationID
}

func NewStore(db *fake.DB) *Store {
	return &Store{
		sources:      fake.TableOf(db, "performance_sources", func(s performance.Source) string { return s.ID }),
		mappings:     fake.TableOf(db, "performance_source_mappings", func(m performance.Mapping) string { return m.ID }),
		facts:        fake.TableOf(db, "performance_facts", func(f performance.Fact) string { return f.ID }),
		imports:      fake.TableOf(db, "performance_imports", func(i performance.Import) string { return i.ID }),
		reviews:      fake.TableOf(db, "performance_review_rows", func(r performance.ReviewRow) string { return r.ID }),
		codes:        fake.TableOf(db, "media_codes", func(c performance.MediaCode) string { return c.ID }),
		Keys:         map[performance.MatchedOn]map[string][]string{},
		NextCode:     9000001,
		aliases:      map[string]performance.Alias{},
		integrations: map[string]performance.IntegrationID{},
	}
}

func (s *Store) ListSources(_ context.Context, id string) ([]performance.SourceView, error) {
	all, err := s.sources.List()
	if err != nil {
		return nil, err
	}
	views := make([]performance.SourceView, 0, len(all))
	for _, src := range all {
		if id != "" && src.ID != id {
			continue
		}
		views = append(views, performance.SourceView{Source: src, FactCount: s.factsOf(src.ID)})
	}
	slices.SortFunc(views, func(a, b performance.SourceView) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return views, nil
}

func (s *Store) factsOf(sourceID string) int {
	all, err := s.facts.List()
	if err != nil {
		return 0
	}
	count := 0
	for _, f := range all {
		if f.SourceID == sourceID {
			count++
		}
	}
	return count
}

func (s *Store) PutSource(ctx context.Context, src performance.Source) error {
	return s.sources.Put(ctx, src)
}

func (s *Store) DeleteSource(ctx context.Context, id string) (bool, error) {
	if s.factsOf(id) > 0 {
		return false, nil
	}
	if _, found, err := s.sources.Get(id); err != nil || !found {
		return false, err
	}
	return true, s.sources.Delete(ctx, id)
}

func (s *Store) ListMappings(_ context.Context, sourceIDs []string) ([]performance.Mapping, error) {
	all, err := s.mappings.List()
	if err != nil {
		return nil, err
	}
	found := make([]performance.Mapping, 0, len(all))
	for _, m := range all {
		if len(sourceIDs) == 0 || slices.Contains(sourceIDs, m.SourceID) {
			found = append(found, m)
		}
	}
	return found, nil
}

func (s *Store) PutMapping(ctx context.Context, m performance.Mapping) error {
	return s.mappings.Put(ctx, m)
}

func (s *Store) DeleteMapping(ctx context.Context, id string) error {
	return s.mappings.Delete(ctx, id)
}

func (s *Store) ActiveSuppliers(ctx context.Context, modelID string) ([]performance.Supplier, error) {
	mappings, err := s.ListMappings(ctx, nil)
	if err != nil {
		return nil, err
	}
	suppliers := []performance.Supplier{}
	for _, m := range mappings {
		if m.ModelID != modelID || !m.SuppliedBy() {
			continue
		}
		src, found, srcErr := s.sources.Get(m.SourceID)
		if srcErr != nil || !found || src.Status != performance.StatusActive {
			continue
		}
		suppliers = append(suppliers, performance.Supplier{
			MappingID: m.ID, SourceID: src.ID, SourceName: src.Name, Meaning: m.Meaning, Stage: m.Stage,
		})
	}
	return suppliers, nil
}

func (s *Store) match(on performance.MatchedOn, keys []string) map[string][]string {
	found := map[string][]string{}
	for _, k := range keys {
		key := strings.ToLower(strings.TrimSpace(k))
		if hit, ok := s.Keys[on][key]; ok {
			found[key] = hit
		}
	}
	return found
}

func (s *Store) MatchByActivityID(_ context.Context, _ authz.Scope, keys []string) (map[string][]string, error) {
	return s.match(performance.MatchedActivityID, keys), nil
}

func (s *Store) MatchByMediaCode(_ context.Context, _ authz.Scope, codes []int64) (map[string][]string, error) {
	keys := make([]string, 0, len(codes))
	for _, c := range codes {
		keys = append(keys, strings.TrimSpace(strings.ToLower(itoa(c))))
	}
	return s.match(performance.MatchedMediaCode, keys), nil
}

func (s *Store) MatchByIntegrationID(_ context.Context, _ authz.Scope, _ string, keys []string) (map[string][]string, error) {
	return s.match(performance.MatchedIntegrationID, keys), nil
}

func (s *Store) MatchByExternalName(_ context.Context, _ authz.Scope, keys []string) (map[string][]string, error) {
	return s.match(performance.MatchedExternalName, keys), nil
}

func (s *Store) ListFactsForActivity(_ context.Context, _ authz.Scope, activityID string, countedOnly bool) ([]performance.FactView, error) {
	all, err := s.facts.List()
	if err != nil {
		return nil, err
	}
	views := []performance.FactView{}
	for _, f := range all {
		if f.ActivityID != activityID || f.ReplacedByID != "" {
			continue
		}
		if countedOnly && f.Held {
			continue
		}
		src, _, _ := s.sources.Get(f.SourceID)
		views = append(views, performance.FactView{Fact: f, SourceName: src.Name})
	}
	return views, nil
}

func (s *Store) PutFact(ctx context.Context, f performance.Fact) error {
	return s.facts.Put(ctx, f)
}

func (s *Store) CurrentFactByReference(_ context.Context, sourceID, reference string) (performance.Fact, bool, error) {
	all, err := s.facts.List()
	if err != nil {
		return performance.Fact{}, false, err
	}
	for _, f := range all {
		if f.SourceID == sourceID && f.SourceReference == reference && f.ReplacedByID == "" {
			return f, true, nil
		}
	}
	return performance.Fact{}, false, nil
}

func (s *Store) ReleaseHeld(ctx context.Context, activityID string, by actor.Actor, at time.Time) (int, error) {
	all, err := s.facts.List()
	if err != nil {
		return 0, err
	}
	released := 0
	for _, f := range all {
		if f.ActivityID != activityID || !f.Held {
			continue
		}
		f.Held = false
		f.Version++
		f.UpdatedAt, f.UpdatedBy = at, by
		if err := s.facts.Put(ctx, f); err != nil {
			return released, err
		}
		released++
	}
	return released, nil
}

func (s *Store) ListImports(_ context.Context, id, sourceID string, limit int) ([]performance.Import, error) {
	all, err := s.imports.List()
	if err != nil {
		return nil, err
	}
	found := []performance.Import{}
	for _, i := range all {
		if (id != "" && i.ID != id) || (sourceID != "" && i.SourceID != sourceID) {
			continue
		}
		found = append(found, i)
	}
	if limit > 0 && len(found) > limit {
		found = found[:limit]
	}
	return found, nil
}

func (s *Store) PutImport(ctx context.Context, i performance.Import) error {
	return s.imports.Put(ctx, i)
}

func (s *Store) ListReviewRows(_ context.Context, status, sourceID string, limit int) ([]performance.ReviewRowView, error) {
	all, err := s.reviews.List()
	if err != nil {
		return nil, err
	}
	views := []performance.ReviewRowView{}
	for _, r := range all {
		if (status != "" && string(r.Status) != status) || (sourceID != "" && r.SourceID != sourceID) {
			continue
		}
		src, _, _ := s.sources.Get(r.SourceID)
		views = append(views, performance.ReviewRowView{ReviewRow: r, SourceName: src.Name})
	}
	if limit > 0 && len(views) > limit {
		views = views[:limit]
	}
	return views, nil
}

func (s *Store) PutReviewRow(ctx context.Context, r performance.ReviewRow) error {
	return s.reviews.Put(ctx, r)
}

func (s *Store) ListMediaCodes(_ context.Context, _ authz.Scope, activityID string) ([]performance.MediaCode, error) {
	all, err := s.codes.List()
	if err != nil {
		return nil, err
	}
	found := []performance.MediaCode{}
	for _, c := range all {
		if c.ActivityID == activityID {
			found = append(found, c)
		}
	}
	return found, nil
}

func (s *Store) TakeNextCode(_ context.Context, _ actor.Actor, _ time.Time) (int64, error) {
	code := s.NextCode
	s.NextCode++
	return code, nil
}

func (s *Store) PutMediaCode(ctx context.Context, c performance.MediaCode) error {
	return s.codes.Put(ctx, c)
}

func (s *Store) PutIntegrationID(_ context.Context, k performance.IntegrationID) error {
	s.integrations[k.ID] = k
	key := strings.ToLower(strings.TrimSpace(k.Value))
	if s.Keys[performance.MatchedIntegrationID] == nil {
		s.Keys[performance.MatchedIntegrationID] = map[string][]string{}
	}
	s.Keys[performance.MatchedIntegrationID][key] = append(
		s.Keys[performance.MatchedIntegrationID][key], k.ActivityID)
	return nil
}

func (s *Store) idsOf(all []string) ([]string, error) { return all, nil }

func (s *Store) FactIDsOf(_ context.Context, activityIDs []string) ([]string, error) {
	all, err := s.facts.List()
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, f := range all {
		if slices.Contains(activityIDs, f.ActivityID) {
			ids = append(ids, f.ID)
		}
	}
	return s.idsOf(ids)
}

func (s *Store) MediaCodeIDsOf(_ context.Context, activityIDs []string) ([]string, error) {
	all, err := s.codes.List()
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, c := range all {
		if slices.Contains(activityIDs, c.ActivityID) {
			ids = append(ids, c.ID)
		}
	}
	return s.idsOf(ids)
}

func (s *Store) AliasIDsOf(_ context.Context, activityIDs []string) ([]string, error) {
	var ids []string
	for _, a := range s.aliases {
		if slices.Contains(activityIDs, a.ActivityID) {
			ids = append(ids, a.ID)
		}
	}
	return s.idsOf(ids)
}

func (s *Store) IntegrationIDsOf(_ context.Context, activityIDs []string) ([]string, error) {
	var ids []string
	for _, k := range s.integrations {
		if slices.Contains(activityIDs, k.ActivityID) {
			ids = append(ids, k.ID)
		}
	}
	return s.idsOf(ids)
}

func (s *Store) GetFact(_ context.Context, id string) (performance.Fact, bool, error) {
	return s.facts.Get(id)
}

func (s *Store) GetMediaCode(_ context.Context, id string) (performance.MediaCode, bool, error) {
	return s.codes.Get(id)
}

func (s *Store) GetAlias(_ context.Context, id string) (performance.Alias, bool, error) {
	a, ok := s.aliases[id]
	return a, ok, nil
}

func (s *Store) GetIntegrationID(_ context.Context, id string) (performance.IntegrationID, bool, error) {
	k, ok := s.integrations[id]
	return k, ok, nil
}

func (s *Store) DeleteFacts(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := s.facts.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteMediaCodes(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := s.codes.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteAliases(_ context.Context, ids []string) error {
	for _, id := range ids {
		delete(s.aliases, id)
	}
	return nil
}

func (s *Store) DeleteIntegrationIDs(_ context.Context, ids []string) error {
	for _, id := range ids {
		delete(s.integrations, id)
	}
	return nil
}

func (s *Store) PutAlias(_ context.Context, a performance.Alias) error {
	s.aliases[a.ID] = a
	return nil
}
