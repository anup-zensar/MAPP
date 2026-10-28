package app

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/stores"
)

// activityWorkflows implements activity.Workflows over the approval workflows that approve activities; a
// workflow of another object type is not found.
type activityWorkflows struct {
	repo stores.ApprovalStore
}

var _ activity.Workflows = activityWorkflows{}

func (w activityWorkflows) GetWorkflow(ctx context.Context, id string) (activity.WorkflowFacts, bool, error) {
	workflow, found, err := w.repo.GetWorkflow(ctx, id)
	if err != nil || !found || workflow.Kind() != approval.KindActivity {
		return activity.WorkflowFacts{}, false, err
	}
	return workflowFacts(workflow), true, nil
}

// ListWorkflows returns every activity workflow, active and inactive, sorted by name ignoring letter case.
func (w activityWorkflows) ListWorkflows(ctx context.Context) ([]activity.WorkflowFacts, error) {
	workflows, err := w.repo.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]activity.WorkflowFacts, 0, len(workflows))
	for _, workflow := range workflows {
		if workflow.Kind() == approval.KindActivity {
			list = append(list, workflowFacts(workflow))
		}
	}
	slices.SortStableFunc(list, func(x, y activity.WorkflowFacts) int {
		return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
	})
	return list, nil
}

func workflowFacts(w approval.Workflow) activity.WorkflowFacts {
	levels := make([]activity.WorkflowLevelFacts, 0, len(w.Levels))
	for _, l := range w.Levels {
		levels = append(levels, activity.WorkflowLevelFacts{Name: l.Name, ApproverIDs: slices.Clone(l.ApproverIDs)})
	}
	return activity.WorkflowFacts{ID: w.ID, Name: w.Name, Active: w.Active, Levels: levels}
}
