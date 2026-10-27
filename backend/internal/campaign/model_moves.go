package campaign

import (
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

// The purposes of the destinations query.
const (
	PurposeMove = "move"
	PurposeCopy = "copy"
)

// CampaignMove is the body of POST /api/v1/campaigns/{campaignId}/move (CampaignMove).
type CampaignMove struct {
	PlanID  string `json:"planId"` // another plan of the same year
	Version *int   `json:"version"`
}

// InitiativeMove is the body of POST /api/v1/initiatives/{initiativeId}/move (InitiativeMove).
type InitiativeMove struct {
	CampaignID string `json:"campaignId"` // another campaign of the same plan
	Version    *int   `json:"version"`
}

// CampaignCopy is the body of POST /api/v1/campaigns/{campaignId}/copy (CampaignCopy).
type CampaignCopy struct {
	PlanID string `json:"planId"`
	Name   string `json:"name"`
}

// InitiativeCopy is the body of POST /api/v1/initiatives/{initiativeId}/copy (InitiativeCopy).
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
