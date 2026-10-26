package settings_test

import (
	"encoding/json"
	"testing"

	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// An update's money field tells "left out", "null" and an amount apart (D-203).
func TestOptionalMoney_LeftOutNullOrAmount_D203(t *testing.T) {
	t.Parallel()
	var update settings.ActivityTypeUpdate
	if err := json.Unmarshal([]byte(`{"version":3}`), &update); err != nil || update.DefaultPlannedCost.Set {
		t.Errorf("left out: %+v, %v", update.DefaultPlannedCost, err)
	}
	update = settings.ActivityTypeUpdate{}
	if err := json.Unmarshal([]byte(`{"defaultPlannedCost":null,"version":3}`), &update); err != nil ||
		!update.DefaultPlannedCost.Set || update.DefaultPlannedCost.Value != nil {
		t.Errorf("null: %+v, %v", update.DefaultPlannedCost, err)
	}
	update = settings.ActivityTypeUpdate{}
	err := json.Unmarshal([]byte(`{"defaultPlannedCost":{"amount":"25000","currency":"USD"},"version":3}`), &update)
	if err != nil || !update.DefaultPlannedCost.Set || *update.DefaultPlannedCost.Value != (money.Input{Amount: "25000", Currency: "USD"}) {
		t.Errorf("amount: %+v, %v", update.DefaultPlannedCost, err)
	}
}
