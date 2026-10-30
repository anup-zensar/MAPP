package dashboard

import (
	"context"

	"mapp/internal/allwork"
	"mapp/internal/approval"
	"mapp/internal/budget"
	"mapp/internal/financials"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/purchaserequest"
)

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
}

// Planning is All work's reads (010): the plans of a year in the person's scope with everything in them.
type Planning interface {
	Years(ctx context.Context) ([]int, error)
	Year(ctx context.Context, year int, today date.Date, columns []string) (allwork.YearData, error)
	Choices(ctx context.Context, personID string) (allwork.Choices, error)
}

type Plans interface {
	YearOf(ctx context.Context, planID string) (int, bool, error)
}

// Spending, Rounds and Purchases read only the records asked for, which come from the plans in scope (B5, D-356).
type Spending interface {
	SpendByMonth(ctx context.Context, a actor.Actor, lineItemIDs []string) (map[string]map[string]financials.Facts, error)
}

type Rounds interface {
	RoundSummaries(ctx context.Context, a actor.Actor, subjectIDs []string) ([]approval.RoundSummary, error)
}

type Purchases interface {
	Summaries(ctx context.Context, lineItemIDs []string) (purchaserequest.Summaries, error)
}

type GlobalBudgets interface {
	YearFigures(ctx context.Context, a actor.Actor, year int) ([]budget.Summary, error)
}

type Directory interface {
	Currencies(ctx context.Context, personID string) (Currencies, error)
	TeamRefs(ctx context.Context, teamIDs []string) (map[string]people.TeamRef, error)
}

// Currencies are the active currencies with their current rate in units per 1 USD, and the person's own.
type Currencies struct {
	Default string
	Items   []Currency
}

type Currency struct {
	Code, Name, Rate string
}

type Deps struct {
	Planning      Planning
	Plans         Plans
	Spending      Spending
	Rounds        Rounds
	Purchases     Purchases
	GlobalBudgets GlobalBudgets
	Directory     Directory
}
