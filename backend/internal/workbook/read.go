package workbook

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/sheet"
)

const (
	maxFileBytes    = 5 << 20
	MaxUploadBytes  = 7 << 20
	maxFileRows     = 5000
	maxFileName     = 255
	fileField       = "file"
	zipSignature    = "PK\x03\x04"
	msgNoLevelSheet = "The file has none of the sheets Campaigns, Initiatives, Activities and Line items — download a fresh template"
)

// sheetRows is one level sheet of an upload: its columns found by heading and its data rows.
type sheetRows struct {
	level   Level
	name    string
	columns []column
	cells   []int // the cell index of each column
	rows    []sheet.Row
}

func (s sheetRows) has(field string) bool {
	return slices.ContainsFunc(s.columns, func(c column) bool { return c.field == field })
}

func (s sheetRows) text(row sheet.Row, field string) string {
	for i, c := range s.columns {
		if c.field == field {
			return row.Cell(s.cells[i])
		}
	}
	return ""
}

// readUpload reads the level sheets of an upload; a problem of the whole file refuses it, every one at once
// (FR-028).
func readUpload(up Upload, ch Choices) ([]sheetRows, []string, error) {
	tables, err := tablesOf(up)
	if err != nil {
		return nil, nil, err
	}
	var sheets []sheetRows
	ignored := []string{}
	var problems errs.Problems
	total := 0
	for _, level := range levels {
		i := slices.IndexFunc(tables, func(t sheet.Table) bool { return normalHeading(t.Name) == normalHeading(sheetName(level)) })
		if i < 0 {
			continue
		}
		s, unknown := matchColumns(level, tables[i], ch)
		ignored = append(ignored, unknown...)
		for _, c := range missingRequired(s) {
			problems.Add(fileField, fmt.Sprintf(msgMissingColumn, sheetName(level), c.heading))
		}
		total += len(s.rows)
		sheets = append(sheets, s)
	}
	switch {
	case len(sheets) == 0:
		problems.Add(fileField, msgNoLevelSheet)
	case total > maxFileRows:
		problems.Add(fileField, msgTooManyRows)
	}
	return sheets, ignored, problems.Err()
}

func tablesOf(up Upload) ([]sheet.Table, error) {
	name := strings.TrimSpace(up.FileName)
	if name == "" || utf8.RuneCountInString(name) > maxFileName {
		return nil, fileProblem(msgFileName)
	}
	data, err := base64.StdEncoding.DecodeString(up.Content)
	switch {
	case err != nil:
		return nil, fileProblem(msgUnreadable)
	case len(data) == 0:
		return nil, fileProblem(msgEmptyFile)
	case len(data) > maxFileBytes:
		return nil, fileProblem(msgTooLarge)
	}
	if bytes.HasPrefix(data, []byte(zipSignature)) {
		tables, xlsxErr := sheet.ReadXLSX(data, maxFileRows+2)
		return tables, readProblem(xlsxErr)
	}
	if !strings.HasSuffix(strings.ToLower(name), ".csv") {
		return nil, fileProblem(msgChooseFile)
	}
	table, err := sheet.ReadCSV(data, maxFileRows+2)
	table.Name = sheetName(LevelActivity)
	return []sheet.Table{table}, readProblem(err)
}

func readProblem(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, sheet.ErrTooManyRows):
		return fileProblem(msgTooManyRows)
	}
	return fileProblem(msgUnreadable)
}

func fileProblem(message string) error {
	return errs.Validation(errs.FieldError{Field: fileField, Message: message})
}

// matchColumns finds the level's columns by heading in the first row; the rows after it, but a row of
// markings, are the data. Headings no column has are named as ignored.
func matchColumns(level Level, t sheet.Table, ch Choices) (sheetRows, []string) {
	s := sheetRows{level: level, name: t.Name, rows: []sheet.Row{}}
	if len(t.Rows) == 0 {
		return s, nil
	}
	known := columnsOf(level, ch)
	ignored := []string{}
	for i, heading := range t.Rows[0].Cells {
		j := columnByHeading(known, heading)
		switch {
		case heading == "":
		case j < 0 || s.has(known[j].field):
			ignored = append(ignored, t.Name+": "+heading)
		default:
			s.columns, s.cells = append(s.columns, known[j]), append(s.cells, i)
		}
	}
	for _, row := range t.Rows[1:] {
		if !isMarkingRow(row) {
			s.rows = append(s.rows, row)
		}
	}
	return s, ignored
}

func isMarkingRow(row sheet.Row) bool {
	markings := []string{markRequired, markPublish, markSubmit, markOptional, markNewEmpty}
	for _, cell := range row.Cells {
		if cell != "" && !slices.Contains(markings, cell) {
			return false
		}
	}
	return true
}

func missingRequired(s sheetRows) []column {
	if len(s.rows) == 0 {
		return nil
	}
	var missing []column
	for _, c := range baseColumns(s.level) {
		if c.required() && !s.has(c.field) {
			missing = append(missing, c)
		}
	}
	return missing
}
