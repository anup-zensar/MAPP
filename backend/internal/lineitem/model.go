// Package lineitem holds line items (feature 007): the costs of an activity, each in its own currency, spread
// over the months, coded to finance, and the SWBS code GSAP returns for them (feature 030).
package lineitem

import (
	"mapp/internal/people"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

type SpreadMethod string

const (
	SpreadEvenly     SpreadMethod = "evenly"
	SpreadFirstMonth SpreadMethod = "firstMonth"
	SpreadLastMonth  SpreadMethod = "lastMonth"
	SpreadCustom     SpreadMethod = "custom"
)

type LineItem struct {
	ID                  string                   `json:"id"`
	ActivityID          string                   `json:"activityId"`
	Name                string                   `json:"name"`
	OwnerID             string                   `json:"ownerId"`
	Description         string                   `json:"description"`
	Currency            string                   `json:"currency"`
	PlannedCost         money.Amount             `json:"plannedCost"`
	Spread              Spread                   `json:"spread"`
	Attributes          settings.AttributeValues `json:"attributes"`
	ContractedAmount    *money.Amount            `json:"contractedAmount"`
	RequestSwbs         bool                     `json:"requestSwbs"`
	FinanceBudgetYear   *int                     `json:"financeBudgetYear"`
	FinanceBudgetItemID string                   `json:"financeBudgetItemId"`
	Swbs
	record.Meta
}

type Spread struct {
	Method SpreadMethod  `json:"method"`
	Months []MonthAmount `json:"months"` // custom only, sorted by month; [] otherwise (never nil)
}

type MonthAmount struct {
	Month  string       `json:"month"`
	Amount money.Amount `json:"amount"`
}

const (
	StateSpread    = "spread"
	StateNotSpread = "notSpread" // Evenly, First or Last month without activity dates (D-032)
)

const (
	BalanceLeft          = "left"
	BalanceAllAssigned   = "allAssigned"
	BalanceExceeded      = "exceeded"
	BalanceNoPlannedCost = "noPlannedCost"
)

const (
	SourceUnassignedBalance     = "unassignedBalance"
	SourceNothingLeftToAssign   = "nothingLeftToAssign"
	SourceNoActivityPlannedCost = "noActivityPlannedCost"
)

const NoDatesColumn = "noDates"

// The API types below follow the contract's schemas: lists are never nil, amounts sent are decimal strings.

type CurrencyView struct {
	Code       string               `json:"code"`
	Name       string               `json:"name"`
	Symbol     string               `json:"symbol"`
	RatePerUSD money.Rate           `json:"ratePerUsd"`
	Active     bool                 `json:"active"`
	RateDate   *date.Date           `json:"rateDate"`
	RateSource *settings.RateSource `json:"rateSource"`
}

type FinanceBudgetItemRef struct {
	ID         string `json:"id"`
	Year       int    `json:"year"`
	BudgetName string `json:"budgetName"`
	Name       string `json:"name"`
	Code       string `json:"code"`
	Active     bool   `json:"active"`
}

type SpreadMonth struct {
	Month                string      `json:"month"`
	Amount               money.Money `json:"amount"`
	AmountUSD            money.Money `json:"amountUsd"`
	OutsideActivityDates bool        `json:"outsideActivityDates"`
	AmountPlan           money.Money `json:"-"`
}

type SpreadView struct {
	Method      SpreadMethod  `json:"method"`
	State       string        `json:"state"`
	Months      []SpreadMonth `json:"months"`
	CustomTotal *money.Money  `json:"customTotal"`
	AddsUp      bool          `json:"addsUp"`
}

type SpreadMonthInput struct {
	Month  string `json:"month"`
	Amount string `json:"amount"`
}

type SpreadInput struct {
	Method SpreadMethod       `json:"method"`
	Months []SpreadMonthInput `json:"months"`
}

type View struct {
	ID                 string                   `json:"id"`
	ActivityID         string                   `json:"activityId"`
	Name               string                   `json:"name"`
	Owner              *people.PersonRef        `json:"owner"`
	Description        string                   `json:"description"`
	Currency           CurrencyView             `json:"currency"`
	PlannedCost        money.Money              `json:"plannedCost"`
	PlannedCostUSD     money.Money              `json:"plannedCostUsd"`
	PlannedCostDisplay money.Money              `json:"plannedCostDisplay"`
	PlannedCostPlan    money.Money              `json:"plannedCostPlan"`
	Spread             SpreadView               `json:"spread"`
	Attributes         settings.AttributeValues `json:"attributes"`
	ContractedAmount   *money.Money             `json:"contractedAmount"`
	RequestSwbs        bool                     `json:"requestSwbs"`
	SwbsStatus         SwbsStatus               `json:"swbsStatus"`
	Swbs
	HXKey             string                  `json:"hxKey"`
	FinanceBudgetYear *int                    `json:"financeBudgetYear"`
	FinanceBudgetItem *FinanceBudgetItemRef   `json:"financeBudgetItem"`
	Missing           []settings.MissingField `json:"missing"`
	record.Meta
}

type ActivityView struct {
	ID                    string           `json:"id"`
	Name                  string           `json:"name"`
	Status                string           `json:"status"`
	RestartsApproval      bool             `json:"restartsApproval"` // deprecated: always false, for older clients
	StartDate             *date.Date       `json:"startDate"`
	EndDate               *date.Date       `json:"endDate"`
	Ongoing               bool             `json:"ongoing"`
	PlannedCost           *money.Money     `json:"plannedCost"`
	Path                  []errs.RecordRef `json:"path"`
	PlanYear              int              `json:"planYear"`
	DefaultSpreadMethod   SpreadMethod     `json:"defaultSpreadMethod"`
	DefaultFinanceBudget  *string          `json:"defaultFinanceBudget"`
	BusinessUnitOptionIDs []string         `json:"businessUnitOptionIds"`
}

// UnassignedBalance is the activity's planned cost minus its line items' planned costs, in the plan currency.
type UnassignedBalance struct {
	PlannedCost  *money.Money `json:"plannedCost"`
	Assigned     money.Money  `json:"assigned"`
	Balance      *money.Money `json:"balance"`
	State        string       `json:"state"`
	RoundingHint bool         `json:"roundingHint"`
}

type SpreadGridColumn struct {
	Key                  string  `json:"key"`
	Month                *string `json:"month"`
	OutsideActivityDates bool    `json:"outsideActivityDates"`
}

type SpreadGridRow struct {
	LineItemID string         `json:"lineItemId"`
	Name       string         `json:"name"`
	Cells      []*money.Money `json:"cells"`
	Total      money.Money    `json:"total"`
}

type SpreadGrid struct {
	Columns []SpreadGridColumn `json:"columns"`
	Rows    []SpreadGridRow    `json:"rows"`
	Totals  []*money.Money     `json:"totals"`
	Total   money.Money        `json:"total"`
}

type Section struct {
	Activity ActivityView          `json:"activity"`
	Items    []View                `json:"items"`
	Balance  UnassignedBalance     `json:"balance"`
	Grid     SpreadGrid            `json:"grid"`
	Access   *authz.LineItemAccess `json:"access,omitempty"`
}

type Details struct {
	LineItem View                  `json:"lineItem"`
	Activity ActivityView          `json:"activity"`
	Balance  UnassignedBalance     `json:"balance"`
	Access   *authz.LineItemAccess `json:"access,omitempty"`
}

type Defaults struct {
	Name                         string            `json:"name"`
	Owner                        *people.PersonRef `json:"owner"`
	Currency                     string            `json:"currency"`
	PlannedCost                  *money.Money      `json:"plannedCost"`
	PlannedCostSource            string            `json:"plannedCostSource"`
	SpreadMethod                 SpreadMethod      `json:"spreadMethod"`
	FinanceBudgetYear            int               `json:"financeBudgetYear"`
	FinanceBudgetItemID          *string           `json:"financeBudgetItemId"`
	FinanceBudgetItemPreselected bool              `json:"financeBudgetItemPreselected"`
	RequestSwbs                  bool              `json:"requestSwbs"`
}

type FormOptions struct {
	Activity           ActivityView           `json:"activity"`
	Balance            UnassignedBalance      `json:"balance"`
	Defaults           Defaults               `json:"defaults"`
	Currencies         []CurrencyView         `json:"currencies"`
	FinanceBudgetYears []int                  `json:"financeBudgetYears"`
	FinanceBudgetItems []FinanceBudgetItemRef `json:"financeBudgetItems"`
}

type Input struct {
	Name                string                   `json:"name"`
	OwnerID             *string                  `json:"ownerId"`
	Description         string                   `json:"description"`
	Currency            string                   `json:"currency"`
	PlannedCost         *string                  `json:"plannedCost"`
	Spread              SpreadInput              `json:"spread"`
	Attributes          settings.AttributeValues `json:"attributes"`
	ContractedAmount    *string                  `json:"contractedAmount"`
	RequestSwbs         bool                     `json:"requestSwbs"`
	FinanceBudgetYear   *int                     `json:"financeBudgetYear"`
	FinanceBudgetItemID *string                  `json:"financeBudgetItemId"`
	RestartApproval     bool                     `json:"restartApproval"` // deprecated and ignored, accepted for older clients
}

type Update struct {
	Name                string                   `json:"name"`
	OwnerID             *string                  `json:"ownerId"`
	Description         string                   `json:"description"`
	Currency            string                   `json:"currency"`
	PlannedCost         *string                  `json:"plannedCost"`
	Spread              SpreadInput              `json:"spread"`
	Attributes          settings.AttributeValues `json:"attributes"`
	ContractedAmount    *string                  `json:"contractedAmount"`
	RequestSwbs         bool                     `json:"requestSwbs"`
	FinanceBudgetYear   *int                     `json:"financeBudgetYear"`
	FinanceBudgetItemID *string                  `json:"financeBudgetItemId"`
	RestartApproval     bool                     `json:"restartApproval"`
	Version             *int                     `json:"version"`
}

type DuplicateInput struct {
	RestartApproval bool `json:"restartApproval"`
}
