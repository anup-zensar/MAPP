package settings

import (
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
