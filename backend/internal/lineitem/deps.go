package lineitem

import (
	"cmp"
	"context"

	"mapp/internal/people"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// Deps are interfaces to the features line items read, to avoid import cycles (D-196); each method runs in ctx's transaction.
type Deps struct {
	Activities     Activities
	Currencies     Currencies
	FinanceBudgets FinanceBudgets
	Attributes     Attributes
	People         People
}

type Activities interface {
	Facts(ctx context.Context, activityID string) (ActivityFacts, bool, error)
	FactsOf(ctx context.Context, activityIDs []string) (map[string]ActivityFacts, error)
}

type ActivityFacts struct {
	ID, Name             string
	OwnerID              string
	Status               string
	StartDate, EndDate   *date.Date
	Ongoing              bool
	PlannedCost          *money.Amount
	Path                 []errs.RecordRef
	PlanYear             int
	DefaultSpread        SpreadMethod
	DefaultFinanceBudget string
	BusinessUnit         []string
	Access               authz.PlanFacts
	PlanCurrency         string
}

func (f ActivityFacts) planCurrency() string {
	return cmp.Or(f.PlanCurrency, money.BaseCurrency)
}

type Currencies interface {
	Currency(ctx context.Context, code string) (settings.Currency, bool, error)
	Currencies(ctx context.Context) ([]settings.Currency, error)
	Rates(ctx context.Context) (money.Rates, error)
	CurrentRates(ctx context.Context) ([]settings.RateInfo, error)
}

type BudgetItem struct {
	ID         string
	Year       int
	BudgetName string
	Name       string
	Code       string
	Active     bool
}

type FinanceBudgets interface {
	Item(ctx context.Context, id string) (BudgetItem, bool, error)
	Items(ctx context.Context) ([]BudgetItem, error)
}

type Attributes interface {
	// Check returns the Problems that block saving (fields "attributes.<id>") and the Values to store.
	Check(ctx context.Context, in settings.AttributeInput) (settings.AttributeResult, error)
	// Missing lists the visible, active, needed-to-submit attributes without a value, in Settings order (D-034).
	Missing(ctx context.Context, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error)
	Describe(ctx context.Context, field string, value any) (label, text string, err error)
}

type People interface {
	Person(ctx context.Context, id string) (people.PersonRef, bool, error)
	// DisplayRateOf gives USD at rate "1" to a person who has not chosen a display currency (D-213).
	DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error)
}
