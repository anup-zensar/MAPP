package lineitem

import (
	"fmt"

	"mapp/internal/settings"
)

const (
	ownerLabel             = "Owner"
	budgetYearLabel        = "Finance budget year"
	budgetItemLabel        = "Finance budget item"
	inactiveItemLabel      = "Choose an active finance budget item — '%s' is no longer active."
	spreadNotAddingUpLabel = "Spread doesn't add up to the planned cost"
)

// Missing lists, in this order, what stops the activity from being submitted (data model §3.6, FR-028); item
// is the linked finance budget item, nil when there is none. Never nil.
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
