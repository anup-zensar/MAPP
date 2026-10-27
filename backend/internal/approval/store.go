package approval

import (
	"context"
	"time"

	"mapp/internal/people"
)

// Store stores approval workflows, rounds, history entries and notifications. The JSON store
// implements it now (store_json.go) and PostgreSQL can later. Every method runs inside the caller's
// transaction, which the context carries. Lists are never nil.
type Store interface {
	// GetWorkflow returns the workflow with id, and false when there is none.
	GetWorkflow(ctx context.Context, id string) (Workflow, bool, error)
	// ListWorkflows returns every workflow, active or not, ordered by id.
	ListWorkflows(ctx context.Context) ([]Workflow, error)
	// PutWorkflow saves w, replacing the workflow with the same id.
	PutWorkflow(ctx context.Context, w Workflow) error
	// DeleteWorkflow removes the workflow with id; removing one that does not exist changes nothing.
	DeleteWorkflow(ctx context.Context, id string) error

	// GetRound returns the round with id, and false when there is none.
	GetRound(ctx context.Context, id string) (Round, bool, error)
	// ListRounds returns every round of every activity, ordered by id.
	ListRounds(ctx context.Context) ([]Round, error)
	// RoundsOf returns the rounds of the activity activityID, ordered by round number.
	RoundsOf(ctx context.Context, activityID string) ([]Round, error)
	// OpenRounds returns every round still open, at most one per activity, ordered by id.
	OpenRounds(ctx context.Context) ([]Round, error)
	// PutRound saves r, replacing the round with the same id.
	PutRound(ctx context.Context, r Round) error

	// PutAction saves a new history entry (entries are never changed or deleted).
	PutAction(ctx context.Context, a Action) error
	// ListActions returns every history entry, oldest first (by id number).
	ListActions(ctx context.Context) ([]Action, error)
	// ActionsOf returns the history entries of the activity activityID, oldest first.
	ActionsOf(ctx context.Context, activityID string) ([]Action, error)

	// GetNotification returns the notification with id, and false when there is none.
	GetNotification(ctx context.Context, id string) (Notification, bool, error)
	// NotificationsOf returns the notifications of the person personID, newest first (time, then id
	// number).
	NotificationsOf(ctx context.Context, personID string) ([]Notification, error)
	// MarkNotificationsRead marks the notifications with ids read at at, all at once.
	MarkNotificationsRead(ctx context.Context, ids []string, at time.Time) error
	// PutNotification saves n, replacing the notification with the same id.
	PutNotification(ctx context.Context, n Notification) error

	// GetDelegation returns the approval delegation with id, and false when there is none (017).
	GetDelegation(ctx context.Context, id string) (Delegation, bool, error)
	// ListDelegations returns every approval delegation, cancelled ones included, ordered by id.
	ListDelegations(ctx context.Context) ([]Delegation, error)
	// PutDelegation saves dl, replacing the delegation with the same id. Delegations are never deleted.
	PutDelegation(ctx context.Context, dl Delegation) error
}

// People is what approvals reads of people and teams.
type People interface {
	List(ctx context.Context) ([]people.Person, error)
}

var (
	_ Store = (*PostgresStore)(nil)
	_ Store = JSONStore{}
)
