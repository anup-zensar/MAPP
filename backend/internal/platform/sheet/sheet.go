// Package sheet writes spreadsheet files with the standard library only (decision D-120): an .xlsx workbook
// of sheets (xlsx_write.go) or a CSV file with notes before the table (csv_write.go). Feature 010 exports
// All work with it; feature 012 adds the readers.
//
// A sheet is rows of cells. Each cell has a kind — text, number, money, date, moment — so that the
// spreadsheet keeps numbers as numbers and dates as dates, with the formats people expect: money #,##0.00,
// dates dd mmm yyyy.
package sheet

import (
	"strconv"
	"time"

	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
)

// kind is what a cell holds.
type kind int

// The kinds of cells.
const (
	kindEmpty kind = iota
	kindText
	kindNumber   // a whole number or a percentage
	kindMoney    // an amount with two decimals
	kindDate     // a calendar date
	kindDateTime // a moment, in UTC
)

// Cell is one cell of a sheet. The zero Cell is empty.
type Cell struct {
	kind   kind
	text   string // text cells; the decimal text of numbers and money
	date   date.Date
	moment time.Time
	bold   bool
}

// Text returns a text cell.
func Text(s string) Cell {
	if s == "" {
		return Cell{}
	}
	return Cell{kind: kindText, text: s}
}

// Bold returns a text cell in bold, such as a heading.
func Bold(s string) Cell {
	c := Text(s)
	c.bold = c.kind != kindEmpty
	return c
}

// Number returns a whole number.
func Number(n int64) Cell {
	return Cell{kind: kindNumber, text: strconv.FormatInt(n, 10)}
}

// Percent returns a whole percentage as a number: 80 for 80 %.
func Percent(n int64) Cell {
	return Number(n)
}

// Money returns an amount as a number with two decimals, shown as #,##0.00.
func Money(a money.Amount) Cell {
	return Cell{kind: kindMoney, text: a.String()}
}

// Date returns a calendar date, shown as dd mmm yyyy; the zero date is an empty cell.
func Date(d date.Date) Cell {
	if d.IsZero() {
		return Cell{}
	}
	return Cell{kind: kindDate, date: d}
}

// DateTime returns a moment in UTC, shown as dd mmm yyyy hh:mm; the zero time is an empty cell.
func DateTime(t time.Time) Cell {
	if t.IsZero() {
		return Cell{}
	}
	return Cell{kind: kindDateTime, moment: t.UTC()}
}

// Sheet is one sheet of a workbook: a name and its rows, top to bottom.
type Sheet struct {
	Name string
	Rows [][]Cell
}

// Workbook is the sheets of an .xlsx file, in order.
type Workbook struct {
	Sheets []Sheet
}
