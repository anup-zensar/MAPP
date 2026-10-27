package financials

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

// The API types below are what the API answers and receives (contract schemas in brackets). Lists are
// never nil. Money is in the reader's display currency unless the field says otherwise.

// Level is the level of a planning record (contract AttributeLevel).
type Level string

// The levels of the planning tree, top-down.
const (
	LevelPlan       Level = "plan"
	LevelCampaign   Level = "campaign"
	LevelInitiative Level = "initiative"
	LevelActivity   Level = "activity"
	LevelLineItem   Level = "lineItem"
)

// levelWords name the levels mid-sentence.
var levelWords = map[Level]string{
	LevelPlan: "plan", LevelCampaign: "campaign", LevelInitiative: "initiative", LevelActivity: "activity",
	LevelLineItem: "line item",
}

// childLevel is the level of each level's breakdown rows; a line item has none.
var childLevel = map[Level]Level{
	LevelPlan: LevelCampaign, LevelCampaign: LevelInitiative, LevelInitiative: LevelActivity, LevelActivity: LevelLineItem,
}

// RecordRef is a planning record (FinancialRecordRef).
type RecordRef struct {
	Type Level  `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Totals are sums of transactions (FinancialTotals).
type Totals struct {
	Committed money.Money `json:"committed"`
	Invoiced  money.Money `json:"invoiced"`
	Paid      money.Money `json:"paid"`
	Accrued   money.Money `json:"accrued"`
}

// Figures are a record's figures (FinancialFigures, FR-011, FR-013).
type Figures struct {
	PlannedCost               *money.Money `json:"plannedCost"`
	Committed                 money.Money  `json:"committed"`
	Invoiced                  money.Money  `json:"invoiced"`
	Paid                      money.Money  `json:"paid"`
	Accrued                   money.Money  `json:"accrued"`
	NotYetCommitted           *money.Money `json:"notYetCommitted"`
	CommittedPercent          *int         `json:"committedPercent"`
	CommittedAbovePlanned     bool         `json:"committedAbovePlanned"`
	LineItemsAbovePlanned     int          `json:"lineItemsAbovePlanned"`
	FirstLineItemAbovePlanned *RecordRef   `json:"firstLineItemAbovePlanned"`
	TransactionCount          int          `json:"transactionCount"`
}

// SourceView says where the GSAP data comes from and how recent it is (FinancialSource, FR-008).
type SourceView struct {
	DataAsOf     *time.Time `json:"dataAsOf"`     // null when no GSAP data was ever loaded
	FeedName     *string    `json:"feedName"`     // of the newest import
	FileName     *string    `json:"fileName"`     // of the newest import; null for the demo data
	AccrualMonth *string    `json:"accrualMonth"` // the latest accrual month in the data; null without accruals
}

// BreakdownRow is one child of a record with its figures (FinancialBreakdownRow).
type BreakdownRow struct {
	Record  RecordRef `json:"record"`
	Figures Figures   `json:"figures"`
}

// RecordView is everything a Financials card shows (FinancialRecordView, FR-015 – FR-019).
type RecordView struct {
	Record         RecordRef        `json:"record"`
	PlanID         string           `json:"planId"`
	Path           []RecordRef      `json:"path"` // ancestors, plan first; empty for a plan
	Figures        Figures          `json:"figures"`
	BreakdownLevel *Level           `json:"breakdownLevel"` // null for a line item
	Breakdown      []BreakdownRow   `json:"breakdown"`
	Latest         []TransactionRow `json:"latest"`    // at most 5, newest first
	SwbsCodes      []string         `json:"swbsCodes"` // line items: the SWBS codes of their transactions
	Source         SourceView       `json:"source"`
}

// Vendor is a transaction's vendor (FinancialVendor).
type Vendor struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// LinkState is the derived link state of a transaction (FinancialLinkState, data model §4.3).
type LinkState string

// The link states.
const (
	StateLinked          LinkState = "linked"          // linked to a live line item
	StateLineItemDeleted LinkState = "lineItemDeleted" // linked to a line item in the recycle bin (or purged)
	StateNotLinked       LinkState = "notLinked"
)

// LinkView is how a transaction is linked (FinancialLink).
type LinkView struct {
	State      LinkState    `json:"state"`
	MatchedBy  *MatchedBy   `json:"matchedBy"`
	Reason     *string      `json:"reason"`
	LineItemID *string      `json:"lineItemId"`
	Path       []RecordRef  `json:"path"` // plan → … → line item when linked; [] otherwise
	ManualBy   *actor.Actor `json:"manualBy"`
	ManualAt   *time.Time   `json:"manualAt"`
}

// TransactionRow is a transaction as lists show it (FinancialTransactionRow).
type TransactionRow struct {
	ID              string       `json:"id"`
	Type            Type         `json:"type"`
	Name            string       `json:"name"`
	DocumentNumber  string       `json:"documentNumber"`
	LineNumber      string       `json:"lineNumber"`
	PONumber        string       `json:"poNumber"`
	POLineNumber    string       `json:"poLineNumber"`
	PostingDate     date.Date    `json:"postingDate"`
	AccrualMonth    *string      `json:"accrualMonth"`
	Amount          money.Money  `json:"amount"`        // in its own currency
	AmountDisplay   *money.Money `json:"amountDisplay"` // null when its currency is no longer in Settings
	CurrencyMissing bool         `json:"currencyMissing"`
	Vendor          *Vendor      `json:"vendor"`
	Description     string       `json:"description"`
	SwbsCode        string       `json:"swbsCode"`
	Link            LinkView     `json:"link"`
}

// ImportRef is the import a transaction came from (FinancialImportRef).
type ImportRef struct {
	ID         string      `json:"id"`
	FeedName   string      `json:"feedName"`
	FileName   string      `json:"fileName"`
	ImportedAt time.Time   `json:"importedAt"`
	ImportedBy actor.Actor `json:"importedBy"`
}

// TransactionView is one transaction with every field of the details panel (FinancialTransaction, FR-022).
type TransactionView struct {
	ID              string       `json:"id"`
	Type            Type         `json:"type"`
	Name            string       `json:"name"`
	DocumentNumber  string       `json:"documentNumber"`
	LineNumber      string       `json:"lineNumber"`
	PONumber        string       `json:"poNumber"`
	POLineNumber    string       `json:"poLineNumber"`
	PostingDate     date.Date    `json:"postingDate"`
	AccrualMonth    *string      `json:"accrualMonth"`
	Amount          money.Money  `json:"amount"`
	AmountUSD       *money.Money `json:"amountUsd"`
	AmountDisplay   *money.Money `json:"amountDisplay"`
	CurrencyMissing bool         `json:"currencyMissing"`
	Vendor          *Vendor      `json:"vendor"`
	Description     string       `json:"description"`
	SwbsCode        string       `json:"swbsCode"`
	TrackingKey     string       `json:"trackingKey"`
	SentLineItemID  string       `json:"sentLineItemId"`
	Link            LinkView     `json:"link"`
	Import          ImportRef    `json:"import"`
	CanLink         bool         `json:"canLink"`
	CanUnlink       bool         `json:"canUnlink"`
	Version         int          `json:"version"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
	UpdatedBy       actor.Actor  `json:"updatedBy"`
}

// TypeCounts are the counts of the Financials page's tabs (FinancialTypeCounts).
type TypeCounts struct {
	All           int `json:"all"`
	PurchaseOrder int `json:"purchaseOrder"`
	Invoice       int `json:"invoice"`
	Payment       int `json:"payment"`
	Accrual       int `json:"accrual"`
	NotLinked     int `json:"notLinked"`
}

// PlanChoice is a plan of the Plan filter (FinancialPlanChoice).
type PlanChoice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Year int    `json:"year"`
}

// Choices are the values of the Financials page's filters (FinancialChoices).
type Choices struct {
	Years       []int        `json:"years"` // newest first
	CurrentYear int          `json:"currentYear"`
	Plans       []PlanChoice `json:"plans"` // the live plans of the selected year, by name
}

// TransactionPage is one page of the Financials page (FinancialTransactionPage, FR-021).
type TransactionPage struct {
	Items        []TransactionRow `json:"items"`
	Total        int              `json:"total"`
	Page         int              `json:"page"`
	PageSize     int              `json:"pageSize"`
	Year         int              `json:"year"`
	Counts       TypeCounts       `json:"counts"`
	Totals       Totals           `json:"totals"`
	RecordFilter *RecordRef       `json:"recordFilter"`
	Choices      Choices          `json:"choices"`
	Source       SourceView       `json:"source"`
}

// LinkRequest is the body of a manual link (FinancialLinkRequest).
type LinkRequest struct {
	LineItemID string `json:"lineItemId"`
	Version    *int   `json:"version"`
}

// UnlinkRequest is the body of an unlink (FinancialUnlinkRequest).
type UnlinkRequest struct {
	Version *int `json:"version"`
}

// Scope is Home's My teams / Everything switch (contract HomeScope).
type Scope string

// The scopes.
const (
	ScopeMyTeams    Scope = "myTeams"
	ScopeEverything Scope = "everything"
)

// Overview is Home's Financials card (FinancialOverview, FR-025).
type Overview struct {
	Year                   int        `json:"year"`
	Scope                  Scope      `json:"scope"`
	Figures                Figures    `json:"figures"`
	ActivitiesAbovePlanned int        `json:"activitiesAbovePlanned"`
	NotLinked              int        `json:"notLinked"`
	Source                 SourceView `json:"source"`
}
