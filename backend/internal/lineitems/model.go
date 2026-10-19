// Package lineitems holds line items (feature 007): the costs of an activity, each in its own currency,
// spread over the months, coded to finance. Line items have no status of their own; they are approved
// with their activity.
//
// The shared foundation wrote the data layer: the stored line item and the API types (this file) and the
// repository (repository.go, store_json.go). Feature 007 adds the calculations, the service, the dependency
// interfaces with their adapters, the history kind and the handler. USD amounts, spreads of the Evenly,
// First and Last methods, the SWBS status and completeness are derived on read, never stored (D-041).
package lineitems

import (
	"northstar/internal/people"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/settings"
)

// SpreadMethod is how a line item's planned cost is spread over the activity's months (004's SpreadMethod).
type SpreadMethod string

// The spread methods.
const (
	SpreadEvenly     SpreadMethod = "evenly"
	SpreadFirstMonth SpreadMethod = "firstMonth"
	SpreadLastMonth  SpreadMethod = "lastMonth"
	SpreadCustom     SpreadMethod = "custom"
)

// LineItem is a line item as stored in the table "line_items" (data model §1).
type LineItem struct {
	ID                  string                   `json:"id"`          // "L-1001"
	ActivityID          string                   `json:"activityId"`  // never changes
	Name                string                   `json:"name"`        // 1–120 characters, unique in the activity ignoring case
	OwnerID             string                   `json:"ownerId"`     // "" when empty
	Description         string                   `json:"description"` // 0–2,000 characters
	Currency            string                   `json:"currency"`    // ISO code of a currency of feature 002
	PlannedCost         money.Amount             `json:"plannedCost"` // in Currency, 0.00 or more
	Spread              Spread                   `json:"spread"`
	Attributes          settings.AttributeValues `json:"attributes"`          // line-item Shell attribute values; never nil
	ContractedAmount    *money.Amount            `json:"contractedAmount"`    // in Currency; nil when empty
	RequestSwbs         bool                     `json:"requestSwbs"`         // Request SWBS code from GSAP
	FinanceBudgetYear   *int                     `json:"financeBudgetYear"`   // nil when empty
	FinanceBudgetItemID string                   `json:"financeBudgetItemId"` // an item of FinanceBudgetYear; "" when empty
	record.Meta
}

// Spread is a line item's spread method and, for a custom spread only, its month amounts.
type Spread struct {
	Method SpreadMethod  `json:"method"`
	Months []MonthAmount `json:"months"` // custom only, sorted by month; [] otherwise (never nil)
}

// MonthAmount is one month of a custom spread, in the line item's currency.
type MonthAmount struct {
	Month  string       `json:"month"`  // "2026-02"
	Amount money.Amount `json:"amount"` // 0.00 or more
}

// SwbsStatus is the derived SWBS request status (data model §3.5, contract SwbsStatus).
type SwbsStatus string

// The SWBS request statuses.
const (
	SwbsNotRequested       SwbsStatus = "notRequested"       // Request SWBS code from GSAP is off
	SwbsWaitingForApproval SwbsStatus = "waitingForApproval" // the activity is not Approved yet
	SwbsReadyToSend        SwbsStatus = "readyToSend"        // the activity is Approved
)

// The states of a spread (SpreadView.State).
const (
	StateSpread    = "spread"
	StateNotSpread = "notSpread" // Evenly, First or Last month without activity dates (D-032)
)

// The states of the unassigned balance (UnassignedBalance.State).
const (
	BalanceLeft          = "left"
	BalanceAllAssigned   = "allAssigned"
	BalanceExceeded      = "exceeded"
	BalanceNoPlannedCost = "noPlannedCost"
)

// Where a new line item's planned cost comes from (Defaults.PlannedCostSource).
const (
	SourceUnassignedBalance     = "unassignedBalance"
	SourceNothingLeftToAssign   = "nothingLeftToAssign"
	SourceNoActivityPlannedCost = "noActivityPlannedCost"
)

// NoDatesColumn is the key of the spread grid's "No dates yet" column.
const NoDatesColumn = "noDates"

// The API types below are what the API returns and receives (contract schema in brackets, JSON names as
// the contract). Lists are never nil. Amounts sent are decimal strings in the line item's currency.

// CurrencyView is a currency as the line-item screens need it (LineItemCurrency).
type CurrencyView struct {
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	Symbol     string     `json:"symbol"`
	RatePerUSD money.Rate `json:"ratePerUsd"` // current rate in units per 1 USD ("1" for USD)
	Active     bool       `json:"active"`
}

// FinanceBudgetItemRef is a finance budget item as line items show it (FinanceBudgetItemRef).
type FinanceBudgetItemRef struct {
	ID         string `json:"id"`
	Year       int    `json:"year"`
	BudgetName string `json:"budgetName"`
	Name       string `json:"name"`
	Code       string `json:"code"`
	Active     bool   `json:"active"`
}

// SpreadMonth is one month of a spread, in the line item's currency and in USD (SpreadMonth).
type SpreadMonth struct {
	Month                string      `json:"month"`
	Amount               money.Money `json:"amount"`
	AmountUSD            money.Money `json:"amountUsd"`
	OutsideActivityDates bool        `json:"outsideActivityDates"`
}

// SpreadView is a line item's spread as the API returns it (Spread).
type SpreadView struct {
	Method      SpreadMethod  `json:"method"`
	State       string        `json:"state"`       // StateSpread or StateNotSpread
	Months      []SpreadMonth `json:"months"`      // empty when not spread
	CustomTotal *money.Money  `json:"customTotal"` // custom only
	AddsUp      bool          `json:"addsUp"`
}

// SpreadMonthInput is one custom month of a request (SpreadMonthInput).
type SpreadMonthInput struct {
	Month  string `json:"month"`  // YYYY-MM
	Amount string `json:"amount"` // decimal amount in the line item's currency
}

// SpreadInput is the spread of a request (SpreadInput); Months are kept only for a custom spread.
type SpreadInput struct {
	Method SpreadMethod       `json:"method"`
	Months []SpreadMonthInput `json:"months"`
}

// View is a line item as the API returns it (LineItem).
type View struct {
	ID                string                   `json:"id"`
	ActivityID        string                   `json:"activityId"`
	Name              string                   `json:"name"`
	Owner             *people.PersonRef        `json:"owner"`
	Description       string                   `json:"description"`
	Currency          CurrencyView             `json:"currency"`
	PlannedCost       money.Money              `json:"plannedCost"`
	PlannedCostUSD    money.Money              `json:"plannedCostUsd"`
	Spread            SpreadView               `json:"spread"`
	Attributes        settings.AttributeValues `json:"attributes"`
	ContractedAmount  *money.Money             `json:"contractedAmount"`
	RequestSwbs       bool                     `json:"requestSwbs"`
	SwbsStatus        SwbsStatus               `json:"swbsStatus"`
	FinanceBudgetYear *int                     `json:"financeBudgetYear"`
	FinanceBudgetItem *FinanceBudgetItemRef    `json:"financeBudgetItem"`
	Missing           []settings.MissingField  `json:"missing"` // what still blocks submitting, in order (data model §3.6)
	record.Meta
}

// ActivityView is what the line-item screens need to know about the activity (LineItemActivity).
type ActivityView struct {
	ID                   string           `json:"id"`
	Name                 string           `json:"name"`
	Status               string           `json:"status"`           // the activity's stored status
	RestartsApproval     bool             `json:"restartsApproval"` // Submitted or Approved
	StartDate            *date.Date       `json:"startDate"`
	EndDate              *date.Date       `json:"endDate"`
	Ongoing              bool             `json:"ongoing"`
	PlannedCost          *money.Money     `json:"plannedCost"` // USD
	Path                 []errs.RecordRef `json:"path"`        // plan, campaign, initiative
	PlanYear             int              `json:"planYear"`
	DefaultSpreadMethod  SpreadMethod     `json:"defaultSpreadMethod"`
	DefaultFinanceBudget *string          `json:"defaultFinanceBudget"` // the plan's default budget name
}

// UnassignedBalance is the activity's planned cost minus its line items' USD planned costs
// (UnassignedBalance, Sys_Gen_Balance).
type UnassignedBalance struct {
	PlannedCost  *money.Money `json:"plannedCost"` // USD; null while the activity has none
	Assigned     money.Money  `json:"assigned"`
	Balance      *money.Money `json:"balance"` // negative when exceeded; null without a planned cost
	State        string       `json:"state"`   // one of the Balance… states
	RoundingHint bool         `json:"roundingHint"`
}

// SpreadGridColumn is one column of the spread grid (SpreadGridColumn).
type SpreadGridColumn struct {
	Key                  string  `json:"key"`   // the month (YYYY-MM) or NoDatesColumn
	Month                *string `json:"month"` // null for the No dates yet column
	OutsideActivityDates bool    `json:"outsideActivityDates"`
}

// SpreadGridRow is one line item's row of the grid, in USD (SpreadGridRow).
type SpreadGridRow struct {
	LineItemID string         `json:"lineItemId"`
	Name       string         `json:"name"`
	Cells      []*money.Money `json:"cells"` // one per column; null where the line item has nothing
	Total      money.Money    `json:"total"`
}

// SpreadGrid is the activity's line items by month, in USD (SpreadGrid).
type SpreadGrid struct {
	Columns []SpreadGridColumn `json:"columns"`
	Rows    []SpreadGridRow    `json:"rows"`
	Totals  []*money.Money     `json:"totals"` // null when every cell of the column is empty
	Total   money.Money        `json:"total"`
}

// Section is the answer of GET /api/activities/{activityId}/line-items (LineItemSection).
type Section struct {
	Activity ActivityView      `json:"activity"`
	Items    []View            `json:"items"` // in creation order
	Balance  UnassignedBalance `json:"balance"`
	Grid     SpreadGrid        `json:"grid"`
}

// Details is the answer of GET /api/line-items/{lineItemId} (LineItemDetails).
type Details struct {
	LineItem View              `json:"lineItem"`
	Activity ActivityView      `json:"activity"`
	Balance  UnassignedBalance `json:"balance"`
}

// Defaults are what a new line item starts with (LineItemDefaults, FR-008).
type Defaults struct {
	Name                         string            `json:"name"`
	Owner                        *people.PersonRef `json:"owner"`
	Currency                     string            `json:"currency"`
	PlannedCost                  *money.Money      `json:"plannedCost"`
	PlannedCostSource            string            `json:"plannedCostSource"` // one of the Source… values
	SpreadMethod                 SpreadMethod      `json:"spreadMethod"`
	FinanceBudgetYear            int               `json:"financeBudgetYear"`
	FinanceBudgetItemID          *string           `json:"financeBudgetItemId"`
	FinanceBudgetItemPreselected bool              `json:"financeBudgetItemPreselected"`
	RequestSwbs                  bool              `json:"requestSwbs"`
}

// FormOptions is the answer of GET /api/activities/{activityId}/line-items/form-options
// (LineItemFormOptions).
type FormOptions struct {
	Activity           ActivityView           `json:"activity"`
	Balance            UnassignedBalance      `json:"balance"`
	Defaults           Defaults               `json:"defaults"`
	Currencies         []CurrencyView         `json:"currencies"`         // active ones, USD first, then by code
	FinanceBudgetYears []int                  `json:"financeBudgetYears"` // ascending
	FinanceBudgetItems []FinanceBudgetItemRef `json:"financeBudgetItems"`
}

// Input is the body of POST /api/activities/{activityId}/line-items (LineItemCreate).
type Input struct {
	Name                string                   `json:"name"`
	OwnerID             *string                  `json:"ownerId"`
	Description         string                   `json:"description"`
	Currency            string                   `json:"currency"`
	PlannedCost         *string                  `json:"plannedCost"` // decimal amount in the currency
	Spread              SpreadInput              `json:"spread"`
	Attributes          settings.AttributeValues `json:"attributes"`
	ContractedAmount    *string                  `json:"contractedAmount"`
	RequestSwbs         bool                     `json:"requestSwbs"`
	FinanceBudgetYear   *int                     `json:"financeBudgetYear"`
	FinanceBudgetItemID *string                  `json:"financeBudgetItemId"`
	RestartApproval     bool                     `json:"restartApproval"` // the person confirmed the activity needs approval again
}

// Update is the body of PATCH /api/line-items/{lineItemId} (LineItemUpdate): the whole form and the version.
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

// DuplicateInput is the body of POST /api/line-items/{lineItemId}/duplicate (LineItemDuplicate).
type DuplicateInput struct {
	RestartApproval bool `json:"restartApproval"`
}
