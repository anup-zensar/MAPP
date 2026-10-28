package activity

import (
	"mapp/internal/overspend"
	"mapp/internal/people"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

type PlanFacts struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Year   int    `json:"year"`
	Status string `json:"status"`
}

type InitiativeFacts struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Owner     *people.PersonRef `json:"owner"`
	StartDate *date.Date        `json:"startDate"`
	EndDate   *date.Date        `json:"endDate"`
}

type ModelFacts struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	InqToMqlRate    money.Percent `json:"inqToMqlRate"`
	MqlToDealRate   money.Percent `json:"mqlToDealRate"`
	AverageDealSize money.Money   `json:"averageDealSize"`
	Active          bool          `json:"active"`
}

type BusinessUnit struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type FormContext struct {
	Plan          PlanFacts       `json:"plan"`
	Campaign      errs.RecordRef  `json:"campaign"`
	Initiative    InitiativeFacts `json:"initiative"`
	Model         ModelFacts      `json:"model"`
	BusinessUnits []BusinessUnit  `json:"businessUnits"`
	LatestEndDate date.Date       `json:"latestEndDate"`
}

type TypeOption struct {
	ID                    string       `json:"id"`
	Name                  string       `json:"name"`
	DefaultPlannedCost    *money.Money `json:"defaultPlannedCost"`
	Active                bool         `json:"active"`
	Allowed               bool         `json:"allowed"`
	DefaultPlannedCostUSD *money.Money `json:"defaultPlannedCostUsd"`
}

type WorkflowLevel struct {
	Name      string             `json:"name"`
	Approvers []people.PersonRef `json:"approvers"`
}

type WorkflowOption struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Active bool            `json:"active"`
	Levels []WorkflowLevel `json:"levels"`
}

type Defaults struct {
	Owner              *people.PersonRef `json:"owner"`
	StartDate          *date.Date        `json:"startDate"`
	EndDate            *date.Date        `json:"endDate"`
	ApprovalWorkflowID *string           `json:"approvalWorkflowId"`
}

type FormView struct {
	Context           FormContext                   `json:"context"`
	ActivityTypes     []TypeOption                  `json:"activityTypes"`
	ApprovalWorkflows []WorkflowOption              `json:"approvalWorkflows"`
	Attributes        []settings.ShellAttributeView `json:"attributes"`
	ExistingNames     []string                      `json:"existingNames"`
	Defaults          *Defaults                     `json:"defaults"`
	OverspendLimits   []overspend.Limit             `json:"overspendLimits"`
}
