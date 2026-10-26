// Package activities holds activities (feature 006): one piece of marketing work under an initiative, with
// its planned cost, goals, Shell attribute values and approval status.
//
// The shared foundation wrote the data layer: the stored activity and the API types (this file), the seams
// features 007 and 008 implement (hooks.go) and the repository (repository.go, store_json.go). Feature 006
// adds the service, the dependency interfaces with their adapters, the history kind and the handler.
package activities

import (
	"mapp/internal/people"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// Status is an activity's stored status (domain model §2.3, contract ActivityStatus).
type Status string

// The stored statuses. Created is set by create, copy, a re-approval save and a restore from the recycle
// bin; the others by feature 008's approval actions.
const (
	StatusCreated   Status = "created"
	StatusSubmitted Status = "submitted"
	StatusDeclined  Status = "declined"
	StatusApproved  Status = "approved"
)

// DisplayStatus is the status shown (contract ActivityDisplayStatus): an approved activity shows In progress
// from its start date and Complete after its end date (never when ongoing).
type DisplayStatus string

// The display statuses: the stored ones and two more for approved activities.
const (
	DisplayCreated    DisplayStatus = "created"
	DisplaySubmitted  DisplayStatus = "submitted"
	DisplayDeclined   DisplayStatus = "declined"
	DisplayApproved   DisplayStatus = "approved"
	DisplayInProgress DisplayStatus = "inProgress"
	DisplayComplete   DisplayStatus = "complete"
)

// Activity is an activity as stored in the table "activities" (data model §1). The campaign and the plan
// are read through the initiative, never stored here. Empty text means "not set"; derived values (goals
// when not typed, display status, unassigned planned cost, readiness) are computed on read (D-041).
type Activity struct {
	ID                   string                   `json:"id"`                 // "A-1001"
	InitiativeID         string                   `json:"initiativeId"`       // changed only by Move (same plan)
	Name                 string                   `json:"name"`               // 1–120 characters, unique in the initiative ignoring case
	ExternalName         string                   `json:"externalName"`       // 0–120 characters
	ActivityTypeID       string                   `json:"activityTypeId"`     // an activity type the plan's model allows (R7)
	OwnerID              string                   `json:"ownerId"`            // a person; "" = not set
	StartDate            *date.Date               `json:"startDate"`          // nil = not set
	EndDate              *date.Date               `json:"endDate"`            // nil = not set or ongoing
	Ongoing              bool                     `json:"ongoing"`            // once saved true it stays true (R6)
	PlannedCost          *money.Money             `json:"plannedCost"`        // in the plan currency (016 FR-011), 0.00 or more; nil = not set
	ApprovalWorkflowID   string                   `json:"approvalWorkflowId"` // an approval workflow of 008; "" = not set
	Description          string                   `json:"description"`        // 0–2,000 characters
	Attributes           settings.AttributeValues `json:"attributes"`         // activity-level Shell attribute values; never nil
	INQ                  *int64                   `json:"inq"`                // 0 or more; nil = not set
	SetLaterStagesMyself bool                     `json:"setLaterStagesMyself"`
	MQL                  *int64                   `json:"mql"`     // stored only when SetLaterStagesMyself
	Revenue              *money.Money             `json:"revenue"` // USD; stored only when SetLaterStagesMyself
	Status               Status                   `json:"status"`
	// SplitFromID is the activity this one was split from (016 FR-028, data-model §4.3); "" otherwise and in
	// older data. Set only by Split; never copied (FR-031).
	SplitFromID string `json:"splitFromId,omitempty"`
	record.Meta
}

// SplitLink is the other activity of a split, as the activity page shows it (SplitLink, 016 FR-031):
// "Split from 'UK launch event' (A-1011)", with Deleted when that activity is in the recycle bin.
type SplitLink struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	PlanID  string `json:"planId"`
	Deleted bool   `json:"deleted"`
}

// The codes of the ready-to-submit problems (data model §4.1); feature 007 adds the codes of line items.
const (
	ProblemMissingField           = "missingField"
	ProblemSplitIncomplete        = "splitIncomplete"
	ProblemDatesOutsideInitiative = "datesOutsideInitiative"
	ProblemTypeNotAllowed         = "typeNotAllowed"
	ProblemInactiveWorkflow       = "inactiveWorkflow"
	ProblemNegativeUnassigned     = "negativeUnassigned"
	ProblemUnassignedRemaining    = "unassignedRemaining" // advice, never blocks (D-204)
)

// The API types below are what the API returns and receives (contract schema in brackets, JSON names as
// the contract). Lists are never nil.

// TypeRef is the activity's type, with whether the plan's model allows it (ActivityTypeRef).
type TypeRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Allowed bool   `json:"allowed"`
}

// WorkflowRef is the activity's approval workflow (ActivityWorkflowRef).
type WorkflowRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// Goals are the activity's goals: INQ as typed; MQL, deals and revenue calculated from the plan's model
// unless SetLaterStagesMyself (ActivityGoals).
type Goals struct {
	INQ                  *int64       `json:"inq"`
	SetLaterStagesMyself bool         `json:"setLaterStagesMyself"`
	MQL                  *int64       `json:"mql"`
	Deals                *int64       `json:"deals"`
	Revenue              *money.Money `json:"revenue"`
	Calculated           bool         `json:"calculated"` // MQL and revenue come from the model
}

// GoalsInput are the goals of a create or update (ActivityGoalsInput): MQL and revenue are used only when
// SetLaterStagesMyself.
type GoalsInput struct {
	INQ                  *int64       `json:"inq"`
	SetLaterStagesMyself bool         `json:"setLaterStagesMyself"`
	MQL                  *int64       `json:"mql"`
	Revenue              *money.Input `json:"revenue"`
}

// LineItemTotals are the activity's line items in short (ActivityLineItemTotals; zero until 007).
type LineItemTotals struct {
	Count        int         `json:"count"`
	PlannedCost  money.Money `json:"plannedCost"`  // in the plan currency (016 FR-015)
	AutoSpread   int         `json:"autoSpread"`   // re-spread when the dates change
	CustomSpread int         `json:"customSpread"` // kept as typed when the dates change
}

// Completeness counts what the ready-to-submit check lists (ActivityCompleteness, D-199).
type Completeness struct {
	MissingFields             int `json:"missingFields"`
	LineItemsNeedingAttention int `json:"lineItemsNeedingAttention"`
}

// SubmitProblem is one item of the ready-to-submit check (ActivitySubmitProblem).
type SubmitProblem struct {
	Code         string  `json:"code"`  // one of the Problem… codes, or one of feature 007
	Field        string  `json:"field"` // the form field that fixes it, such as ownerId or attributes.<id>
	LineItemID   *string `json:"lineItemId"`
	LineItemName *string `json:"lineItemName"`
	Message      string  `json:"message"`
}

// SubmitCheck is everything that stops the activity from being submitted, all at once (ActivitySubmitCheck).
type SubmitCheck struct {
	Ready        bool            `json:"ready"`
	ProblemCount int             `json:"problemCount"` // items of Activity and LineItems; advice never counts
	Activity     []SubmitProblem `json:"activity"`
	LineItems    []SubmitProblem `json:"lineItems"`
	Advice       []SubmitProblem `json:"advice"`
}

// View is an activity as the API returns it (Activity).
type View struct {
	ID                    string                   `json:"id"`
	Name                  string                   `json:"name"`
	ExternalName          string                   `json:"externalName"`
	Plan                  errs.RecordRef           `json:"plan"`
	Campaign              errs.RecordRef           `json:"campaign"`
	Initiative            errs.RecordRef           `json:"initiative"`
	ActivityType          TypeRef                  `json:"activityType"`
	Owner                 *people.PersonRef        `json:"owner"`
	StartDate             *date.Date               `json:"startDate"`
	EndDate               *date.Date               `json:"endDate"`
	Ongoing               bool                     `json:"ongoing"`
	PlannedCost           *money.Money             `json:"plannedCost"`
	ApprovalWorkflow      *WorkflowRef             `json:"approvalWorkflow"`
	Description           string                   `json:"description"`
	Attributes            settings.AttributeValues `json:"attributes"`
	Goals                 Goals                    `json:"goals"`
	Status                Status                   `json:"status"`
	DisplayStatus         DisplayStatus            `json:"displayStatus"`
	UnassignedPlannedCost *money.Money             `json:"unassignedPlannedCost"` // null when there is neither a planned cost nor a line item
	Display               Display                  `json:"display"`               // the amounts above in the reader's display currency (D-220)
	LineItems             LineItemTotals           `json:"lineItems"`
	Completeness          Completeness             `json:"completeness"`
	Readiness             SubmitCheck              `json:"readiness"`
	Access                *authz.ActivityAccess    `json:"access,omitempty"` // what the reader may do (014 FR-011)
	SplitFrom             *SplitLink               `json:"splitFrom"`        // 016 FR-031: null when not split from another
	SplitInto             []SplitLink              `json:"splitInto"`        // 016 FR-031: the activities split off it; never nil
	record.Meta
}

// Summary is an activity as a row of the plan tree (ActivitySummary).
type Summary struct {
	ID                    string            `json:"id"`
	Name                  string            `json:"name"`
	InitiativeID          string            `json:"initiativeId"`
	ActivityType          TypeRef           `json:"activityType"`
	Owner                 *people.PersonRef `json:"owner"`
	StartDate             *date.Date        `json:"startDate"`
	EndDate               *date.Date        `json:"endDate"`
	Ongoing               bool              `json:"ongoing"`
	PlannedCost           *money.Money      `json:"plannedCost"`
	UnassignedPlannedCost *money.Money      `json:"unassignedPlannedCost"`
	Display               Display           `json:"display"` // the amounts above in the reader's display currency (D-220)
	Status                Status            `json:"status"`
	DisplayStatus         DisplayStatus     `json:"displayStatus"`
	Completeness          Completeness      `json:"completeness"`
	Version               int               `json:"version"`
	SplitFromID           *string           `json:"splitFromId"` // 016 FR-031: null when not split from another
}

// SummaryPage is the answer of GET /api/v1/plans/{planId}/activities (ActivitySummaryPage).
type SummaryPage struct {
	Items []Summary `json:"items"`
	Total int       `json:"total"`
	// Access is what the reader may do with the records of the plan (014 FR-011).
	Access *authz.PlanContentAccess `json:"access,omitempty"`
}

// Input is the body of POST /api/v1/initiatives/{initiativeId}/activities (ActivityCreate). Dates are texts so
// that a wrong date is a field problem ("Enter a valid date."); amounts are money inputs.
type Input struct {
	Name               string                   `json:"name"`
	ExternalName       string                   `json:"externalName"`
	ActivityTypeID     string                   `json:"activityTypeId"`
	OwnerID            *string                  `json:"ownerId"`
	StartDate          *string                  `json:"startDate"`
	EndDate            *string                  `json:"endDate"` // ignored when Ongoing
	Ongoing            bool                     `json:"ongoing"`
	PlannedCost        *money.Input             `json:"plannedCost"`
	ApprovalWorkflowID *string                  `json:"approvalWorkflowId"`
	Description        string                   `json:"description"`
	Attributes         settings.AttributeValues `json:"attributes"`
	Goals              GoalsInput               `json:"goals"`
}

// Update is the body of PATCH /api/v1/activities/{activityId} (ActivityUpdate): every editable field (null
// clears an optional one), the version, and whether the person confirmed that the activity needs approval
// again (RestartApproval, D-189).
type Update struct {
	Name               string                   `json:"name"`
	ExternalName       string                   `json:"externalName"`
	ActivityTypeID     string                   `json:"activityTypeId"`
	OwnerID            *string                  `json:"ownerId"`
	StartDate          *string                  `json:"startDate"`
	EndDate            *string                  `json:"endDate"`
	Ongoing            bool                     `json:"ongoing"`
	PlannedCost        *money.Input             `json:"plannedCost"`
	ApprovalWorkflowID *string                  `json:"approvalWorkflowId"`
	Description        string                   `json:"description"`
	Attributes         settings.AttributeValues `json:"attributes"`
	Goals              GoalsInput               `json:"goals"`
	Version            *int                     `json:"version"`
	RestartApproval    bool                     `json:"restartApproval"`
}

// MoveInput is the body of POST /api/v1/activities/{activityId}/move (ActivityMove).
type MoveInput struct {
	InitiativeID string `json:"initiativeId"` // another initiative of the same plan
	Version      *int   `json:"version"`
}

// CopyInput is the body of POST /api/v1/activities/{activityId}/copy (ActivityCopy).
type CopyInput struct {
	InitiativeID string `json:"initiativeId"` // its own initiative or another of the same plan
}

// Destination is an initiative an activity can be moved or copied to (ActivityDestination).
type Destination struct {
	Campaign    errs.RecordRef `json:"campaign"`
	Initiative  errs.RecordRef `json:"initiative"`
	StartDate   *date.Date     `json:"startDate"`
	EndDate     *date.Date     `json:"endDate"`
	Current     bool           `json:"current"`
	MoveProblem *string        `json:"moveProblem"` // null when it can be moved here
	CopyProblem *string        `json:"copyProblem"` // null when it can be copied here
}

// DestinationPage is the answer of GET /api/v1/activities/{activityId}/destinations (ActivityDestinationPage).
type DestinationPage struct {
	Items []Destination `json:"items"`
	Total int           `json:"total"`
}

// PlanFacts is the plan of the form (ActivityPlanFacts).
type PlanFacts struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Year   int    `json:"year"`
	Status string `json:"status"` // the plan's status: "draft" or "published"
}

// InitiativeFacts is the initiative of the form (ActivityInitiativeFacts).
type InitiativeFacts struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Owner     *people.PersonRef `json:"owner"`
	StartDate *date.Date        `json:"startDate"`
	EndDate   *date.Date        `json:"endDate"`
}

// ModelFacts is the plan's model as the goals of the form need it (ActivityModelFacts).
type ModelFacts struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	InqToMqlRate    money.Percent `json:"inqToMqlRate"`
	MqlToDealRate   money.Percent `json:"mqlToDealRate"`
	AverageDealSize money.Money   `json:"averageDealSize"`
	Active          bool          `json:"active"`
}

// BusinessUnit is one business unit the plan includes (ActivityBusinessUnit).
type BusinessUnit struct {
	ID    string `json:"id"` // the option id of the plan-level Business unit attribute
	Label string `json:"label"`
}

// FormContext is the plan context of the activity form (ActivityFormContext).
type FormContext struct {
	Plan          PlanFacts       `json:"plan"`
	Campaign      errs.RecordRef  `json:"campaign"`
	Initiative    InitiativeFacts `json:"initiative"`
	Model         ModelFacts      `json:"model"`
	BusinessUnits []BusinessUnit  `json:"businessUnits"`
	LatestEndDate date.Date       `json:"latestEndDate"` // 31 December of the plan's year + 10 (R5)
}

// TypeOption is an activity type the form offers (ActivityTypeOption).
type TypeOption struct {
	ID                 string       `json:"id"`
	Name               string       `json:"name"`
	DefaultPlannedCost *money.Money `json:"defaultPlannedCost"` // 016 FR-016: converted into the plan currency
	Active             bool         `json:"active"`
	Allowed            bool         `json:"allowed"`
	// DefaultPlannedCostUSD is the type's default as kept in Settings, for the form's note "Default 5,000.00 USD
	// = 3,950.00 GBP at today's rate" (016 FR-016).
	DefaultPlannedCostUSD *money.Money `json:"defaultPlannedCostUsd"`
}

// WorkflowLevel is one level of a workflow the form offers (ActivityWorkflowLevel).
type WorkflowLevel struct {
	Name      string             `json:"name"`
	Approvers []people.PersonRef `json:"approvers"`
}

// WorkflowOption is an approval workflow the form offers (ActivityWorkflowOption).
type WorkflowOption struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Active bool            `json:"active"`
	Levels []WorkflowLevel `json:"levels"`
}

// Defaults are what a new activity starts with (ActivityDefaults).
type Defaults struct {
	Owner              *people.PersonRef `json:"owner"` // the initiative's owner, when active
	StartDate          *date.Date        `json:"startDate"`
	EndDate            *date.Date        `json:"endDate"`
	ApprovalWorkflowID *string           `json:"approvalWorkflowId"` // when exactly one workflow is active
}

// FormView is the answer of the two form queries (ActivityForm): Defaults only for a new activity.
type FormView struct {
	Context           FormContext                   `json:"context"`
	ActivityTypes     []TypeOption                  `json:"activityTypes"`
	ApprovalWorkflows []WorkflowOption              `json:"approvalWorkflows"`
	Attributes        []settings.ShellAttributeView `json:"attributes"`
	ExistingNames     []string                      `json:"existingNames"`
	Defaults          *Defaults                     `json:"defaults"`
}
