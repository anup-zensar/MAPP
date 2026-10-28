package approval

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

type Activities interface {
	Activity(ctx context.Context, id string) (ActivityFacts, bool, error)
	ListOf(ctx context.Context, ids []string) ([]ActivityFacts, error)
	WorkflowFields(ctx context.Context) (map[string]string, error)
	ListInScope(ctx context.Context, scope authz.Scope, ids []string) ([]ActivityFacts, error)
	SetStatus(ctx context.Context, a actor.Actor, id, status string) (ActivityFacts, error)
	ReadyCheck(ctx context.Context, a actor.Actor, id string) (ReadyCheck, error)
	LineItems(ctx context.Context, a actor.Actor, activityID string) ([]PreviewLineItem, error)
}

type ActivityFacts struct {
	ID, Name              string
	Status                string
	DisplayStatus         string
	Path                  []errs.RecordRef
	OwnerID               string
	StartDate, EndDate    *date.Date
	Ongoing               bool
	PlannedCost           *money.Money
	UnassignedPlannedCost *money.Money
	TypeName, Description string
	WorkflowID            string
	record.Meta
	Access authz.PlanFacts

	Plan           PlanFacts
	ActivityTypeID string
	BusinessUnits  []Choice
}

type PlanFacts struct {
	ID          string
	TeamIDs     []string
	OwnerID     string
	CreatedByID string
}

type Choice struct {
	ID, Label string
	Active    bool
}

type Permission string

const (
	PermissionDecide          Permission = "approvals.decide"
	PermissionDecideOnBehalf  Permission = "approvals.decideOnBehalf"
	PermissionManageWorkflows Permission = "workflows.manage"
)

type Grant struct {
	Allowed bool
	Refusal string
}

type RoleInfo struct {
	RoleRef
	Approves bool
}

type Access interface {
	CanApprove(ctx context.Context, personIDs []string) (map[string]bool, error)
	Check(ctx context.Context, personID string, p Permission, plan *PlanFacts) (Grant, error)
	NoSelfApproval(ctx context.Context) (bool, error)
	Roles(ctx context.Context) ([]RoleInfo, error)
	RoleHolders(ctx context.Context, roleID string, plan PlanFacts) ([]string, error)
}

type RuleChoices interface {
	ActivityTypes(ctx context.Context) ([]Choice, error)
	BusinessUnits(ctx context.Context) ([]Choice, error)
}

type ReadyCheck struct {
	Problems              []SubmitProblem
	UnassignedPlannedCost *money.Money
	Missing               []MissingFieldRef
}

const (
	StatusCreated   = "created"
	StatusSubmitted = "submitted"
	StatusDeclined  = "declined"
	StatusApproved  = "approved"
)

const (
	GroupActivity = "activity"
	GroupLineItem = "lineItem"
)
