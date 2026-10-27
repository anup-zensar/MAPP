package history

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
)

// Tables of the test kinds.
const (
	parentsTable       = "parents"
	childrenTable      = "children"
	grandchildrenTable = "grandchildren"
)

// testRecord is the record of every test kind: a parent, a child under a parent, or a grandchild under a
// child.
type testRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
	Status   string `json:"status,omitempty"` // only the tests of restore rules (011) give one
	record.Meta
}

// newRecord returns a record that Priya created when the test started.
func newRecord(id, name, parentID string) testRecord {
	return testRecord{ID: id, Name: name, ParentID: parentID, Meta: record.NewMeta(priya, start)}
}

// standardKinds returns the kinds parent, child and grandchild, parents first, for a test to adjust and
// register.
func (f *fixture) standardKinds() []Kind {
	return []Kind{
		f.testKind("parent", "parents", "", parentsTable),
		f.testKind("child", "children", "parent", childrenTable),
		f.testKind("grandchild", "grandchildren", "child", grandchildrenTable),
	}
}

// testKind returns a kind whose records are testRecords kept in table; its singular is its name. Remove
// notes every id in f.removed, so tests can check the order.
func (f *fixture) testKind(name, plural, parent, table string) Kind {
	k := Kind{
		Name: name, Singular: name, Plural: plural, Parent: parent,
		Snapshot: func(ctx context.Context, id string) (Snapshot, bool, error) {
			r, found, err := store.Get[testRecord](ctx, table, id)
			if err != nil || !found {
				return Snapshot{}, false, err
			}
			data, err := json.Marshal(r)
			return Snapshot{ID: r.ID, Name: r.Name, ParentID: r.ParentID, Data: data}, true, err
		},
		Remove: func(ctx context.Context, id string) error {
			f.removed = append(f.removed, id)
			return store.Delete(ctx, table, id)
		},
		Put: func(ctx context.Context, s Snapshot) error {
			var r testRecord
			if err := json.Unmarshal(s.Data, &r); err != nil {
				return err
			}
			return store.Put(ctx, table, r.ID, r)
		},
	}
	if parent != "" {
		k.ChildIDs = func(ctx context.Context, parentID string) ([]string, error) {
			records, err := store.List[testRecord](ctx, table)
			if err != nil {
				return nil, err
			}
			var childIDs []string
			for _, r := range slices.Backward(records) { // backwards, so the test sees history sort them
				if r.ParentID == parentID {
					childIDs = append(childIDs, r.ID)
				}
			}
			return childIDs, nil
		}
	}
	return k
}

// describeOwner is a Describe hook that names the person in the field "owner", as a kind holding people's
// ids would. It leaves every other field to the defaults.
func describeOwner(_ context.Context, field string, value any) (string, string, error) {
	if field != "owner" {
		return "", "", nil
	}
	names := map[string]string{"U-1002": "Priya Raman", "U-1003": "Daniel Okoro"}
	id, _ := value.(string)
	return "Owner", names[id], nil
}

// newStatusFixture returns a fixture with the tree of addTree in which C-1, G-1 and G-2 are approved, and
// restore rules like those of the planning kinds (004–008): a child or grandchild restored from the bin
// comes back as created (with a note), a child whose name is taken gets " (restored)", and a grandchild
// restored under a stored, approved child sends that child back to created and logs it, as 007's
// AfterRestore does. An Undo keeps everything as it was.
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
			taken, err := nameTaken(ctx, r)
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

// nameTaken reports whether another stored child of the same parent has the name of r.
func nameTaken(ctx context.Context, r testRecord) (bool, error) {
	children, err := store.List[testRecord](ctx, childrenTable)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(children, func(c testRecord) bool {
		return c.ID != r.ID && c.ParentID == r.ParentID && c.Name == r.Name
	}), nil
}

// sendParentBack sends the stored child parentID back to created when it is approved, and logs that
// through history, as 007's AfterRestore does for an activity whose line item comes back.
func (f *fixture) sendParentBack(ctx context.Context, rc RestoreContext, parentID string) error {
	parent, found, err := store.Get[testRecord](ctx, childrenTable, parentID)
	if err != nil || !found || parent.Status != "approved" {
		return err
	}
	parent.Status = "created"
	parent.Touch(rc.Actor, rc.Now)
	if err := store.Put(ctx, childrenTable, parent.ID, parent); err != nil {
		return err
	}
	_, err = f.service.Log(ctx, rc.Actor, LogInput{Kind: "child", RecordID: parent.ID, RecordName: parent.Name,
		Action: ActionStatusChanged, Verb: "sent back to created",
		Changes: []Change{{Field: "status", Old: raw(`"approved"`), New: raw(`"created"`)}}})
	return err
}

// withKind returns a second service on the fixture's data that also knows a kind called name under the
// kind "parent": the entries it logs are about a kind the fixture's service does not know.
func (f *fixture) withKind(t *testing.T, name string) *Service {
	t.Helper()
	other := NewService(f.db, NewJSONStore(), f.service.ids, f.clock.Now)
	for _, k := range append(f.standardKinds(), f.testKind(name, name+"s", "parent", "others")) {
		if err := other.Register(k); err != nil {
			t.Fatalf("Register(%s): %v", k.Name, err)
		}
	}
	return other
}
