package app

import (
	"context"
	"reflect"
	"testing"
	"time"

	"northstar/internal/activities"
	"northstar/internal/approvals"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
)

// Feature 006 sees the stored approval workflows — inactive ones too, with their active flag and levels —
// sorted by name, so the activity form offers only the active ones and labels the others (FR-007).
func TestAdapter_WorkflowFacts_FR007(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	repo := approvals.NewJSONRepository()
	meta := record.NewMeta(actor.System, time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC))
	err = db.Write(t.Context(), func(ctx context.Context) error {
		for _, w := range []approvals.Workflow{
			{ID: "WF-1001", Name: "Lubricants EU approval", Active: false, Levels: []approvals.Level{{ApproverIDs: []string{"U-1003"}}}, Meta: meta},
			{ID: "WF-1002", Name: "China BC approval workflow", Active: true,
				Levels: []approvals.Level{{Name: "Region", ApproverIDs: []string{"U-1005"}}, {ApproverIDs: []string{"U-1001"}}}, Meta: meta},
		} {
			if err := repo.PutWorkflow(ctx, w); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed workflows: %v", err)
	}
	adapter := activityWorkflows{repo: repo}

	var list []activities.WorkflowFacts
	var one activities.WorkflowFacts
	var found, missing bool
	err = db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		if list, err = adapter.ListWorkflows(ctx); err != nil {
			return err
		}
		if one, found, err = adapter.GetWorkflow(ctx, "WF-1001"); err != nil {
			return err
		}
		_, missing, err = adapter.GetWorkflow(ctx, "WF-9999")
		return err
	})

	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := []activities.WorkflowFacts{
		{ID: "WF-1002", Name: "China BC approval workflow", Active: true, Levels: []activities.WorkflowLevelFacts{
			{Name: "Region", ApproverIDs: []string{"U-1005"}}, {ApproverIDs: []string{"U-1001"}},
		}},
		{ID: "WF-1001", Name: "Lubricants EU approval", Active: false, Levels: []activities.WorkflowLevelFacts{{ApproverIDs: []string{"U-1003"}}}},
	}
	if !reflect.DeepEqual(list, want) {
		t.Errorf("list =\n%+v\nwant\n%+v", list, want)
	}
	if !found || !reflect.DeepEqual(one, want[1]) || missing {
		t.Errorf("GetWorkflow = %+v %v, unknown found %v; want the inactive WF-1001 and no unknown one", one, found, missing)
	}
}
