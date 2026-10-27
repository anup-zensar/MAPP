package approval

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// ListWorkflows returns every approval workflow, active and inactive, sorted by name ignoring letter case,
// each with its levels and their approvers now, its levels text, how many activities use it and wait for
// approval with it, and its warnings (FR-001, FR-005).
func (s *Service) ListWorkflows(ctx context.Context, _ actor.Actor) ([]WorkflowView, error) {
	var views []WorkflowView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		workflows, err := s.repo.ListWorkflows(ctx)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(workflows, func(x, y Workflow) int {
			return cmp.Or(cmp.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), cmp.Compare(x.ID, y.ID))
		})
		views = make([]WorkflowView, 0, len(workflows))
		for _, w := range workflows {
			views = append(views, workflowView(w, u, rc))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return views, nil
}

// GetWorkflow returns the approval workflow id as ListWorkflows does, with the activities that use it, by
// plan name and then activity name (FR-005).
func (s *Service) GetWorkflow(ctx context.Context, _ actor.Actor, id string) (WorkflowDetails, error) {
	var details WorkflowDetails
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		w, err := s.storedWorkflow(ctx, id)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return err
		}
		details = WorkflowDetails{Workflow: workflowView(w, u, rc), Activities: []WorkflowUsage{}}
		using, err := s.activities.ListOf(ctx, u.usedBy[w.ID])
		if err != nil {
			return err
		}
		for _, facts := range using {
			details.Activities = append(details.Activities, WorkflowUsage{
				Activity: errs.RecordRef{Type: activityKind, ID: facts.ID, Name: facts.Name}, Path: nonNilPath(facts.Path),
				DisplayStatus: facts.DisplayStatus, Waiting: u.waiting[w.ID][facts.ID],
			})
		}
		slices.SortStableFunc(details.Activities, func(x, y WorkflowUsage) int {
			return cmp.Or(cmp.Compare(strings.ToLower(planName(x.Path)), strings.ToLower(planName(y.Path))),
				cmp.Compare(strings.ToLower(x.Activity.Name), strings.ToLower(y.Activity.Name)))
		})
		return nil
	})
	if err != nil {
		return WorkflowDetails{}, err
	}
	return details, nil
}

// DeleteWorkflow moves the approval workflow id to the recycle bin and returns the bin entry for Undo
// (FR-008). A workflow that any activity uses is refused with in_use: it can be set inactive instead.
func (s *Service) DeleteWorkflow(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if _, err := s.storedWorkflow(ctx, id); err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		switch n := len(u.usedBy[id]); n {
		case 0:
		case 1:
			return errs.InUse(inUseOneMessage)
		default:
			return errs.InUse(fmt.Sprintf(inUseManyMessage, n))
		}
		entry, err = s.history.MoveToBin(ctx, a, workflowKind, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// storedWorkflow returns the stored workflow id, or not_found with the plan's message.
func (s *Service) storedWorkflow(ctx context.Context, id string) (Workflow, error) {
	w, found, err := s.repo.GetWorkflow(ctx, id)
	if err != nil {
		return Workflow{}, err
	}
	if !found {
		return Workflow{}, errs.NotFound(workflowGoneMessage)
	}
	return w, nil
}

// workflowNamed returns the workflow among workflows, other than exceptID, whose name is name ignoring
// letter case, and false when there is none. Names are stored without surrounding spaces.
func workflowNamed(workflows []Workflow, name, exceptID string) (Workflow, bool) {
	for _, w := range workflows {
		if w.ID != exceptID && strings.EqualFold(w.Name, name) {
			return w, true
		}
	}
	return Workflow{}, false
}

// planName is the name of the plan of a path, "" when the path has none.
func planName(path []errs.RecordRef) string {
	if len(path) == 0 {
		return ""
	}
	return path[0].Name
}

// usages is who uses each workflow (data model §1): by workflow id, the stored activities that have it in
// their Approval workflow field or have a round with it, in the order of the activities, and those of them
// with an open round of it.
type usages struct {
	usedBy  map[string][]string        // workflow id → activity ids
	waiting map[string]map[string]bool // workflow id → activity id → waiting
}

// usage reads who uses each workflow, inside the caller's transaction: the activities' workflow fields and
// the rounds, never the activities themselves, which only GetWorkflow lists.
func (s *Service) usage(ctx context.Context) (usages, error) {
	fields, err := s.activities.WorkflowFields(ctx)
	if err != nil {
		return usages{}, err
	}
	rounds, err := s.repo.ListRounds(ctx)
	if err != nil {
		return usages{}, err
	}
	byActivity := map[string][]Round{}
	for _, r := range rounds {
		byActivity[r.ActivityID] = append(byActivity[r.ActivityID], r)
	}
	u := usages{usedBy: map[string][]string{}, waiting: map[string]map[string]bool{}}
	for _, activityID := range inIDOrder(slices.Collect(maps.Keys(fields))) {
		uses := map[string]bool{}
		if fields[activityID] != "" {
			uses[fields[activityID]] = true
		}
		for _, r := range byActivity[activityID] {
			uses[r.WorkflowID] = true
			if r.Outcome == OutcomeOpen {
				u.wait(r.WorkflowID, activityID)
			}
		}
		for id := range uses {
			u.usedBy[id] = append(u.usedBy[id], activityID)
		}
	}
	return u, nil
}

func (u usages) wait(workflowID, activityID string) {
	if u.waiting[workflowID] == nil {
		u.waiting[workflowID] = map[string]bool{}
	}
	u.waiting[workflowID][activityID] = true
}

// inIDOrder sorts readable ids as the stores list records: shorter ids first, then by text.
func inIDOrder(recordIDs []string) []string {
	slices.SortFunc(recordIDs, func(x, y string) int { return cmp.Or(cmp.Compare(len(x), len(y)), strings.Compare(x, y)) })
	return recordIDs
}
