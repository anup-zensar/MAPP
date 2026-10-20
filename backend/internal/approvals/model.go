// Package approvals holds activity approvals (feature 008): approval workflows (a Settings record), the
// approval rounds of activities with their history entries, and in-app notifications.
//
// The shared foundation wrote the data layer: the stored records and the API types (this file) and the
// repository (repository.go, store_json.go). Feature 008 adds the Activities port with its adapter, the
// service, the history kind and the handler. The package never imports activities.
package approvals

import (
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
)

// Workflow is an approval workflow: a named, ordered set of approval levels (table "approval_workflows",
// data model §1). Feature 006's seed writes the two demo workflows in this shape (D-103, D-201).
type Workflow struct {
	ID          string  `json:"id"`          // "WF-1001"
	Name        string  `json:"name"`        // 1–120 characters, unique among all workflows ignoring case
	Description string  `json:"description"` // 0–2,000 characters
	Active      bool    `json:"active"`
	Levels      []Level `json:"levels"` // at least one; the first is level 1
	record.Meta
}

// Level is one level of a workflow: a value inside it, without an id of its own (008-P11).
type Level struct {
	Name        string   `json:"name"`        // 0–60 characters; "" means "Level <n>"
	ApproverIDs []string `json:"approverIds"` // at least one person; any one of them approves the level (D-205)
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
	record.Meta
}

// RoundLevel is a round's copy of one workflow level, with its approvers' names at submission.
type RoundLevel struct {
	Number    int           `json:"number"` // 1-based
	Name      string        `json:"name"`   // "" means "Level <n>"
	Approvers []actor.Actor `json:"approvers"`
}

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
)

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

// LevelView is one level of a workflow as the API shows it (ApprovalWorkflowLevel).
type LevelView struct {
	Number    int          `json:"number"`
	Name      string       `json:"name"`
	Label     string       `json:"label"` // the name, or "Level <number>"
	Approvers []PersonView `json:"approvers"`
}

// WorkflowView is a workflow as the API returns it (ApprovalWorkflow).
type WorkflowView struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Active       bool        `json:"active"`
	Levels       []LevelView `json:"levels"`
	LevelsText   string      `json:"levelsText"` // "Level 1 · Carlos Mendes → Level 2 · Alex Morgan"
	UsedByCount  int         `json:"usedByCount"`
	WaitingCount int         `json:"waitingCount"`
	Warnings     []string    `json:"warnings"` // "Level 2 has no active approver"
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

// LevelInput is one level of a create or update (ApprovalWorkflowLevelInput).
type LevelInput struct {
	Name        string   `json:"name"`
	ApproverIDs []string `json:"approverIds"`
}

// WorkflowInput is the body of POST /api/approval-workflows (ApprovalWorkflowCreate).
type WorkflowInput struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Levels      []LevelInput `json:"levels"`
}

// WorkflowUpdate is the body of PATCH /api/approval-workflows/{workflowId} (ApprovalWorkflowUpdate): only
// the fields to change (D-203), Levels being the complete new list.
type WorkflowUpdate struct {
	Name        *string       `json:"name"`
	Description *string       `json:"description"`
	Levels      *[]LevelInput `json:"levels"`
	Version     *int          `json:"version"`
}

// StatusChange is the body of POST …/deactivate and …/activate (ApprovalWorkflowStatusChange).
type StatusChange struct {
	Version *int `json:"version"`
}

// Decision is the decision on one level of a round (ApprovalDecision).
type Decision struct {
	Action     string        `json:"action"` // "approved" or "declined"
	Actor      actor.Actor   `json:"actor"`
	OnBehalfOf []actor.Actor `json:"onBehalfOf"`
	At         time.Time     `json:"at"`
	Comment    string        `json:"comment"`
}

// The states of a level in a round's chain (ChainLevel.State).
const (
	LevelApproved   = "approved"
	LevelDeclined   = "declined"
	LevelWaiting    = "waiting"
	LevelNotReached = "notReached"
	LevelNotDecided = "notDecided" // waiting when the round was recalled, restarted or cancelled
)

// ChainLevel is one level of a round as the Approval section shows it (ApprovalChainLevel).
type ChainLevel struct {
	Number    int          `json:"number"`
	Label     string       `json:"label"`
	Approvers []PersonView `json:"approvers"`
	State     string       `json:"state"`
	Decision  *Decision    `json:"decision"`
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
}

// ActionAvailability says whether an approval action can be used, and why not (ApprovalActionAvailability).
type ActionAvailability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// Viewer is the signed-in person's relation to the activity (ApprovalViewer).
type Viewer struct {
	IsOwner           bool         `json:"isOwner"`
	IsCurrentApprover bool         `json:"isCurrentApprover"`
	OnBehalfOf        []PersonView `json:"onBehalfOf"`
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
