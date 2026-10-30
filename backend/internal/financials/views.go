package financials

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

// The API types below are what the API answers and receives (contract schemas in brackets). Lists are
// never nil. Money is in the reader's display currency unless the field says otherwise.

type Level string

const (
	LevelPlan       Level = "plan"
	LevelCampaign   Level = "campaign"
	LevelInitiative Level = "initiative"
	LevelActivity   Level = "activity"
	LevelLineItem   Level = "lineItem"
)

func levelWords() map[Level]string {
	return map[Level]string{
		LevelPlan: "plan", LevelCampaign: "campaign", LevelInitiative: "initiative", LevelActivity: "activity",
		LevelLineItem: "line item",
	}
}

func childLevel() map[Level]Level {
	return map[Level]Level{
		LevelPlan: LevelCampaign, LevelCampaign: LevelInitiative, LevelInitiative: LevelActivity, LevelActivity: LevelLineItem,
	}
}

type RecordRef struct {
	Type Level  `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Totals struct {
	Committed money.Money `json:"committed"`
	Invoiced  money.Money `json:"invoiced"`
	Paid      money.Money `json:"paid"`
	Accrued   money.Money `json:"accrued"`
}

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

type SourceView struct {
	DataAsOf     *time.Time `json:"dataAsOf"` // null when no GSAP data was ever loaded
	FeedName     *string    `json:"feedName"`
	FileName     *string    `json:"fileName"`     // of the newest import; null for the demo data
	AccrualMonth *string    `json:"accrualMonth"` // the latest accrual month in the data; null without accruals
}

type BreakdownRow struct {
	Record  RecordRef `json:"record"`
	Figures Figures   `json:"figures"`
}

type RecordView struct {
	Record         RecordRef        `json:"record"`
	PlanID         string           `json:"planId"`
	Path           []RecordRef      `json:"path"` // ancestors, plan first; empty for a plan
	Figures        Figures          `json:"figures"`
	BreakdownLevel *Level           `json:"breakdownLevel"`
	Breakdown      []BreakdownRow   `json:"breakdown"`
	Latest         []TransactionRow `json:"latest"` // at most 5, newest first
	SwbsCodes      []string         `json:"swbsCodes"`
	Source         SourceView       `json:"source"`
}

type Vendor struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type LinkState string

const (
	StateLinked          LinkState = "linked"
	StateLineItemDeleted LinkState = "lineItemDeleted" // linked to a line item in the recycle bin (or purged)
	StateNotLinked       LinkState = "notLinked"
)

type LinkView struct {
	State      LinkState    `json:"state"`
	MatchedBy  *MatchedBy   `json:"matchedBy"`
	Reason     *string      `json:"reason"`
	LineItemID *string      `json:"lineItemId"`
	Path       []RecordRef  `json:"path"`
	ManualBy   *actor.Actor `json:"manualBy"`
	ManualAt   *time.Time   `json:"manualAt"`
}

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
	Amount          money.Money  `json:"amount"`
	AmountDisplay   *money.Money `json:"amountDisplay"` // null when its currency is no longer in Settings
	CurrencyMissing bool         `json:"currencyMissing"`
	Vendor          *Vendor      `json:"vendor"`
	Description     string       `json:"description"`
	SwbsCode        string       `json:"swbsCode"`
	Link            LinkView     `json:"link"`
}

type ImportRef struct {
	ID         string      `json:"id"`
	FeedName   string      `json:"feedName"`
	FileName   string      `json:"fileName"`
	ImportedAt time.Time   `json:"importedAt"`
	ImportedBy actor.Actor `json:"importedBy"`
}

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

type TypeCounts struct {
	All           int  `json:"all"`
	PurchaseOrder int  `json:"purchaseOrder"`
	Invoice       int  `json:"invoice"`
	Payment       int  `json:"payment"`
	Accrual       int  `json:"accrual"`
	NotLinked     *int `json:"notLinked,omitempty"` // nil for people who may not see not-linked transactions
}

type PlanChoice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Year int    `json:"year"`
}

type Choices struct {
	Years       []int        `json:"years"`
	CurrentYear int          `json:"currentYear"`
	Plans       []PlanChoice `json:"plans"`
}

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

type LinkRequest struct {
	LineItemID string `json:"lineItemId"`
	Version    *int   `json:"version"`
}

type UnlinkRequest struct {
	Version *int `json:"version"`
}

type Scope string

const (
	ScopeMyTeams    Scope = "myTeams"
	ScopeEverything Scope = "everything"
)

type Overview struct {
	Year                   int        `json:"year"`
	Scope                  Scope      `json:"scope"`
	Figures                Figures    `json:"figures"`
	ActivitiesAbovePlanned int        `json:"activitiesAbovePlanned"`
	NotLinked              int        `json:"notLinked"`
	Source                 SourceView `json:"source"`
}
