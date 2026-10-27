package approval

import (
	"context"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// CreateWorkflow creates an active approval workflow from in, its texts without surrounding spaces, after
// checking every rule at once (FR-002, FR-003; 017 FR-001 – FR-008, FR-025): the name, the description, the
// reminder and escalation days and each level with its approver source, conditions and backup approver. It
// is logged as created (FR-026).
func (s *Service) CreateWorkflow(ctx context.Context, a actor.Actor, in WorkflowInput) (WorkflowView, error) {
	name, description := strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	levels, amountProblems := levelsOf(in.Levels)
	var view WorkflowView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return err
		}
		w := Workflow{Name: name, Description: description, Active: true, Levels: levels,
			ReminderDays: in.ReminderDays, EscalationDays: in.EscalationDays}
		var problems errs.Problems
		if err = s.checkWorkflow(ctx, &problems, workflowCheck{rc: rc, w: w, amountProblems: amountProblems}); err != nil {
			return err
		}
		if err = problems.Err(); err != nil {
			return err
		}
		if w.ID, err = s.ids.Next(ctx, ids.PrefixApprovalWorkflow); err != nil {
			return err
		}
		w.Meta = record.NewMeta(a, s.now())
		if err = s.repo.PutWorkflow(ctx, w); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{Kind: workflowKind, RecordID: w.ID, RecordName: w.Name, Action: history.ActionCreated})
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, w, rc)
		return err
	})
	if err != nil {
		return WorkflowView{}, err
	}
	return view, nil
}
