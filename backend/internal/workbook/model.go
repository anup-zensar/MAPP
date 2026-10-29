// Package workbook is the plan workbook of feature 012: one .xlsx layout per plan, downloaded empty or with the
// plan's records, checked and imported row by row through the owning areas, and undone.
package workbook

import (
	"encoding/json"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

type Level string

const (
	LevelCampaign   Level = "campaign"
	LevelInitiative Level = "initiative"
	LevelActivity   Level = "activity"
	LevelLineItem   Level = "lineItem"
)

const (
	ActionCreated = "created"
	ActionUpdated = "updated"
	StatusDone    = "imported"
	StatusUndone  = "undone"
)

type Upload struct {
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

type Problem struct {
	Sheet  string `json:"sheet"`
	Row    int    `json:"row"`
	Column string `json:"column"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

type LevelCounts struct {
	Level     Level `json:"level"`
	Create    int   `json:"create"`
	Update    int   `json:"update"`
	Unchanged int   `json:"unchanged"`
	LeftOut   int   `json:"leftOut"`
}

type Check struct {
	FileName        string        `json:"fileName"`
	Rows            int           `json:"rows"`
	Ready           int           `json:"ready"`
	WithProblems    int           `json:"withProblems"`
	Levels          []LevelCounts `json:"levels"`
	Problems        []Problem     `json:"problems"`
	IgnoredColumns  []string      `json:"ignoredColumns"`
	NeedMoreFields  int           `json:"needMoreFields"`
	NothingToChange bool          `json:"nothingToChange"`
}

type LevelTotals struct {
	Campaigns   int `json:"campaigns"`
	Initiatives int `json:"initiatives"`
	Activities  int `json:"activities"`
	LineItems   int `json:"lineItems"`
}

type RecordRef struct {
	ID     string `json:"id"`
	Level  Level  `json:"level"`
	Name   string `json:"name"`
	Action string `json:"action"`
	Exists bool   `json:"exists"`
}

type UndoPreview struct {
	LevelTotals
	AlreadyDeleted int `json:"alreadyDeleted"`
}

type Kept struct {
	ID     string `json:"id"`
	Level  Level  `json:"level"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type Summary struct {
	ID          string       `json:"id"`
	PlanID      string       `json:"planId"`
	FileName    string       `json:"fileName"`
	Status      string       `json:"status"`
	ImportedAt  time.Time    `json:"importedAt"`
	ImportedBy  actor.Actor  `json:"importedBy"`
	RowsRead    int          `json:"rowsRead"`
	RowsLeftOut int          `json:"rowsLeftOut"`
	Created     LevelTotals  `json:"created"`
	Updated     LevelTotals  `json:"updated"`
	UndoneAt    *time.Time   `json:"undoneAt"`
	UndoneBy    *actor.Actor `json:"undoneBy"`
}

type Detail struct {
	Import   Summary      `json:"import"`
	Problems []Problem    `json:"problems"`
	Records  []RecordRef  `json:"records"`
	Undo     *UndoPreview `json:"undo"`
	Kept     []Kept       `json:"kept"`
}

type Page struct {
	Items []Summary `json:"items"`
	Total int       `json:"total"`
}

// Import is a stored import: the records it created and updated, and the rows it left out.
type Import struct {
	ID          string
	PlanID      string
	FileName    string
	ImportedAt  time.Time
	ImportedBy  actor.Actor
	RowsRead    int
	RowsLeftOut int
	Records     []ImportRecord
	Problems    []Problem
	UndoneAt    *time.Time
	UndoneBy    *actor.Actor
}

type ImportRecord struct {
	Level    Level
	RecordID string
	Name     string
	ParentID string
	Action   string
}

// Values are the workbook's fields of one record, as the owning areas store them; "" and nil are empty.
type Values struct {
	Name           string
	Description    string
	ExternalName   string
	OwnerID        string
	StartDate      *date.Date
	EndDate        *date.Date
	Budget         *money.Amount
	PlannedCost    *money.Amount
	Ongoing        bool
	ActivityTypeID string
	WorkflowID     string
	Currency       string
	Attributes     map[string]json.RawMessage
}
