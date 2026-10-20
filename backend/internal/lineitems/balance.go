package lineitems

import (
	"slices"

	"northstar/internal/platform/money"
)

// roundingHintLimit is the largest balance, either way, that shows the rounding hint: 0.05 in the plan
// currency (D-145, 016 data-model §3.1).
var roundingHintLimit = money.FromCents(5)

// Balance works out the activity's unassigned planned cost, Sys_Gen_Balance (data model §3.7, FR-002): the
// activity's planned cost minus the sum of its line items' planned costs in the plan currency planned (016
// FR-015), whose own currencies are currencies.
//
//   - left: some planned cost is not assigned yet (advice only, Q-008);
//   - allAssigned: exactly 0.00 is left;
//   - exceeded: the line items cost more than the planned cost (blocks submitting, D-016);
//   - noPlannedCost: the activity has no planned cost, so there is no balance (D-144).
//
// The rounding hint shows when the balance is a few cents (at most 0.05) either way and a line item uses
// another currency than the plan's (D-145).
func Balance(f ActivityFacts, planned []money.Amount, currencies []string) UnassignedBalance {
	code := f.planCurrency()
	assigned := money.Sum(planned...)
	result := UnassignedBalance{Assigned: money.Money{Amount: assigned, Currency: code}, State: BalanceNoPlannedCost}
	if f.PlannedCost == nil {
		return result
	}
	plannedCost := money.Money{Amount: *f.PlannedCost, Currency: code}
	balance := money.Money{Amount: f.PlannedCost.Sub(assigned), Currency: code}
	result.PlannedCost, result.Balance = &plannedCost, &balance
	switch balance.Amount.Sign() {
	case 1:
		result.State = BalanceLeft
	case 0:
		result.State = BalanceAllAssigned
	default:
		result.State = BalanceExceeded
	}
	small := balance.Amount.Cmp(roundingHintLimit) <= 0 && balance.Amount.Neg().Cmp(roundingHintLimit) <= 0
	foreign := slices.ContainsFunc(currencies, func(c string) bool { return c != code })
	result.RoundingHint = !balance.Amount.IsZero() && small && foreign
	return result
}
