package lineitems

import (
	"slices"

	"northstar/internal/platform/money"
)

// roundingHintLimit is the largest balance, either way, that shows the rounding hint: 0.05 USD (D-145).
var roundingHintLimit = money.FromCents(5)

// Balance works out the activity's unassigned planned cost, Sys_Gen_Balance (data model §3.7, FR-002): the
// activity's planned cost minus the sum of its line items' USD planned costs usd, whose currencies are
// currencies.
//
//   - left: some planned cost is not assigned yet (advice only, Q-008);
//   - allAssigned: exactly 0.00 is left;
//   - exceeded: the line items cost more than the planned cost (blocks submitting, D-016);
//   - noPlannedCost: the activity has no planned cost, so there is no balance (D-144).
//
// The rounding hint shows when the balance is a few cents (at most 0.05 USD) either way and a line item uses
// another currency than USD (D-145).
func Balance(f ActivityFacts, usd []money.Amount, currencies []string) UnassignedBalance {
	assigned := money.Sum(usd...)
	result := UnassignedBalance{Assigned: money.USD(assigned), State: BalanceNoPlannedCost}
	if f.PlannedCost == nil {
		return result
	}
	planned := money.USD(*f.PlannedCost)
	balance := money.USD(f.PlannedCost.Sub(assigned))
	result.PlannedCost, result.Balance = &planned, &balance
	switch balance.Amount.Sign() {
	case 1:
		result.State = BalanceLeft
	case 0:
		result.State = BalanceAllAssigned
	default:
		result.State = BalanceExceeded
	}
	small := balance.Amount.Cmp(roundingHintLimit) <= 0 && balance.Amount.Neg().Cmp(roundingHintLimit) <= 0
	foreign := slices.ContainsFunc(currencies, func(code string) bool { return code != money.BaseCurrency })
	result.RoundingHint = !balance.Amount.IsZero() && small && foreign
	return result
}
