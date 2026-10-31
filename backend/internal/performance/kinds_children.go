package performance

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
)

// childSpec builds the history.Kind of a record that hangs off an activity. Registering these is what lets
// the recycle bin collect them before it deletes the activity: without them the activity's DELETE fails on
// their foreign key, and binning an activity that has performance data is refused (FR-021).
type childSpec[T any] struct {
	name, singular, plural string
	ids                    func(ctx context.Context, activityIDs []string) ([]string, error)
	get                    func(ctx context.Context, id string) (T, bool, error)
	remove                 func(ctx context.Context, ids []string) error
	put                    func(ctx context.Context, record T) error
	parts                  func(record T) (id, parentID, label string)
	restore                func(record T) T
}

func (c childSpec[T]) kind() history.Kind {
	return history.Kind{
		Name: c.name, Singular: c.singular, Plural: c.plural, Parent: activityKind,
		Snapshot: func(ctx context.Context, id string) (history.Snapshot, bool, error) {
			record, found, err := c.get(ctx, id)
			if err != nil || !found {
				return history.Snapshot{}, false, err
			}
			data, err := json.Marshal(record)
			if err != nil {
				return history.Snapshot{}, false, err
			}
			recordID, parentID, label := c.parts(record)
			return history.Snapshot{ID: recordID, Name: label, ParentID: parentID, Data: data}, true, nil
		},
		ChildIDs: func(ctx context.Context, parentID string) ([]string, error) {
			return c.ids(ctx, []string{parentID})
		},
		Remove: func(ctx context.Context, id string) error {
			return c.remove(ctx, []string{id})
		},
		Put: func(ctx context.Context, s history.Snapshot) error {
			var record T
			if err := json.Unmarshal(s.Data, &record); err != nil {
				return err
			}
			return c.put(ctx, record)
		},
		AllChildIDs:    c.ids,
		RemoveAll:      c.remove,
		PrepareRestore: c.prepareRestore,
	}
}

// Skipped on an Undo: an Undo keeps the activity's status, a restore returns it to Created (R14).
func (c childSpec[T]) prepareRestore(_ context.Context, rc history.RestoreContext, s *history.Snapshot) ([]string, error) {
	if c.restore == nil || rc.Undo {
		return nil, nil
	}
	var record T
	if err := json.Unmarshal(s.Data, &record); err != nil {
		return nil, err
	}
	data, err := json.Marshal(c.restore(record))
	if err != nil {
		return nil, err
	}
	s.Data = data
	return nil, nil
}

func (s *Service) childKinds() []history.Kind {
	return []history.Kind{
		childSpec[Fact]{
			name: factKind, singular: "performance fact", plural: "performance facts",
			ids: s.repo.FactIDsOf, get: s.repo.GetFact, remove: s.repo.DeleteFacts, put: s.repo.PutFact,
			parts: func(f Fact) (string, string, string) { return f.ID, f.ActivityID, f.SourceReference },
			// A restore returns the activity to Created, so facts must come back held (FR-020, FR-021).
			restore: func(f Fact) Fact { f.Held = true; return f },
		}.kind(),
		childSpec[MediaCode]{
			name: mediaCodeKind, singular: "media code", plural: "media codes",
			ids: s.repo.MediaCodeIDsOf, get: s.repo.GetMediaCode, remove: s.repo.DeleteMediaCodes,
			put:   s.repo.PutMediaCode,
			parts: func(c MediaCode) (string, string, string) { return c.ID, c.ActivityID, codeText(c.Code) },
		}.kind(),
		childSpec[Alias]{
			name: aliasKind, singular: "external name alias", plural: "external name aliases",
			ids: s.repo.AliasIDsOf, get: s.repo.GetAlias, remove: s.repo.DeleteAliases, put: s.repo.PutAlias,
			parts: func(a Alias) (string, string, string) { return a.ID, a.ActivityID, a.Name },
		}.kind(),
		childSpec[IntegrationID]{
			name: integrationKind, singular: "integration ID", plural: "integration IDs",
			ids: s.repo.IntegrationIDsOf, get: s.repo.GetIntegrationID, remove: s.repo.DeleteIntegrationIDs,
			put:   s.repo.PutIntegrationID,
			parts: func(k IntegrationID) (string, string, string) { return k.ID, k.ActivityID, k.Value },
		}.kind(),
	}
}
