package settings

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// RateInfo is a currency's current rate and where it came from (contract RateUsed, FR-009): what the web app
// shows beside every converted amount.
type RateInfo struct {
	Currency       string      `json:"currency"`
	Rate           money.Rate  `json:"rate"`
	InverseRate    money.Rate  `json:"inverseRate"`
	RateDate       *date.Date  `json:"rateDate"`       // null for USD
	Source         *RateSource `json:"source"`         // null for USD
	ImportFileName *string     `json:"importFileName"` // the GSAP file of an imported rate; null otherwise
}

// CurrentRatePage is the answer of GET /api/v1/exchange-rates/current (contract CurrentRatePage).
type CurrentRatePage struct {
	Today date.Date  `json:"today"`
	Items []RateInfo `json:"items"`
	Total int        `json:"total"`
}

// The API types of the rate history (contract ExchangeRateRecord, ExchangeRatePage, ExchangeRateCreate,
// ExchangeRateUpdate).

// ExchangeRateView is one dated rate as the API answers it (ExchangeRateRecord).
type ExchangeRateView struct {
	ID             string     `json:"id"`
	Currency       string     `json:"currency"`
	Date           date.Date  `json:"date"`
	Rate           money.Rate `json:"rate"`
	InverseRate    money.Rate `json:"inverseRate"`
	Source         RateSource `json:"source"`
	ImportID       *string    `json:"importId"`
	ImportFileName *string    `json:"importFileName"`
	Current        bool       `json:"current"` // the rate conversions use today
	Future         bool       `json:"future"`  // dated after today ("From <date>")
	record.Meta
}

// ExchangeRatePage is a currency's rate history, newest date first, with its current rate (ExchangeRatePage).
type ExchangeRatePage struct {
	Items   []ExchangeRateView `json:"items"`
	Total   int                `json:"total"`
	Current RateInfo           `json:"current"`
}

// RateInput is the body of POST /api/v1/currencies/{code}/rates and its impact check (ExchangeRateCreate).
type RateInput struct {
	Date string `json:"date"`
	Rate string `json:"rate"`
}

// RateUpdate is the body of PATCH /api/v1/exchange-rates/{rateId} and its impact check (ExchangeRateUpdate).
type RateUpdate struct {
	Rate    string `json:"rate"`
	Version *int   `json:"version"`
}

// RateImportUpload is the body of the two import operations (contract RateImportUpload): the file in base64.
type RateImportUpload struct {
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

// RateOutcome is what importing one row does (contract RateImportOutcome).
type RateOutcome string

// The outcomes of a row (data-model §2.4).
const (
	OutcomeAdd     RateOutcome = "add"     // a new rate
	OutcomeReplace RateOutcome = "replace" // a different rate for the same currency and day
	OutcomePresent RateOutcome = "present" // the same rate is stored already: skipped
	OutcomeNotUsed RateOutcome = "notUsed" // another rate type, not quoted from USD, USD itself, or a currency not set up
	OutcomeProblem RateOutcome = "problem" // an unreadable date or rate, or a row twice in the file
)

// RateImportRow is one reviewed row (contract RateImportRow).
type RateImportRow struct {
	Row          int         `json:"row"`
	Outcome      RateOutcome `json:"outcome"`
	Currency     *string     `json:"currency"`
	Date         *date.Date  `json:"date"`
	Rate         *money.Rate `json:"rate"`
	PreviousRate *money.Rate `json:"previousRate"` // replace only
	Message      *string     `json:"message"`      // why a row is not used or has a problem; old → new for replace
}

// RateImportCounts counts the rows of a review by outcome (contract RateImportCounts).
type RateImportCounts struct {
	Add     int `json:"add"`
	Replace int `json:"replace"`
	Present int `json:"present"`
	NotUsed int `json:"notUsed"`
	Problem int `json:"problem"`
}

// RateImportCheck is the review of a rate file (contract RateImportCheck): every row, in file order.
type RateImportCheck struct {
	FileName string           `json:"fileName"`
	Feed     string           `json:"feed"`
	Rows     []RateImportRow  `json:"rows"`
	Counts   RateImportCounts `json:"counts"`
}

// RateImportView is a rate import as the API answers it (contract RateImport).
type RateImportView struct {
	ID         string      `json:"id"`
	FileName   string      `json:"fileName"`
	Feed       string      `json:"feed"`
	FirstDate  *date.Date  `json:"firstDate"`
	LastDate   *date.Date  `json:"lastDate"`
	Added      int         `json:"added"`
	Replaced   int         `json:"replaced"`
	Unchanged  int         `json:"unchanged"`
	NotUsed    int         `json:"notUsed"`
	Problems   int         `json:"problems"`
	ImportedAt time.Time   `json:"importedAt"`
	ImportedBy actor.Actor `json:"importedBy"`
}

// RateImportPage is the latest imports, newest first (contract RateImportPage).
type RateImportPage struct {
	Items []RateImportView `json:"items"`
	Total int              `json:"total"`
}
