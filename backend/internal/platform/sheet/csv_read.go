package sheet

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

// ReadCSV reads a CSV file in UTF-8, with or without the byte-order mark, separated by commas, semicolons or
// tabs (whichever the first line holds most of), as one table without a name. More than maxRows rows
// holding text is ErrTooManyRows; a file that is not UTF-8 text is ErrUnreadable.
func ReadCSV(data []byte, maxRows int) (Table, error) {
	data = bytes.TrimPrefix(data, []byte(byteOrderMark))
	if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return Table{}, ErrUnreadable
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = delimiterOf(data)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	table := Table{Rows: []Row{}}
	number := 0
	for {
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			return table, nil
		}
		if err != nil {
			return Table{}, ErrUnreadable
		}
		number++
		cells := make([]string, 0, min(len(record), maxColumns))
		for _, field := range record[:min(len(record), maxColumns)] {
			cells = append(cells, strings.TrimSpace(field))
		}
		if !hasText(cells) {
			continue
		}
		if len(table.Rows) == maxRows {
			return Table{}, ErrTooManyRows
		}
		table.Rows = append(table.Rows, Row{Number: number, Cells: cells})
	}
}

func delimiterOf(data []byte) rune {
	first, _, _ := bytes.Cut(data, []byte("\n"))
	best, bestCount := ',', bytes.Count(first, []byte(","))
	for _, d := range []rune{';', '\t'} {
		if n := bytes.Count(first, []byte(string(d))); n > bestCount {
			best, bestCount = d, n
		}
	}
	return best
}
