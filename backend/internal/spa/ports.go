package spa

import (
	"context"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

// The ports of spa: what it reads from other features. internal/app implements them over 004–007 (through All
// work's plan reader), 015 and 032; every method runs inside the caller's transaction, which ctx carries.

type Level string

const (
	LevelPlan       Level = "plan"
	LevelCampaign   Level = "campaign"
	LevelInitiative Level = "initiative"
	LevelActivity   Level = "activity"
	LevelLineItem   Level = "lineItem"
)

type Plan struct {
	ID     string
	Name   string
	Year   int
	Status string
	BCPIR  bool
	Budget *money.Amount
	Access authz.PlanFacts
}

// Record is a live campaign, initiative, activity or line item with its owning feature's values, amounts in USD.
type Record struct {
	Level        Level
	ID           string
	ParentID     string
	Order        int
	Name         string
	Owner        string
	Status       string
	ActivityType string
	StartDate    *date.Date
	Months       []string
	PlannedCost  *money.Amount
	Budget       *money.Amount
	Spread       []MonthAmount
	Linked       bool
	Values       map[string][]Value
}

type MonthAmount struct {
	Month string
	USD   money.Amount
}

// Value is one value of a dimension; Share is set for a % split.
type Value struct {
	Label string
	Share *money.Percent
}

// Dimension is a grouping of the planned cost, the default first; Channel marks Channel / audience.
type Dimension struct {
	Key     string
	Label   string
	Level   Level
	Channel bool
}

type Planning interface {
	Plan(ctx context.Context, planID string) (Plan, bool, error)
	Records(ctx context.Context, p Plan) ([]Record, error)
	Dimensions(ctx context.Context) ([]Dimension, error)
}

type Spend struct {
	Committed money.Amount
	Invoiced  money.Amount
	Actual    money.Amount
}

func (s Spend) plus(o Spend) Spend {
	return Spend{Committed: s.Committed.Add(o.Committed), Invoiced: s.Invoiced.Add(o.Invoiced), Actual: s.Actual.Add(o.Actual)}
}

type Finance interface {
	SeesFinance(ctx context.Context, a actor.Actor, planID string) (bool, error)
	SpendByMonth(ctx context.Context) (map[string]map[string]Spend, error)
	DataAsOf(ctx context.Context) (*time.Time, error)
}

type PurchaseRequest struct {
	ID     string
	Name   string
	Vendor string
	Status string
	Lines  []PurchaseLine
}

type PurchaseLine struct {
	LineItemID string
	USD        money.Amount
}

type Purchases interface {
	Drawn(ctx context.Context, lineItemIDs []string) ([]PurchaseRequest, error)
	CanRaise(ctx context.Context, p Plan) (authz.Availability, error)
}

type Displays interface {
	Display(ctx context.Context, personID string) (money.Display, error)
}

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
}

type Deps struct {
	Planning  Planning
	Finance   Finance
	Purchases Purchases
	Displays  Displays
}
