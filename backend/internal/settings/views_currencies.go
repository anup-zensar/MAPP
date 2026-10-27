package settings

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// CurrencyView is a currency as the API returns it (contract Currency).
type CurrencyView struct {
	Code          string      `json:"code"`
	Name          string      `json:"name"`
	Symbol        string      `json:"symbol"`
	Rate          money.Rate  `json:"rate"`
	InverseRate   money.Rate  `json:"inverseRate"`
	RateChangedAt time.Time   `json:"rateChangedAt"` // when the current rate was recorded (016)
	RateChangedBy actor.Actor `json:"rateChangedBy"`
	// The current rate's date and source (016 FR-003): "rate of 19 Oct 2026 · GSAP import".
	RateDate           date.Date  `json:"rateDate"`
	RateSource         RateSource `json:"rateSource"`
	RateImportFileName *string    `json:"rateImportFileName"`
	Active             bool       `json:"active"`
	Usage              Usage      `json:"usage"`
	Locks              []Lock     `json:"locks"`
	record.Meta
}

// CurrencyPage is the answer of GET /api/v1/currencies (contract CurrencyPage): USD first, then by code.
type CurrencyPage struct {
	Items []CurrencyView `json:"items"`
	Total int            `json:"total"`
}

// CurrencyInput is the body of POST /api/v1/currencies (contract CurrencyCreate); the rate is text ("" when
// not filled in).
type CurrencyInput struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Rate   string `json:"rate"`
}

// CurrencyUpdate is the body of PATCH /api/v1/currencies/{code} and of its impact check (contract
// CurrencyUpdate). The code never changes.
type CurrencyUpdate struct {
	Name    *string `json:"name"`
	Symbol  *string `json:"symbol"`
	Rate    *string `json:"rate"`
	Version *int    `json:"version"`
}
