package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/record"
)

// PostgresStore is the Repository on PostgreSQL. It holds the approval workflows so far; rounds, history
// entries, notifications and delegations follow (feature 023 T021), and until then it is not a Repository.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) GetWorkflow(ctx context.Context, id string) (Workflow, bool, error) {
	row, err := s.db.Queries(ctx).GetWorkflow(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workflow{}, false, nil
	}
	if err != nil {
		return Workflow{}, false, fmt.Errorf("select approval workflow %s: %w", id, err)
	}
	workflows, err := s.withLevels(ctx, []sqlcdb.ApprovalWorkflow{row}, &id)
	if err != nil {
		return Workflow{}, false, err
	}
	return workflows[0], true, nil
}

func (s *PostgresStore) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	rows, err := s.db.Queries(ctx).ListWorkflows(ctx)
	if err != nil {
		return nil, fmt.Errorf("select approval workflows: %w", err)
	}
	return s.withLevels(ctx, rows, nil)
}

// PutWorkflow saves w with its levels normalised, which is how the JSON store reads every level back, so a
// workflow read from either store is the same.
func (s *PostgresStore) PutWorkflow(ctx context.Context, w Workflow) error {
	q := s.db.Queries(ctx)
	err := q.PutWorkflow(ctx, sqlcdb.PutWorkflowParams{
		ID: w.ID, Name: w.Name, Description: w.Description, Active: w.Active, ReminderDays: w.ReminderDays,
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
	if err := s.db.Queries(ctx).DeleteWorkflow(ctx, id); err != nil {
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
		BackupApproverID: database.NullIfEmpty(l.BackupApproverID),
	})
	if err != nil {
		return err
	}
	return q.InsertWorkflowLevelApprovers(ctx, sqlcdb.InsertWorkflowLevelApproversParams{
		WorkflowID: workflowID, LevelNumber: number, PersonIds: l.ApproverIDs,
	})
}

// withLevels reads the levels of the workflows rows (of the one workflowID, or of all) in one query.
func (s *PostgresStore) withLevels(ctx context.Context, rows []sqlcdb.ApprovalWorkflow, workflowID *string) ([]Workflow, error) {
	levelRows, err := s.db.Queries(ctx).ListWorkflowLevels(ctx, workflowID)
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
		ID: row.ID, Name: row.Name, Description: row.Description, Active: row.Active, Levels: levels,
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
		BackupApproverID: database.EmptyIfNull(row.BackupApproverID),
	}
	err := errors.Join(json.Unmarshal(row.AppliesWhen, &l.AppliesWhen), json.Unmarshal(row.AutoApproveWhen, &l.AutoApproveWhen))
	if err != nil {
		return Level{}, fmt.Errorf("decode level %d of approval workflow %s: %w", row.Number, row.WorkflowID, err)
	}
	return l, nil
}
