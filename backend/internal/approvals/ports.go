package approvals

import (
	"context"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/authz"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
)

// Activities is what approvals needs from activities (feature 006) and their line items (feature 007).
// internal/app implements it over those features (approvals_adapter.go); the service tests use a fake. Every
// method runs inside the caller's transaction, which ctx carries, so approvals never imports activities.
type Activities interface {
	// Activity returns the stored activity id, and false when it is not stored (never created, or in the
	// recycle bin).
	Activity(ctx context.Context, id string) (ActivityFacts, bool, error)
	// List returns every stored activity.
	List(ctx context.Context) ([]ActivityFacts, error)
	// SetStatus stores the activity's new status with record.Meta.Touch — a new version, last changed by a —
	// and returns the activity as stored now. It writes no log entry: approvals writes it (FR-026).
	SetStatus(ctx context.Context, a actor.Actor, id, status string) (ActivityFacts, error)
	// ReadyCheck runs the ready-to-submit check of features 006 and 007 (006 FR-023, 007 FR-028).
	ReadyCheck(ctx context.Context, a actor.Actor, id string) (ReadyCheck, error)
	// LineItems returns the activity's line items, in their order (FR-032).
	LineItems(ctx context.Context, a actor.Actor, activityID string) ([]PreviewLineItem, error)
}

// ActivityFacts is an activity as approvals needs it.
type ActivityFacts struct {
	ID, Name              string
	Status                string           // StatusCreated, StatusSubmitted, StatusDeclined or StatusApproved
	DisplayStatus         string           // the status, or "inProgress" / "complete" for approved ones (006 FR-021)
	Path                  []errs.RecordRef // plan, campaign, initiative: top level first
	OwnerID               string           // "" when empty
	StartDate, EndDate    *date.Date       // nil when empty (EndDate also when ongoing)
	Ongoing               bool
	PlannedCost           *money.Money // USD; nil when empty
	UnassignedPlannedCost *money.Money // planned cost minus the line items in USD; nil with neither
	TypeName, Description string
	WorkflowID            string // the activity's Approval workflow field; "" when empty
	record.Meta                  // version, and who changed the activity last, and when
	// Access is the activity's plan as access decisions need it (feature 014): who sees the activity and who
	// may submit, recall or decide on it.
	Access authz.PlanFacts
}

// ReadyCheck is the result of the ready-to-submit check: every problem that blocks submitting, all at once
// (FR-010 a–d), and a positive unassigned planned cost, which never blocks (domain model §4.3).
type ReadyCheck struct {
	Problems              []SubmitProblem
	UnassignedPlannedCost *money.Money // set only when positive
	// Missing is the activity's completeness: its own empty needed-to-submit fields among the problems,
	// as the preview lists them (FR-032).
	Missing []MissingFieldRef
}

// The stored statuses of an activity (006, contract ActivityStatus).
const (
	StatusCreated   = "created"
	StatusSubmitted = "submitted"
	StatusDeclined  = "declined"
	StatusApproved  = "approved"
)

// The groups of the ready-to-submit problems (SubmitProblem.Group).
const (
	GroupActivity = "activity"
	GroupLineItem = "lineItem"
)
