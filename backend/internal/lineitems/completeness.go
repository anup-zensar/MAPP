package lineitems

import (
	"fmt"

	"northstar/internal/settings"
)

// The labels of the missing list (plan, "Completeness labels").
const (
	ownerLabel             = "Owner"
	budgetYearLabel        = "Finance budget year"
	budgetItemLabel        = "Finance budget item"
	inactiveItemLabel      = "Choose an active finance budget item — '%s' is no longer active."
	spreadNotAddingUpLabel = "Spread doesn't add up to the planned cost"
)

// Missing lists what still stops the line item's activity from being submitted, in this order (data model
// §3.6, FR-028): an empty owner; each visible, active, needed-to-submit Shell attribute without a value
// (attributeMissing, from feature 002); an empty finance budget year; an empty finance budget item; a linked
// finance budget item that is inactive (FR-016); a custom spread that does not add up (FR-019). item is the
// linked finance budget item, nil when there is none. An inactive currency, owner or option is a kept value,
// not a problem. The list is never nil.
func Missing(li LineItem, attributeMissing []settings.MissingField, item *BudgetItem, addsUp bool) []settings.MissingField {
	missing := []settings.MissingField{}
	if li.OwnerID == "" {
		missing = append(missing, settings.MissingField{Field: "ownerId", Label: ownerLabel})
	}
	missing = append(missing, attributeMissing...)
	if li.FinanceBudgetYear == nil {
		missing = append(missing, settings.MissingField{Field: "financeBudgetYear", Label: budgetYearLabel})
	}
	switch {
	case li.FinanceBudgetItemID == "":
		missing = append(missing, settings.MissingField{Field: "financeBudgetItemId", Label: budgetItemLabel})
	case item != nil && !item.Active:
		missing = append(missing, settings.MissingField{Field: "financeBudgetItemId", Label: fmt.Sprintf(inactiveItemLabel, item.Name)})
	}
	if !addsUp {
		missing = append(missing, settings.MissingField{Field: "spread", Label: spreadNotAddingUpLabel})
	}
	return missing
}

// SwbsStatusOf is the line item's SWBS request status (data model §3.5, FR-024): Not requested while
// Request SWBS code from GSAP is off; Waiting for activity approval until the activity is Approved; Ready to
// send to GSAP once it is (also while it shows In progress or Complete). Nothing is sent in this phase (D-008).
func SwbsStatusOf(li LineItem, f ActivityFacts) SwbsStatus {
	switch {
	case !li.RequestSwbs:
		return SwbsNotRequested
	case f.Status == "approved":
		return SwbsReadyToSend
	}
	return SwbsWaitingForApproval
}
