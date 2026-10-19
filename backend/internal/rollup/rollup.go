// Package rollup defines, once, how the planning containers — plans, campaigns and initiatives — add up
// what is below them (005 plan "Roll-ups", decision D-090): budget flowing down, cost rolling up, approval
// progress, timing and goal sums. The functions are pure: no storage and no clock of their own. Every
// value is computed on read and never stored, so a change below shows on the next read (FR-012).
//
// All amounts are USD (budgets and planned costs are stored in USD), so sums are exact and need no
// rounding; the only rounding is the whole-percent budget used, half up.
package rollup

import (
	"math/big"

	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
)

// The approval progress states (contract ApprovalProgress).
const (
	NoActivities = "noActivities"
	InProgress   = "inProgress"
	AllApproved  = "allApproved"
)

// The timings of a record compared with today (contract Campaign.timing).
const (
	NotStarted = "notStarted"
	Running    = "running"
	Finished   = "finished"
)

// zero is 0.00 USD.
var zero = money.USD(money.Amount{})

// Sum adds the amounts that are present. It is nil when none is: nothing below has a planned cost, which
// is shown as "—" (0.00 is a real value).
func Sum(amounts []*money.Money) *money.Money {
	var total *money.Money
	for _, a := range amounts {
		if a == nil {
			continue
		}
		if total == nil {
			sum := zero
			total = &sum
		}
		total.Amount = total.Amount.Add(a.Amount)
	}
	return total
}

// Total adds the amounts that are present; 0.00 USD when none is (budgets allocated to children).
func Total(amounts []*money.Money) money.Money {
	if sum := Sum(amounts); sum != nil {
		return *sum
	}
	return zero
}

// Remaining is budget − allocated: what is left to give to the children. It is nil when there is no
// budget ("No budget to allocate from") and may be negative: over-allocating never blocks (D-015).
func Remaining(budget *money.Money, allocated money.Money) *money.Money {
	if budget == nil {
		return nil
	}
	left := money.USD(budget.Amount.Sub(allocated.Amount))
	return &left
}

// OverBy returns how much remaining is below 0.00 ("Over-allocated by …"), as a positive amount; nil when
// remaining is nil or not negative.
func OverBy(remaining *money.Money) *money.Money {
	if remaining == nil || remaining.Amount.Sign() >= 0 {
		return nil
	}
	over := money.USD(remaining.Amount.Neg())
	return &over
}

// BudgetUsedPercent is planned × 100 ÷ budget, rounded half up to a whole percent. It works on cents with
// integers only: (2 × planned¢ × 100 + budget¢) ÷ (2 × budget¢), in math/big so that no amount can
// overflow. It is nil without a budget, with a budget of 0.00, or without a planned cost (planned costs are
// never negative, so the percent never is either).
func BudgetUsedPercent(planned, budget *money.Money) *int {
	if planned == nil || budget == nil || budget.Amount.Sign() <= 0 || planned.Amount.Sign() < 0 {
		return nil
	}
	plannedCents, budgetCents := big.NewInt(planned.Amount.Cents()), big.NewInt(budget.Amount.Cents())
	twiceBudget := new(big.Int).Mul(budgetCents, big.NewInt(2))
	numerator := new(big.Int).Mul(plannedCents, big.NewInt(200))
	numerator.Add(numerator, budgetCents)
	percent := int(new(big.Int).Quo(numerator, twiceBudget).Int64())
	return &percent
}

// OverBudget reports planned > budget, compared exactly, so 100.4 % shows "100 %" in red. It is false when
// either is nil.
func OverBudget(planned, budget *money.Money) bool {
	return planned != nil && budget != nil && planned.Amount.Cmp(budget.Amount) > 0
}

// LeftToPlan is budget − planned cost (0.00 when there is none); nil without a budget. Negative when over
// budget.
func LeftToPlan(budget, planned *money.Money) *money.Money {
	if budget == nil {
		return nil
	}
	used := zero
	if planned != nil {
		used = *planned
	}
	return Remaining(budget, used)
}

// Progress is how far the activities below a record are approved (contract ApprovalSummary).
type Progress struct {
	State           string // NoActivities, InProgress or AllApproved
	Approved, Total int
}

// ApprovalProgress returns the progress of approved out of total activities: No activities without any,
// All approved when every one is, In progress otherwise — also "0 of 5" (domain model §2.2).
func ApprovalProgress(approved, total int) Progress {
	state := InProgress
	switch {
	case total == 0:
		state = NoActivities
	case approved == total:
		state = AllApproved
	}
	return Progress{State: state, Approved: approved, Total: total}
}

// Timing compares a record's dates with today (D-091): Not started before the start, Running from the start
// to the end (both included), Finished after the end. It is "" when a date is missing ("Timing appears once
// dates are set").
func Timing(start, end *date.Date, today date.Date) string {
	switch {
	case start == nil || end == nil:
		return ""
	case today.Before(*start):
		return NotStarted
	case today.After(*end):
		return Finished
	}
	return Running
}

// Goals are goal sums: INQ and MQL as whole numbers, revenue in USD; each nil when nothing below has it.
type Goals struct {
	INQ, MQL *int64
	Revenue  *money.Money
}

// AddGoals adds up goals value by value; a value stays nil when no goal has it.
func AddGoals(goals []Goals) Goals {
	var sum Goals
	revenues := make([]*money.Money, 0, len(goals))
	for _, g := range goals {
		sum.INQ = addCount(sum.INQ, g.INQ)
		sum.MQL = addCount(sum.MQL, g.MQL)
		revenues = append(revenues, g.Revenue)
	}
	sum.Revenue = Sum(revenues)
	return sum
}

// addCount returns total + n, where a nil n adds nothing and a nil total is "none yet".
func addCount(total, n *int64) *int64 {
	if n == nil {
		return total
	}
	sum := *n
	if total != nil {
		sum += *total
	}
	return &sum
}

// BudgetAndCost is budget flowing down and cost rolling up for one plan, campaign or initiative (contract
// BudgetAndCost).
type BudgetAndCost struct {
	AllocatedToChildren  money.Money
	RemainingForChildren *money.Money
	OverAllocatedBy      *money.Money
	PlannedCost          *money.Money
	BudgetUsedPercent    *int
	OverBudget           bool
	LeftToPlan           *money.Money
}

// Container combines the rules above for one record with budget, the budgets of its children and its
// planned cost (the sum of the planned costs below it).
func Container(budget *money.Money, childBudgets []*money.Money, plannedCost *money.Money) BudgetAndCost {
	allocated := Total(childBudgets)
	remaining := Remaining(budget, allocated)
	return BudgetAndCost{
		AllocatedToChildren:  allocated,
		RemainingForChildren: remaining,
		OverAllocatedBy:      OverBy(remaining),
		PlannedCost:          plannedCost,
		BudgetUsedPercent:    BudgetUsedPercent(plannedCost, budget),
		OverBudget:           OverBudget(plannedCost, budget),
		LeftToPlan:           LeftToPlan(budget, plannedCost),
	}
}
