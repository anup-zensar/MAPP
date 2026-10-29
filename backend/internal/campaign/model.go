// Package campaign holds campaigns and initiatives (feature 005): a campaign is a theme inside one plan,
// an initiative a group of similar work inside one campaign. Neither has a stored status (D-012): every
// figure is derived on read.
//
// The shared foundation wrote the data layer: the stored records and the API types (this file), the
// activity seam feature 006 implements (hooks.go) and the repository (store.go, store_json.go).
// Feature 005 adds the service, the dependency interfaces with their adapters, the roll-ups, the history
// kinds and the handler. The package never imports the plan or activity packages.
package campaign

import (
	"mapp/internal/overspend"
	"mapp/internal/people"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

type Campaign struct {
	ID            string                   `json:"id"`
	PlanID        string                   `json:"planId"`
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	OwnerID       *string                  `json:"ownerId"`
	StartDate     *date.Date               `json:"startDate"`
	EndDate       *date.Date               `json:"endDate"`
	Budget        *money.Money             `json:"budget"`
	Attributes    settings.AttributeValues `json:"attributes"`
	BudgetPhasing rollup.Phasing           `json:"budgetPhasing"`
	Overspend     overspend.Mode           `json:"overspend"`
	record.Meta
}

type Initiative struct {
	ID            string                   `json:"id"`
	CampaignID    string                   `json:"campaignId"`
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	OwnerID       *string                  `json:"ownerId"`
	StartDate     *date.Date               `json:"startDate"`
	EndDate       *date.Date               `json:"endDate"`
	Budget        *money.Money             `json:"budget"`
	Attributes    settings.AttributeValues `json:"attributes"`
	BudgetPhasing rollup.Phasing           `json:"budgetPhasing"`
	Overspend     overspend.Mode           `json:"overspend"`
	record.Meta
}

const (
	TimingNotStarted = "notStarted"
	TimingRunning    = "running"
	TimingFinished   = "finished"
)

const (
	ProgressNoActivities = "noActivities"
	ProgressInProgress   = "inProgress"
	ProgressAllApproved  = "allApproved"
)

type PlanRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Year int    `json:"year"`
}

type CampaignRef struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	StartDate *date.Date `json:"startDate"`
	EndDate   *date.Date `json:"endDate"`
}

type BudgetAndCost struct {
	AllocatedToChildren  money.Money  `json:"allocatedToChildren"`
	RemainingForChildren *money.Money `json:"remainingForChildren"`
	OverAllocatedBy      *money.Money `json:"overAllocatedBy"`
	PlannedCost          *money.Money `json:"plannedCost"`
	BudgetUsedPercent    *int         `json:"budgetUsedPercent"`
	OverBudget           bool         `json:"overBudget"`
	LeftToPlan           *money.Money `json:"leftToPlan"`
}

type ApprovalSummary struct {
	State    string `json:"state"`
	Approved int    `json:"approved"`
	Total    int    `json:"total"`
}

type RecordCompleteness struct {
	Missing                    []settings.MissingField `json:"missing"`
	ItemsBelowNeedingAttention int                     `json:"itemsBelowNeedingAttention"`
}

type GoalSums struct {
	INQ     *int64       `json:"inq"`
	MQL     *int64       `json:"mql"`
	Revenue *money.Money `json:"revenue"`
}

type CampaignCounts struct {
	Initiatives        int `json:"initiatives"`
	Activities         int `json:"activities"`
	ApprovedActivities int `json:"approvedActivities"`
	LineItems          int `json:"lineItems"`
}

type InitiativeCounts struct {
	Activities         int `json:"activities"`
	ApprovedActivities int `json:"approvedActivities"`
	LineItems          int `json:"lineItems"`
}

type InitiativeView struct {
	ID                    string                   `json:"id"`
	PlanID                string                   `json:"planId"`
	CampaignID            string                   `json:"campaignId"`
	Name                  string                   `json:"name"`
	Description           string                   `json:"description"`
	OwnerID               *string                  `json:"ownerId"`
	Owner                 *people.PersonRef        `json:"owner"`
	StartDate             *date.Date               `json:"startDate"`
	EndDate               *date.Date               `json:"endDate"`
	Budget                *money.Money             `json:"budget"`
	Attributes            settings.AttributeValues `json:"attributes"`
	Plan                  PlanRef                  `json:"plan"`
	Campaign              CampaignRef              `json:"campaign"`
	ParentRemainingBudget *money.Money             `json:"parentRemainingBudget"`
	BudgetAndCost         BudgetAndCost            `json:"budgetAndCost"`
	Display               DisplayAmounts           `json:"display"`
	ApprovalProgress      ApprovalSummary          `json:"approvalProgress"`
	Timing                *string                  `json:"timing"`
	Completeness          RecordCompleteness       `json:"completeness"`
	Goals                 GoalSums                 `json:"goals"`
	Counts                InitiativeCounts         `json:"counts"`
	Access                *authz.ContainerAccess   `json:"access,omitempty"`
	BudgetPhasing         rollup.PhasingView       `json:"budgetPhasing"`
	Overspend             overspend.Control        `json:"overspend"`
	record.Meta
}

type CampaignView struct {
	ID                    string                   `json:"id"`
	PlanID                string                   `json:"planId"`
	Name                  string                   `json:"name"`
	Description           string                   `json:"description"`
	OwnerID               *string                  `json:"ownerId"`
	Owner                 *people.PersonRef        `json:"owner"`
	StartDate             *date.Date               `json:"startDate"`
	EndDate               *date.Date               `json:"endDate"`
	Budget                *money.Money             `json:"budget"`
	Attributes            settings.AttributeValues `json:"attributes"`
	Plan                  PlanRef                  `json:"plan"`
	OutsidePlanYear       bool                     `json:"outsidePlanYear"`
	ParentRemainingBudget *money.Money             `json:"parentRemainingBudget"`
	BudgetAndCost         BudgetAndCost            `json:"budgetAndCost"`
	Display               DisplayAmounts           `json:"display"`
	ApprovalProgress      ApprovalSummary          `json:"approvalProgress"`
	Timing                *string                  `json:"timing"`
	Completeness          RecordCompleteness       `json:"completeness"`
	Goals                 GoalSums                 `json:"goals"`
	Counts                CampaignCounts           `json:"counts"`
	Initiatives           []InitiativeView         `json:"initiatives"`
	Access                *authz.ContainerAccess   `json:"access,omitempty"`
	BudgetPhasing         rollup.PhasingView       `json:"budgetPhasing"`
	Overspend             overspend.Control        `json:"overspend"`
	record.Meta
}

type PlanCampaignTotals struct {
	Campaigns            int          `json:"campaigns"`
	Initiatives          int          `json:"initiatives"`
	Activities           int          `json:"activities"`
	AllocatedToCampaigns money.Money  `json:"allocatedToCampaigns"`
	PlanBudget           *money.Money `json:"planBudget"`
	RemainingPlanBudget  *money.Money `json:"remainingPlanBudget"`
	OverAllocatedBy      *money.Money `json:"overAllocatedBy"`
	PlannedCost          *money.Money `json:"plannedCost"`
}

type PlanCampaigns struct {
	Plan          PlanRef                  `json:"plan"`
	PlanOwner     *people.PersonRef        `json:"planOwner"`
	Items         []CampaignView           `json:"items"`
	Total         int                      `json:"total"`
	Totals        PlanCampaignTotals       `json:"totals"`
	DisplayTotals PlanCampaignTotals       `json:"displayTotals"`
	Access        *authz.PlanContentAccess `json:"access,omitempty"`
}

type CampaignInput struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	OwnerID     *string                  `json:"ownerId"`
	StartDate   *date.Date               `json:"startDate"`
	EndDate     *date.Date               `json:"endDate"`
	Budget      *money.Input             `json:"budget"`
	Attributes  settings.AttributeValues `json:"attributes"`
	Overspend   *string                  `json:"overspend"`
}

type CampaignUpdate struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	OwnerID     *string                  `json:"ownerId"`
	StartDate   *date.Date               `json:"startDate"`
	EndDate     *date.Date               `json:"endDate"`
	Budget      *money.Input             `json:"budget"`
	Attributes  settings.AttributeValues `json:"attributes"`
	Overspend   *string                  `json:"overspend"`
	Version     *int                     `json:"version"`
}

type InitiativeInput struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	OwnerID     *string                  `json:"ownerId"`
	StartDate   *date.Date               `json:"startDate"`
	EndDate     *date.Date               `json:"endDate"`
	Budget      *money.Input             `json:"budget"`
	Attributes  settings.AttributeValues `json:"attributes"`
	Overspend   *string                  `json:"overspend"`
}

type InitiativeUpdate struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	OwnerID     *string                  `json:"ownerId"`
	StartDate   *date.Date               `json:"startDate"`
	EndDate     *date.Date               `json:"endDate"`
	Budget      *money.Input             `json:"budget"`
	Attributes  settings.AttributeValues `json:"attributes"`
	Overspend   *string                  `json:"overspend"`
	Version     *int                     `json:"version"`
}
