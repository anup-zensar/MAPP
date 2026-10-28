// Package approval approves records — activities and purchase requests — through approval workflows, and
// keeps their rounds, history entries and notifications (features 008, 017, 031).
package approval

import (
	"encoding/json"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

type Workflow struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Active         bool        `json:"active"`
	ObjectType     SubjectKind `json:"objectType"`
	Levels         []Level     `json:"levels"`
	ReminderDays   *int        `json:"reminderDays"`
	EscalationDays *int        `json:"escalationDays"`
	record.Meta
}

// Kind is what the workflow approves; a workflow stored without an object type approves activities.
func (w Workflow) Kind() SubjectKind {
	if w.ObjectType == "" {
		return KindActivity
	}
	return w.ObjectType
}

type Level struct {
	Name             string     `json:"name"`
	ApproverType     string     `json:"approverType"`
	ApproverIDs      []string   `json:"approverIds"`
	RoleID           string     `json:"roleId"`
	Mode             string     `json:"mode"`
	AppliesWhen      *Condition `json:"appliesWhen"`
	AutoApproveWhen  *Condition `json:"autoApproveWhen"`
	BackupApproverID string     `json:"backupApproverId"`
	RecordField      string     `json:"recordField"`
}

const (
	SourcePeople      = "people"
	SourceManager     = "manager"
	SourceRole        = "role"
	SourceOwner       = "owner"
	SourceRecordField = "recordField"
)

// RecordFieldMoAApprover is the purchase request's MoA approver, the record field a level may name.
const RecordFieldMoAApprover = "moaApprover"

const (
	ModeAny = "any"
	ModeAll = "all"
)

type Condition struct {
	PlannedCostAtLeast *money.Amount `json:"plannedCostAtLeast"`
	PlannedCostBelow   *money.Amount `json:"plannedCostBelow"`
	BusinessUnitIDs    []string      `json:"businessUnitIds"`
	ActivityTypeIDs    []string      `json:"activityTypeIds"`
}

type PersonView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Initials   string `json:"initials"`
	Active     bool   `json:"active"`
	CanApprove *bool  `json:"canApprove,omitempty"`
}

type WorkflowRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type LevelView struct {
	Number          int             `json:"number"`
	Name            string          `json:"name"`
	Label           string          `json:"label"`
	Approvers       []PersonView    `json:"approvers"`
	ApproverType    string          `json:"approverType"`
	Role            *RoleRef        `json:"role"`
	Mode            string          `json:"mode"`
	AppliesWhen     *ConditionView  `json:"appliesWhen"`
	AutoApproveWhen *ConditionView  `json:"autoApproveWhen"`
	BackupApprover  *PersonView     `json:"backupApprover"`
	RecordField     *RecordFieldRef `json:"recordField"`
	Summary         string          `json:"summary"`
}

type RecordFieldRef struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type RoleRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BuiltIn string `json:"builtIn"`
}

type ConditionView struct {
	PlannedCostAtLeast *money.Money `json:"plannedCostAtLeast"`
	PlannedCostBelow   *money.Money `json:"plannedCostBelow"`
	BusinessUnits      []OptionRef  `json:"businessUnits"`
	ActivityTypes      []SettingRef `json:"activityTypes"`
	Text               string       `json:"text"`
}

type OptionRef struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Active bool   `json:"active"`
}

type SettingRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type WorkflowView struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Active      bool        `json:"active"`
	ObjectType  SubjectKind `json:"objectType"`
	// ObjectTypeFixed: the workflow is used, so what it approves can no longer change.
	ObjectTypeFixed bool        `json:"objectTypeFixed"`
	Levels          []LevelView `json:"levels"`
	LevelsText      string      `json:"levelsText"`
	UsedByCount     int         `json:"usedByCount"`
	WaitingCount    int         `json:"waitingCount"`
	Warnings        []string    `json:"warnings"`
	ReminderDays    *int        `json:"reminderDays"`
	EscalationDays  *int        `json:"escalationDays"`
	record.Meta
}

type WorkflowPage struct {
	Items []WorkflowView `json:"items"`
	Total int            `json:"total"`
}

type WorkflowUsage struct {
	Activity      errs.RecordRef   `json:"activity"`
	Path          []errs.RecordRef `json:"path"`
	DisplayStatus string           `json:"displayStatus"`
	Waiting       bool             `json:"waiting"`
}

type WorkflowDetails struct {
	Workflow   WorkflowView    `json:"workflow"`
	Activities []WorkflowUsage `json:"activities"`
}

type LevelInput struct {
	Name             string          `json:"name"`
	ApproverIDs      []string        `json:"approverIds"`
	ApproverType     string          `json:"approverType"`
	RoleID           string          `json:"roleId"`
	Mode             string          `json:"mode"`
	AppliesWhen      *ConditionInput `json:"appliesWhen"`
	AutoApproveWhen  *ConditionInput `json:"autoApproveWhen"`
	BackupApproverID string          `json:"backupApproverId"`
	RecordField      string          `json:"recordField"`
}

type ConditionInput struct {
	PlannedCostAtLeast *string  `json:"plannedCostAtLeast"`
	PlannedCostBelow   *string  `json:"plannedCostBelow"`
	BusinessUnitIDs    []string `json:"businessUnitIds"`
	ActivityTypeIDs    []string `json:"activityTypeIds"`
}

type WorkflowInput struct {
	Name           string       `json:"name"`
	ObjectType     SubjectKind  `json:"objectType"`
	Description    string       `json:"description"`
	Levels         []LevelInput `json:"levels"`
	ReminderDays   *int         `json:"reminderDays"`
	EscalationDays *int         `json:"escalationDays"`
}

type WorkflowUpdate struct {
	Name           *string       `json:"name"`
	ObjectType     *SubjectKind  `json:"objectType"`
	Description    *string       `json:"description"`
	Levels         *[]LevelInput `json:"levels"`
	ReminderDays   OptionalDays  `json:"reminderDays"`
	EscalationDays OptionalDays  `json:"escalationDays"`
	Version        *int          `json:"version"`
}

// OptionalDays is days an update may leave out (Sent false), turn off (Sent, Days nil) or set.
type OptionalDays struct {
	Sent bool
	Days *int
}

func (o *OptionalDays) UnmarshalJSON(data []byte) error {
	o.Sent = true
	return json.Unmarshal(data, &o.Days)
}

func (o OptionalDays) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Days)
}

type StatusChange struct {
	Version *int `json:"version"`
}

type RuleOptions struct {
	ActivityTypes []SettingRef        `json:"activityTypes"`
	BusinessUnits []OptionRef         `json:"businessUnits"`
	Roles         []RoleRef           `json:"roles"`
	ObjectTypes   []KindOption        `json:"objectTypes"`
	RecordFields  []RecordFieldOption `json:"recordFields"`
}

type KindOption struct {
	ID    SubjectKind `json:"id"`
	Label string      `json:"label"`
}

type RecordFieldOption struct {
	ObjectType SubjectKind `json:"objectType"`
	ID         string      `json:"id"`
	Label      string      `json:"label"`
}
