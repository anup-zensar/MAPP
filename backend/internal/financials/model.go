// Package financials holds the GSAP financial transactions of feature 015 (BUS.FIN.080): purchase-order
// lines, invoices, payments and accruals as GSAP sent them, linked to line items, and the figures —
// committed, invoiced, paid, accrued, not yet committed — worked out from them for every line item,
// activity, initiative, campaign and plan.
//
// Transactions are read-only in NorthStar: they arrive through a Feed (feed.go) — now the simulated GSAP
// extract (a CSV file sent to the API) and the demo data — and NorthStar never creates, edits, deletes or
// sends one (constitution II, D-242). The only change a person makes is a manual link of a transaction the
// rules could not link (FR-026). Every figure is calculated on read and never stored (FR-012).
//
// The package reads the planning tree, the exchange rates, the display currency and the access rules
// through the consumer interfaces of ports.go, which internal/app implements over features 002–007 and
// 014 (D-196, D-252).
package financials

import (
	"strings"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
)

// Type is the kind of GSAP transaction (FR-001, D-243).
type Type string

// The four types, in the order an import applies them (data model §4.1).
const (
	TypePurchaseOrder Type = "purchaseOrder" // a PO line: committed
	TypeInvoice       Type = "invoice"       // invoiced; credit notes are negative
	TypePayment       Type = "payment"       // paid (Shell's actuals)
	TypeAccrual       Type = "accrual"       // accrued, for its accrual month
)

// Types lists the four types in the order of application: purchase orders first, so that invoices,
// payments and accruals of the same delivery find their PO.
var Types = []Type{TypePurchaseOrder, TypeInvoice, TypePayment, TypeAccrual}

// typeWords are the type words of readable names: "PO 4500001001 / 10", "Invoice 5100002002".
var typeWords = map[Type]string{
	TypePurchaseOrder: "PO", TypeInvoice: "Invoice", TypePayment: "Payment", TypeAccrual: "Accrual",
}

// MatchedBy says which rule linked a transaction to its line item (FR-007), or that a person did
// (FR-026). Empty means not linked.
type MatchedBy string

// The ways a transaction is linked.
const (
	MatchedByLineItemID  MatchedBy = "lineItemId"
	MatchedByTrackingKey MatchedBy = "trackingKey"
	MatchedByPONumber    MatchedBy = "poNumber"
	MatchedBySwbsCode    MatchedBy = "swbsCode"
	MatchedByManual      MatchedBy = "manual"
	MatchedByNone        MatchedBy = "" // not linked
)

// Source is where an import came from (data model §2).
type Source string

// The sources of GSAP data now: the simulated extract uploaded through the API, and the demo seed.
const (
	SourceFile Source = "file"
	SourceDemo Source = "demo"
)

// Fields are a transaction's fields as GSAP sent them (FR-001): everything an import may change. An
// import compares them to tell an updated row from an unchanged one (data model §4.1).
type Fields struct {
	Type           Type         `json:"type"`
	DocumentNumber string       `json:"documentNumber"` // GSAP document number; for POs the PO number
	LineNumber     string       `json:"lineNumber"`     // PO line or invoice item, 1–6 digits; "" when none
	PONumber       string       `json:"poNumber"`       // POs: their DocumentNumber; others: the PO referred to; "" when none
	POLineNumber   string       `json:"poLineNumber"`   // the PO line referred to; "" when none (always "" on POs)
	PostingDate    date.Date    `json:"postingDate"`
	AccrualMonth   string       `json:"accrualMonth"` // accruals: "2026-09"; others ""
	Amount         money.Amount `json:"amount"`       // as GSAP sent it, in Currency; negative for credits
	Currency       string       `json:"currency"`     // ISO code of a currency of Settings (002)
	VendorCode     string       `json:"vendorCode"`
	VendorName     string       `json:"vendorName"`
	Description    string       `json:"description"`
	SwbsCode       string       `json:"swbsCode"`
	TrackingKey    string       `json:"trackingKey"`    // the HX key as sent, upper case: "HX1003"
	SentLineItemID string       `json:"sentLineItemId"` // the line_item_id column as sent: "L-1003"
}

// key identifies a transaction among GSAP's data: type + document number + line (FR-006).
type key struct {
	Type           Type
	DocumentNumber string
	LineNumber     string
}

// key returns the identity of the fields.
func (f Fields) key() key {
	return key{Type: f.Type, DocumentNumber: f.DocumentNumber, LineNumber: f.LineNumber}
}

// Name returns the readable name of the transaction (data model §1): "PO 4500001001 / 10",
// "Invoice 5100002002".
func (f Fields) Name() string {
	name := typeWords[f.Type] + " " + f.DocumentNumber
	if f.LineNumber != "" {
		name += " / " + f.LineNumber
	}
	return name
}

// Transaction is one GSAP transaction as stored in the table "financial_transactions" (data model §1).
type Transaction struct {
	ID string `json:"id"` // "FT-1001"
	Fields

	LineItemID      string      `json:"lineItemId"`      // the linked line item; "" when not linked
	MatchedBy       MatchedBy   `json:"matchedBy"`       // "" when not linked
	NotLinkedReason string      `json:"notLinkedReason"` // "" when linked
	ManualLink      *ManualLink `json:"manualLink"`      // only when MatchedBy is manual
	ImportID        string      `json:"importId"`        // the import that added or last changed the GSAP fields
	record.Meta
}

// ManualLink says who linked a transaction by hand, and when (FR-026).
type ManualLink struct {
	By actor.Actor `json:"by"`
	At time.Time   `json:"at"` // UTC
}

// linked reports whether the transaction is linked to a line item (live or not).
func (t Transaction) linked() bool {
	return t.LineItemID != ""
}

// Import is one delivery of GSAP data, as stored in the table "financial_imports" (data model §2). Its
// CreatedBy and CreatedAt say who imported it and when.
type Import struct {
	ID        string `json:"id"` // "FIM-1001"
	Source    Source `json:"source"`
	FeedName  string `json:"feedName"` // "Simulated GSAP extract", "Demo data (simulated GSAP)"
	FileName  string `json:"fileName"` // "" for the demo data
	Rows      int    `json:"rows"`     // data rows read: New + Updated + Unchanged + Refused
	New       int    `json:"new"`
	Updated   int    `json:"updated"`
	Unchanged int    `json:"unchanged"`
	Refused   int    `json:"refused"`
	NotLinked int    `json:"notLinked"` // accepted rows of this delivery left not linked
	Relinked  int    `json:"relinked"`  // earlier not-linked transactions this import linked (FR-007)
	record.Meta
}

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

// ImportRequest is the body of the import check and the import (FinancialImportRequest): the file as
// base64.
type ImportRequest struct {
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

// ImportProblem is a problem of one row of a file (FinancialImportProblem).
type ImportProblem struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Message string `json:"message"`
}

// NotLinkedRow is an accepted row left not linked, with the reason (FinancialImportNotLinkedRow).
type NotLinkedRow struct {
	Row    int    `json:"row"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// ImportCounts are the outcome counts of an import (FinancialImportCounts).
type ImportCounts struct {
	Rows      int `json:"rows"`
	New       int `json:"new"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Refused   int `json:"refused"`
	NotLinked int `json:"notLinked"`
	Relinked  int `json:"relinked"`
}

// ImportCheck is what an import would do (FinancialImportCheck, FR-004).
type ImportCheck struct {
	FileName       string          `json:"fileName"`
	Counts         ImportCounts    `json:"counts"`
	Problems       []ImportProblem `json:"problems"` // at most 200
	ProblemsTotal  int             `json:"problemsTotal"`
	NotLinked      []NotLinkedRow  `json:"notLinked"` // at most 200
	NotLinkedTotal int             `json:"notLinkedTotal"`
	UnusedColumns  []string        `json:"unusedColumns"`
}

// ImportView is an import as the API answers it (FinancialImport).
type ImportView struct {
	ID         string       `json:"id"`
	Source     Source       `json:"source"`
	FeedName   string       `json:"feedName"`
	FileName   string       `json:"fileName"`
	ImportedAt time.Time    `json:"importedAt"`
	ImportedBy actor.Actor  `json:"importedBy"`
	Counts     ImportCounts `json:"counts"`
}

// ImportResult is the answer of an import (FinancialImportResult).
type ImportResult struct {
	Import         ImportView      `json:"import"`
	Problems       []ImportProblem `json:"problems"`
	ProblemsTotal  int             `json:"problemsTotal"`
	NotLinked      []NotLinkedRow  `json:"notLinked"`
	NotLinkedTotal int             `json:"notLinkedTotal"`
}

// ImportPage is the import history (FinancialImportPage).
type ImportPage struct {
	Items []ImportView `json:"items"`
	Total int          `json:"total"`
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

// view returns the import as the API answers it.
func (i Import) view() ImportView {
	return ImportView{
		ID: i.ID, Source: i.Source, FeedName: i.FeedName, FileName: i.FileName, ImportedAt: i.CreatedAt,
		ImportedBy: i.CreatedBy, Counts: i.counts(),
	}
}

// counts returns the import's counts as the API answers them.
func (i Import) counts() ImportCounts {
	return ImportCounts{
		Rows: i.Rows, New: i.New, Updated: i.Updated, Unchanged: i.Unchanged, Refused: i.Refused,
		NotLinked: i.NotLinked, Relinked: i.Relinked,
	}
}

// ref returns the import as a transaction's "from import".
func (i Import) ref() ImportRef {
	return ImportRef{ID: i.ID, FeedName: i.FeedName, FileName: i.FileName, ImportedAt: i.CreatedAt, ImportedBy: i.CreatedBy}
}

// lower is strings.ToLower, for searches and sorts that ignore letter case.
func lower(s string) string {
	return strings.ToLower(s)
}
