package approval

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/record"
)

// Delegation is one person's delegation of their approvals to another person for a date range (table
// "approval_delegations", 017 FR-022, data-model §7). It is cancelled, never deleted, for the audit trail.
type Delegation struct {
	ID          string       `json:"id"`         // "DLG-1001"
	PersonID    string       `json:"personId"`   // the delegating person
	DelegateID  string       `json:"delegateId"` // who decides for them; holds approvals.decide
	StartDate   date.Date    `json:"startDate"`  // first day, inclusive (UTC calendar dates)
	EndDate     date.Date    `json:"endDate"`    // last day, inclusive; at most 90 days after StartDate
	Note        string       `json:"note"`       // 0–500 characters
	CancelledAt *time.Time   `json:"cancelledAt"`
	CancelledBy *actor.Actor `json:"cancelledBy"`
	record.Meta
}

// The states of a delegation (DelegationView.State), derived from its dates and today.
const (
	DelegationPlanned   = "planned"
	DelegationCurrent   = "current"
	DelegationEnded     = "ended"
	DelegationCancelled = "cancelled"
)

// state is the delegation's state on the day today (data-model §7).
func (dl Delegation) state(today date.Date) string {
	switch {
	case dl.CancelledAt != nil:
		return DelegationCancelled
	case dl.EndDate.Before(today):
		return DelegationEnded
	case dl.StartDate.After(today):
		return DelegationPlanned
	default:
		return DelegationCurrent
	}
}

// DelegationView is a delegation as the API answers it (ApprovalDelegation).
type DelegationView struct {
	ID          string       `json:"id"`
	Person      PersonView   `json:"person"`
	Delegate    PersonView   `json:"delegate"`
	StartDate   date.Date    `json:"startDate"`
	EndDate     date.Date    `json:"endDate"`
	Note        string       `json:"note"`
	State       string       `json:"state"`     // DelegationPlanned, DelegationCurrent, DelegationEnded or DelegationCancelled
	CanChange   bool         `json:"canChange"` // the signed-in person may edit and cancel it now
	CancelledAt *time.Time   `json:"cancelledAt"`
	CancelledBy *actor.Actor `json:"cancelledBy"`
	record.Meta
}

// DelegationPage is the answer of GET /api/v1/approval-delegations (ApprovalDelegationPage).
type DelegationPage struct {
	Items           []DelegationView `json:"items"`
	Total           int              `json:"total"`
	CanManageOthers bool             `json:"canManageOthers"` // the signed-in person holds workflows.manage
}

// DelegationInput is the body of POST /api/v1/approval-delegations (ApprovalDelegationCreate): dates as texts,
// so that a wrong date is a field problem.
type DelegationInput struct {
	PersonID   string `json:"personId"` // "" = the signed-in person
	DelegateID string `json:"delegateId"`
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	Note       string `json:"note"`
}

// DelegationUpdate is the body of PATCH /api/v1/approval-delegations/{delegationId} (ApprovalDelegationUpdate):
// only the fields to change, plus the version (D-203).
type DelegationUpdate struct {
	DelegateID *string `json:"delegateId"`
	StartDate  *string `json:"startDate"`
	EndDate    *string `json:"endDate"`
	Note       *string `json:"note"`
	Version    *int    `json:"version"`
}

// DelegationCancel is the body of POST /api/v1/approval-delegations/{delegationId}/cancel.
type DelegationCancel struct {
	Version *int `json:"version"`
}
