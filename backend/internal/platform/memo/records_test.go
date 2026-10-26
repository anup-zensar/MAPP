package memo_test

import (
	"context"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/memo"
)

type child struct{ id, parent string }

var stored = []child{{"A-1001", "I-1"}, {"A-999", "I-2"}, {"A-1002", "I-1"}}

type reads struct{ byID, byParent [][]string }

func (r *reads) records() memo.Records[child] {
	return memo.Records[child]{
		Area: "activities", ID: func(c child) string { return c.id }, Parent: func(c child) string { return c.parent },
		Read: func(_ context.Context, ids, parentIDs []string) ([]child, error) {
			var found []child
			for _, c := range stored {
				if slices.Contains(ids, c.id) || slices.Contains(parentIDs, c.parent) {
					found = append(found, c)
				}
			}
			if ids != nil {
				r.byID = append(r.byID, ids)
			} else {
				r.byParent = append(r.byParent, parentIDs)
			}
			return found, nil
		},
	}
}

// Records answer in table order — shorter ids first — leave out ids that are not stored, and read each
// parent's children and each record once per transaction, the records of a parent included (028 FR-002).
func TestRecords_TableOrderAndOneReadEach_028_FR002(t *testing.T) {
	t.Parallel()
	ctx := memo.With(t.Context())
	r := &reads{}

	children, err := r.records().ByParent(ctx, []string{"I-2", "I-1", "I-3"})
	if err != nil {
		t.Fatalf("ByParent: %v", err)
	}
	again, _ := r.records().ByParent(ctx, []string{"I-1"})
	byID, _ := r.records().ByID(ctx, []string{"A-1002", "A-999", "A-404", "A-1002"})

	if diff := cmp.Diff([]child{{"A-999", "I-2"}, {"A-1001", "I-1"}, {"A-1002", "I-1"}}, children, cmp.AllowUnexported(child{})); diff != "" {
		t.Errorf("ByParent (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]child{{"A-1001", "I-1"}, {"A-1002", "I-1"}}, again, cmp.AllowUnexported(child{})); diff != "" {
		t.Errorf("ByParent again (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]child{{"A-999", "I-2"}, {"A-1002", "I-1"}}, byID, cmp.AllowUnexported(child{})); diff != "" {
		t.Errorf("ByID (-want +got):\n%s", diff)
	}
	want := reads{byParent: [][]string{{"I-2", "I-1", "I-3"}}, byID: [][]string{{"A-404"}}}
	if diff := cmp.Diff(want, *r, cmp.AllowUnexported(reads{})); diff != "" {
		t.Errorf("reads (-want +got):\n%s", diff)
	}
}

// A change forgets the area, so the next read sees it (028 FR-006).
func TestRecords_ReadAgainAfterForget_028_FR006(t *testing.T) {
	t.Parallel()
	ctx := memo.With(t.Context())
	r := &reads{}
	_, _ = r.records().ByID(ctx, []string{"A-1001"})

	memo.Forget(ctx, "activities")
	_, _ = r.records().ByID(ctx, []string{"A-1001"})

	if len(r.byID) != 2 {
		t.Errorf("reads by id = %v, want two", r.byID)
	}
}
