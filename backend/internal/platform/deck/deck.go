// Package deck writes pages of figures — cards, a table, a bar chart, notes — as a PowerPoint presentation or a
// PDF document with the standard library only (ADR-0017): each page is laid out once, then drawn by either writer.
package deck

import "time"

type Colour string

const (
	Ink            Colour = "292929"
	Subtle         Colour = "757575"
	Hairline       Colour = "E0E0E0"
	Band           Colour = "F5F5F5"
	White          Colour = "FFFFFF"
	Danger         Colour = "DD1D21"
	ChartBudget    Colour = "5B6B7A"
	ChartPlanned   Colour = "336094"
	ChartCommitted Colour = "008192"
	ChartActual    Colour = "008557"
)

type Align int

const (
	Left Align = iota
	Right
	Center
)

type Deck struct {
	Title   string
	Author  string
	Created time.Time
	Footer  string
	Pages   []Page
}

// Page is one page as the caller describes it; a table too long for it continues on pages titled
// "<Title> (continued)", and a chart sits beside the table when there is one.
type Page struct {
	Title    string
	Subtitle string
	Cards    []Card
	Chart    *Chart
	Table    *Table
	Notes    []string
}

type Card struct {
	Label, Value, Note string
	Problem            bool
}

type Column struct {
	Title  string
	Weight int
	Align  Align
}

type Row struct {
	Cells []string
	Bold  bool
}

type Table struct {
	Columns []Column
	Rows    []Row
}

type Series struct {
	Name   string
	Colour Colour
	Values []int64
}

type Chart struct {
	Categories []string
	Series     []Series
	MaxLabel   string
}
