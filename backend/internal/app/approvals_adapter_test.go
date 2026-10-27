package app

import (
	"context"
	"reflect"
	"testing"
	"time"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/approval/approvaltest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/record"
)

// Feature 006 sees the stored approval workflows — inactive ones too, with their active flag and levels —
// sorted by name, so the activity form offers only the active ones and labels the others (FR-007).
func TestAdapter_WorkflowFacts_FR007(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	repo := approvaltest.NewStore(db)
	meta := record.NewMeta(actor.System, time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC))
	err := db.Write(t.Context(), func(ctx context.Context) error {
		for _, w := range []approval.Workflow{
			{ID: "WF-1001", Name: "Lubricants EU approval", Active: false, Levels: []approval.Level{{ApproverIDs: []string{"U-1003"}}}, Meta: meta},
			{ID: "WF-1002", Name: "China BC approval workflow", Active: true,
				Levels: []approval.Level{{Name: "Region", ApproverIDs: []string{"U-1005"}}, {ApproverIDs: []string{"U-1001"}}}, Meta: meta},
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
	var one activity.WorkflowFacts
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
	want := []activity.WorkflowFacts{
		{ID: "WF-1002", Name: "China BC approval workflow", Active: true, Levels: []activity.WorkflowLevelFacts{
			{Name: "Region", ApproverIDs: []string{"U-1005"}}, {ApproverIDs: []string{"U-1001"}},
		}},
		{ID: "WF-1001", Name: "Lubricants EU approval", Active: false, Levels: []activity.WorkflowLevelFacts{{ApproverIDs: []string{"U-1003"}}}},
	}
	if !reflect.DeepEqual(list, want) {
		t.Errorf("list =\n%+v\nwant\n%+v", list, want)
	}
	if !found || !reflect.DeepEqual(one, want[1]) || missing {
		t.Errorf("GetWorkflow = %+v %v, unknown found %v; want the inactive WF-1001 and no unknown one", one, found, missing)
	}
}

// Deleting a Submitted activity leaves its open round stored exactly as it was, so that an Undo brings the
// activity back Submitted with the same round (008-P3, D-031); the round only ends when a restore brings
// the activity back Created (FR-023).
func TestAdapter_ActivityDeletingKeepsRound_008P3(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	repo := approvaltest.NewStore(db)
	at := time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	priya := actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	open := approval.Round{
		ID: "AR-1001", ActivityID: "A-1004", Number: 1, WorkflowID: "WF-1001", WorkflowName: "Lubricants EU approval",
		Levels: []approval.RoundLevel{{Number: 1, Approvers: []actor.Actor{{ID: "U-1003", Name: "Daniel Okoro"}},
			ApproverType: approval.SourcePeople, Mode: approval.ModeAny, Resolution: approval.ResolutionDecide, Excluded: []approval.Excluded{}}},
		CurrentLevel: 1, Outcome: approval.OutcomeOpen, SubmittedAt: at, SubmittedBy: priya, Meta: record.NewMeta(priya, at),
	}
	hooks := activityApprovalHooks{} // the delete needs nothing of the approvals service

	var after approval.Round
	err := db.Write(t.Context(), func(ctx context.Context) error {
		if err := repo.PutRound(ctx, open); err != nil {
			return err
		}
		if err := hooks.ActivityDeleting(ctx, priya, activity.Activity{ID: "A-1004", Status: activity.StatusSubmitted}); err != nil {
			return err
		}
		var err error
		after, _, err = repo.GetRound(ctx, open.ID)
		return err
	})

	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if !reflect.DeepEqual(after, open) {
		t.Errorf("round after the delete =\n%+v\nwant it unchanged\n%+v", after, open)
	}
}
