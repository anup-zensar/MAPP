package approval

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"mapp/internal/platform/errs"
)

// days returns a pointer to n, for reminder and escalation days.
func days(n int) *int {
	return &n
}

// Saving a workflow reports every problem of its rules at once, each on its field (017 FR-008, US2 scenario
// 5, data-model §1–§2): an unknown approver source, a missing or non-approving role, an unknown mode, empty
// conditions, wrong amounts, unknown business units and types, a missing or unusable backup approver, wrong
// day counts.
func TestCreateWorkflow_RuleProblemsAtOnce_FR008(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.cannotApprove[sara.ID] = true
	f.deactivatePerson(t, mei)
	text := func(s string) *string { return &s }

	_, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{
		Name: "Rules", ReminderDays: days(61), EscalationDays: days(5),
		Levels: []LevelInput{
			{ApproverType: "boss"},
			{ApproverType: SourceRole, AppliesWhen: &ConditionInput{},
				AutoApproveWhen: &ConditionInput{PlannedCostAtLeast: text("50000.00"), PlannedCostBelow: text("20000.00")}, BackupApproverID: "U-9999"},
			{ApproverType: SourceRole, RoleID: viewerRole.ID, Mode: "some", AppliesWhen: &ConditionInput{PlannedCostAtLeast: text("1.234")},
				AutoApproveWhen: &ConditionInput{BusinessUnitIDs: []string{"SO-9999"}}, BackupApproverID: sara.ID},
			{ApproverIDs: []string{daniel.ID}, AppliesWhen: &ConditionInput{ActivityTypeIDs: []string{eventType, eventType}},
				AutoApproveWhen: &ConditionInput{PlannedCostBelow: text("-5.00")}, BackupApproverID: daniel.ID},
			{ApproverIDs: []string{sara.ID}, BackupApproverID: mei.ID},
			{ApproverType: SourceRole, RoleID: "RL-9999", BackupApproverID: alex.ID},
		},
	})

	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "reminderDays", Message: "Remind after: enter a whole number of days from 1 to 60."},
		errs.FieldError{Field: "levels.1.approverType", Message: "Level 1: choose who approves."},
		errs.FieldError{Field: "levels.1.backupApproverId", Message: "Level 1: choose a backup approver, or turn escalation off."},
		errs.FieldError{Field: "levels.2.roleId", Message: "Level 2: choose a role."},
		errs.FieldError{Field: "levels.2.appliesWhen", Message: "Level 2: 'Only needed when' needs at least one condition."},
		errs.FieldError{Field: "levels.2.autoApproveWhen", Message: "Level 2: the 'at least' amount must be lower than the 'below' amount."},
		errs.FieldError{Field: "levels.2.backupApproverId", Message: "Level 2: choose the backup approver from the list."},
		errs.FieldError{Field: "levels.3.roleId", Message: "Level 3: people with the role 'Viewer' can't approve activities. Choose a role that approves."},
		errs.FieldError{Field: "levels.3.mode", Message: "Level 3: choose 'Any one approves' or 'All must approve'."},
		errs.FieldError{Field: "levels.3.appliesWhen", Message: "Level 3: enter amounts of 0.00 or more with at most two decimals."},
		errs.FieldError{Field: "levels.3.autoApproveWhen", Message: "Level 3: choose business units from the list."},
		errs.FieldError{Field: "levels.3.backupApproverId", Message: "Level 3: Sara Novak can't approve activities. Give them a role that approves first."},
		errs.FieldError{Field: "levels.4.appliesWhen", Message: "Level 4: choose activity types from the list."},
		errs.FieldError{Field: "levels.4.autoApproveWhen", Message: "Level 4: enter amounts of 0.00 or more with at most two decimals."},
		errs.FieldError{Field: "levels.4.backupApproverId", Message: "Level 4: Daniel Okoro is already an approver of this level — choose someone else."},
		errs.FieldError{Field: "levels.5.approverIds", Message: "Sara Novak can't approve activities. Give them a role that approves first."},
		errs.FieldError{Field: "levels.5.backupApproverId", Message: "Level 5: Mei Lin is inactive — choose an active person."},
		errs.FieldError{Field: "levels.6.roleId", Message: "Level 6: choose a role from the list."},
	))
	if views := f.workflowViews(t, alex); len(views) != 2 {
		t.Errorf("%d workflows, want the 2 of the fixture: nothing is saved", len(views))
	}
}

// A workflow may remind after 1–60 days and escalate after 1–90 days, more than the reminder days; either
// can be off (null). Escalation needs a backup approver on every level (017 FR-006, FR-007). The timings and
// backups are saved, shown and changed like any field; null turns them off.
func TestWorkflow_BackupAndTimings_FR006_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{
		Name: "Timed", ReminderDays: days(3), EscalationDays: days(5),
		Levels: []LevelInput{{ApproverIDs: []string{daniel.ID}, BackupApproverID: alex.ID}, {ApproverIDs: []string{carlos.ID}, BackupApproverID: alex.ID}},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	if *view.ReminderDays != 3 || *view.EscalationDays != 5 || view.Levels[0].BackupApprover == nil ||
		view.Levels[0].BackupApprover.Name != "Alex Morgan" || len(view.Warnings) != 0 {
		t.Errorf("view = %+v, want 3 and 5 days with Alex Morgan as backup approver", view)
	}

	tests := []struct {
		name               string
		reminder, escalate *int
		want               []errs.FieldError
	}{
		{"escalation not after the reminder", days(5), days(5),
			[]errs.FieldError{{Field: "escalationDays", Message: "Escalate after must be more days than Remind after."}}},
		{"too many days", days(0), days(91), []errs.FieldError{
			{Field: "reminderDays", Message: "Remind after: enter a whole number of days from 1 to 60."},
			{Field: "escalationDays", Message: "Escalate after: enter a whole number of days from 1 to 90."}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{
				Name: "Wrong days", ReminderDays: tt.reminder, EscalationDays: tt.escalate,
				Levels: []LevelInput{{ApproverIDs: []string{daniel.ID}, BackupApproverID: alex.ID}},
			})
			assertError(t, err, errs.Validation(tt.want...))
		})
	}
	if _, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{
		Name: "Reminders only", ReminderDays: days(2), Levels: []LevelInput{{ApproverIDs: []string{daniel.ID}}},
	}); err != nil {
		t.Errorf("reminders without escalation need no backup approver: %v", err)
	}

	var update WorkflowUpdate
	if err := json.Unmarshal([]byte(`{"escalationDays": null, "version": 1}`), &update); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !update.EscalationDays.Sent || update.EscalationDays.Days != nil || update.ReminderDays.Sent {
		t.Fatalf("update = %+v, want escalation sent as null and reminders not sent", update)
	}
	off, err := f.service.UpdateWorkflow(t.Context(), alex, view.ID, update)
	if err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}
	if off.EscalationDays != nil || *off.ReminderDays != 3 || off.Version != 2 {
		t.Errorf("after turning escalation off = %+v, want no escalation, reminders kept, version 2", off)
	}
}

// What the source does not use is cleared on save (017 FR-001, FR-002, plan "Service operations"): approvers
// of line-manager and role levels, the role of other levels; a line-manager level is always "any one
// approves"; a level without a source is named people.
func TestUpdateWorkflow_Normalises_FR001_FR002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view, err := f.service.UpdateWorkflow(t.Context(), alex, chinaFlow, WorkflowUpdate{
		Levels: &[]LevelInput{
			{ApproverType: SourceManager, ApproverIDs: []string{carlos.ID}, RoleID: approverRole.ID, Mode: ModeAll},
			{ApproverType: SourceRole, ApproverIDs: []string{carlos.ID}, RoleID: approverRole.ID, Mode: ModeAll},
			{ApproverIDs: []string{alex.ID}, RoleID: approverRole.ID},
		},
		Version: version(1),
	})
	if err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}

	stored, _ := f.storedWorkflow(t, chinaFlow)
	want := []Level{
		{ApproverType: SourceManager, ApproverIDs: []string{}, Mode: ModeAny},
		{ApproverType: SourceRole, ApproverIDs: []string{}, RoleID: approverRole.ID, Mode: ModeAll},
		{ApproverType: SourcePeople, ApproverIDs: []string{alex.ID}, Mode: ModeAny},
	}
	if !reflect.DeepEqual(stored.Levels, want) {
		t.Errorf("levels =\n%+v\nwant\n%+v", stored.Levels, want)
	}
	if view.LevelsText != "Level 1 · Line manager → Level 2 · Role: Approver · all must approve → Level 3 · Alex Morgan" ||
		view.Levels[1].Role == nil || view.Levels[1].Role.Name != "Approver" || view.Levels[0].Role != nil {
		t.Errorf("view = %q, roles %+v / %+v", view.LevelsText, view.Levels[0].Role, view.Levels[1].Role)
	}
}

// A backup approver and a role level must be able to approve (017 FR-025, 014 FR-028); one who lost the
// permission or became inactive after the save stays, with a warning, as does a role that no longer exists.
func TestWorkflow_BackupAndRoleMustApprove_FR025(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.cannotApprove[sara.ID] = true

	_, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{Name: "Viewer backup", Levels: []LevelInput{
		{ApproverIDs: []string{daniel.ID}, BackupApproverID: sara.ID},
		{ApproverType: SourceRole, RoleID: viewerRole.ID},
	}})
	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "levels.1.backupApproverId", Message: "Level 1: Sara Novak can't approve activities. Give them a role that approves first."},
		errs.FieldError{Field: "levels.2.roleId", Message: "Level 2: people with the role 'Viewer' can't approve activities. Choose a role that approves."},
	))

	created, err := f.service.CreateWorkflow(t.Context(), alex, WorkflowInput{Name: "Backups", Levels: []LevelInput{
		{ApproverIDs: []string{daniel.ID}, BackupApproverID: carlos.ID},
		{ApproverIDs: []string{daniel.ID}, BackupApproverID: mei.ID},
		{ApproverType: SourceRole, RoleID: approverRole.ID},
	}})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	f.access.cannotApprove[carlos.ID] = true
	f.deactivatePerson(t, mei)
	f.write(t, func(ctx context.Context) error { // the role no longer exists
		w, _, err := f.service.repo.GetWorkflow(ctx, created.ID)
		if err != nil {
			return err
		}
		w.Levels[2].RoleID = "RL-9999"
		return f.service.repo.PutWorkflow(ctx, w)
	})

	var view WorkflowView
	for _, w := range f.workflowViews(t, alex) {
		if w.ID == created.ID {
			view = w
		}
	}
	want := []string{
		"Level 1: the backup approver Carlos Mendes can't approve activities",
		"Level 2: the backup approver Mei Lin is inactive",
		"Level 3: role 'RL-9999' no longer exists",
	}
	if !reflect.DeepEqual(view.Warnings, want) {
		t.Errorf("warnings = %q, want %q", view.Warnings, want)
	}
	if _, err := f.service.UpdateWorkflow(t.Context(), alex, created.ID, WorkflowUpdate{Name: textPtr("Backups kept"), Version: version(1)}); err == nil ||
		!strings.Contains(err.Error(), "validation_failed") {
		t.Errorf("saving with the gone role = %v, want the role problem", err)
	}
}

// Changes of a workflow's rules are logged with readable texts (017 FR-035): the levels as their rule
// summaries, the reminder and escalation days as "3 days" or "Off".
func TestWorkflowChanges_LoggedWithRules_FR035(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	created := f.createWorkflow(t, alex, "Temp flow", []string{carlos.ID})
	below := "10000.00"

	_, err := f.service.UpdateWorkflow(t.Context(), alex, created.ID, WorkflowUpdate{
		Levels:       &[]LevelInput{{ApproverType: SourceManager, AutoApproveWhen: &ConditionInput{PlannedCostBelow: &below}, BackupApproverID: alex.ID}},
		ReminderDays: OptionalDays{Sent: true, Days: days(1)}, EscalationDays: OptionalDays{Sent: true, Days: days(2)}, Version: version(1),
	})
	if err != nil {
		t.Fatalf("UpdateWorkflow: %v", err)
	}

	entry := f.log(t, created.ID)[0]
	var got []string
	for _, c := range entry.Changes {
		got = append(got, c.Label+": "+c.OldText+" → "+c.NewText)
	}
	want := []string{
		"Escalate after: Off → 2 days",
		"Levels: Level 1 · Carlos Mendes → Level 1 · Line manager · automatic when planned cost is below $10,000.00",
		"Remind after: Off → 1 day",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("changes =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
