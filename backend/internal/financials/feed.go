package financials

import "context"

// Feed turns one delivery from a source system into incoming rows (FR-003, data model §3). The simulated
// GSAP extract (CSVFeed, feed_csv.go) and the demo data (DemoFeed, feed_demo.go) are feeds now. A real
// GSAP feed (PGSI 941 files) replaces the CSV feed later without a change to the service, the handler, the
// contract or the figures: everything after Read is the same for every feed (SC-006).
type Feed interface {
	// Name is the feed's name as the screens show it: "Simulated GSAP extract".
	Name() string
	// Source says whether the delivery is a file or the demo data.
	Source() Source
	// Read parses a delivery. A problem with the whole delivery is an *errs.Error (validation_failed on
	// "content" or "fileName"); nothing of it is used. Row problems are returned with their row numbers,
	// and those rows are not in the result. unused are the column names the feed does not read.
	Read(ctx context.Context, d Delivery) (rows []Incoming, problems []RowProblem, unused []string, err error)
}

// Delivery is one delivery of GSAP data.
type Delivery struct {
	FileName string
	Content  []byte // the decoded file; nil for the demo feed
}

// Incoming is one row of a delivery, checked by its feed (data model §3).
type Incoming struct {
	Row int // the file's row number: 2 is the first data row, as a spreadsheet numbers it
	Fields
}

// RowProblem is one problem of one row, shown as "Row 3 · amount: …".
type RowProblem struct {
	Row     int
	Column  string // the column's name, such as "amount"; "" for a problem with the whole row
	Message string
}
