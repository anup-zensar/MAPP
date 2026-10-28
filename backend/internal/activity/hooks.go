package activity

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// LineItems, Workflows and ApprovalHooks are adapters wired in internal/app (D-196); each runs in the caller's Write.
type LineItems interface {
	Facts(ctx context.Context, activityIDs []string) (map[string]LineItemFacts, error)
	CopyAll(ctx context.Context, a actor.Actor, fromActivityID, toActivityID string, yearShift int) (int, error)
	CopyMany(ctx context.Context, a actor.Actor, pairs []CopyPair, yearShift int) (int, error)
	DatesChanged(ctx context.Context, a actor.Actor, before, after Activity) error
	Items(ctx context.Context, activityID string) ([]LineItemBrief, error)
	MoveToActivity(ctx context.Context, a actor.Actor, lineItemIDs []string, toActivityID string) error
}

type CopyPair struct {
	FromID, ToID string
}

type LineItemBrief struct {
	ID, Name        string
	PlannedCost     money.Money
	PlannedCostPlan money.Money
}

type LineItemFacts struct {
	Count           int
	PlannedCostPlan money.Amount
	Currency        string
	AutoSpread      int
	CustomSpread    int
	Problems        []LineItemProblem
}

type LineItemProblem struct {
	LineItemID, LineItemName string
	Code                     string
	Field                    string
	Message                  string
}

type Workflows interface {
	GetWorkflow(ctx context.Context, id string) (WorkflowFacts, bool, error)
	ListWorkflows(ctx context.Context) ([]WorkflowFacts, error)
}

type WorkflowFacts struct {
	ID, Name string
	Active   bool
	Levels   []WorkflowLevelFacts
}

type WorkflowLevelFacts struct {
	Name        string
	ApproverIDs []string
}

type ApprovalHooks interface {
	// ApprovalRestarted runs after the activity's own planned cost, dates, ongoing flag or a split sent it back to Created (D-322).
	ApprovalRestarted(ctx context.Context, a actor.Actor, before, after Activity, changes []history.Change) error
	ActivityDeleting(ctx context.Context, a actor.Actor, act Activity) error
	// ActivityRestored runs after a recycle-bin restore, never after an Undo, and returns notes for the "restored" log entry.
	ActivityRestored(ctx context.Context, rc history.RestoreContext, activityID string) (notes []string, err error)
}
