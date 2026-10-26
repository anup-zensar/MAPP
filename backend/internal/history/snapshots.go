package history

import (
	"context"
	"fmt"
	"maps"
	"slices"
)

// readSnapshots keeps the records one action reads, by kind and id: a page of log entries, the recycle bin
// or a delete reads each kind's records at once (Kind.SnapshotsOf) and every record above them once.
type readSnapshots struct {
	s    *Service
	kept map[recordKey]*Snapshot // nil: read, and not stored
}

func (s *Service) newReadSnapshots() *readSnapshots {
	return &readSnapshots{s: s, kept: map[recordKey]*Snapshot{}}
}

func (r *readSnapshots) loadWithAncestors(ctx context.Context, keys []recordKey) error {
	for len(keys) > 0 {
		read, err := r.load(ctx, keys)
		if err != nil {
			return err
		}
		keys = keys[:0:0]
		for _, snapshot := range read {
			if k, found := r.s.findKind(snapshot.Kind); found && k.Parent != "" {
				keys = append(keys, recordKey{k.Parent, snapshot.ParentID})
			}
		}
	}
	return nil
}

// load reads the records of keys not read yet, one call per kind, and returns those found with their Kind.
func (r *readSnapshots) load(ctx context.Context, keys []recordKey) ([]Snapshot, error) {
	var found []Snapshot
	missing := r.missingByKind(keys)
	for _, kindName := range slices.Sorted(maps.Keys(missing)) {
		k, known := r.s.findKind(kindName)
		if !known {
			continue
		}
		read, err := r.readKind(ctx, k, missing[kindName])
		if err != nil {
			return nil, err
		}
		for _, snapshot := range read {
			snapshot.Kind = k.Name
			found = append(found, snapshot)
		}
	}
	return found, nil
}

func (r *readSnapshots) missingByKind(keys []recordKey) map[string][]string {
	missing := map[string][]string{}
	asked := map[recordKey]bool{}
	for _, key := range keys {
		if _, read := r.kept[key]; !read && !asked[key] {
			asked[key] = true
			missing[key.kind] = append(missing[key.kind], key.id)
		}
	}
	return missing
}

func (r *readSnapshots) readKind(ctx context.Context, k Kind, ids []string) ([]Snapshot, error) {
	var read []Snapshot
	if k.SnapshotsOf != nil {
		found, err := k.SnapshotsOf(ctx, ids)
		if err != nil {
			return nil, err
		}
		read = found
	} else {
		for _, id := range ids {
			snapshot, found, err := k.Snapshot(ctx, id)
			if err != nil {
				return nil, err
			}
			if found {
				read = append(read, snapshot)
			}
		}
	}
	for _, id := range ids {
		r.kept[recordKey{k.Name, id}] = nil
	}
	for i := range read {
		r.kept[recordKey{k.Name, read[i].ID}] = &read[i]
	}
	return read, nil
}

func (r *readSnapshots) get(ctx context.Context, k Kind, id string) (Snapshot, bool, error) {
	if kept, read := r.kept[recordKey{k.Name, id}]; read {
		if kept == nil {
			return Snapshot{}, false, nil
		}
		return *kept, true, nil
	}
	read, err := r.readKind(ctx, k, []string{id})
	if err != nil || len(read) == 0 {
		return Snapshot{}, false, err
	}
	return read[0], true, nil
}

func (r *readSnapshots) isStored(ctx context.Context, ref RecordRef) (bool, error) {
	k, found := r.s.findKind(ref.Type)
	if !found {
		return false, nil
	}
	_, stored, err := r.get(ctx, k, ref.ID)
	return stored, err
}

// ancestors is Service.ancestors through the records read: the path down to the record id, top level first.
func (r *readSnapshots) ancestors(ctx context.Context, kindName, id string) ([]RecordRef, error) {
	path := []RecordRef{}
	for kindName != "" {
		k, err := r.s.registeredKind(kindName)
		if err != nil {
			return nil, err
		}
		snapshot, found, err := r.get(ctx, k, id)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("history: the %s %s is not stored", k.Singular, id)
		}
		path = append(path, RecordRef{Type: k.Name, ID: id, Name: snapshot.Name})
		kindName, id = k.Parent, snapshot.ParentID
	}
	slices.Reverse(path)
	return path, nil
}
