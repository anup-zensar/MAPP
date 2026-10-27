// Package financials holds the GSAP financial transactions of feature 015 (BUS.FIN.080): purchase-order
// lines, invoices, payments and accruals as GSAP sent them, linked to line items, and the figures —
// committed, invoiced, paid, accrued, not yet committed — worked out from them for every line item,
// activity, initiative, campaign and plan.
//
// Transactions are read-only in MAPP: they arrive through a Feed (feed.go) — now the simulated GSAP
// extract (a CSV file sent to the API) and the demo data — and MAPP never creates, edits, deletes or
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

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
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
