package history

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// A record and everything under it are read, removed and put back one level of the tree at a time, each
// kind's records together where the kind can, so deleting or restoring a plan costs a few statements per
// level whatever its size.

type child struct {
	parent   string
	snapshot Snapshot
}

type tree map[recordKey]map[string][]Snapshot

// collect returns the snapshots of the record id of kind k and of everything under it: the record first,
// then each child followed by everything under that child, so parents always come before their children.
// Child kinds come in the order they were registered and the records of each kind in the order of their
// ids, so the result never depends on the order in which a repository lists records.
func (s *Service) collect(ctx context.Context, k Kind, id string) ([]Snapshot, error) {
	read := s.newReadSnapshots()
	top, found, err := read.get(ctx, k, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("history: there is no %s %s", k.Singular, id)
	}
	top.Kind = k.Name
	below := tree{}
	for level := []Snapshot{top}; len(level) > 0; {
		if level, err = s.readLevelBelow(ctx, read, level, below); err != nil {
			return nil, err
		}
	}
	return s.inOrder(below, top), nil
}

func (s *Service) readLevelBelow(ctx context.Context, read *readSnapshots, level []Snapshot, below tree) ([]Snapshot, error) {
	var next []Snapshot
	for _, kindName := range kindsOf(level) {
		parents := idsOfKind(level, kindName)
		for _, childKind := range s.childKinds(kindName) {
			children, err := s.childrenOf(ctx, read, childKind, parents)
			if err != nil {
				return nil, err
			}
			for _, c := range children {
				parent := recordKey{kindName, c.parent}
				if below[parent] == nil {
					below[parent] = map[string][]Snapshot{}
				}
				below[parent][childKind.Name] = append(below[parent][childKind.Name], c.snapshot)
				next = append(next, c.snapshot)
			}
		}
	}
	return next, nil
}

func (s *Service) childrenOf(ctx context.Context, read *readSnapshots, k Kind, parents []string) ([]child, error) {
	if k.AllChildIDs == nil {
		var children []child
		for _, parent := range parents {
			ids, err := k.ChildIDs(ctx, parent)
			if err != nil {
				return nil, err
			}
			found, err := s.snapshotsOfKind(ctx, read, k, ids)
			if err != nil {
				return nil, err
			}
			for _, snapshot := range found {
				children = append(children, child{parent: parent, snapshot: snapshot})
			}
		}
		return children, nil
	}
	ids, err := k.AllChildIDs(ctx, parents)
	if err != nil {
		return nil, err
	}
	found, err := s.snapshotsOfKind(ctx, read, k, ids)
	children := make([]child, 0, len(found))
	for _, snapshot := range found {
		children = append(children, child{parent: snapshot.ParentID, snapshot: snapshot})
	}
	return children, err
}

func (s *Service) snapshotsOfKind(ctx context.Context, read *readSnapshots, k Kind, ids []string) ([]Snapshot, error) {
	if _, err := read.load(ctx, keysOf(k.Name, ids)); err != nil {
		return nil, err
	}
	found := make([]Snapshot, 0, len(ids))
	for _, id := range ids {
		snapshot, stored, err := read.get(ctx, k, id)
		if err != nil {
			return nil, err
		}
		if !stored {
			return nil, fmt.Errorf("history: there is no %s %s", k.Singular, id)
		}
		snapshot.Kind = k.Name
		found = append(found, snapshot)
	}
	return found, nil
}

// inOrder returns node, then each child kind in turn, its records by id, each followed by what is under it.
func (s *Service) inOrder(below tree, node Snapshot) []Snapshot {
	list := []Snapshot{node}
	children := below[recordKey{node.Kind, node.ID}]
	for _, childKind := range s.childKinds(node.Kind) {
		kids := slices.Clone(children[childKind.Name])
		slices.SortFunc(kids, func(x, y Snapshot) int { return strings.Compare(x.ID, y.ID) })
		for _, kid := range kids {
			list = append(list, s.inOrder(below, kid)...)
		}
	}
	return list
}

// remove deletes children first: backwards one at a time, or kind by kind once a kind removes many at once.
func (s *Service) remove(ctx context.Context, snapshots []Snapshot) error {
	groups := groupsByKind(snapshots)
	if !s.anyRemovesAll(groups) {
		for _, snapshot := range slices.Backward(snapshots) {
			if err := s.removeOne(ctx, snapshot); err != nil {
				return err
			}
		}
		return nil
	}
	for _, g := range slices.Backward(groups) {
		k, err := s.registeredKind(g.kind)
		if err != nil {
			return err
		}
		ids := make([]string, 0, len(g.positions))
		for _, i := range slices.Backward(g.positions) {
			ids = append(ids, snapshots[i].ID)
		}
		if err := removeIDs(ctx, k, ids); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) anyRemovesAll(groups []kindGroup) bool {
	return slices.ContainsFunc(groups, func(g kindGroup) bool {
		k, found := s.findKind(g.kind)
		return found && k.RemoveAll != nil
	})
}

func (s *Service) removeOne(ctx context.Context, snapshot Snapshot) error {
	k, err := s.registeredKind(snapshot.Kind)
	if err != nil {
		return err
	}
	return k.Remove(ctx, snapshot.ID)
}

func removeIDs(ctx context.Context, k Kind, ids []string) error {
	if k.RemoveAll != nil {
		return k.RemoveAll(ctx, ids)
	}
	for _, id := range ids {
		if err := k.Remove(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// restoring is one restore's records, which PrepareRestore may change, and its notes by position.
type restoring struct {
	rc        RestoreContext
	snapshots []Snapshot
	notes     [][]string
}

// prepare runs k's PrepareRestore on its records at positions. Their stored siblings are read at once first,
// which each PrepareRestore then finds kept; the records come back together afterwards, as names unique
// among siblings when deleted cannot clash with each other.
func (r *restoring) prepare(ctx context.Context, k Kind, positions []int) error {
	if k.PrepareRestore == nil {
		return nil
	}
	if k.AllChildIDs != nil {
		var parents []string
		for _, i := range positions {
			if !slices.Contains(parents, r.snapshots[i].ParentID) {
				parents = append(parents, r.snapshots[i].ParentID)
			}
		}
		if _, err := k.AllChildIDs(ctx, parents); err != nil {
			return err
		}
	}
	for _, i := range positions {
		notes, err := k.PrepareRestore(ctx, r.rc, &r.snapshots[i])
		if err != nil {
			return err
		}
		r.notes[i] = notes
	}
	return nil
}

func (r *restoring) put(ctx context.Context, k Kind, positions []int) error {
	if k.PutAll != nil {
		group := make([]Snapshot, 0, len(positions))
		for _, i := range positions {
			group = append(group, r.snapshots[i])
		}
		return k.PutAll(ctx, group)
	}
	for _, i := range positions {
		if err := k.Put(ctx, r.snapshots[i]); err != nil {
			return err
		}
	}
	return nil
}

type kindGroup struct {
	kind      string
	positions []int
}

// groupsByKind groups snapshots by kind, kinds in the order they first appear: parents' kinds first.
func groupsByKind(snapshots []Snapshot) []kindGroup {
	var groups []kindGroup
	for i, snapshot := range snapshots {
		g := slices.IndexFunc(groups, func(g kindGroup) bool { return g.kind == snapshot.Kind })
		if g < 0 {
			groups = append(groups, kindGroup{kind: snapshot.Kind})
			g = len(groups) - 1
		}
		groups[g].positions = append(groups[g].positions, i)
	}
	return groups
}

func kindsOf(level []Snapshot) []string {
	var kinds []string
	for _, snapshot := range level {
		if !slices.Contains(kinds, snapshot.Kind) {
			kinds = append(kinds, snapshot.Kind)
		}
	}
	return kinds
}

func idsOfKind(level []Snapshot, kindName string) []string {
	var ids []string
	for _, snapshot := range level {
		if snapshot.Kind == kindName {
			ids = append(ids, snapshot.ID)
		}
	}
	return ids
}

func keysOf(kindName string, ids []string) []recordKey {
	keys := make([]recordKey, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, recordKey{kindName, id})
	}
	return keys
}
