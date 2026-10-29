package sheet

import (
	"encoding/xml"
	"errors"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	lastSerialDay   = 2958465 // 31 Dec 9999
	date1904Offset  = 1462
	maxNumberDigits = 9
)

type stylesPart struct {
	NumFmts []struct {
		ID   int    `xml:"numFmtId,attr"`
		Code string `xml:"formatCode,attr"`
	} `xml:"numFmts>numFmt"`
	Xfs []struct {
		NumFmtID int `xml:"numFmtId,attr"`
	} `xml:"cellXfs>xf"`
}

// dateStyles holds the cell styles whose number format shows a date.
type dateStyles map[int]bool

type cellPart struct {
	Ref    string   `xml:"r,attr"`
	Type   string   `xml:"t,attr"`
	Style  int      `xml:"s,attr"`
	Value  string   `xml:"v"`
	Inline richText `xml:"is"`
}

type rowPart struct {
	Number int        `xml:"r,attr"`
	Cells  []cellPart `xml:"c"`
}

func (p stylesPart) dateStyles() dateStyles {
	custom := map[int]string{}
	for _, f := range p.NumFmts {
		custom[f.ID] = f.Code
	}
	styles := dateStyles{}
	for i, xf := range p.Xfs {
		code, isCustom := custom[xf.NumFmtID]
		styles[i] = isCustom && isDateFormat(code) || !isCustom && isBuiltInDate(xf.NumFmtID)
	}
	return styles
}

func isBuiltInDate(id int) bool {
	return (id >= 14 && id <= 22) || (id >= 45 && id <= 47)
}

// isDateFormat reports whether a format code shows a day or a year, quoted texts and [colours] left out.
func isDateFormat(code string) bool {
	var plain strings.Builder
	quoted, bracketed := false, false
	for _, r := range strings.ToLower(code) {
		switch {
		case r == '"':
			quoted = !quoted
		case r == '[' && !quoted:
			bracketed = true
		case r == ']' && !quoted:
			bracketed = false
		case !quoted && !bracketed:
			plain.WriteRune(r)
		}
	}
	return strings.ContainsAny(plain.String(), "dy")
}

// readSheet reads the rows of the worksheet part name that hold text.
func (f *xlsxFile) readSheet(name string, maxRows int) ([]Row, error) {
	part, found := f.parts[name]
	if !found || part.UncompressedSize64 > maxPartBytes {
		return nil, ErrUnreadable
	}
	rc, err := part.Open()
	if err != nil {
		return nil, ErrUnreadable
	}
	defer func() { _ = rc.Close() }()
	d := xml.NewDecoder(io.LimitReader(rc, maxPartBytes))
	rows := []Row{}
	previous := 0
	for {
		rp, done, err := nextRow(d)
		if done || err != nil {
			return rows, err
		}
		row := f.row(rp, previous)
		previous = row.Number
		if !hasText(row.Cells) {
			continue
		}
		if len(rows) == maxRows {
			return nil, ErrTooManyRows
		}
		rows = append(rows, row)
	}
}

// nextRow decodes the next row element; done is true at the end of the part.
func nextRow(d *xml.Decoder) (rowPart, bool, error) {
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return rowPart{}, true, nil
		}
		if err != nil {
			return rowPart{}, false, ErrUnreadable
		}
		start, isStart := tok.(xml.StartElement)
		if !isStart || start.Name.Local != "row" {
			continue
		}
		var rp rowPart
		if err := d.DecodeElement(&rp, &start); err != nil {
			return rowPart{}, false, ErrUnreadable
		}
		return rp, false, nil
	}
}

func (f *xlsxFile) row(rp rowPart, previous int) Row {
	number := rp.Number
	if number <= previous {
		number = previous + 1
	}
	cells := []string{}
	next := 0
	for _, c := range rp.Cells {
		column := next
		if i, ok := columnIndex(c.Ref); ok {
			column = i
		}
		next = column + 1
		if column >= maxColumns {
			continue
		}
		for len(cells) <= column {
			cells = append(cells, "")
		}
		cells[column] = strings.TrimSpace(f.text(c))
	}
	return Row{Number: number, Cells: cells}
}

func (f *xlsxFile) text(c cellPart) string {
	switch c.Type {
	case "s":
		return f.sharedString(c.Value)
	case "inlineStr":
		return c.Inline.String()
	case "b":
		if strings.TrimSpace(c.Value) == "1" {
			return "TRUE"
		}
		return "FALSE"
	case "d":
		date, _, _ := strings.Cut(c.Value, "T")
		return date
	case "str", "e":
		return c.Value
	}
	if d, ok := f.serialDate(c); ok {
		return d
	}
	return plainNumber(c.Value)
}

func (f *xlsxFile) sharedString(value string) string {
	i, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || i < 0 || i >= len(f.strings) {
		return ""
	}
	return f.strings[i]
}

// serialDate turns the day number of a date cell into YYYY-MM-DD; the time of day is dropped.
func (f *xlsxFile) serialDate(c cellPart) (string, bool) {
	if !f.dates[c.Style] {
		return "", false
	}
	serial, err := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
	if err != nil || serial < 1 || serial > lastSerialDay {
		return "", false
	}
	days := int(math.Floor(serial))
	if f.date1904 {
		days += date1904Offset
	}
	return excelEpoch.AddDate(0, 0, days).Format(time.DateOnly), true
}

// plainNumber writes a number without an exponent and without the noise of binary fractions (0.1 + 0.2).
func plainNumber(value string) string {
	value = strings.TrimSpace(value)
	_, decimals, _ := strings.Cut(value, ".")
	if !strings.ContainsAny(value, "eE") && len(decimals) <= maxNumberDigits {
		return value
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsInf(n, 0) || math.IsNaN(n) {
		return value
	}
	return strings.TrimSuffix(strings.TrimRight(strconv.FormatFloat(n, 'f', maxNumberDigits, 64), "0"), ".")
}

// columnIndex reads the column of a cell reference: "A1" → 0, "AB12" → 27.
func columnIndex(ref string) (int, bool) {
	index := 0
	letters := 0
	for _, r := range ref {
		if r < 'A' || r > 'Z' {
			break
		}
		index = index*26 + int(r-'A') + 1
		letters++
		if letters > 3 {
			return 0, false
		}
	}
	return index - 1, letters > 0
}
