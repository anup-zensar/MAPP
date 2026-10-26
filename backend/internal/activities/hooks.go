package activities

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// The seams below are declared by activities and implemented by later features through adapters in
// internal/app (D-196): LineItems by 007, Workflows and ApprovalHooks by 008. Until they are wired, the
// activities service uses its own defaults (no line items, the seed's workflows, no hooks). Every method
// runs inside the caller's transaction, which ctx carries.

// LineItems is what activities needs from line items (feature 007).
type LineItems interface {
	// Facts returns, per activity id, its line-item totals and problems; a missing entry means the activity
	// has no line items.
	Facts(ctx context.Context, activityIDs []string) (map[string]LineItemFacts, error)
	// CopyAll copies every line item of fromActivityID into toActivityID with new IDs (R13), moving the
	// custom months and the finance budget year by yearShift years (D-195, 007-P9), inside the caller's
	// Write; it writes no log entries and returns how many it copied.
	CopyAll(ctx context.Context, a actor.Actor, fromActivityID, toActivityID string, yearShift int) (int, error)
	// CopyMany is CopyAll for many activities at once, activity by activity in the order of pairs.
	CopyMany(ctx context.Context, a actor.Actor, pairs []CopyPair, yearShift int) (int, error)
	// DatesChanged is told after the activity's dates or ongoing changed, inside the same Write (007 FR-022).
	DatesChanged(ctx context.Context, a actor.Actor, before, after Activity) error
	// Items returns the line items of the activity activityID in creation order, with their planned costs in
	// their own currency and in the plan currency (016 FR-026, the Split dialog).
	Items(ctx context.Context, activityID string) ([]LineItemBrief, error)
	// MoveToActivity moves the line items lineItemIDs to the activity toActivityID keeping their IDs, values
	// and spreads, logging each as moved, inside the caller's Write (016 FR-028, FR-030).
	MoveToActivity(ctx context.Context, a actor.Actor, lineItemIDs []string, toActivityID string) error
}

type CopyPair struct {
	FromID, ToID string
}

// LineItemBrief is a line item in short (016 FR-026): its planned cost in its own currency and in the plan
// currency.
type LineItemBrief struct {
	ID, Name        string
	PlannedCost     money.Money
	PlannedCostPlan money.Money
}

// LineItemFacts are one activity's line items in short.
type LineItemFacts struct {
	Count           int
	PlannedCostPlan money.Amount // sum of the line items' planned costs in Currency, the plan currency (016 FR-015)
	Currency        string       // the plan currency of PlannedCostPlan; "" when there are no line items
	AutoSpread      int          // spread Evenly, First or Last month: re-spread on a date change
	CustomSpread    int          // custom spread: kept on a date change
	Problems        []LineItemProblem
}

// LineItemProblem is one problem of a line item that blocks submitting its activity.
type LineItemProblem struct {
	LineItemID, LineItemName string
	Code                     string // e.g. "missingField", "customSpreadMismatch", "inactiveFinanceBudgetItem", "splitIncomplete"
	Field                    string // the line-item form field, e.g. "financeBudgetItemId" or "attributes.<id>"
	Message                  string // e.g. "GL code — needed to submit"
}

// Workflows is where activities finds approval workflows (feature 008 owns them).
type Workflows interface {
	// GetWorkflow returns the workflow with id, active or not, and false when there is none.
	GetWorkflow(ctx context.Context, id string) (WorkflowFacts, bool, error)
	// ListWorkflows returns every workflow, active and inactive, sorted by name.
	ListWorkflows(ctx context.Context) ([]WorkflowFacts, error)
}

// WorkflowFacts is an approval workflow as activities needs it.
type WorkflowFacts struct {
	ID, Name string
	Active   bool
	Levels   []WorkflowLevelFacts // in order: the first is level 1
}

// WorkflowLevelFacts is one level of a workflow.
type WorkflowLevelFacts struct {
	Name        string // "" means "Level <n>"
	ApproverIDs []string
}

// ApprovalHooks let feature 008 react to activity changes inside the same Write (D-196).
type ApprovalHooks interface {
	// ApprovalRestarted is called after a re-approval change sent a Submitted or Approved activity back to
	// Created and its log entry was written (006 FR-027; also through RestartApproval for 007's line-item
	// changes); changes are that entry's described changes.
	ApprovalRestarted(ctx context.Context, a actor.Actor, before, after Activity, changes []history.Change) error
	// ActivityDeleting is called before the activity moves to the recycle bin.
	ActivityDeleting(ctx context.Context, a actor.Actor, act Activity) error
	// ActivityRestored is called after a restore from the recycle bin (never after an Undo) reset the
	// activity to Created; it returns notes for the "restored" log entry (008 cancels the open round).
	ActivityRestored(ctx context.Context, rc history.RestoreContext, activityID string) (notes []string, err error)
}
