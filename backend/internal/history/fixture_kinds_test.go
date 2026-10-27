package history

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"mapp/internal/platform/record"
)

const (
	parentsTable       = "parents"
	childrenTable      = "children"
	grandchildrenTable = "grandchildren"
)

type testRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
	Status   string `json:"status,omitempty"`
	record.Meta
}

func newRecord(id, name, parentID string) testRecord {
	return testRecord{ID: id, Name: name, ParentID: parentID, Meta: record.NewMeta(priya, start)}
}

func (f *fixture) standardKinds() []Kind {
	return []Kind{
		f.testKind("parent", "parents", "", parentsTable),
		f.testKind("child", "children", "parent", childrenTable),
		f.testKind("grandchild", "grandchildren", "child", grandchildrenTable),
	}
}

// testKind notes every id it removes in f.removed, so tests can check the order.
func (f *fixture) testKind(name, plural, parent, table string) Kind {
	records := f.records(table)
	k := Kind{
		Name: name, Singular: name, Plural: plural, Parent: parent,
		Snapshot: func(ctx context.Context, id string) (Snapshot, bool, error) {
			r, found, err := records.Get(id)
			if err != nil || !found {
				return Snapshot{}, false, err
			}
			data, err := json.Marshal(r)
			return Snapshot{ID: r.ID, Name: r.Name, ParentID: r.ParentID, Data: data}, true, err
		},
		Remove: func(ctx context.Context, id string) error {
			f.removed = append(f.removed, id)
			return records.Delete(ctx, id)
		},
		Put: func(ctx context.Context, s Snapshot) error {
			var r testRecord
			if err := json.Unmarshal(s.Data, &r); err != nil {
				return err
			}
			return records.Put(ctx, r)
		},
	}
	if parent != "" {
		k.ChildIDs = func(_ context.Context, parentID string) ([]string, error) {
			all, err := records.List()
			if err != nil {
				return nil, err
			}
			var childIDs []string
			for _, r := range slices.Backward(all) { // backwards, so the test sees history sort them
				if r.ParentID == parentID {
					childIDs = append(childIDs, r.ID)
				}
			}
			return childIDs, nil
		}
	}
	return k
}

func describeOwner(_ context.Context, field string, value any) (string, string, error) {
	if field != "owner" {
		return "", "", nil
	}
	names := map[string]string{"U-1002": "Priya Raman", "U-1003": "Daniel Okoro"}
	id, _ := value.(string)
	return "Owner", names[id], nil
}

// newStatusFixture approves C-1, G-1 and G-2 and gives the planning kinds' restore rules: a restore (not an
// Undo) brings a record back as created, renames a child whose name is taken and sends an approved parent back.
func newStatusFixture(t *testing.T) *fixture {
	t.Helper()
	f := newBareFixture(t)
	kinds := f.standardKinds()
	backToCreated := func(ctx context.Context, rc RestoreContext, s *Snapshot) ([]string, error) {
		if rc.Undo {
			return nil, nil
		}
		var r testRecord
		if err := json.Unmarshal(s.Data, &r); err != nil {
			return nil, err
		}
		var notes []string
		if s.Kind == "child" {
			taken, err := f.nameTaken(r)
			if err != nil {
				return nil, err
			}
			if taken {
				r.Name += " (restored)"
				notes = append(notes, "Renamed to '"+r.Name+"' because the name was taken.")
			}
		}
		if r.Status == "approved" {
			r.Status = "created"
			notes = append(notes, "'"+s.Name+"' comes back as created.")
		}
		r.Touch(rc.Actor, rc.Now)
		data, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		s.Name, s.Data = r.Name, data
		if s.Kind == "grandchild" {
			return notes, f.sendParentBack(ctx, rc, r.ParentID)
		}
		return notes, nil
	}
	kinds[1].PrepareRestore = backToCreated
	kinds[2].PrepareRestore = backToCreated
	f.register(t, kinds...)
	f.addTree(t)
	approved := func(r testRecord) testRecord { r.Status = "approved"; return r }
	f.put(t, childrenTable, approved(newRecord("C-1", "Child one", "P-1")))
	f.put(t, grandchildrenTable, approved(newRecord("G-1", "Grandchild one", "C-1")), approved(newRecord("G-2", "Grandchild two", "C-1")))
	return f
}

func (f *fixture) nameTaken(r testRecord) (bool, error) {
	children, err := f.records(childrenTable).List()
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(children, func(c testRecord) bool {
		return c.ID != r.ID && c.ParentID == r.ParentID && c.Name == r.Name
	}), nil
}

func (f *fixture) sendParentBack(ctx context.Context, rc RestoreContext, parentID string) error {
	parent, found, err := f.records(childrenTable).Get(parentID)
	if err != nil || !found || parent.Status != "approved" {
		return err
	}
	parent.Status = "created"
	parent.Touch(rc.Actor, rc.Now)
	if putErr := f.records(childrenTable).Put(ctx, parent); putErr != nil {
		return putErr
	}
	_, err = f.service.Log(ctx, rc.Actor, LogInput{Kind: "child", RecordID: parent.ID, RecordName: parent.Name,
		Action: ActionStatusChanged, Verb: "sent back to created",
		Changes: []Change{{Field: "status", Old: raw(`"approved"`), New: raw(`"created"`)}}})
	return err
}

// withKind returns a second service that also knows the kind name: the fixture's service does not know it.
func (f *fixture) withKind(t *testing.T, name string) *Service {
	t.Helper()
	other := NewService(f.db, newFakeStore(f.db), f.service.ids, f.clock.Now)
	for _, k := range append(f.standardKinds(), f.testKind(name, name+"s", "parent", "others")) {
		if err := other.Register(k); err != nil {
			t.Fatalf("Register(%s): %v", k.Name, err)
		}
	}
	return other
}
