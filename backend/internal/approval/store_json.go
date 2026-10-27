package approval

import (
	"cmp"
	"context"
	"slices"

	"mapp/internal/platform/store"
)

// The JSON store tables of this package (data model §1–§4).
const (
	workflowsTable     = "approval_workflows"
	roundsTable        = "approval_rounds"
	actionsTable       = "approval_actions"
	notificationsTable = "notifications"
	delegationsTable   = "approval_delegations" // 017 data-model §7
)

// JSONStore is the Store of the JSON store.
type JSONStore struct{}

// NewJSONStore returns the Store of the JSON store.
func NewJSONStore() JSONStore {
	return JSONStore{}
}

// GetWorkflow reads one workflow.
func (JSONStore) GetWorkflow(ctx context.Context, id string) (Workflow, bool, error) {
	w, found, err := store.Get[Workflow](ctx, workflowsTable, id)
	return w.normalised(), found, err
}

// ListWorkflows reads the whole approval_workflows table.
func (JSONStore) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	workflows, err := store.List[Workflow](ctx, workflowsTable)
	for i := range workflows {
		workflows[i] = workflows[i].normalised()
	}
	return workflows, err
}

// PutWorkflow saves a workflow under its id.
func (JSONStore) PutWorkflow(ctx context.Context, w Workflow) error {
	return store.Put(ctx, workflowsTable, w.ID, w)
}

// DeleteWorkflow removes one workflow.
func (JSONStore) DeleteWorkflow(ctx context.Context, id string) error {
	return store.Delete(ctx, workflowsTable, id)
}

// GetRound reads one round.
func (JSONStore) GetRound(ctx context.Context, id string) (Round, bool, error) {
	round, found, err := store.Get[Round](ctx, roundsTable, id)
	return round.normalised(), found, err
}

// ListRounds reads the whole approval_rounds table.
func (JSONStore) ListRounds(ctx context.Context) ([]Round, error) {
	rounds, err := store.List[Round](ctx, roundsTable)
	for i := range rounds {
		rounds[i] = rounds[i].normalised()
	}
	return rounds, err
}

// RoundsOf reads every round to find those of one activity, ordered by their number.
func (r JSONStore) RoundsOf(ctx context.Context, activityID string) ([]Round, error) {
	all, err := r.ListRounds(ctx)
	if err != nil {
		return nil, err
	}
	rounds := where(all, func(round Round) bool { return round.ActivityID == activityID })
	slices.SortStableFunc(rounds, func(a, b Round) int { return cmp.Compare(a.Number, b.Number) })
	return rounds, nil
}

func (r JSONStore) OpenRounds(ctx context.Context) ([]Round, error) {
	all, err := r.ListRounds(ctx)
	if err != nil {
		return nil, err
	}
	return where(all, func(round Round) bool { return round.Outcome == OutcomeOpen }), nil
}

// PutRound saves a round under its id.
func (JSONStore) PutRound(ctx context.Context, round Round) error {
	return store.Put(ctx, roundsTable, round.ID, round)
}

// PutAction saves a history entry under its id.
func (JSONStore) PutAction(ctx context.Context, a Action) error {
	return store.Put(ctx, actionsTable, a.ID, a)
}

// ListActions reads the whole approval_actions table; the store orders it by id, which is the order the
// entries were written in.
func (JSONStore) ListActions(ctx context.Context) ([]Action, error) {
	return store.List[Action](ctx, actionsTable)
}

// ActionsOf reads every history entry to find those of one activity.
func (r JSONStore) ActionsOf(ctx context.Context, activityID string) ([]Action, error) {
	all, err := r.ListActions(ctx)
	if err != nil {
		return nil, err
	}
	return where(all, func(a Action) bool { return a.ActivityID == activityID }), nil
}

// GetNotification reads one notification.
func (JSONStore) GetNotification(ctx context.Context, id string) (Notification, bool, error) {
	return store.Get[Notification](ctx, notificationsTable, id)
}

// NotificationsOf reads every notification to find those of one person, newest first. The store lists
// them by id (creation order); reversed, the newer id comes first, and the stable sort by time keeps it so
// among notifications of the same time.
func (JSONStore) NotificationsOf(ctx context.Context, personID string) ([]Notification, error) {
	all, err := store.List[Notification](ctx, notificationsTable)
	if err != nil {
		return nil, err
	}
	mine := where(all, func(n Notification) bool { return n.RecipientID == personID })
	slices.Reverse(mine) // newest id first
	slices.SortStableFunc(mine, func(a, b Notification) int { return b.At.Compare(a.At) })
	return mine, nil
}

// PutNotification saves a notification under its id.
func (JSONStore) PutNotification(ctx context.Context, n Notification) error {
	return store.Put(ctx, notificationsTable, n.ID, n)
}

// where returns the records that keep accepts, in order; never nil.
func where[T any](records []T, keep func(T) bool) []T {
	kept := []T{}
	for _, r := range records {
		if keep(r) {
			kept = append(kept, r)
		}
	}
	return kept
}

// GetDelegation reads one delegation.
func (JSONStore) GetDelegation(ctx context.Context, id string) (Delegation, bool, error) {
	return store.Get[Delegation](ctx, delegationsTable, id)
}

// ListDelegations reads the whole approval_delegations table, ordered by id.
func (JSONStore) ListDelegations(ctx context.Context) ([]Delegation, error) {
	return store.List[Delegation](ctx, delegationsTable)
}

// PutDelegation saves a delegation under its id.
func (JSONStore) PutDelegation(ctx context.Context, dl Delegation) error {
	return store.Put(ctx, delegationsTable, dl.ID, dl)
}
