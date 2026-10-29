// Package overspend lets each budget warn or refuse when planned cost goes over it (feature 037): the setting,
// the organisation default, and the guard that refuses a save taking a budget set to Refuse over it.
package overspend

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

type Mode string

const (
	Warn   Mode = "warn"
	Refuse Mode = "refuse"
)

// Normalised reads anything but Refuse as Warn, the setting of every budget saved before feature 037.
func (m Mode) Normalised() Mode {
	if m == Refuse {
		return Refuse
	}
	return Warn
}

func (m Mode) Refuses() bool {
	return m == Refuse
}

type Kind string

const (
	KindPlan         Kind = "plan"
	KindCampaign     Kind = "campaign"
	KindInitiative   Kind = "initiative"
	KindGlobalBudget Kind = "globalBudget"
)

var kindOrder = map[Kind]int{KindPlan: 1, KindCampaign: 2, KindInitiative: 3, KindGlobalBudget: 4} //nolint:gochecknoglobals // a fixed lookup

var kindWords = map[Kind]string{ //nolint:gochecknoglobals // a fixed lookup
	KindPlan: "plan", KindCampaign: "campaign", KindInitiative: "initiative", KindGlobalBudget: "global budget",
}

// Budget is one budget with the planned cost under it, in the budget's currency (FR-008); never stored.
type Budget struct {
	Kind    Kind
	ID      string
	Name    string
	Mode    Mode
	Amount  money.Money
	Planned money.Money
	PlanIDs []string
}

type GlobalBudget struct {
	ID     string
	Name   string
	Mode   Mode
	Amount money.Amount
	Plans  []PlanCost
}

type PlanCost struct {
	PlanID  string
	Planned money.Money
}

type Overage struct {
	Budget      errs.RecordRef
	Amount      money.Money
	PlannedCost money.Money
	OverBy      money.Money
}

type Control struct {
	Mode      Mode               `json:"mode"`
	CanChange authz.Availability `json:"canChange"`
}

// Limit is a Refuse budget above an activity and what is left to plan in it; negative when already over (FR-014).
type Limit struct {
	Budget     errs.RecordRef `json:"budget"`
	LeftToPlan money.Money    `json:"leftToPlan"`
}

type Path struct {
	PlanID, CampaignID, InitiativeID string
	PlanCurrency                     string
}

type Configuration struct {
	DefaultMode Mode `json:"defaultMode"`
	record.Meta
}

func DefaultConfiguration() Configuration {
	return Configuration{DefaultMode: Warn, Meta: record.Meta{Version: 1, CreatedBy: actor.System, UpdatedBy: actor.System}}
}

type ConfigurationView struct {
	DefaultMode             Mode               `json:"defaultMode"`
	ApprovalOverPlannedCost Mode               `json:"approvalOverPlannedCost"`
	Version                 int                `json:"version"`
	UpdatedAt               time.Time          `json:"updatedAt"`
	UpdatedBy               actor.Actor        `json:"updatedBy"`
	CanChange               authz.Availability `json:"canChange"`
}

type ConfigurationInput struct {
	Version     int    `json:"version"`
	DefaultMode string `json:"defaultMode"`
}
