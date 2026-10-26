package activities

import (
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// Messages of the plan's "User-visible messages" table about goals.
const (
	wholeNumberOrMoreMessage = "Enter a whole number of 0 or more."
	revenueNegativeMessage   = "Revenue can't be negative."
)

// storedGoals are the goals as an activity stores them.
type storedGoals struct {
	INQ                  *int64
	SetLaterStagesMyself bool
	MQL                  *int64
	Revenue              *money.Money
}

// parseGoals reads the goals of a request (FR-018), adding problems under goals.inq, goals.mql and
// goals.revenue: whole numbers of 0 or more, revenue in USD and not negative. MQL and revenue are kept only
// when the person sets them (SetLaterStagesMyself); otherwise they are calculated when read.
func parseGoals(problems *errs.Problems, in GoalsInput) storedGoals {
	goals := storedGoals{INQ: in.INQ, SetLaterStagesMyself: in.SetLaterStagesMyself}
	if in.INQ != nil && *in.INQ < 0 {
		problems.Add("goals.inq", wholeNumberOrMoreMessage)
	}
	if !in.SetLaterStagesMyself {
		return goals
	}
	goals.MQL = in.MQL
	if in.MQL != nil && *in.MQL < 0 {
		problems.Add("goals.mql", wholeNumberOrMoreMessage)
	}
	if in.Revenue != nil {
		goals.Revenue = parseUSD(problems, "goals.revenue", *in.Revenue, revenueNegativeMessage)
	}
	return goals
}

// goalsOf returns the goals of a as shown (domain model §4.5): typed MQL and revenue when the person sets
// them, otherwise MQL, deals and revenue calculated from INQ with the model's current values; all empty
// without INQ.
func goalsOf(a Activity, model settings.Model) Goals {
	if a.SetLaterStagesMyself {
		return Goals{INQ: a.INQ, SetLaterStagesMyself: true, MQL: a.MQL, Revenue: a.Revenue}
	}
	goals := Goals{INQ: a.INQ, Calculated: true}
	if a.INQ == nil {
		return goals
	}
	mql, deals, revenue := waterfall(*a.INQ, model)
	goals.MQL, goals.Deals, goals.Revenue = &mql, &deals, revenue
	return goals
}

// waterfall works out MQL, deals and revenue from inq with the model (domain model §4.5): MQL = INQ→MQL
// rate of INQ and deals = MQL→deal rate of MQL, both rounded half-up to whole numbers; revenue = deals ×
// average deal size, in USD (nil when too large to keep). 250,000 with Lubricants 1.0 (2 %, 5 %,
// 10,000.00 USD) → 5,000, 250, 2,500,000.00 USD. Feature 002's settings.Waterfall does the same.
func waterfall(inq int64, model settings.Model) (mql, deals int64, revenue *money.Money) {
	mql = model.InqToMqlRate.OfWhole(inq)
	deals = model.MqlToDealRate.OfWhole(mql)
	amount, err := model.AverageDealSize.Amount.MulInt(deals)
	if err != nil {
		return mql, deals, nil
	}
	usd := money.USD(amount)
	return mql, deals, &usd
}
