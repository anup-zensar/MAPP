package approvals_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"mapp/internal/approvals"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

var (
	pgMeta  = record.NewMeta(actor.System, time.Date(2026, time.March, 12, 9, 0, 0, 0, time.UTC))
	pgExact = cmpopts.EquateComparable(money.Amount{})
)

// pgStore gives a store whose database already holds the approvers U-1001 and U-1002.
func pgStore(t *testing.T) (*database.DB, *approvals.PostgresStore) {
	t.Helper()
	db := dbtest.NewDB(t)
	directory := people.NewPostgresStore(db)
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(
			directory.Put(ctx, people.Person{ID: "U-1001", Name: "Priya Raman", Email: "priya@example.com", Meta: pgMeta}),
			directory.Put(ctx, people.Person{ID: "U-1002", Name: "Daniel Okoro", Email: "daniel@example.com", Meta: pgMeta}))
	})
	return db, approvals.NewPostgresStore(db)
}

func pgWrite(t *testing.T, db *database.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Write(t.Context(), fn); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
}

func pgWorkflow(id, name string, levels ...approvals.Level) approvals.Workflow {
	return approvals.Workflow{ID: id, Name: name, Description: "Approval for " + name, Active: true, Levels: levels,
		Meta: pgMeta}
}

func TestPostgresStoreWorkflowsFR008(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	threshold := money.FromCents(1000000)
	reminder, escalation := 3, 5
	twoLevels := pgWorkflow("WF-1001", "Large spend",
		approvals.Level{Name: "Finance", ApproverType: approvals.SourcePeople, ApproverIDs: []string{"U-1002", "U-1001"},
			Mode: approvals.ModeAll, BackupApproverID: "U-1001",
			AppliesWhen: &approvals.Condition{PlannedCostAtLeast: &threshold, BusinessUnitIDs: []string{"SO-1001"}}},
		approvals.Level{ApproverType: approvals.SourceManager, ApproverIDs: []string{}, Mode: approvals.ModeAny,
			AutoApproveWhen: &approvals.Condition{ActivityTypeIDs: []string{"AT-1001"}}})
	twoLevels.ReminderDays, twoLevels.EscalationDays = &reminder, &escalation
	byRole := pgWorkflow("WF-1002", "Role approval", approvals.Level{ApproverType: approvals.SourceRole,
		ApproverIDs: []string{}, RoleID: "RL-9999", Mode: approvals.ModeAny})
	pgWrite(t, db, func(ctx context.Context) error {
		return errors.Join(s.PutWorkflow(ctx, byRole), s.PutWorkflow(ctx, twoLevels), s.PutWorkflow(ctx, pgWorkflow(
			"WF-1003", "Deleted", approvals.Level{ApproverType: approvals.SourceManager, Mode: approvals.ModeAny})))
	})
	pgWrite(t, db, func(ctx context.Context) error { return s.DeleteWorkflow(ctx, "WF-1003") })

	got, err := s.ListWorkflows(t.Context())

	if err != nil {
		t.Fatalf("ListWorkflows() error: %v", err)
	}
	if diff := cmp.Diff([]approvals.Workflow{twoLevels, byRole}, got, pgExact); diff != "" {
		t.Errorf("ListWorkflows() mismatch (-want +got):\n%s", diff)
	}
	twoLevels.Levels = twoLevels.Levels[1:]
	pgWrite(t, db, func(ctx context.Context) error { return s.PutWorkflow(ctx, twoLevels) })
	if one, found, err := s.GetWorkflow(t.Context(), "WF-1001"); err != nil || !found || !cmp.Equal(twoLevels, one, pgExact) {
		t.Errorf("GetWorkflow() after replacing its levels = %+v, found %v, error %v; want %+v", one, found, err, twoLevels)
	}
}

func TestPostgresStoreWorkflowLevelsReadAsTheJSONStoreReadsThem017P19(t *testing.T) {
	t.Parallel()
	db, s := pgStore(t)
	saved := pgWorkflow("WF-1001", "Before 017", approvals.Level{Name: "Level 1", ApproverIDs: []string{"U-1001"}},
		approvals.Level{Name: "Level 2"})
	pgWrite(t, db, func(ctx context.Context) error { return s.PutWorkflow(ctx, saved) })

	got, found, err := s.GetWorkflow(t.Context(), "WF-1001")

	if err != nil || !found {
		t.Fatalf("GetWorkflow() = found %v, error %v", found, err)
	}
	want := pgWorkflow("WF-1001", "Before 017",
		approvals.Level{Name: "Level 1", ApproverType: approvals.SourcePeople, ApproverIDs: []string{"U-1001"}, Mode: approvals.ModeAny},
		approvals.Level{Name: "Level 2", ApproverType: approvals.SourcePeople, ApproverIDs: []string{}, Mode: approvals.ModeAny})
	if diff := cmp.Diff(want, got, pgExact); diff != "" {
		t.Errorf("GetWorkflow() mismatch (-want +got):\n%s", diff)
	}
	if saved.Levels[1].ApproverType != "" {
		t.Errorf("PutWorkflow() changed the caller's levels: %+v", saved.Levels)
	}
}

func TestPostgresStoreRefusesBrokenWorkflowsFR004(t *testing.T) {
	t.Parallel()
	reminder, escalation := 5, 5
	escalatesTooEarly := pgWorkflow("WF-1002", "Escalating", approvals.Level{ApproverType: approvals.SourceManager, Mode: approvals.ModeAny})
	escalatesTooEarly.ReminderDays, escalatesTooEarly.EscalationDays = &reminder, &escalation
	tests := []struct {
		name     string
		workflow approvals.Workflow
	}{
		{name: "a name that differs only in letter case", workflow: pgWorkflow("WF-1002", "LARGE SPEND",
			approvals.Level{ApproverType: approvals.SourceManager, Mode: approvals.ModeAny})},
		{name: "an escalation not later than the reminder", workflow: escalatesTooEarly},
		{name: "an approver who does not exist", workflow: pgWorkflow("WF-1002", "Unknown approver",
			approvals.Level{ApproverType: approvals.SourcePeople, ApproverIDs: []string{"U-9999"}, Mode: approvals.ModeAny})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, s := pgStore(t)
			pgWrite(t, db, func(ctx context.Context) error {
				return s.PutWorkflow(ctx, pgWorkflow("WF-1001", "Large spend",
					approvals.Level{ApproverType: approvals.SourceManager, Mode: approvals.ModeAny}))
			})

			err := db.Write(t.Context(), func(ctx context.Context) error { return s.PutWorkflow(ctx, tt.workflow) })

			if err == nil {
				t.Errorf("Write() error = nil, want the database to refuse the change")
			}
		})
	}
}
