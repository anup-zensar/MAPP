package approval

import (
	"context"
	"time"

	"mapp/internal/people"
)

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists by id, except RoundsOf (by round number) and NotificationsOf (newest first, then by id); history
// entries and delegations are never deleted.
type Store interface {
	GetWorkflow(ctx context.Context, id string) (Workflow, bool, error)
	ListWorkflows(ctx context.Context) ([]Workflow, error)
	PutWorkflow(ctx context.Context, w Workflow) error
	DeleteWorkflow(ctx context.Context, id string) error

	GetRound(ctx context.Context, id string) (Round, bool, error)
	ListRounds(ctx context.Context) ([]Round, error)
	RoundsOf(ctx context.Context, subjectID string) ([]Round, error)
	OpenRounds(ctx context.Context) ([]Round, error)
	PutRound(ctx context.Context, r Round) error

	PutAction(ctx context.Context, a Action) error
	ListActions(ctx context.Context) ([]Action, error)
	ActionsOf(ctx context.Context, subjectID string) ([]Action, error)

	GetNotification(ctx context.Context, id string) (Notification, bool, error)
	NotificationsOf(ctx context.Context, personID string) ([]Notification, error)
	MarkNotificationsRead(ctx context.Context, ids []string, at time.Time) error
	PutNotification(ctx context.Context, n Notification) error

	GetDelegation(ctx context.Context, id string) (Delegation, bool, error)
	ListDelegations(ctx context.Context) ([]Delegation, error)
	PutDelegation(ctx context.Context, dl Delegation) error
}

type People interface {
	List(ctx context.Context) ([]people.Person, error)
}

var _ Store = (*PostgresStore)(nil)
