// Package bulk runs one action on many selected campaigns, initiatives and activities (feature 012): each
// record through its owning area's single operation, with a preview that is the same run, discarded.
package bulk

import (
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
)

type Level string

const (
	LevelCampaign   Level = "campaign"
	LevelInitiative Level = "initiative"
	LevelActivity   Level = "activity"
)

type Kind string

const (
	ActionChangeOwner Kind = "changeOwner"
	ActionChangeField Kind = "changeField"
	ActionSubmit      Kind = "submit"
	ActionCopy        Kind = "copy"
	ActionMove        Kind = "move"
	ActionDelete      Kind = "delete"
)

const (
	OutcomeDone    = "done"
	OutcomeSkipped = "skipped"
	OutcomeFailed  = "failed"
)

const (
	StatusCreated   = "created"
	StatusSubmitted = "submitted"
	StatusDeclined  = "declined"
	StatusApproved  = "approved"
)

type FieldValue struct {
	Date      *date.Date `json:"date,omitempty"`
	OptionIDs []string   `json:"optionIds,omitempty"`
	YesNo     *bool      `json:"yesNo,omitempty"`
}

type FieldChange struct {
	Key   string      `json:"key"`
	Value *FieldValue `json:"value"`
}

type Request struct {
	Action          Kind         `json:"action"`
	RecordIDs       []string     `json:"recordIds"`
	OwnerID         string       `json:"ownerId"`
	Field           *FieldChange `json:"field"`
	TargetID        string       `json:"targetId"`
	RestartApproval bool         `json:"restartApproval"`
	Comment         string       `json:"comment"`
}

// Record is a selected record as the owning area shows it to the person acting.
type Record struct {
	ID        string
	Level     Level
	Name      string
	PlanID    string
	PlanName  string
	Year      int
	Ancestors []string
	Status    string
	Shown     string
}

// Below counts what goes with a record that is deleted, and how many activities among it and below it are
// Submitted or Approved.
type Below struct {
	Counts     BelowCounts
	InApproval int
}

type RecordRef struct {
	RecordID string `json:"recordId"`
	Level    Level  `json:"level"`
	Name     string `json:"name"`
	PlanID   string `json:"planId"`
	Status   string `json:"status,omitempty"`
}

type Outcome struct {
	RecordID    string   `json:"recordId"`
	Level       Level    `json:"level"`
	Name        string   `json:"name"`
	PlanID      string   `json:"planId"`
	Outcome     string   `json:"outcome"`
	Reasons     []string `json:"reasons"`
	NewRecordID string   `json:"newRecordId,omitempty"`
}

type Effect struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
	Text  string `json:"text"`
}

type LevelCounts struct {
	Campaigns   int `json:"campaigns"`
	Initiatives int `json:"initiatives"`
	Activities  int `json:"activities"`
}

type BelowCounts struct {
	Campaigns   int `json:"campaigns"`
	Initiatives int `json:"initiatives"`
	Activities  int `json:"activities"`
	LineItems   int `json:"lineItems"`
}

type DeleteSummary struct {
	Selected            LevelCounts `json:"selected"`
	Below               BelowCounts `json:"below"`
	SubmittedOrApproved int         `json:"submittedOrApproved"`
}

type Counts struct {
	Done    int `json:"done"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

type Result struct {
	ID                 *string                `json:"id"`
	Preview            bool                   `json:"preview"`
	Action             Kind                   `json:"action"`
	Verb               string                 `json:"verb"`
	Note               string                 `json:"note"`
	Selected           LevelCounts            `json:"selected"`
	Counts             Counts                 `json:"counts"`
	Outcomes           []Outcome              `json:"outcomes"`
	Reapproval         []RecordRef            `json:"reapproval"`
	Effects            []Effect               `json:"effects"`
	DeleteSummary      *DeleteSummary         `json:"deleteSummary"`
	RecycleBinEntries  []history.BinEntryView `json:"recycleBinEntries"`
	UndoAvailableUntil *time.Time             `json:"undoAvailableUntil"`
}

type UndoResult struct {
	BulkActionID string                  `json:"bulkActionId"`
	Restored     []history.RestoreResult `json:"restored"`
}

// Action is a stored bulk action: what Undo of a bulk delete needs.
type Action struct {
	ID          string
	Kind        Kind
	Note        string
	Actor       actor.Actor
	RanAt       time.Time
	BinEntryIDs []string
	UndoneAt    *time.Time
	UndoneBy    *actor.Actor
}

type FieldOption struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	ParentOptionIDs []string `json:"parentOptionIds"`
}

type Field struct {
	Key       string        `json:"key"`
	Label     string        `json:"label"`
	Kind      string        `json:"kind"`
	Required  bool          `json:"required"`
	Options   []FieldOption `json:"options"`
	DependsOn *string       `json:"dependsOn"`
	Condition *string       `json:"condition"`
}

type FieldList struct {
	Level      Level   `json:"level"`
	Items      []Field `json:"items"`
	Total      int     `json:"total"`
	NotOffered string  `json:"notOffered"`
}

type TargetQuery struct {
	Action    Kind     `json:"action"`
	RecordIDs []string `json:"recordIds"`
	Search    string   `json:"search"`
}

type Target struct {
	ID        string     `json:"id"`
	Level     string     `json:"level"`
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	PlanID    string     `json:"planId"`
	Year      int        `json:"year"`
	StartDate *date.Date `json:"startDate,omitempty"`
	EndDate   *date.Date `json:"endDate,omitempty"`
}

type TargetPage struct {
	Items []Target `json:"items"`
	Total int      `json:"total"`
	Note  *string  `json:"note"`
}

// Consequences are what changes below campaigns that land in another plan (FR-018).
type Consequences struct {
	OtherPlanName   string
	ModelName       string
	TypesNotAllowed int
	GroupsHidden    []string
	GroupsShown     []string
}
