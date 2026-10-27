package activities

import (
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// SplitLineItem is a line item the Split dialog lists (ActivitySplitLineItem).
type SplitLineItem struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	PlannedCost     money.Money `json:"plannedCost"`     // in the line item's currency
	PlannedCostPlan money.Money `json:"plannedCostPlan"` // in the plan currency
}

// SplitOptions are what the Split dialog offers (ActivitySplitOptions, FR-026).
type SplitOptions struct {
	ActivityID    string          `json:"activityId"`
	Version       int             `json:"version"`
	Status        Status          `json:"status"`
	PlannedCost   *money.Money    `json:"plannedCost"`
	PlanCurrency  string          `json:"planCurrency"`
	SuggestedName string          `json:"suggestedName"`
	NeedsRestart  bool            `json:"needsRestart"`
	LineItems     []SplitLineItem `json:"lineItems"`
	Destinations  []Destination   `json:"destinations"`
}

// SplitRequest is the body of the split preview and of the split (ActivitySplitRequest).
type SplitRequest struct {
	Name            string   `json:"name"`
	InitiativeID    string   `json:"initiativeId"`
	LineItemIDs     []string `json:"lineItemIds"`
	PlannedCost     *string  `json:"plannedCost"` // in the plan currency; null or "0" moves none
	RestartApproval bool     `json:"restartApproval"`
	Version         *int     `json:"version"`
}

// SplitSide is one activity as the split would leave it (ActivitySplitSide, FR-027).
type SplitSide struct {
	Name                  string       `json:"name"`
	InitiativeName        string       `json:"initiativeName"`
	PlannedCost           *money.Money `json:"plannedCost"`
	LineItemNames         []string     `json:"lineItemNames"`
	UnassignedPlannedCost *money.Money `json:"unassignedPlannedCost"` // null without a planned cost
	UnassignedState       string       `json:"unassignedState"`       // left, allAssigned, exceeded or noPlannedCost
	Status                Status       `json:"status"`
}

// SplitPreview is both activities as the split would leave them, and every problem it would answer
// (ActivitySplitPreview). A side is nil when it cannot be worked out.
type SplitPreview struct {
	Original *SplitSide        `json:"original"`
	Created  *SplitSide        `json:"created"`
	Problems []errs.FieldError `json:"problems"`
}

// SplitResult is the original and the new activity after a split (ActivitySplitResult).
type SplitResult struct {
	Activity    View `json:"activity"`
	NewActivity View `json:"newActivity"`
}

// The states of an unassigned planned cost (as feature 007's balance).
const (
	unassignedLeft          = "left"
	unassignedAllAssigned   = "allAssigned"
	unassignedExceeded      = "exceeded"
	unassignedNoPlannedCost = "noPlannedCost"
)
