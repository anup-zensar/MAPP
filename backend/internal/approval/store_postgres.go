package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/memo"
	"mapp/internal/platform/record"
)

// keptArea is what the transaction memo keeps of approvals: the workflows.
const keptArea = "approvals"

// PostgresStore is the Store on PostgreSQL: workflows here, rounds and history entries in
// store_postgres_rounds.go, notifications and delegations in store_postgres_notifications.go.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) GetWorkflow(ctx context.Context, id string) (Workflow, bool, error) {
	workflows, err := s.workflows(ctx)
	if err != nil {
		return Workflow{}, false, err
	}
	i := slices.IndexFunc(workflows, func(w Workflow) bool { return w.ID == id })
	if i < 0 {
		return Workflow{}, false, nil
	}
	return copyOfWorkflow(workflows[i]), true, nil
}

func (s *PostgresStore) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	workflows, err := s.workflows(ctx)
	if err != nil {
		return nil, err
	}
	copies := make([]Workflow, 0, len(workflows))
	for _, w := range workflows {
		copies = append(copies, copyOfWorkflow(w))
	}
	return copies, nil
}

// workflows reads every workflow with its levels once per transaction (memo): a page checks the workflow of
// each activity it shows, and only this store changes them.
func (s *PostgresStore) workflows(ctx context.Context) ([]Workflow, error) {
	return memo.Get(ctx, keptArea+":workflows", func(ctx context.Context) ([]Workflow, error) {
		rows, err := s.db.Queries(ctx).ListWorkflows(ctx)
		if err != nil {
			return nil, fmt.Errorf("select approval workflows: %w", err)
		}
		return s.withLevels(ctx, rows)
	})
}

func (s *PostgresStore) change(ctx context.Context) *sqlcdb.Queries {
	memo.Forget(ctx, keptArea)
	return s.db.Queries(ctx)
}

// copyOfWorkflow returns w with a list of levels of its own, so a caller never changes the kept one.
func copyOfWorkflow(w Workflow) Workflow {
	w.Levels = slices.Clone(w.Levels)
	return w
}

// PutWorkflow saves w with feature 017's defaults in its levels, so every workflow reads back alike.
func (s *PostgresStore) PutWorkflow(ctx context.Context, w Workflow) error {
	q := s.change(ctx)
	err := q.PutWorkflow(ctx, sqlcdb.PutWorkflowParams{
		ID: w.ID, Name: w.Name, Description: w.Description, Active: w.Active, ObjectType: string(w.Kind()), ReminderDays: w.ReminderDays,
		EscalationDays: w.EscalationDays, Version: w.Version, CreatedAt: w.CreatedAt, CreatedByID: w.CreatedBy.ID,
		CreatedByName: w.CreatedBy.Name, UpdatedAt: w.UpdatedAt, UpdatedByID: w.UpdatedBy.ID,
		UpdatedByName: w.UpdatedBy.Name,
	})
	if err == nil {
		err = q.DeleteWorkflowLevels(ctx, w.ID)
	}
	for i, level := range normalisedLevels(slices.Clone(w.Levels)) {
		if err == nil {
			err = s.insertLevel(ctx, w.ID, i+1, level)
		}
	}
	if err != nil {
		return fmt.Errorf("save approval workflow %s: %w", w.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteWorkflow(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteWorkflow(ctx, id); err != nil {
		return fmt.Errorf("delete approval workflow %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) insertLevel(ctx context.Context, workflowID string, number int, l Level) error {
	appliesWhen, appliesErr := json.Marshal(l.AppliesWhen)
	autoApproveWhen, autoErr := json.Marshal(l.AutoApproveWhen)
	if err := errors.Join(appliesErr, autoErr); err != nil {
		return err
	}
	q := s.db.Queries(ctx)
	err := q.InsertWorkflowLevel(ctx, sqlcdb.InsertWorkflowLevelParams{
		WorkflowID: workflowID, Number: number, Name: l.Name, ApproverType: l.ApproverType, RoleID: l.RoleID,
		Mode: l.Mode, AppliesWhen: appliesWhen, AutoApproveWhen: autoApproveWhen,
		BackupApproverID: database.NullIfEmpty(l.BackupApproverID), RecordField: l.RecordField,
	})
	if err != nil {
		return err
	}
	return q.InsertWorkflowLevelApprovers(ctx, sqlcdb.InsertWorkflowLevelApproversParams{
		WorkflowID: workflowID, LevelNumber: number, PersonIds: l.ApproverIDs,
	})
}

// withLevels reads the levels of every workflow in one query.
func (s *PostgresStore) withLevels(ctx context.Context, rows []sqlcdb.ApprovalWorkflow) ([]Workflow, error) {
	levelRows, err := s.db.Queries(ctx).ListWorkflowLevels(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("select approval workflow levels: %w", err)
	}
	levels := map[string][]Level{}
	for _, row := range levelRows {
		l, err := levelFromRow(row.ApprovalWorkflowLevel, row.ApproverIds)
		if err != nil {
			return nil, err
		}
		levels[row.ApprovalWorkflowLevel.WorkflowID] = append(levels[row.ApprovalWorkflowLevel.WorkflowID], l)
	}
	workflows := make([]Workflow, 0, len(rows))
	for _, row := range rows {
		workflows = append(workflows, workflowFromRow(row, levels[row.ID]))
	}
	return workflows, nil
}

func workflowFromRow(row sqlcdb.ApprovalWorkflow, levels []Level) Workflow {
	if levels == nil {
		levels = []Level{}
	}
	return Workflow{
		ID: row.ID, Name: row.Name, Description: row.Description, Active: row.Active, ObjectType: SubjectKind(row.ObjectType), Levels: levels,
		ReminderDays: row.ReminderDays, EscalationDays: row.EscalationDays,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}

func levelFromRow(row sqlcdb.ApprovalWorkflowLevel, approverIDs []string) (Level, error) {
	l := Level{
		Name: row.Name, ApproverType: row.ApproverType, ApproverIDs: approverIDs, RoleID: row.RoleID, Mode: row.Mode,
		BackupApproverID: database.EmptyIfNull(row.BackupApproverID), RecordField: row.RecordField,
	}
	err := errors.Join(json.Unmarshal(row.AppliesWhen, &l.AppliesWhen), json.Unmarshal(row.AutoApproveWhen, &l.AutoApproveWhen))
	if err != nil {
		return Level{}, fmt.Errorf("decode level %d of approval workflow %s: %w", row.Number, row.WorkflowID, err)
	}
	return l, nil
}
