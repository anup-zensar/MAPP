package campaign

import (
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
)

// amountOf returns the amount of a budget; nil without one.
func amountOf(budget *money.Money) *money.Amount {
	if budget == nil {
		return nil
	}
	return &budget.Amount
}

// budgetAndCost returns the roll-up of package rollup as the API returns it.
func budgetAndCost(c rollup.BudgetAndCost) BudgetAndCost {
	return BudgetAndCost{
		AllocatedToChildren:  c.AllocatedToChildren,
		RemainingForChildren: c.RemainingForChildren,
		OverAllocatedBy:      c.OverAllocatedBy,
		PlannedCost:          c.PlannedCost,
		BudgetUsedPercent:    c.BudgetUsedPercent,
		OverBudget:           c.OverBudget,
		LeftToPlan:           c.LeftToPlan,
	}
}

// approvalSummary is the approval progress of approved out of total activities (FR-013).
func approvalSummary(approved, total int) ApprovalSummary {
	p := rollup.ApprovalProgress(approved, total)
	return ApprovalSummary{State: p.State, Approved: p.Approved, Total: p.Total}
}

// timing is a record's timing against today; nil without dates (FR-013).
func timing(start, end *date.Date, today date.Date) *string {
	t := rollup.Timing(start, end, today)
	if t == "" {
		return nil
	}
	return &t
}

// outsideYear reports R4: the dates start before 1 January or end after 31 December of year (FR-008).
func outsideYear(start, end *date.Date, year int) bool {
	return (start != nil && start.Before(date.YearStart(year))) || (end != nil && end.After(date.YearEnd(year)))
}
