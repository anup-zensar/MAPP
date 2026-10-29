package sheet

import "errors"

// A part of an .xlsx file unpacks to at most maxPartBytes, so a small file cannot unpack to gigabytes.
const (
	maxPartBytes = 50 << 20
	maxColumns   = 200
)

var (
	ErrUnreadable  = errors.New("sheet: the file is not a readable .xlsx or .csv file")
	ErrTooManyRows = errors.New("sheet: the file has more rows than allowed")
)

// Table is one sheet of a file as text: its name ("" for a CSV file) and its rows that hold any text.
type Table struct {
	Name string
	Rows []Row
}

type Row struct {
	Number int
	Cells  []string
}

func (r Row) Cell(i int) string {
	if i < 0 || i >= len(r.Cells) {
		return ""
	}
	return r.Cells[i]
}

func hasText(cells []string) bool {
	for _, c := range cells {
		if c != "" {
			return true
		}
	}
	return false
}
