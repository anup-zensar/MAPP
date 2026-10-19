package approvals

import (
	"cmp"
	"context"
	"slices"

	"northstar/internal/platform/store"
)

// The JSON store tables of this package (data model §1–§4).
const (
	workflowsTable     = "approval_workflows"
	roundsTable        = "approval_rounds"
	actionsTable       = "approval_actions"
	notificationsTable = "notifications"
)

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// GetWorkflow reads one workflow.
func (jsonRepository) GetWorkflow(ctx context.Context, id string) (Workflow, bool, error) {
	return store.Get[Workflow](ctx, workflowsTable, id)
}

// ListWorkflows reads the whole approval_workflows table.
func (jsonRepository) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	return store.List[Workflow](ctx, workflowsTable)
}

// PutWorkflow saves a workflow under its id.
func (jsonRepository) PutWorkflow(ctx context.Context, w Workflow) error {
	return store.Put(ctx, workflowsTable, w.ID, w)
}

// DeleteWorkflow removes one workflow.
func (jsonRepository) DeleteWorkflow(ctx context.Context, id string) error {
	return store.Delete(ctx, workflowsTable, id)
}

// GetRound reads one round.
func (jsonRepository) GetRound(ctx context.Context, id string) (Round, bool, error) {
	return store.Get[Round](ctx, roundsTable, id)
}

// ListRounds reads the whole approval_rounds table.
func (jsonRepository) ListRounds(ctx context.Context) ([]Round, error) {
	return store.List[Round](ctx, roundsTable)
}

// RoundsOf reads every round to find those of one activity, ordered by their number.
func (r jsonRepository) RoundsOf(ctx context.Context, activityID string) ([]Round, error) {
	all, err := r.ListRounds(ctx)
	if err != nil {
		return nil, err
	}
	rounds := where(all, func(round Round) bool { return round.ActivityID == activityID })
	slices.SortStableFunc(rounds, func(a, b Round) int { return cmp.Compare(a.Number, b.Number) })
	return rounds, nil
}

// PutRound saves a round under its id.
func (jsonRepository) PutRound(ctx context.Context, round Round) error {
	return store.Put(ctx, roundsTable, round.ID, round)
}

// PutAction saves a history entry under its id.
func (jsonRepository) PutAction(ctx context.Context, a Action) error {
	return store.Put(ctx, actionsTable, a.ID, a)
}

// ListActions reads the whole approval_actions table; the store orders it by id, which is the order the
// entries were written in.
func (jsonRepository) ListActions(ctx context.Context) ([]Action, error) {
	return store.List[Action](ctx, actionsTable)
}

// ActionsOf reads every history entry to find those of one activity.
func (r jsonRepository) ActionsOf(ctx context.Context, activityID string) ([]Action, error) {
	all, err := r.ListActions(ctx)
	if err != nil {
		return nil, err
	}
	return where(all, func(a Action) bool { return a.ActivityID == activityID }), nil
}

// GetNotification reads one notification.
func (jsonRepository) GetNotification(ctx context.Context, id string) (Notification, bool, error) {
	return store.Get[Notification](ctx, notificationsTable, id)
}

// NotificationsOf reads every notification to find those of one person, newest first. The store lists
// them by id (creation order); reversed, the newer id comes first, and the stable sort by time keeps it so
// among notifications of the same time.
func (jsonRepository) NotificationsOf(ctx context.Context, personID string) ([]Notification, error) {
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
func (jsonRepository) PutNotification(ctx context.Context, n Notification) error {
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
