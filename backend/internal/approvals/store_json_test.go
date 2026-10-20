package approvals_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"northstar/internal/approvals"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
)

var (
	seededAt = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	daniel   = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	priya    = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
)

// Workflows, rounds, history entries and notifications are saved in their own tables and read back exactly;
// an activity's rounds come by number, its entries oldest first, a person's notifications newest first
// (data model §1–§4).
func TestJSONRepository_RoundTrip(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	repo := approvals.NewJSONRepository()
	meta := record.NewMeta(actor.System, seededAt)
	workflow := approvals.Workflow{
		ID: "WF-1001", Name: "Lubricants EU approval", Description: "Lubricants Europe activities.", Active: true,
		Levels: []approvals.Level{{Name: "", ApproverType: approvals.SourcePeople, ApproverIDs: []string{"U-1003"}, Mode: approvals.ModeAny}},
		Meta:   meta,
	}
	ended := seededAt.Add(-8 * 24 * time.Hour)
	first := approvals.Round{
		ID: "AR-1001", ActivityID: "A-1004", Number: 1, WorkflowID: "WF-1001", WorkflowName: "Lubricants EU approval",
		Levels: []approvals.RoundLevel{{Number: 1, Approvers: []actor.Actor{daniel}, ApproverType: approvals.SourcePeople,
			Mode: approvals.ModeAny, Resolution: approvals.ResolutionDecide, Excluded: []approvals.Excluded{}}},
		CurrentLevel: 1, Outcome: approvals.OutcomeDeclined, SubmittedAt: seededAt.Add(-9 * 24 * time.Hour), SubmittedBy: priya,
		EndedAt: &ended, Meta: meta,
	}
	second := first
	second.ID, second.Number, second.Outcome, second.EndedAt = "AR-1002", 2, approvals.OutcomeOpen, nil
	actions := []approvals.Action{
		{ID: "AH-1001", RoundID: "AR-1001", RoundNumber: 1, ActivityID: "A-1004", Action: approvals.ActionSubmitted, Level: 1,
			Approvers: []actor.Actor{daniel}, Actor: priya, OnBehalfOf: []actor.Actor{}, At: first.SubmittedAt, Changes: []approvals.ChangeText{}},
		{ID: "AH-1002", RoundID: "AR-1001", RoundNumber: 1, ActivityID: "A-1004", Action: approvals.ActionDeclined, Level: 1,
			Approvers: []actor.Actor{daniel}, Actor: daniel, OnBehalfOf: []actor.Actor{}, At: ended,
			Comment: "Please split the agency fee by country before re-submitting.", Changes: []approvals.ChangeText{}},
	}
	older := approvals.Notification{
		ID: "NT-1001", RecipientID: "U-1002", Kind: approvals.NotifyDeclined, Title: "Activity declined", ActivityID: "A-1004",
		ActivityName: "Trade magazine ads", Actor: daniel, At: ended, Read: true, ReadAt: &ended,
	}
	newer := approvals.Notification{
		ID: "NT-1002", RecipientID: "U-1002", Kind: approvals.NotifyApproved, Title: "Activity approved", ActivityID: "A-1002",
		ActivityName: "Social always-on", Actor: daniel, At: seededAt,
	}
	sameTime := newer
	sameTime.ID = "NT-1003"
	forDaniel := approvals.Notification{ID: "NT-1004", RecipientID: "U-1003", Kind: approvals.NotifyApprovalNeeded, At: seededAt}

	err = db.Write(t.Context(), func(ctx context.Context) error {
		for _, err := range []error{
			repo.PutWorkflow(ctx, workflow), repo.PutRound(ctx, second), repo.PutRound(ctx, first),
			repo.PutAction(ctx, actions[0]), repo.PutAction(ctx, actions[1]),
			repo.PutNotification(ctx, older), repo.PutNotification(ctx, newer), repo.PutNotification(ctx, sameTime), repo.PutNotification(ctx, forDaniel),
		} {
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = db.Read(t.Context(), func(ctx context.Context) error {
		if got, found, err := repo.GetWorkflow(ctx, "WF-1001"); err != nil || !found || !reflect.DeepEqual(got, workflow) {
			t.Errorf("GetWorkflow = %+v, %v, %v", got, found, err)
		}
		if got, found, err := repo.GetRound(ctx, "AR-1001"); err != nil || !found || !reflect.DeepEqual(got, first) {
			t.Errorf("GetRound = %+v, %v, %v", got, found, err)
		}
		rounds, err := repo.RoundsOf(ctx, "A-1004")
		if err != nil || len(rounds) != 2 || rounds[0].Number != 1 || rounds[1].Number != 2 {
			t.Errorf("RoundsOf = %+v, %v", rounds, err)
		}
		if got, err := repo.ActionsOf(ctx, "A-1004"); err != nil || !reflect.DeepEqual(got, actions) {
			t.Errorf("ActionsOf = %+v, %v", got, err)
		}
		mine, err := repo.NotificationsOf(ctx, "U-1002")
		if err != nil || len(mine) != 3 || mine[0].ID != "NT-1003" || mine[1].ID != "NT-1002" || mine[2].ID != "NT-1001" {
			t.Errorf("NotificationsOf = %+v, %v; want NT-1003, NT-1002, NT-1001", mine, err)
		}
		if got, found, err := repo.GetNotification(ctx, "NT-1001"); err != nil || !found || !reflect.DeepEqual(got, older) {
			t.Errorf("GetNotification = %+v, %v, %v", got, found, err)
		}
		if none, err := repo.RoundsOf(ctx, "A-1001"); err != nil || none == nil || len(none) != 0 {
			t.Errorf("RoundsOf(no rounds) = %#v, %v; want an empty list", none, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	err = db.Write(t.Context(), func(ctx context.Context) error { return repo.DeleteWorkflow(ctx, "WF-1001") })
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = db.Read(t.Context(), func(ctx context.Context) error {
		if all, _ := repo.ListWorkflows(ctx); len(all) != 0 {
			t.Errorf("workflows after the delete = %+v", all)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}
