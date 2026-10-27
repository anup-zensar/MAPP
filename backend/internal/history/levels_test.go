package history

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
)

// batched gives k the functions for many records at once, built on its own, and notes every call in calls.
func batched(k Kind, calls *[]string) Kind {
	if k.ChildIDs != nil {
		k.AllChildIDs = func(ctx context.Context, parentIDs []string) ([]string, error) {
			*calls = append(*calls, "children of "+fmt.Sprint(parentIDs))
			var all []string
			for _, parent := range parentIDs {
				ids, err := k.ChildIDs(ctx, parent)
				if err != nil {
					return nil, err
				}
				all = append(all, ids...)
			}
			return all, nil
		}
	}
	k.SnapshotsOf = func(ctx context.Context, ids []string) ([]Snapshot, error) {
		var found []Snapshot
		for _, id := range ids {
			if s, stored, err := k.Snapshot(ctx, id); err != nil || stored {
				found = append(found, s)
			}
		}
		return found, nil
	}
	k.RemoveAll = func(ctx context.Context, ids []string) error {
		*calls = append(*calls, "remove "+fmt.Sprint(ids))
		return removeAll(ctx, k, ids)
	}
	k.PutAll = func(ctx context.Context, snapshots []Snapshot) error {
		*calls = append(*calls, fmt.Sprintf("put %d %s", len(snapshots), k.Name))
		for _, s := range snapshots {
			if err := k.Put(ctx, s); err != nil {
				return err
			}
		}
		return nil
	}
	return k
}

func removeAll(ctx context.Context, k Kind, ids []string) error {
	for _, id := range ids {
		if err := k.Remove(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// noteEach gives k a PrepareRestore that notes the record it prepares, so the order of the notes shows.
func noteEach(k Kind) Kind {
	k.PrepareRestore = func(_ context.Context, _ RestoreContext, s *Snapshot) ([]string, error) {
		return []string{s.ID + "."}, nil
	}
	return k
}

// Kinds that read, remove and put back many records at once keep a delete and its restore exactly as one
// record at a time does: the same snapshots in tree order, the same records back, the notes in tree order;
// each level costs one call per kind (028 FR-004).
func TestMoveToBinAndRestore_ManyAtOnceAsOneByOne_028_FR004(t *testing.T) {
	t.Parallel()
	var calls []string
	one, many := newBareFixture(t), newBareFixture(t)
	for _, k := range one.standardKinds() {
		one.register(t, noteEach(k))
	}
	for _, k := range many.standardKinds() {
		many.register(t, batched(noteEach(k), &calls))
	}
	results := make([]RestoreResult, 0, 2)
	deleted := make([][]Snapshot, 0, 2)
	for _, f := range []*fixture{one, many} {
		f.addTree(t)
		before := f.allRecords(t)
		entry := f.moveToBin(t, "parent", "P-1")
		deleted = append(deleted, entry.Snapshots)
		f.clock.Advance(time.Hour)
		restored, err := f.service.Restore(t.Context(), priya, entry.ID)
		if err != nil {
			t.Fatalf("Restore: %v", err)
		}
		if after := f.allRecords(t); !reflect.DeepEqual(after, before) {
			t.Errorf("records after the restore = %+v, want %+v", after, before)
		}
		results = append(results, restored)
	}

	if !reflect.DeepEqual(deleted[0], deleted[1]) || results[0] != results[1] {
		t.Errorf("one by one: %s, %+v; many at once: %s, %+v; want the same", asJSON(deleted[0]), results[0],
			asJSON(deleted[1]), results[1])
	}
	wantCalls := []string{
		"children of [P-1]", "children of [C-2 C-1]",
		"remove [G-3 G-2 G-1]", "remove [C-2 C-1]", "remove [P-1]",
		"put 1 parent", "children of [P-1]", "put 2 child", "children of [C-1 C-2]", "put 3 grandchild",
	}
	if !slices.Equal(calls, wantCalls) {
		t.Errorf("calls = %q, want %q", calls, wantCalls)
	}
}

// A page of the log reads where its records are now kind by kind and gives the same answer as reading them
// one at a time: stored with their current place, or in the recycle bin (028 FR-002).
func TestQueryLog_RecordsNowManyAtOnceAsOneByOne_028_FR002(t *testing.T) {
	t.Parallel()
	var calls []string
	one, many := newBareFixture(t), newBareFixture(t)
	one.register(t, one.standardKinds()...)
	for _, k := range many.standardKinds() {
		many.register(t, batched(k, &calls))
	}
	pages := make([]LogPage, 0, 2)
	for _, f := range []*fixture{one, many} {
		f.addTree(t)
		for _, in := range []LogInput{
			{Kind: "grandchild", RecordID: "G-1", RecordName: "Grandchild one", Action: ActionChanged},
			{Kind: "grandchild", RecordID: "G-3", RecordName: "Grandchild three", Action: ActionChanged},
			{Kind: "child", RecordID: "C-2", RecordName: "Child two", Action: ActionChanged},
		} {
			f.log(t, in)
		}
		f.moveToBin(t, "child", "C-2")
		page, err := f.service.QueryLog(t.Context(), priya, LogQuery{Limit: 50})
		if err != nil {
			t.Fatalf("QueryLog: %v", err)
		}
		pages = append(pages, page)
	}
	if !reflect.DeepEqual(pages[0], pages[1]) {
		t.Errorf("one by one:\n%s\nmany at once:\n%s", asJSON(pages[0]), asJSON(pages[1]))
	}
}
