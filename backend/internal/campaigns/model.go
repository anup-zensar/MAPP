// Package campaigns holds campaigns and initiatives (feature 005): a campaign is a theme inside one plan,
// an initiative a group of similar work inside one campaign. Neither has a stored status (D-012): every
// figure is derived on read.
//
// The shared foundation wrote the data layer: the stored records and the API types (this file), the
// activity seam feature 006 implements (hooks.go) and the repository (repository.go, store_json.go).
// Feature 005 adds the service, the dependency interfaces with their adapters, the roll-ups, the history
// kinds and the handler. The package never imports plans or activities.
package campaigns

import (
	"northstar/internal/people"
	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/settings"
)

// Campaign is a campaign as stored in the table "campaigns" (data model §1).
type Campaign struct {
	ID          string                   `json:"id"`          // "C-1001"
	PlanID      string                   `json:"planId"`      // its plan; changed only by Move
	Name        string                   `json:"name"`        // 1–120 characters, unique in the plan ignoring case
	Description string                   `json:"description"` // 0–2,000 characters
	OwnerID     *string                  `json:"ownerId"`     // a person; nil when none
	StartDate   *date.Date               `json:"startDate"`   // both dates or neither
	EndDate     *date.Date               `json:"endDate"`
	Budget      *money.Money             `json:"budget"`     // USD, 0.00 or more; nil when none
	Attributes  settings.AttributeValues `json:"attributes"` // campaign-level Shell attribute values; never nil
	record.Meta
}

// Initiative is an initiative as stored in the table "initiatives" (data model §2). Its plan is its
// campaign's: an initiative does not store it.
type Initiative struct {
	ID          string                   `json:"id"`          // "I-1001"
	CampaignID  string                   `json:"campaignId"`  // its campaign; changed only by Move
	Name        string                   `json:"name"`        // 1–120 characters, unique in the campaign ignoring case
	Description string                   `json:"description"` // 0–2,000 characters
	OwnerID     *string                  `json:"ownerId"`
	StartDate   *date.Date               `json:"startDate"` // inside the campaign's dates when the campaign has both (R3)
	EndDate     *date.Date               `json:"endDate"`
	Budget      *money.Money             `json:"budget"`
	Attributes  settings.AttributeValues `json:"attributes"` // initiative-level Shell attribute values; never nil
	record.Meta
}

// The timings of a campaign or initiative compared with today (contract Campaign.timing; null without
// dates).
const (
	TimingNotStarted = "notStarted"
	TimingRunning    = "running"
	TimingFinished   = "finished"
)

// The approval progress states (004's ApprovalProgress values; campaigns does not import plans).
const (
	ProgressNoActivities = "noActivities"
	ProgressInProgress   = "inProgress"
	ProgressAllApproved  = "allApproved"
)

// The purposes of the destinations query.
const (
	PurposeMove = "move"
	PurposeCopy = "copy"
)

// The API types below are what the API returns and receives (contract schema in brackets, JSON names as
// the contract). Lists are never nil. Amounts in requests are money inputs, so problems become field
// messages; dates in requests are date.Date, so a wrong date is refused as an unreadable request (D-101).

// PlanRef is the plan a campaign or initiative belongs to (PlanRef).
type PlanRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Year int    `json:"year"`
}

// CampaignRef is the campaign an initiative belongs to, with its dates (CampaignRef).
type CampaignRef struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	StartDate *date.Date `json:"startDate"`
	EndDate   *date.Date `json:"endDate"`
}

// BudgetAndCost is budget flowing down and cost rolling up for one campaign or initiative, all USD
// (BudgetAndCost).
type BudgetAndCost struct {
	AllocatedToChildren  money.Money  `json:"allocatedToChildren"`
	RemainingForChildren *money.Money `json:"remainingForChildren"` // null without a budget; may be negative
	OverAllocatedBy      *money.Money `json:"overAllocatedBy"`
	PlannedCost          *money.Money `json:"plannedCost"`       // null when nothing below has one
	BudgetUsedPercent    *int         `json:"budgetUsedPercent"` // half up; null without budget, with 0.00 or without planned cost
	OverBudget           bool         `json:"overBudget"`
	LeftToPlan           *money.Money `json:"leftToPlan"` // null without a budget
}

// ApprovalSummary is the approval progress of the activities below with its counts (ApprovalSummary).
type ApprovalSummary struct {
	State    string `json:"state"` // ProgressNoActivities, ProgressInProgress or ProgressAllApproved
	Approved int    `json:"approved"`
	Total    int    `json:"total"`
}

// RecordCompleteness is the record's own missing needed-to-publish fields and how many records below are
// incomplete (RecordCompleteness, D-199).
type RecordCompleteness struct {
	Missing                    []settings.MissingField `json:"missing"` // ownerId "Owner", startDate "Dates", attributes.<id>
	ItemsBelowNeedingAttention int                     `json:"itemsBelowNeedingAttention"`
}

// GoalSums are the sums of the goals of the activities below; each null when no activity has it (GoalSums).
type GoalSums struct {
	INQ     *int64       `json:"inq"`
	MQL     *int64       `json:"mql"`
	Revenue *money.Money `json:"revenue"`
}

// CampaignCounts are the records below a campaign (CampaignCounts).
type CampaignCounts struct {
	Initiatives        int `json:"initiatives"`
	Activities         int `json:"activities"`
	ApprovedActivities int `json:"approvedActivities"`
	LineItems          int `json:"lineItems"`
}

// InitiativeCounts are the records below an initiative (InitiativeCounts).
type InitiativeCounts struct {
	Activities         int `json:"activities"`
	ApprovedActivities int `json:"approvedActivities"`
	LineItems          int `json:"lineItems"`
}

// InitiativeView is an initiative as the API returns it (Initiative).
type InitiativeView struct {
	ID                    string                   `json:"id"`
	PlanID                string                   `json:"planId"` // its campaign's plan (derived)
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
	ParentRemainingBudget *money.Money             `json:"parentRemainingBudget"` // the campaign's remaining budget
	BudgetAndCost         BudgetAndCost            `json:"budgetAndCost"`
	ApprovalProgress      ApprovalSummary          `json:"approvalProgress"`
	Timing                *string                  `json:"timing"` // null without dates
	Completeness          RecordCompleteness       `json:"completeness"`
	Goals                 GoalSums                 `json:"goals"`
	Counts                InitiativeCounts         `json:"counts"`
	record.Meta
}

// CampaignView is a campaign as the API returns it, with its initiatives (Campaign).
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
	OutsidePlanYear       bool                     `json:"outsidePlanYear"`       // R4 warning; never blocks
	ParentRemainingBudget *money.Money             `json:"parentRemainingBudget"` // the plan's remaining budget
	BudgetAndCost         BudgetAndCost            `json:"budgetAndCost"`
	ApprovalProgress      ApprovalSummary          `json:"approvalProgress"`
	Timing                *string                  `json:"timing"`
	Completeness          RecordCompleteness       `json:"completeness"`
	Goals                 GoalSums                 `json:"goals"`
	Counts                CampaignCounts           `json:"counts"`
	Initiatives           []InitiativeView         `json:"initiatives"` // by start date, then name; undated last
	record.Meta
}

// PlanCampaignTotals is the plan tree's totals row (PlanCampaignTotals).
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

// PlanCampaigns is the answer of GET /api/plans/{planId}/campaigns (PlanCampaigns).
type PlanCampaigns struct {
	Plan      PlanRef            `json:"plan"`
	PlanOwner *people.PersonRef  `json:"planOwner"` // the default owner of a new campaign (R2)
	Items     []CampaignView     `json:"items"`
	Total     int                `json:"total"`
	Totals    PlanCampaignTotals `json:"totals"`
}

// CampaignInput is the body of POST /api/plans/{planId}/campaigns (CampaignCreate).
type CampaignInput struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	OwnerID     *string                  `json:"ownerId"`
	StartDate   *date.Date               `json:"startDate"`
	EndDate     *date.Date               `json:"endDate"`
	Budget      *money.Input             `json:"budget"` // USD only
	Attributes  settings.AttributeValues `json:"attributes"`
}

// CampaignUpdate is the body of PATCH /api/campaigns/{campaignId} (CampaignUpdate): the whole form, null or
// "" clearing a field, and the version (D-203).
type CampaignUpdate struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	OwnerID     *string                  `json:"ownerId"`
	StartDate   *date.Date               `json:"startDate"`
	EndDate     *date.Date               `json:"endDate"`
	Budget      *money.Input             `json:"budget"`
	Attributes  settings.AttributeValues `json:"attributes"`
	Version     *int                     `json:"version"`
}

// InitiativeInput is the body of POST /api/campaigns/{campaignId}/initiatives (InitiativeCreate).
type InitiativeInput struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	OwnerID     *string                  `json:"ownerId"`
	StartDate   *date.Date               `json:"startDate"`
	EndDate     *date.Date               `json:"endDate"`
	Budget      *money.Input             `json:"budget"`
	Attributes  settings.AttributeValues `json:"attributes"`
}

// InitiativeUpdate is the body of PATCH /api/initiatives/{initiativeId} (InitiativeUpdate): the whole
// form; the campaign changes only with Move.
type InitiativeUpdate struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	OwnerID     *string                  `json:"ownerId"`
	StartDate   *date.Date               `json:"startDate"`
	EndDate     *date.Date               `json:"endDate"`
	Budget      *money.Input             `json:"budget"`
	Attributes  settings.AttributeValues `json:"attributes"`
	Version     *int                     `json:"version"`
}

// CampaignMove is the body of POST /api/campaigns/{campaignId}/move (CampaignMove).
type CampaignMove struct {
	PlanID  string `json:"planId"` // another plan of the same year
	Version *int   `json:"version"`
}

// InitiativeMove is the body of POST /api/initiatives/{initiativeId}/move (InitiativeMove).
type InitiativeMove struct {
	CampaignID string `json:"campaignId"` // another campaign of the same plan
	Version    *int   `json:"version"`
}

// CampaignCopy is the body of POST /api/campaigns/{campaignId}/copy (CampaignCopy).
type CampaignCopy struct {
	PlanID string `json:"planId"`
	Name   string `json:"name"`
}

// InitiativeCopy is the body of POST /api/initiatives/{initiativeId}/copy (InitiativeCopy).
type InitiativeCopy struct {
	CampaignID string `json:"campaignId"`
	Name       string `json:"name"`
}

// TypeNotAllowed is an activity whose type the destination plan's model does not allow (TypeNotAllowed).
type TypeNotAllowed struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	TypeName string `json:"typeName"`
}

// MoveConsequences is what changes below a campaign that lands in a plan with another model or business
// unit (MoveConsequences).
type MoveConsequences struct {
	ModelName           string           `json:"modelName"`
	ModelChanges        bool             `json:"modelChanges"`
	BusinessUnitChanges bool             `json:"businessUnitChanges"`
	TypesNotAllowed     []TypeNotAllowed `json:"typesNotAllowed"`
	GroupsHidden        []string         `json:"groupsHidden"`
	GroupsShown         []string         `json:"groupsShown"`
}

// Destination is a plan (for a campaign) or a campaign (for an initiative) to move or copy to
// (Destination).
type Destination struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Current           bool              `json:"current"`
	Available         bool              `json:"available"`
	UnavailableReason *string           `json:"unavailableReason"`
	PlanStatus        *string           `json:"planStatus"` // plans only: "draft" or "published"
	StartDate         *date.Date        `json:"startDate"`  // campaigns only
	EndDate           *date.Date        `json:"endDate"`
	SuggestedName     *string           `json:"suggestedName"` // copies only
	RemainingAfter    *money.Money      `json:"remainingAfter"`
	OverAllocatedBy   *money.Money      `json:"overAllocatedBy"`
	Consequences      *MoveConsequences `json:"consequences"`
}

// DestinationList is the answer of the destinations queries (DestinationList).
type DestinationList struct {
	Items []Destination `json:"items"`
	Total int           `json:"total"`
	Rule  string        `json:"rule"`
}
