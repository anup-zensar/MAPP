package activities

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"northstar/internal/approvals"
)

// seededWorkflows reads the approval workflows the seed wrote in feature 008's table (D-103, D-201).
type seededWorkflows struct {
	repo approvals.Repository
}

// NewSeededWorkflows returns the Workflows of a build without feature 008: the workflows of the table
// "approval_workflows", read through repo. Feature 008 replaces it with its own adapter.
func NewSeededWorkflows(repo approvals.Repository) Workflows {
	return seededWorkflows{repo: repo}
}

// GetWorkflow returns the workflow with id, active or not.
func (w seededWorkflows) GetWorkflow(ctx context.Context, id string) (WorkflowFacts, bool, error) {
	workflow, found, err := w.repo.GetWorkflow(ctx, id)
	if err != nil || !found {
		return WorkflowFacts{}, false, err
	}
	return workflowFacts(workflow), true, nil
}

// ListWorkflows returns every workflow, active and inactive, sorted by name ignoring letter case.
func (w seededWorkflows) ListWorkflows(ctx context.Context) ([]WorkflowFacts, error) {
	stored, err := w.repo.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]WorkflowFacts, 0, len(stored))
	for _, workflow := range stored {
		list = append(list, workflowFacts(workflow))
	}
	slices.SortStableFunc(list, func(x, y WorkflowFacts) int {
		return cmp.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
	})
	return list, nil
}

// workflowFacts is a stored workflow as activities needs it.
func workflowFacts(w approvals.Workflow) WorkflowFacts {
	levels := make([]WorkflowLevelFacts, 0, len(w.Levels))
	for _, l := range w.Levels {
		levels = append(levels, WorkflowLevelFacts{Name: l.Name, ApproverIDs: slices.Clone(l.ApproverIDs)})
	}
	return WorkflowFacts{ID: w.ID, Name: w.Name, Active: w.Active, Levels: levels}
}
