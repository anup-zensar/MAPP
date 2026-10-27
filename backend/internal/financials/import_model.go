package financials

import (
	"time"

	"mapp/internal/platform/actor"
)

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
