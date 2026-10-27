package settings

import (
	"encoding/json"

	"mapp/internal/history"
	"mapp/internal/platform/money"
)

// planRef is a plan using the model modelID, as feature 004's source reports it.
func planRef(id, name, modelID string, values AttributeValues) Reference {
	return Reference{Record: history.RecordRef{Type: "plan", ID: id, Name: name}, Level: LevelPlan, PlanID: id,
		ModelID: modelID, Attributes: values}
}

// activityRef is an activity of the plan planID and the activity type typeID.
func activityRef(id, planID, typeID string, values AttributeValues) Reference {
	return Reference{Record: history.RecordRef{Type: "activity", ID: id, Name: "Activity " + id},
		Level: LevelActivity, PlanID: planID, ActivityID: id, ActivityTypeID: typeID, Attributes: values}
}

// lineItemRef is a line item of the activity activityID of the plan P-1001 with a planned cost in currency.
func lineItemRef(id, activityID, currency, cost string) Reference {
	amount, err := money.ParseAmount(cost)
	if err != nil {
		panic(err)
	}
	return Reference{Record: history.RecordRef{Type: "lineItem", ID: id, Name: "Line item " + id},
		Level: LevelLineItem, PlanID: "P-1001", ActivityID: activityID, Currency: currency, PlannedCost: &amount}
}

func inRecycleBin(ref Reference) Reference {
	ref.InBin = true
	return ref
}

// personRef is a person showing amounts in the display currency currency.
func personRef(id, currency string) Reference {
	return Reference{Record: history.RecordRef{Type: "person", ID: id, Name: "Person " + id}, Currency: currency}
}

// values builds attribute values from pairs of attribute id and JSON value.
func values(pairs ...string) AttributeValues {
	v := AttributeValues{}
	for i := 0; i+1 < len(pairs); i += 2 {
		v[pairs[i]] = json.RawMessage(pairs[i+1])
	}
	return v
}
