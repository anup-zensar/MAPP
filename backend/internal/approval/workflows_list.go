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
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// ListWorkflows returns every approval workflow by name ignoring case, with its usage and warnings.
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

// GetWorkflow returns the workflow and the records that use it, by plan and then name.
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
		using, err := s.usingRecords(ctx, u.usedBy[w.ID], u)
		if err != nil {
			return err
		}
		details = WorkflowDetails{Workflow: workflowView(w, u, rc), Activities: make([]WorkflowUsage, 0, len(using))}
		for _, sub := range using {
			details.Activities = append(details.Activities, WorkflowUsage{
				Activity: sub.ref(), Path: nonNilPath(sub.Path), DisplayStatus: sub.DisplayStatus, Waiting: u.waiting[w.ID][sub.ID],
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

// usingRecords reads the records among ids: every stored activity, and the purchase requests the person sees.
func (s *Service) usingRecords(ctx context.Context, ids []string, u usages) ([]subject, error) {
	var activityIDs, purchaseIDs []string
	for _, id := range ids {
		if u.purchases[id] {
			purchaseIDs = append(purchaseIDs, id)
		} else {
			activityIDs = append(activityIDs, id)
		}
	}
	acts, err := s.activities.ListOf(ctx, nonNilIDs(activityIDs))
	if err != nil {
		return nil, err
	}
	list := make([]subject, 0, len(ids))
	for _, f := range acts {
		list = append(list, activitySubject(f))
	}
	if s.purchases == nil || len(purchaseIDs) == 0 {
		return list, nil
	}
	prs, err := s.purchases.ListInScope(ctx, authz.ScopeOf(ctx), purchaseIDs)
	for _, f := range prs {
		list = append(list, purchaseSubject(KindPurchaseRequest, f))
	}
	return list, err
}

// DeleteWorkflow moves an unused workflow to the recycle bin; a used one is refused with in_use.
func (s *Service) DeleteWorkflow(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		w, err := s.storedWorkflow(ctx, id)
		if err != nil {
			return err
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		switch n := len(u.usedBy[id]); n {
		case 0:
		case 1:
			return errs.InUse(fmt.Sprintf(inUseOneMessage, w.Kind().word()))
		default:
			return errs.InUse(fmt.Sprintf(inUseManyMessage, n, w.Kind().records()))
		}
		entry, err = s.history.MoveToBin(ctx, a, workflowKind, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

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

func workflowNamed(workflows []Workflow, name, exceptID string) (Workflow, bool) {
	for _, w := range workflows {
		if w.ID != exceptID && strings.EqualFold(w.Name, name) {
			return w, true
		}
	}
	return Workflow{}, false
}

func planName(path []errs.RecordRef) string {
	if len(path) == 0 {
		return ""
	}
	return path[0].Name
}

// usages: by workflow id, the stored records that name it or have a round with it, and those waiting on it.
type usages struct {
	usedBy    map[string][]string
	waiting   map[string]map[string]bool
	purchases map[string]bool
}

func (s *Service) usage(ctx context.Context) (usages, error) {
	fields, err := s.activities.WorkflowFields(ctx)
	if err != nil {
		return usages{}, err
	}
	rounds, err := s.repo.ListRounds(ctx)
	if err != nil {
		return usages{}, err
	}
	stored, err := s.storedPurchaseRequests(ctx, rounds)
	if err != nil {
		return usages{}, err
	}
	byRecord := map[string][]Round{}
	for _, r := range rounds {
		byRecord[r.SubjectID] = append(byRecord[r.SubjectID], r)
	}
	u := usages{usedBy: map[string][]string{}, waiting: map[string]map[string]bool{}, purchases: map[string]bool{}}
	for _, id := range stored {
		u.purchases[id] = true
	}
	for _, recordID := range inIDOrder(append(slices.Collect(maps.Keys(fields)), stored...)) {
		uses := map[string]bool{}
		if fields[recordID] != "" {
			uses[fields[recordID]] = true
		}
		for _, r := range byRecord[recordID] {
			uses[r.WorkflowID] = true
			if r.Outcome == OutcomeOpen {
				u.wait(r.WorkflowID, recordID)
			}
		}
		for id := range uses {
			u.usedBy[id] = append(u.usedBy[id], recordID)
		}
	}
	return u, nil
}

// storedPurchaseRequests returns the ids of the stored purchase requests that have rounds.
func (s *Service) storedPurchaseRequests(ctx context.Context, rounds []Round) ([]string, error) {
	var ids []string
	for _, r := range recordsOf(rounds) {
		if r.kind.isPurchaseRequest() {
			ids = append(ids, r.id)
		}
	}
	if s.purchases == nil || len(ids) == 0 {
		return nil, nil
	}
	prs, err := s.purchases.ListInScope(ctx, authz.Scope{Everything: true, TeamIDs: []string{}}, ids)
	stored := make([]string, 0, len(prs))
	for _, f := range prs {
		stored = append(stored, f.ID)
	}
	return stored, err
}

func (u usages) wait(workflowID, recordID string) {
	if u.waiting[workflowID] == nil {
		u.waiting[workflowID] = map[string]bool{}
	}
	u.waiting[workflowID][recordID] = true
}

// inIDOrder sorts readable ids as the stores list records: shorter ids first, then by text.
func inIDOrder(recordIDs []string) []string {
	slices.SortFunc(recordIDs, func(x, y string) int { return cmp.Or(cmp.Compare(len(x), len(y)), strings.Compare(x, y)) })
	return recordIDs
}
