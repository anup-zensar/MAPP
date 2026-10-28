// Package swbs sends SWBS requests (feature 030): the line items that are ready go to GSAP through an adapter —
// the simulated GSAP until live integration (ADR-0016) — and every run is kept with the codes it returned.
package swbs

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/record"
)

type Source string

const (
	SourceSimulated Source = "simulatedGsap"
	SourceDemo      Source = "demoData"
)

// Run is stored in "swbs_runs" with its codes, in send order, in "swbs_run_codes"; runs are only ever added.
type Run struct {
	ID     string `json:"id"`
	Source Source `json:"source"`
	Codes  []Code `json:"codes"`
	record.Meta
}

type Code struct {
	LineItemID   string `json:"lineItemId"`
	LineItemName string `json:"lineItemName"`
	CompanyCode  string `json:"companyCode"`
	SwbsCode     string `json:"swbsCode"`
}

type RunInput struct {
	LineItemIDs []string `json:"lineItemIds"`
}

type RunView struct {
	ID     string      `json:"id"`
	Source Source      `json:"source"`
	RanAt  time.Time   `json:"ranAt"`
	RanBy  actor.Actor `json:"ranBy"`
	Sent   int         `json:"sent"`
	Codes  []Code      `json:"codes"`
}

type RunPage struct {
	Items       []RunView `json:"items"`
	Total       int       `json:"total"`
	ReadyToSend int       `json:"readyToSend"`
}

func (r Run) view() RunView {
	codes := r.Codes
	if codes == nil {
		codes = []Code{}
	}
	return RunView{ID: r.ID, Source: r.Source, RanAt: r.CreatedAt, RanBy: r.CreatedBy, Sent: len(codes), Codes: codes}
}
