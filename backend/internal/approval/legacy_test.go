package approval

import (
	"context"
	"reflect"
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/record"
)

// Records saved without the fields of feature 017 read with its defaults, as the PostgreSQL store saves them.
func TestLegacyRecords_ReadWithDefaults_017P19(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	repo := newFakeStore(db)
	meta := record.NewMeta(actor.System, start)
	err := db.Write(t.Context(), func(ctx context.Context) error {
		if err := repo.PutWorkflow(ctx, Workflow{ID: "WF-1001", Name: "Lubricants EU approval", Active: true,
			Levels: []Level{{ApproverIDs: []string{"U-1003"}}}, Meta: meta}); err != nil {
			return err
		}
		return repo.PutRound(ctx, Round{ID: "AR-1001", SubjectKind: KindActivity, SubjectID: "A-1004", Number: 1, WorkflowID: "WF-1001",
			WorkflowName: "Lubricants EU approval", Levels: []RoundLevel{{Number: 1, Approvers: []actor.Actor{daniel}}},
			CurrentLevel: 1, Outcome: OutcomeOpen, SubmittedAt: start, SubmittedBy: priya, Meta: meta})
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	w, _, err := repo.GetWorkflow(t.Context(), "WF-1001")
	if err != nil {
		t.Fatalf("GetWorkflow: %v", err)
	}
	checkLegacyWorkflow(t, w, meta)
	r, _, err := repo.GetRound(t.Context(), "AR-1001")
	if err != nil {
		t.Fatalf("GetRound: %v", err)
	}
	checkLegacyRound(t, r)
}

func checkLegacyWorkflow(t *testing.T, w Workflow, meta record.Meta) {
	t.Helper()
	wantLevel := Level{ApproverType: SourcePeople, ApproverIDs: []string{"U-1003"}, Mode: ModeAny}
	if !reflect.DeepEqual(w.Levels, []Level{wantLevel}) || w.ReminderDays != nil || w.EscalationDays != nil || w.Meta != meta {
		t.Errorf("workflow = %+v, want the 017 defaults", w)
	}
}

func checkLegacyRound(t *testing.T, r Round) {
	t.Helper()
	wantRoundLevel := RoundLevel{Number: 1, Approvers: []actor.Actor{daniel}, ApproverType: SourcePeople,
		Mode: ModeAny, Resolution: ResolutionDecide, Excluded: []Excluded{}}
	if !reflect.DeepEqual(r.Levels, []RoundLevel{wantRoundLevel}) || r.Facts != nil || r.ReminderDays != nil || r.AutoApproved {
		t.Errorf("round = %+v, want the 017 defaults", r)
	}
}
