package app

import (
	"context"
	"testing"
	"time"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/approval/approvaltest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/record"
)

func TestActivityWorkflowsOnlyActivity(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-007 workflows of other object types are neither offered nor found", func(t *testing.T) {
		t.Parallel()
		db := fake.NewDB()
		repo := approvaltest.NewStore(db)
		meta := record.NewMeta(actor.System, time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC))
		level := []approval.Level{{ApproverIDs: []string{"U-1003"}}}
		err := db.Write(t.Context(), func(ctx context.Context) error {
			for _, w := range []approval.Workflow{
				{ID: "WF-1001", Name: "Lubricants EU approval", Active: true, Levels: level, Meta: meta},
				{ID: "WF-1004", Name: "Purchase request approval", Active: true, ObjectType: approval.KindPurchaseRequest, Levels: level, Meta: meta},
				{ID: "WF-1005", Name: "PR changes", Active: true, ObjectType: approval.KindModifiedPurchaseRequest, Levels: level, Meta: meta},
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
		var list []activity.WorkflowFacts
		var foundPR bool
		err = db.Read(t.Context(), func(ctx context.Context) (err error) {
			if list, err = adapter.ListWorkflows(ctx); err != nil {
				return err
			}
			_, foundPR, err = adapter.GetWorkflow(ctx, "WF-1004")
			return err
		})
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if len(list) != 1 || list[0].ID != "WF-1001" || foundPR {
			t.Errorf("list = %+v, purchase-request workflow found = %v; want only WF-1001", list, foundPR)
		}
	})
}
