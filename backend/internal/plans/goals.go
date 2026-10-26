package plans

import (
	"math/big"

	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// Why a goal stage can't be worked out from the model (plan "User-visible messages"). Only working backwards
// divides, so only a goal typed as MQL or Revenue can meet a rate or deal size of 0.
const (
	reasonInqToMqlRate  = "The model's INQ→MQL rate is 0, so this can't be worked out."
	reasonMqlToDealRate = "The model's MQL→deal rate is 0, so this can't be worked out."
	reasonDealSize      = "The model's average deal size is 0, so this can't be worked out."
)

// tenThousand turns a rate in hundredths of a percent into a fraction: 2.00 % is 200 ÷ 10,000.
const tenThousand = 10_000

// waterfall is a plan goal at every stage (domain model §4.5, D-075), revenue in USD. A stage is nil when it
// can't be worked out; Reasons then says why.
type waterfall struct {
	INQ, MQL, Deals *int64
	Revenue         *money.Amount
	Reasons         GoalReasons
}

// planWaterfall works out the goal of every stage from the goal typed at stage (amount, as stored: a whole
// number for INQ and MQL, a USD amount for Revenue) and the model's rates and average deal size: forwards
// from INQ, partly backwards from MQL, backwards from Revenue (plan "Goal waterfall"). Every stage is
// rounded half up to a whole number. Without a goal amount every stage is nil, with no reason.
//
//	INQ 250,000 with 2 %, 5 % and 10,000.00 → MQL 5,000 → deals 250 → revenue 2,500,000.00
func planWaterfall(stage GoalStage, amount string, m settings.Model) waterfall {
	r1, r2 := m.InqToMqlRate.Hundredths(), m.MqlToDealRate.Hundredths()
	dealSize := m.AverageDealSize.Amount.Cents()
	var w waterfall
	switch stage {
	case StageINQ:
		inq, ok := wholeNumber(amount)
		if !ok {
			return w
		}
		w.INQ = &inq
		w.MQL = ratio(inq, r1, tenThousand)
		w.Deals = ratioOf(w.MQL, r2, tenThousand)
	case StageMQL:
		mql, ok := wholeNumber(amount)
		if !ok {
			return w
		}
		w.MQL = &mql
		w.INQ = ratio(mql, tenThousand, r1)
		if r1 == 0 {
			w.Reasons.INQ = reasonInqToMqlRate
		}
		w.Deals = ratio(mql, r2, tenThousand)
	case StageRevenue:
		revenue, err := money.ParseAmount(amount)
		if err != nil {
			return w
		}
		w.Revenue = &revenue
		w.Deals = ratio(revenue.Cents(), 1, dealSize)
		w.MQL = ratioOf(w.Deals, tenThousand, r2)
		w.INQ = ratioOf(w.MQL, tenThousand, r1)
		// A stage worked out from a stage that could not be worked out gets that stage's reason.
		switch {
		case dealSize == 0:
			w.Reasons = GoalReasons{Deals: reasonDealSize, MQL: reasonDealSize, INQ: reasonDealSize}
		case r2 == 0:
			w.Reasons = GoalReasons{MQL: reasonMqlToDealRate, INQ: reasonMqlToDealRate}
		case r1 == 0:
			w.Reasons = GoalReasons{INQ: reasonInqToMqlRate}
		}
		return w
	default:
		return w
	}
	// Revenue = deals × average deal size, from INQ and MQL alike.
	if w.Deals != nil {
		if revenue, err := m.AverageDealSize.Amount.MulInt(*w.Deals); err == nil {
			w.Revenue = &revenue
		}
	}
	return w
}

// wholeNumber reads a stored INQ or MQL goal, and false when there is none.
func wholeNumber(text string) (int64, bool) {
	n, ok := new(big.Int).SetString(text, 10)
	if !ok || !n.IsInt64() {
		return 0, false
	}
	return n.Int64(), true
}

// ratio returns n × mul ÷ div rounded half up (n ≥ 0, mul ≥ 0), worked out exactly with math/big; nil when
// div is 0 or the result is too large for a whole number.
func ratio(n, mul, div int64) *int64 {
	if div <= 0 {
		return nil
	}
	quotient, remainder := new(big.Int).QuoRem(new(big.Int).Mul(big.NewInt(n), big.NewInt(mul)), big.NewInt(div), new(big.Int))
	if new(big.Int).Lsh(remainder, 1).Cmp(big.NewInt(div)) >= 0 { // the remainder is at least half of div
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() {
		return nil
	}
	result := quotient.Int64()
	return &result
}

// ratioOf is ratio for a stage that may itself be nil, which gives nil.
func ratioOf(n *int64, mul, div int64) *int64 {
	if n == nil {
		return nil
	}
	return ratio(*n, mul, div)
}

// plannedShare is how much of the plan goal the activities already plan, as a whole percent (FR-016):
// the activities' total of the goal stage × 100 ÷ the plan goal of that stage, rounded half up. It is nil
// without a plan goal or with a goal of 0. INQ 150,000 of 250,000 → 60.
func plannedShare(stage GoalStage, w waterfall, activities GoalTotals) *int {
	var goal *int64
	var planned int64
	switch stage {
	case StageINQ:
		goal, planned = w.INQ, activities.INQ
	case StageMQL:
		goal, planned = w.MQL, activities.MQL
	case StageRevenue:
		if w.Revenue != nil {
			cents := w.Revenue.Cents()
			goal = &cents
		}
		planned = activities.Revenue.Amount.Cents()
	}
	if goal == nil || *goal == 0 {
		return nil
	}
	share := ratio(planned, 100, *goal)
	if share == nil {
		return nil
	}
	percent := int(*share)
	return &percent
}

// goalsOf returns the plan's goals beside its activities' goals, amounts in the reader's display currency
// (FR-009, FR-016).
func goalsOf(p Plan, m settings.Model, activities GoalTotals, c converter) Goals {
	w := planWaterfall(p.GoalStage, p.GoalAmount, m)
	return Goals{
		Stage: p.GoalStage,
		Plan: GoalFigures{
			INQ: w.INQ, MQL: w.MQL, Deals: w.Deals, Revenue: c.optional(w.Revenue), Reasons: w.Reasons,
		},
		Activities:     GoalTotals{INQ: activities.INQ, MQL: activities.MQL, Revenue: c.of(activities.Revenue.Amount)},
		PlannedPercent: plannedShare(p.GoalStage, w, activities),
	}
}
