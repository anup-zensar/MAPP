package activity

import (
	"mapp/internal/people"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

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
