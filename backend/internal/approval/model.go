// Package approval holds activity approvals (feature 008): approval workflows (a Settings record), the
// approval rounds of activities with their history entries, and in-app notifications.
//
// The shared foundation wrote the data layer: the stored records and the API types (model*.go) and the
// store (store.go, store_json.go). Feature 008 adds the Activities port with its adapter, the
// service, the history kind and the handler. The package never imports activities.
package approval

import (
	"encoding/json"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// Workflow is an approval workflow: a named, ordered set of approval levels (table "approval_workflows",
// data model §1). Feature 006's seed writes the two demo workflows in this shape (D-103, D-201).
type Workflow struct {
	ID          string  `json:"id"`          // "WF-1001"
	Name        string  `json:"name"`        // 1–120 characters, unique among all workflows ignoring case
	Description string  `json:"description"` // 0–2,000 characters
	Active      bool    `json:"active"`
	Levels      []Level `json:"levels"` // at least one; the first is level 1
	// ReminderDays and EscalationDays are the days a level may wait before its approvers are reminded (1–60)
	// and before it is escalated to its backup approver (1–90, more than ReminderDays); nil = off (017 FR-007).
	ReminderDays   *int `json:"reminderDays"`
	EscalationDays *int `json:"escalationDays"`
	record.Meta
}

// Level is one level of a workflow: a value inside it, without an id of its own (008-P11). Feature 017 gives
// it an approver source, a mode, two conditions and a backup approver (data-model §2). Levels stored before
// 017 read as named people, any one approves, always needed, never automatic (normalised, data-model §10).
type Level struct {
	Name         string   `json:"name"`         // 0–60 characters; "" means "Level <n>"
	ApproverType string   `json:"approverType"` // SourcePeople, SourceManager or SourceRole
	ApproverIDs  []string `json:"approverIds"`  // SourcePeople: at least one person; empty for the other sources
	RoleID       string   `json:"roleId"`       // SourceRole: a 014 role that includes approvals.decide; "" otherwise
	Mode         string   `json:"mode"`         // ModeAny or ModeAll; always ModeAny for SourceManager
	// AppliesWhen is the "Only needed when" condition (nil = always needed); AutoApproveWhen the "Approve
	// automatically when" condition (nil = never automatic) (017 FR-003, FR-004).
	AppliesWhen      *Condition `json:"appliesWhen"`
	AutoApproveWhen  *Condition `json:"autoApproveWhen"`
	BackupApproverID string     `json:"backupApproverId"` // "" = none; required on every level when the workflow escalates
}

// The approver sources of a level (Level.ApproverType, 017 FR-001).
const (
	SourcePeople  = "people"  // the named people of ApproverIDs
	SourceManager = "manager" // the activity owner's line manager (people.Person.ManagerID)
	SourceRole    = "role"    // the active holders of a 014 role whose scope covers the activity's plan
)

// The modes of a level (Level.Mode, 017 FR-002).
const (
	ModeAny = "any" // the first approval completes the level
	ModeAll = "all" // every resolved approver must approve
)

// Condition is a level's "Only needed when" or "Approve automatically when" rule (017 FR-005): every part it
// holds must match. A value without an id; at least one part is set.
type Condition struct {
	PlannedCostAtLeast *money.Amount `json:"plannedCostAtLeast"` // USD; planned cost ≥ it
	PlannedCostBelow   *money.Amount `json:"plannedCostBelow"`   // USD; planned cost < it
	BusinessUnitIDs    []string      `json:"businessUnitIds"`    // one of the plan's business units is listed
	ActivityTypeIDs    []string      `json:"activityTypeIds"`    // the activity's type is listed
}

// The API types below are what the API returns and receives (contract schema in brackets, JSON names as
// the contract). Lists are never nil. Activity statuses are texts: approvals does not import activities.

// PersonView is a designated approver or owner, with the live active flag (ApprovalPerson).
type PersonView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Initials string `json:"initials"`
	Active   bool   `json:"active"`
	// CanApprove is false when the person holds no role that approves (feature 014, FR-028): the chain flags
	// them "(can't approve — no approving role)". Always sent once access control is wired (UseApprovers);
	// nil before.
	CanApprove *bool `json:"canApprove,omitempty"`
}

// WorkflowRef is a workflow referred to by an activity (WorkflowRef).
type WorkflowRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// LevelView is one level of a workflow as the API shows it (ApprovalWorkflowLevel), with its rule (017).
type LevelView struct {
	Number          int            `json:"number"`
	Name            string         `json:"name"`
	Label           string         `json:"label"` // the name, or "Level <number>"
	Approvers       []PersonView   `json:"approvers"`
	ApproverType    string         `json:"approverType"`
	Role            *RoleRef       `json:"role"` // role levels only
	Mode            string         `json:"mode"`
	AppliesWhen     *ConditionView `json:"appliesWhen"`
	AutoApproveWhen *ConditionView `json:"autoApproveWhen"`
	BackupApprover  *PersonView    `json:"backupApprover"`
	Summary         string         `json:"summary"` // this level's part of the levels text
}

// RoleRef is a 014 role as a level names it (RoleRef).
type RoleRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BuiltIn string `json:"builtIn"` // which seeded role it is; "" for roles made in Settings
}

// ConditionView is a level condition as the API answers it, with its text (ApprovalCondition).
type ConditionView struct {
	PlannedCostAtLeast *money.Money `json:"plannedCostAtLeast"`
	PlannedCostBelow   *money.Money `json:"plannedCostBelow"`
	BusinessUnits      []OptionRef  `json:"businessUnits"`
	ActivityTypes      []SettingRef `json:"activityTypes"`
	Text               string       `json:"text"` // "planned cost is below $10,000.00"
}

// OptionRef is a Shell attribute option, such as a business unit, with its status (OptionRef).
type OptionRef struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Active bool   `json:"active"`
}

// SettingRef is a Settings record, such as an activity type, with its status (SettingRef).
type SettingRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// WorkflowView is a workflow as the API returns it (ApprovalWorkflow).
type WorkflowView struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Active         bool        `json:"active"`
	Levels         []LevelView `json:"levels"`
	LevelsText     string      `json:"levelsText"` // "Level 1 · Carlos Mendes → Level 2 · Alex Morgan"
	UsedByCount    int         `json:"usedByCount"`
	WaitingCount   int         `json:"waitingCount"`
	Warnings       []string    `json:"warnings"` // "Level 2 has no active approver"
	ReminderDays   *int        `json:"reminderDays"`
	EscalationDays *int        `json:"escalationDays"`
	record.Meta
}

// WorkflowPage is the answer of GET /api/v1/approval-workflows (ApprovalWorkflowPage).
type WorkflowPage struct {
	Items []WorkflowView `json:"items"`
	Total int            `json:"total"`
}

// WorkflowUsage is an activity counted in a workflow's usedByCount (ApprovalWorkflowUsage).
type WorkflowUsage struct {
	Activity      errs.RecordRef   `json:"activity"`
	Path          []errs.RecordRef `json:"path"` // plan, campaign, initiative
	DisplayStatus string           `json:"displayStatus"`
	Waiting       bool             `json:"waiting"`
}

// WorkflowDetails is the answer of GET /api/v1/approval-workflows/{workflowId} (ApprovalWorkflowDetails).
type WorkflowDetails struct {
	Workflow   WorkflowView    `json:"workflow"`
	Activities []WorkflowUsage `json:"activities"`
}

// LevelInput is one level of a create or update (ApprovalWorkflowLevelInput). A level without an approver
// type or mode is a named-people level where any one approves, as before feature 017.
type LevelInput struct {
	Name             string          `json:"name"`
	ApproverIDs      []string        `json:"approverIds"`
	ApproverType     string          `json:"approverType"`
	RoleID           string          `json:"roleId"`
	Mode             string          `json:"mode"`
	AppliesWhen      *ConditionInput `json:"appliesWhen"`
	AutoApproveWhen  *ConditionInput `json:"autoApproveWhen"`
	BackupApproverID string          `json:"backupApproverId"`
}

// ConditionInput is a level condition as sent (ApprovalConditionInput): amounts as USD decimal texts, so
// that a wrong amount is a field problem rather than an unreadable request.
type ConditionInput struct {
	PlannedCostAtLeast *string  `json:"plannedCostAtLeast"`
	PlannedCostBelow   *string  `json:"plannedCostBelow"`
	BusinessUnitIDs    []string `json:"businessUnitIds"`
	ActivityTypeIDs    []string `json:"activityTypeIds"`
}

// WorkflowInput is the body of POST /api/v1/approval-workflows (ApprovalWorkflowCreate).
type WorkflowInput struct {
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	Levels         []LevelInput `json:"levels"`
	ReminderDays   *int         `json:"reminderDays"`   // nil = no reminders
	EscalationDays *int         `json:"escalationDays"` // nil = no escalation
}

// WorkflowUpdate is the body of PATCH /api/v1/approval-workflows/{workflowId} (ApprovalWorkflowUpdate): only
// the fields to change (D-203), Levels being the complete new list. The days are sent only to change them,
// null turning them off.
type WorkflowUpdate struct {
	Name           *string       `json:"name"`
	Description    *string       `json:"description"`
	Levels         *[]LevelInput `json:"levels"`
	ReminderDays   OptionalDays  `json:"reminderDays"`
	EscalationDays OptionalDays  `json:"escalationDays"`
	Version        *int          `json:"version"`
}

// OptionalDays is a number of days that an update may leave out (Sent false), turn off (Sent, Days nil) or
// set. encoding/json calls UnmarshalJSON only for a field that is present, null included.
type OptionalDays struct {
	Sent bool
	Days *int
}

// UnmarshalJSON reads null or a whole number of days.
func (o *OptionalDays) UnmarshalJSON(data []byte) error {
	o.Sent = true
	return json.Unmarshal(data, &o.Days)
}

// MarshalJSON writes the days, or null.
func (o OptionalDays) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Days)
}

// StatusChange is the body of POST …/deactivate and …/activate (ApprovalWorkflowStatusChange).
type StatusChange struct {
	Version *int `json:"version"`
}

// RuleOptions is what a level's rule may name (ApprovalRuleOptions): activity types and business units for
// conditions, and the 014 roles that approve for role levels (017 FR-001, FR-005).
type RuleOptions struct {
	ActivityTypes []SettingRef `json:"activityTypes"`
	BusinessUnits []OptionRef  `json:"businessUnits"`
	Roles         []RoleRef    `json:"roles"`
}
