// Package approvals holds activity approvals (feature 008): approval workflows (a Settings record), the
// approval rounds of activities with their history entries, and in-app notifications.
//
// The shared foundation wrote the data layer: the stored records and the API types (this file) and the
// repository (repository.go, store_json.go). Feature 008 adds the Activities port with its adapter, the
// service, the history kind and the handler. The package never imports activities.
package approvals

import (
	"encoding/json"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
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

// Outcome is how an approval round stands or ended (data model §2).
type Outcome string

// The round outcomes. At most one round per activity is open, exactly while the activity is Submitted.
const (
	OutcomeOpen      Outcome = "open"
	OutcomeApproved  Outcome = "approved"
	OutcomeDeclined  Outcome = "declined"
	OutcomeRecalled  Outcome = "recalled"
	OutcomeRestarted Outcome = "restarted" // a re-approval change (006, 007)
	OutcomeCancelled Outcome = "cancelled" // the activity was restored from the recycle bin
)

// Round is one attempt to get one activity approved (table "approval_rounds", data model §2). It keeps
// its own copy of the workflow's levels, so a later edit of the workflow never changes it (FR-006).
type Round struct {
	ID           string       `json:"id"`           // "AR-1001"
	ActivityID   string       `json:"activityId"`   // never changes
	Number       int          `json:"number"`       // 1 for the activity's first round, then the highest + 1
	WorkflowID   string       `json:"workflowId"`   // the activity's workflow at submission
	WorkflowName string       `json:"workflowName"` // its name at submission
	Levels       []RoundLevel `json:"levels"`       // copy of the workflow's levels at submission
	CurrentLevel int          `json:"currentLevel"` // 1 at submission; after the round ends, the level it ended at
	Outcome      Outcome      `json:"outcome"`
	SubmittedAt  time.Time    `json:"submittedAt"`
	SubmittedBy  actor.Actor  `json:"submittedBy"`
	Comment      string       `json:"comment"` // the submit comment, 0–2,000 characters
	EndedAt      *time.Time   `json:"endedAt"` // nil while open
	// Facts are the values the rules used at submission (017 FR-010); nil for rounds stored before 017.
	Facts *RuleFacts `json:"facts"`
	// ReminderDays and EscalationDays are copied from the workflow at submission (008 FR-006 holds).
	ReminderDays   *int `json:"reminderDays"`
	EscalationDays *int `json:"escalationDays"`
	AutoApproved   bool `json:"autoApproved"` // no level needed a decision: approved at submission (017 FR-014)
	record.Meta
}

// RuleFacts are the values of an activity the rules look at, taken once at submission (data-model §3).
type RuleFacts struct {
	PlannedCost   *money.Money `json:"plannedCost"`   // USD; nil when the activity has none
	ActivityType  *FactRef     `json:"activityType"`  // nil when the activity has no type
	BusinessUnits []FactOption `json:"businessUnits"` // the plan's business units; never nil
}

// FactRef is a Settings record as the rules saw it: its id and name at submission.
type FactRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FactOption is a Shell attribute option as the rules saw it: its id and label at submission.
type FactOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// RoundLevel is a round's copy of one workflow level, with its approvers' names at submission. Feature 017
// keeps in it what the rules decided for the level and who was found to approve it (data-model §4).
type RoundLevel struct {
	Number    int           `json:"number"`    // 1-based
	Name      string        `json:"name"`      // "" means "Level <n>"
	Approvers []actor.Actor `json:"approvers"` // the resolved deciders; empty for skipped and automatic levels
	// The level's rule, copied: its approver source, the text of how its approvers were found ("" for named
	// people, "line manager of Priya Raman", "role Approver"), its role and its mode.
	ApproverType string `json:"approverType"`
	Source       string `json:"source"`
	RoleID       string `json:"roleId"`
	RoleName     string `json:"roleName"`
	Mode         string `json:"mode"`
	// What the rules decided at submission (ResolutionDecide, ResolutionSkipped, ResolutionAutoApproved), and
	// why for the last two.
	Resolution string `json:"resolution"`
	RuleText   string `json:"ruleText"`
	// The people left out when the approvers were resolved, whether the backup approver became the approver,
	// whether nobody can approve but someone deciding on behalf, and the backup who can take the level on
	// escalation (nil when none can, data-model §3 steps 4–6).
	Excluded   []Excluded   `json:"excluded"`
	BackupUsed bool         `json:"backupUsed"`
	NoApprover bool         `json:"noApprover"`
	Backup     *actor.Actor `json:"backup"`
	// StartedAt is when the round reached the level; RemindedAt and EscalatedAt when the job reminded its
	// approvers and escalated it (017 FR-026 – FR-029). nil until then.
	StartedAt   *time.Time `json:"startedAt"`
	RemindedAt  *time.Time `json:"remindedAt"`
	EscalatedAt *time.Time `json:"escalatedAt"`
}

// The outcomes of a level at submission (RoundLevel.Resolution, 017 FR-010).
const (
	ResolutionDecide       = "decide"       // its approvers decide
	ResolutionSkipped      = "skipped"      // Not needed: its "Only needed when" condition did not match
	ResolutionAutoApproved = "autoApproved" // Approved automatically: its "Approve automatically when" matched
)

// Excluded is a person left out of a level when its approvers were resolved, and why (017 FR-011).
type Excluded struct {
	Person actor.Actor `json:"person"`
	Reason string      `json:"reason"` // ReasonInactive, ReasonCannotApprove, ReasonOwner or ReasonSubmitter
}

// The reasons a person is left out of a level (Excluded.Reason), in the order they are checked.
const (
	ReasonInactive      = "inactive"
	ReasonCannotApprove = "cannotApprove" // holds no role with approvals.decide (014)
	ReasonOwner         = "owner"         // owns the activity (014's access rule, 017 FR-018)
	ReasonSubmitter     = "submitter"     // is submitting it (014's access rule, 017 FR-018)
)

// ActionKind is what an approval history entry records.
type ActionKind string

// The approval history actions.
const (
	ActionSubmitted   ActionKind = "submitted"
	ActionResubmitted ActionKind = "resubmitted"
	ActionApproved    ActionKind = "approved"
	ActionDeclined    ActionKind = "declined"
	ActionRecalled    ActionKind = "recalled"
	ActionRestarted   ActionKind = "restarted"
	ActionCancelled   ActionKind = "cancelled"
	// Feature 017: what MAPP records by itself (actor.System).
	ActionAutoApproved ActionKind = "autoApproved" // a level approved automatically at submission
	ActionSkipped      ActionKind = "skipped"      // a level not needed at submission
	ActionReminded     ActionKind = "reminded"     // the level's approvers were reminded
	ActionEscalated    ActionKind = "escalated"    // the level was escalated to its backup approver
)

// Capacity is the capacity a decision was made in (017 FR-019, data-model §6).
type Capacity string

// The capacities of a decision. Decisions stored before 017 have none ("").
const (
	CapacityApprover Capacity = "approver" // a resolved approver of the level
	CapacityDelegate Capacity = "delegate" // the current delegate of a waiting approver
	CapacityBackup   Capacity = "backup"   // the level's backup approver, after escalation
	CapacityOnBehalf Capacity = "onBehalf" // someone allowed to decide on behalf of the waiting approvers
)

// Action is one approval history entry: one step of a round (table "approval_actions", data model §3).
// Written once, never changed or deleted, so it has no record.Meta (008-P4).
type Action struct {
	ID          string        `json:"id"` // "AH-1001"; its number orders entries of the same millisecond
	RoundID     string        `json:"roundId"`
	RoundNumber int           `json:"roundNumber"`
	ActivityID  string        `json:"activityId"` // redundant with the round, for direct lookups
	Action      ActionKind    `json:"action"`
	Level       int           `json:"level"`      // the level concerned; 0 for a restart of an approved activity
	LevelName   string        `json:"levelName"`  // copied from the round level; "" when Level is 0
	Approvers   []actor.Actor `json:"approvers"`  // the level's designated approvers; empty when Level is 0
	Actor       actor.Actor   `json:"actor"`      // who acted (cancelled: who deleted the activity)
	OnBehalfOf  []actor.Actor `json:"onBehalfOf"` // approved/declined by someone else than the approvers: the approvers
	At          time.Time     `json:"at"`
	Comment     string        `json:"comment"` // submit, approve, decline comment or recall reason
	Changes     []ChangeText  `json:"changes"` // restarted only: the changed values
	// Feature 017 (data-model §5): the capacity of a decision and, for a delegate's, the approver they acted
	// for; the rule of an autoApproved or skipped entry; the backup approver of an escalation; and the whole
	// days a reminded or escalated level had waited.
	Capacity     Capacity     `json:"capacity"`
	AsDelegateOf *actor.Actor `json:"asDelegateOf"`
	Rule         string       `json:"rule"`
	EscalatedTo  *actor.Actor `json:"escalatedTo"`
	WaitedDays   int          `json:"waitedDays"`
}

// ChangeText is one changed value of a restart, copied from the saving feature's log entry.
type ChangeText struct {
	Label   string `json:"label"`
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

// NotificationKind is what a notification is about.
type NotificationKind string

// The notification kinds.
const (
	NotifyApprovalNeeded NotificationKind = "approvalNeeded"
	NotifyLevelApproved  NotificationKind = "levelApproved"
	NotifyApproved       NotificationKind = "approved"
	NotifyDeclined       NotificationKind = "declined"
	NotifyRecalled       NotificationKind = "recalled"
	NotifyRestarted      NotificationKind = "restarted"
	NotifyAutoApproved   NotificationKind = "autoApproved" // 017: approved at submission, to the owner
	NotifyReminder       NotificationKind = "reminder"     // 017: a level has waited the reminder days
	NotifyEscalated      NotificationKind = "escalated"    // 017: a level was escalated to its backup approver
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

// Notification is an in-app message to one person (table "notifications", data model §4). Not a business
// record: no version, and marking it read writes no log entry (008-P9).
type Notification struct {
	ID           string           `json:"id"`          // "NT-1001"
	RecipientID  string           `json:"recipientId"` // never the actor of the action (FR-036)
	Kind         NotificationKind `json:"kind"`
	Title        string           `json:"title"` // fixed at creation
	Text         string           `json:"text"`  // one line naming the activity and the actor
	ActivityID   string           `json:"activityId"`
	ActivityName string           `json:"activityName"` // the activity's name at the time
	Actor        actor.Actor      `json:"actor"`        // who caused it
	At           time.Time        `json:"at"`
	Read         bool             `json:"read"`
	ReadAt       *time.Time       `json:"readAt"` // nil while unread
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

// WorkflowPage is the answer of GET /api/approval-workflows (ApprovalWorkflowPage).
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

// WorkflowDetails is the answer of GET /api/approval-workflows/{workflowId} (ApprovalWorkflowDetails).
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

// WorkflowInput is the body of POST /api/approval-workflows (ApprovalWorkflowCreate).
type WorkflowInput struct {
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	Levels         []LevelInput `json:"levels"`
	ReminderDays   *int         `json:"reminderDays"`   // nil = no reminders
	EscalationDays *int         `json:"escalationDays"` // nil = no escalation
}

// WorkflowUpdate is the body of PATCH /api/approval-workflows/{workflowId} (ApprovalWorkflowUpdate): only
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

// Decision is a decision on one level of a round (ApprovalDecision).
type Decision struct {
	Action       string        `json:"action"` // "approved" or "declined"
	Actor        actor.Actor   `json:"actor"`
	OnBehalfOf   []actor.Actor `json:"onBehalfOf"`
	At           time.Time     `json:"at"`
	Comment      string        `json:"comment"`
	Capacity     Capacity      `json:"capacity"`     // "" for decisions recorded before 017
	AsDelegateOf *actor.Actor  `json:"asDelegateOf"` // delegate decisions only
}

// The states of a level in a round's chain (ChainLevel.State).
const (
	LevelApproved     = "approved"
	LevelDeclined     = "declined"
	LevelWaiting      = "waiting"
	LevelNotReached   = "notReached"
	LevelNotDecided   = "notDecided"   // waiting when the round was recalled, restarted or cancelled
	LevelSkipped      = "skipped"      // 017: Not needed
	LevelAutoApproved = "autoApproved" // 017: Approved automatically
)

// ChainLevel is one level of a round as the Approval section shows it (ApprovalChainLevel), with what the
// rules decided, who was found to approve it and how far it got (017 FR-033).
type ChainLevel struct {
	Number        int            `json:"number"`
	Label         string         `json:"label"`
	Approvers     []PersonView   `json:"approvers"`
	State         string         `json:"state"`
	Decision      *Decision      `json:"decision"` // the decision that completed or declined it
	Mode          string         `json:"mode"`
	Source        string         `json:"source"`
	RuleText      string         `json:"ruleText"`
	Decisions     []Decision     `json:"decisions"` // every decision on the level, oldest first
	ApprovedCount int            `json:"approvedCount"`
	RequiredCount int            `json:"requiredCount"`
	Excluded      []ExcludedView `json:"excluded"`
	Backup        *PersonView    `json:"backup"`
	BackupUsed    bool           `json:"backupUsed"`
	WaitingSince  *time.Time     `json:"waitingSince"`
	RemindedAt    *time.Time     `json:"remindedAt"`
	EscalatedAt   *time.Time     `json:"escalatedAt"`
	EscalatedTo   *PersonView    `json:"escalatedTo"`
	NoApprover    bool           `json:"noApprover"`
	Advice        string         `json:"advice"`
}

// ExcludedView is a person left out of a level, and why (ApprovalExcludedPerson).
type ExcludedView struct {
	Person PersonView `json:"person"`
	Reason string     `json:"reason"`
}

// ChangeView is one changed value of a restart, with its sentence (ApprovalChange).
type ChangeView struct {
	Label   string `json:"label"`
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
	Text    string `json:"text"` // "<label> changed from <old> to <new>", "<label> added: <new>", …
}

// HistoryEntryView is one approval history entry as the API shows it (ApprovalHistoryEntry).
type HistoryEntryView struct {
	ID          string        `json:"id"`
	RoundNumber int           `json:"roundNumber"`
	Level       int           `json:"level"`
	LevelLabel  string        `json:"levelLabel"`
	Approvers   []PersonView  `json:"approvers"`
	Actor       actor.Actor   `json:"actor"`
	OnBehalfOf  []actor.Actor `json:"onBehalfOf"`
	Action      string        `json:"action"`
	At          time.Time     `json:"at"`
	Comment     string        `json:"comment"`
	Changes     []ChangeView  `json:"changes"`
	// Feature 017 (FR-034): a decision's capacity, the rule of an automatic entry, the backup approver of an
	// escalation and the days a reminded or escalated level had waited.
	Capacity     Capacity     `json:"capacity"`
	AsDelegateOf *actor.Actor `json:"asDelegateOf"`
	Rule         string       `json:"rule"`
	EscalatedTo  *actor.Actor `json:"escalatedTo"`
	WaitedDays   int          `json:"waitedDays"`
}

// RuleFactsView is the values the rules used at submission (ApprovalRuleFacts).
type RuleFactsView struct {
	PlannedCost   *money.Money    `json:"plannedCost"`
	ActivityType  *errs.RecordRef `json:"activityType"`
	BusinessUnits []OptionRef     `json:"businessUnits"`
}

// RoundWorkflow is the workflow of a round as it was at submission.
type RoundWorkflow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RoundView is a round as the API returns it (ApprovalRound).
type RoundView struct {
	ID           string             `json:"id"`
	Number       int                `json:"number"`
	Outcome      Outcome            `json:"outcome"`
	Workflow     RoundWorkflow      `json:"workflow"`
	SubmittedAt  time.Time          `json:"submittedAt"`
	SubmittedBy  actor.Actor        `json:"submittedBy"`
	Comment      string             `json:"comment"`
	EndedAt      *time.Time         `json:"endedAt"`
	CurrentLevel int                `json:"currentLevel"`
	LevelCount   int                `json:"levelCount"`
	Levels       []ChainLevel       `json:"levels"`
	History      []HistoryEntryView `json:"history"` // newest first
	// Feature 017: the values the rules used, the timings copied at submission, and whether it was approved
	// at submission.
	Facts          *RuleFactsView `json:"facts"`
	ReminderDays   *int           `json:"reminderDays"`
	EscalationDays *int           `json:"escalationDays"`
	AutoApproved   bool           `json:"autoApproved"`
}

// DeclineNote is the latest decline, shown while the activity is Declined (DeclineNote).
type DeclineNote struct {
	RoundNumber int           `json:"roundNumber"`
	Level       int           `json:"level"`
	LevelLabel  string        `json:"levelLabel"`
	Actor       actor.Actor   `json:"actor"`
	OnBehalfOf  []actor.Actor `json:"onBehalfOf"`
	At          time.Time     `json:"at"`
	Comment     string        `json:"comment"`
	Capacity    Capacity      `json:"capacity"`
	// AsDelegateOf is the approver a delegate declined for.
	AsDelegateOf *actor.Actor `json:"asDelegateOf"`
}

// ActionAvailability says whether an approval action can be used, and why not (ApprovalActionAvailability).
type ActionAvailability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// Viewer is the signed-in person's relation to the activity (ApprovalViewer). Feature 017 adds whether and
// in which capacity they may decide the current level, and why not (data-model §6).
type Viewer struct {
	IsOwner            bool         `json:"isOwner"`
	IsCurrentApprover  bool         `json:"isCurrentApprover"`
	OnBehalfOf         []PersonView `json:"onBehalfOf"` // the waiting approvers, for capacity onBehalf only
	CanDecide          bool         `json:"canDecide"`
	Capacity           *Capacity    `json:"capacity"`     // nil when they may not decide
	AsDelegateOf       []PersonView `json:"asDelegateOf"` // capacity delegate: every approver they act for
	CannotDecideReason string       `json:"cannotDecideReason"`
	IsSubmitter        bool         `json:"isSubmitter"`
}

// ActionSet is the availability of each approval action (ActivityApproval.actions).
type ActionSet struct {
	Submit   ActionAvailability `json:"submit"`
	Resubmit ActionAvailability `json:"resubmit"`
	Approve  ActionAvailability `json:"approve"`
	Decline  ActionAvailability `json:"decline"`
	Recall   ActionAvailability `json:"recall"`
}

// ActivityApproval is an activity's Approval section (ActivityApproval).
type ActivityApproval struct {
	ActivityID      string       `json:"activityId"`
	ActivityName    string       `json:"activityName"`
	ActivityVersion int          `json:"activityVersion"` // sent back with Submit
	Status          string       `json:"status"`
	DisplayStatus   string       `json:"displayStatus"`
	OwnerID         string       `json:"ownerId"`
	Owner           *PersonView  `json:"owner"` // the owner with their name now; nil without one (D-214)
	Workflow        *WorkflowRef `json:"workflow"`
	UpcomingLevels  []LevelView  `json:"upcomingLevels"`
	Rounds          []RoundView  `json:"rounds"` // newest first
	LatestDecline   *DeclineNote `json:"latestDecline"`
	Viewer          Viewer       `json:"viewer"`
	Actions         ActionSet    `json:"actions"`
}

// SubmitProblem is one problem that stops a submission (SubmitProblem).
type SubmitProblem struct {
	Field      string `json:"field"`
	Group      string `json:"group"` // "activity" or "lineItem"
	LineItemID string `json:"lineItemId"`
	Message    string `json:"message"`
}

// Submission is the answer of GET /api/activities/{activityId}/approval/submission (ApprovalSubmission).
type Submission struct {
	ActivityID            string          `json:"activityId"`
	ActivityName          string          `json:"activityName"`
	ActivityVersion       int             `json:"activityVersion"`
	Kind                  string          `json:"kind"` // "submit" or "resubmit"
	Available             bool            `json:"available"`
	Reason                string          `json:"reason"`
	RoundNumber           int             `json:"roundNumber"`
	Ready                 bool            `json:"ready"`
	Problems              []SubmitProblem `json:"problems"`
	Workflow              *WorkflowRef    `json:"workflow"`
	Levels                []LevelView     `json:"levels"`
	UnassignedPlannedCost *money.Money    `json:"unassignedPlannedCost"` // a positive one, as a note
	// Outcomes say what submitting would do with each level (017 FR-013); empty without a usable workflow.
	Outcomes []SubmissionLevel `json:"outcomes"`
	// ApprovedAutomatically: no level needs a decision, so confirming approves the activity (017 FR-014).
	ApprovedAutomatically bool `json:"approvedAutomatically"`
}

// SubmissionLevel is what a submission would do with one level (ApprovalSubmissionLevel).
type SubmissionLevel struct {
	Number     int            `json:"number"`
	Label      string         `json:"label"`
	Outcome    string         `json:"outcome"` // ResolutionDecide, ResolutionSkipped or ResolutionAutoApproved
	Approvers  []PersonView   `json:"approvers"`
	Source     string         `json:"source"`
	Mode       string         `json:"mode"`
	RuleText   string         `json:"ruleText"`
	Excluded   []ExcludedView `json:"excluded"`
	BackupUsed bool           `json:"backupUsed"`
	NoApprover bool           `json:"noApprover"`
	Advice     string         `json:"advice"`
}

// SubmitInput is the body of POST …/approval/submit (SubmitRequest).
type SubmitInput struct {
	Version *int   `json:"version"` // the activity version the person saw
	Comment string `json:"comment"`
}

// DecisionInput is the body of POST …/approval/approve and …/decline (DecisionRequest).
type DecisionInput struct {
	RoundID string `json:"roundId"`
	Level   *int   `json:"level"` // the level the person is deciding
	Comment string `json:"comment"`
}

// RecallInput is the body of POST …/approval/recall (RecallRequest).
type RecallInput struct {
	RoundID string `json:"roundId"`
	Reason  string `json:"reason"`
}

// RowRound is the round of an Approvals row (ApprovalRowRound).
type RowRound struct {
	ID           string       `json:"id"`
	Number       int          `json:"number"`
	Outcome      Outcome      `json:"outcome"`
	CurrentLevel int          `json:"currentLevel"`
	LevelCount   int          `json:"levelCount"`
	LevelLabel   string       `json:"levelLabel"`
	Approvers    []PersonView `json:"approvers"`
	// 017: the current level's mode and, on "all must approve" levels, how many have approved of how many.
	Mode          string `json:"mode"`
	ApprovedCount int    `json:"approvedCount"`
	RequiredCount int    `json:"requiredCount"`
}

// MyDecision is the signed-in person's latest decision on an activity (MyDecision).
type MyDecision struct {
	Action      string    `json:"action"`
	At          time.Time `json:"at"`
	RoundNumber int       `json:"roundNumber"`
	Level       int       `json:"level"`
}

// RowActivity is the activity of an Approvals row (ApprovalRow.activity).
type RowActivity struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	DisplayStatus string `json:"displayStatus"`
}

// Row is one activity of the Approvals page (ApprovalRow).
type Row struct {
	Activity        RowActivity      `json:"activity"`
	Path            []errs.RecordRef `json:"path"`
	Owner           *PersonView      `json:"owner"`
	PlannedCost     *money.Money     `json:"plannedCost"` // USD
	Workflow        WorkflowRef      `json:"workflow"`
	Round           RowRound         `json:"round"`
	SubmittedAt     time.Time        `json:"submittedAt"`
	WaitingSince    *time.Time       `json:"waitingSince"`
	StartDate       *date.Date       `json:"startDate"`
	StartDatePassed bool             `json:"startDatePassed"`
	MyDecision      *MyDecision      `json:"myDecision"` // decided tab only
	// Feature 017 (FR-031): in Waiting for me, the approvers the person acts for as delegate; the backup
	// approver of an escalated level; whether the level is overdue; the whole days it has waited (nil without
	// an open round); and whether nobody waiting can act.
	AsDelegateOf []PersonView `json:"asDelegateOf"`
	EscalatedTo  *PersonView  `json:"escalatedTo"`
	Overdue      bool         `json:"overdue"`
	WaitingDays  *int         `json:"waitingDays"`
	NoApprover   bool         `json:"noApprover"`
}

// RowPage is the answer of GET /api/approvals?tab= (ApprovalRowPage).
type RowPage struct {
	Items []Row `json:"items"`
	Total int   `json:"total"`
}

// The tabs of the Approvals page.
const (
	TabWaiting = "waiting"
	TabOpen    = "open"
	TabDecided = "decided"
)

// Counts are the summary cards of the Approvals page (ApprovalCounts).
type Counts struct {
	WaitingForMe      int `json:"waitingForMe"`
	AllOpen           int `json:"allOpen"`
	ApprovedThisMonth int `json:"approvedThisMonth"`
	DeclinedThisMonth int `json:"declinedThisMonth"`
}

// MissingFieldRef is an empty needed-to-submit field of the previewed activity (MissingFieldRef).
type MissingFieldRef struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// PreviewLineItem is a line item of the preview (PreviewLineItem).
type PreviewLineItem struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	PlannedCost money.Money `json:"plannedCost"` // in its own currency
	// PlannedCostDisplay is its USD planned cost as Activities.LineItems gives it; Preview answers it in the
	// reader's display currency (D-213). Nil when its currency has no rate.
	PlannedCostDisplay *money.Money `json:"plannedCostDisplay"`
	SpendType          string       `json:"spendType"`
	GLCode             string       `json:"glCode"`
	FinanceBudgetItem  string       `json:"financeBudgetItem"`
}

// PreviewActivity is the activity of the preview (ApprovalPreview.activity).
type PreviewActivity struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Status        string            `json:"status"`
	DisplayStatus string            `json:"displayStatus"`
	Missing       []MissingFieldRef `json:"missing"`
	Path          []errs.RecordRef  `json:"path"`
	Owner         *PersonView       `json:"owner"`
	StartDate     *date.Date        `json:"startDate"`
	EndDate       *date.Date        `json:"endDate"`
	Ongoing       bool              `json:"ongoing"`
	PlannedCost   *money.Money      `json:"plannedCost"`
	TypeName      string            `json:"typeName"`
	Description   string            `json:"description"`
}

// Preview is the answer of GET /api/approvals/activities/{activityId} (ApprovalPreview).
type Preview struct {
	Activity              PreviewActivity   `json:"activity"`
	LineItems             []PreviewLineItem `json:"lineItems"`
	UnassignedPlannedCost *money.Money      `json:"unassignedPlannedCost"`
	Approval              ActivityApproval  `json:"approval"`
}

// NotificationActivity is the activity of a notification, and whether it is still stored.
type NotificationActivity struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
}

// NotificationView is a notification as the API returns it (Notification).
type NotificationView struct {
	ID       string               `json:"id"`
	Kind     NotificationKind     `json:"kind"`
	Title    string               `json:"title"`
	Text     string               `json:"text"`
	Activity NotificationActivity `json:"activity"`
	Actor    actor.Actor          `json:"actor"`
	At       time.Time            `json:"at"`
	Read     bool                 `json:"read"`
}

// NotificationPage is the answer of GET /api/notifications (NotificationPage).
type NotificationPage struct {
	Items  []NotificationView `json:"items"`  // at most 50, newest first
	Total  int                `json:"total"`  // all of the person's notifications
	Unread int                `json:"unread"` // all unread ones
}

// ReadAll is the answer of POST /api/notifications/read-all (NotificationReadAll).
type ReadAll struct {
	Marked int `json:"marked"`
	Unread int `json:"unread"` // always 0 after the call
}

// RuleOptions is what a level's rule may name (ApprovalRuleOptions): activity types and business units for
// conditions, and the 014 roles that approve for role levels (017 FR-001, FR-005).
type RuleOptions struct {
	ActivityTypes []SettingRef `json:"activityTypes"`
	BusinessUnits []OptionRef  `json:"businessUnits"`
	Roles         []RoleRef    `json:"roles"`
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

// DelegationPage is the answer of GET /api/approval-delegations (ApprovalDelegationPage).
type DelegationPage struct {
	Items           []DelegationView `json:"items"`
	Total           int              `json:"total"`
	CanManageOthers bool             `json:"canManageOthers"` // the signed-in person holds workflows.manage
}

// DelegationInput is the body of POST /api/approval-delegations (ApprovalDelegationCreate): dates as texts,
// so that a wrong date is a field problem.
type DelegationInput struct {
	PersonID   string `json:"personId"` // "" = the signed-in person
	DelegateID string `json:"delegateId"`
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	Note       string `json:"note"`
}

// DelegationUpdate is the body of PATCH /api/approval-delegations/{delegationId} (ApprovalDelegationUpdate):
// only the fields to change, plus the version (D-203).
type DelegationUpdate struct {
	DelegateID *string `json:"delegateId"`
	StartDate  *string `json:"startDate"`
	EndDate    *string `json:"endDate"`
	Note       *string `json:"note"`
	Version    *int    `json:"version"`
}

// DelegationCancel is the body of POST /api/approval-delegations/{delegationId}/cancel.
type DelegationCancel struct {
	Version *int `json:"version"`
}

// The ageing buckets of the Monitoring tab (ApprovalAgeingBucket.bucket), in order.
const (
	BucketUnder1    = "under1"
	BucketDays1to2  = "days1to2"
	BucketDays3to5  = "days3to5"
	BucketDays6to10 = "days6to10"
	BucketOver10    = "over10"
)

// AgeingBucket is one bar of "How long approvals have waited" (ApprovalAgeingBucket).
type AgeingBucket struct {
	Bucket string `json:"bucket"`
	Label  string `json:"label"`
	Count  int    `json:"count"`
}

// WaitingByLevel is one row of "Waiting by level" (ApprovalLevelWaiting).
type WaitingByLevel struct {
	Workflow           RoundWorkflow `json:"workflow"`
	Level              int           `json:"level"`
	LevelLabel         string        `json:"levelLabel"`
	Waiting            int           `json:"waiting"`
	Overdue            int           `json:"overdue"`
	OldestWaitingSince time.Time     `json:"oldestWaitingSince"`
}

// MonitoringCards are the four summary cards of the Monitoring tab (ApprovalMonitoring.cards).
type MonitoringCards struct {
	Overdue               int     `json:"overdue"`
	Escalated             int     `json:"escalated"`
	AverageDecisionDays   *string `json:"averageDecisionDays"` // one decimal, "2.4"; nil when nothing was decided
	DecidedThisMonth      int     `json:"decidedThisMonth"`
	AutoApprovedThisMonth int     `json:"autoApprovedThisMonth"`
}

// Monitoring is the answer of GET /api/approvals/monitoring (ApprovalMonitoring).
type Monitoring struct {
	Cards             MonitoringCards  `json:"cards"`
	Ageing            []AgeingBucket   `json:"ageing"` // always the five buckets, in order
	ByLevel           []WaitingByLevel `json:"byLevel"`
	Overdue           []Row            `json:"overdue"` // oldest waiting first
	CanRunEscalations bool             `json:"canRunEscalations"`
}

// The actions of an escalation run's events (EscalationEvent.action).
const (
	EventReminded     = "reminded"
	EventEscalated    = "escalated"
	EventNotEscalated = "notEscalated"
)

// EscalationEvent is one reminder, escalation or level that could not be escalated (EscalationEvent).
type EscalationEvent struct {
	Activity    errs.RecordRef `json:"activity"`
	RoundNumber int            `json:"roundNumber"`
	Level       int            `json:"level"`
	Action      string         `json:"action"`
	To          []PersonView   `json:"to"` // reminded: who was told; escalated: the backup approver; else empty
	WaitedDays  int            `json:"waitedDays"`
}

// EscalationRun is the answer of POST /api/approvals/escalations/run (EscalationRunResult).
type EscalationRun struct {
	Reminded     int               `json:"reminded"`
	Escalated    int               `json:"escalated"`
	Events       []EscalationEvent `json:"events"`
	NotEscalated []EscalationEvent `json:"notEscalated"`
}
