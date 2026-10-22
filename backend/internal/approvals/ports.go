package approvals

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
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

	// Feature 017: what the approval rules and 014's scopes look at — the activity's plan, its activity type
	// ("" when empty) and the plan's business units with their labels now (data-model §3).
	Plan           PlanFacts
	ActivityTypeID string
	BusinessUnits  []Choice
}

// PlanFacts is what feature 014's scopes look at in an activity's plan (authz.PlanFacts): its id, teams,
// owner and creator.
type PlanFacts struct {
	ID          string
	TeamIDs     []string
	OwnerID     string
	CreatedByID string
}

// Choice is a value a rule condition may name — an activity type or a business unit — with its label and
// whether it is active.
type Choice struct {
	ID, Label string
	Active    bool
}

// Permission is one of feature 014's permissions that approvals asks about.
type Permission string

// The 014 permissions of approvals (014 Table A).
const (
	PermissionDecide          Permission = "approvals.decide"         // approve or decline when named as an approver
	PermissionDecideOnBehalf  Permission = "approvals.decideOnBehalf" // decide on behalf of the named approvers (scoped)
	PermissionManageWorkflows Permission = "workflows.manage"         // create and change approval workflows and delegations
)

// Grant says whether a person holds a permission and, when not, the 403 message that explains it.
type Grant struct {
	Allowed bool
	Refusal string // 014's refusal message; "" when allowed
}

// RoleInfo is a 014 role as approvals needs it: its reference and whether it includes approvals.decide.
type RoleInfo struct {
	RoleRef
	Approves bool
}

// Access is what approvals needs to know from feature 014 (access control) about people's roles (017 plan,
// Ports; D-290): who may approve, decide on behalf and manage workflows, the access rule "Nobody approves an
// activity they own or submitted", the roles and their holders. internal/app implements it over 014's
// internal/access hooks; until it is set (UseAccess), everyone holds every permission and the rule is on.
// Every method runs inside the caller's transaction, which ctx carries.
type Access interface {
	// CanApprove reports, for each of personIDs, whether they hold approvals.decide (014 hook CanApprove).
	CanApprove(ctx context.Context, personIDs []string) (map[string]bool, error)
	// Check reports whether the person personID holds the permission p — approvals.decideOnBehalf in a scope
	// that covers plan (nil: in any scope) — and, when not, 014's refusal message.
	Check(ctx context.Context, personID string, p Permission, plan *PlanFacts) (Grant, error)
	// NoSelfApproval reports whether 014's access rule "Nobody approves an activity they own or submitted" is
	// on (D-302).
	NoSelfApproval(ctx context.Context) (bool, error)
	// Roles returns every 014 role, sorted by name, with whether it includes approvals.decide.
	Roles(ctx context.Context) ([]RoleInfo, error)
	// RoleHolders returns the active people who hold the role roleID with a scope that covers plan, sorted by
	// name (014 hook RoleHolders).
	RoleHolders(ctx context.Context, roleID string, plan PlanFacts) ([]string, error)
}

// RuleChoices lists what a rule condition may name (017 FR-005), over feature 002's readers: every activity
// type and every option of the plan-level Business unit attribute, active or not, as Settings sorts them.
type RuleChoices interface {
	ActivityTypes(ctx context.Context) ([]Choice, error)
	BusinessUnits(ctx context.Context) ([]Choice, error)
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
