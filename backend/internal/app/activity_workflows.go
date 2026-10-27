package app

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/stores"
)

// activityWorkflows implements activity.Workflows (feature 006) over the stored approval workflows, so
// the activity form offers the active ones and labels the inactive ones (FR-007; 008 plan, Reconciliation).
// Its reads join the caller's transaction.
type activityWorkflows struct {
	repo stores.ApprovalStore
}

// The compiler checks here that activityWorkflows implements activity.Workflows.
var _ activity.Workflows = activityWorkflows{}

// GetWorkflow returns the workflow id, active or not, and false when there is none.
func (w activityWorkflows) GetWorkflow(ctx context.Context, id string) (activity.WorkflowFacts, bool, error) {
	workflow, found, err := w.repo.GetWorkflow(ctx, id)
	if err != nil || !found {
		return activity.WorkflowFacts{}, false, err
	}
	return workflowFacts(workflow), true, nil
}

// ListWorkflows returns every workflow, active and inactive, sorted by name ignoring letter case.
func (w activityWorkflows) ListWorkflows(ctx context.Context) ([]activity.WorkflowFacts, error) {
	workflows, err := w.repo.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]activity.WorkflowFacts, 0, len(workflows))
	for _, workflow := range workflows {
		list = append(list, workflowFacts(workflow))
	}
	slices.SortStableFunc(list, func(x, y activity.WorkflowFacts) int {
		return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
	})
	return list, nil
}

// workflowFacts returns a stored workflow as activities needs it.
func workflowFacts(w approval.Workflow) activity.WorkflowFacts {
	levels := make([]activity.WorkflowLevelFacts, 0, len(w.Levels))
	for _, l := range w.Levels {
		levels = append(levels, activity.WorkflowLevelFacts{Name: l.Name, ApproverIDs: slices.Clone(l.ApproverIDs)})
	}
	return activity.WorkflowFacts{ID: w.ID, Name: w.Name, Active: w.Active, Levels: levels}
}
