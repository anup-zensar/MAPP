package settings

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"io"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The reading of an option import file (FR-031, D-051): a .csv or .xlsx file, read in memory with the
// standard library — never written to disk or logged, because it may hold business codes.

// importRow is one row of an import file: its number in the file (the header is row 1) and the texts of
// its first two columns, the option and its parent option, as they are in the file.
type importRow struct {
	number        int
	label, parent string
}

// The files readRows refuses.
var (
	errOldExcel   = errors.New("settings: an .xls file, which MAPP does not read")
	errUnreadable = errors.New("settings: the file can't be read as .csv or .xlsx")
)

// The first bytes of the file types readRows tells apart.
var (
	zipSignature = []byte("PK\x03\x04")                       // .xlsx: a zip archive
	xlsSignature = []byte("\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1") // .xls: Excel 97–2003
	byteOrderUTF = []byte("\xEF\xBB\xBF")                     // the byte-order mark Excel writes in UTF-8 CSV files
)

// maxUnzippedBytes is the most readRows unpacks from one part of an .xlsx file: far more than a sheet of
// 5,000 rows, and a guard against a small file that unpacks to gigabytes.
const maxUnzippedBytes = 20 << 20

// readRows returns the rows of an import file in order: an .xlsx file (it starts with the zip signature)
// from its first worksheet, columns A and B, with the row numbers the file gives; anything else as CSV in
// UTF-8 — without its byte-order mark, separated by ';' when the first line has more ';' than ',' — with the
// line numbers as row numbers. A row whose first two cells are empty is left out, as CSV leaves out a blank
// line. An .xls file is refused with errOldExcel, anything else it cannot read with errUnreadable.
func readRows(content []byte) ([]importRow, error) {
	switch {
	case bytes.HasPrefix(content, zipSignature):
		return readXLSX(content)
	case bytes.HasPrefix(content, xlsSignature):
		return nil, errOldExcel
	}
	return readCSV(content)
}

// readCSV reads the rows of a CSV file.
func readCSV(content []byte) ([]importRow, error) {
	content = bytes.TrimPrefix(content, byteOrderUTF)
	if !utf8.Valid(content) {
		return nil, errUnreadable
	}
	reader := csv.NewReader(bytes.NewReader(content))
	firstLine, _, _ := bytes.Cut(content, []byte("\n"))
	if bytes.Count(firstLine, []byte(";")) > bytes.Count(firstLine, []byte(",")) {
		reader.Comma = ';'
	}
	reader.FieldsPerRecord = -1 // rows may have one column or more
	reader.LazyQuotes = true    // a stray quote in a label is part of it
	rows := []importRow{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			return rows, nil
		}
		if err != nil {
			return nil, errUnreadable
		}
		line, _ := reader.FieldPos(0)
		rows = appendRow(rows, importRow{number: line, label: cellOf(record, 0), parent: cellOf(record, 1)})
	}
}

// cellOf returns the cell i of a CSV record, or "" when the record is shorter.
func cellOf(record []string, i int) string {
	if i < len(record) {
		return record[i]
	}
	return ""
}

// appendRow adds row to rows unless its first two cells are empty.
func appendRow(rows []importRow, row importRow) []importRow {
	if strings.TrimSpace(row.label) == "" && strings.TrimSpace(row.parent) == "" {
		return rows
	}
	return append(rows, row)
}

// The parts of an .xlsx file readXLSX reads (Office Open XML, ECMA-376). encoding/xml matches the element
// names whatever their namespace.
type (
	xlsxWorkbook struct {
		Sheets []struct {
			RelationID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	xlsxRelations struct {
		Relations []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	// xlsxText is a shared string (<si>) or an inline string (<is>): plain text, or rich-text runs.
	xlsxText struct {
		Text string `xml:"t"`
		Runs []struct {
			Text string `xml:"t"`
		} `xml:"r"`
	}
	xlsxSharedStrings struct {
		Items []xlsxText `xml:"si"`
	}
	xlsxSheet struct {
		Rows []struct {
			Number int `xml:"r,attr"`
			Cells  []struct {
				Ref    string   `xml:"r,attr"` // "B12"; may be left out
				Type   string   `xml:"t,attr"` // "s" shared string, "inlineStr", otherwise the value as it is
				Value  string   `xml:"v"`
				Inline xlsxText `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
)

// String returns the text of a shared or inline string.
func (x xlsxText) String() string {
	text := x.Text
	for _, run := range x.Runs {
		text += run.Text
	}
	return text
}

// readXLSX reads the rows of an .xlsx file: columns A and B of its first worksheet. A cell holds a shared
// string, an inline string, or a value written as it is (a number such as 7250080).
func readXLSX(content []byte) ([]importRow, error) {
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, errUnreadable
	}
	parts := map[string]*zip.File{}
	for _, f := range archive.File {
		parts[f.Name] = f
	}
	var shared xlsxSharedStrings
	if part, found := parts["xl/sharedStrings.xml"]; found {
		if err := decodePart(part, &shared); err != nil {
			return nil, err
		}
	}
	part, found := parts[firstSheetPath(parts)]
	if !found {
		return nil, errUnreadable
	}
	var sheet xlsxSheet
	if err := decodePart(part, &sheet); err != nil {
		return nil, err
	}

	rows := []importRow{}
	number := 0
	for _, r := range sheet.Rows {
		number++
		if r.Number > 0 {
			number = r.Number
		}
		row := importRow{number: number}
		for i, c := range r.Cells {
			column := i
			if c.Ref != "" {
				column = columnOf(c.Ref)
			}
			if column != 0 && column != 1 {
				continue // only columns A and B are read
			}
			text := c.Value
			switch c.Type {
			case "s":
				index, err := strconv.Atoi(c.Value)
				if err != nil || index < 0 || index >= len(shared.Items) {
					return nil, errUnreadable
				}
				text = shared.Items[index].String()
			case "inlineStr":
				text = c.Inline.String()
			}
			if column == 0 {
				row.label = text
			} else {
				row.parent = text
			}
		}
		rows = appendRow(rows, row)
	}
	return rows, nil
}

// firstSheetPath returns where the first worksheet of the workbook is: the target of the first sheet's
// relation, or the usual place when the workbook does not say.
func firstSheetPath(parts map[string]*zip.File) string {
	const usual = "xl/worksheets/sheet1.xml"
	var workbook xlsxWorkbook
	var relations xlsxRelations
	workbookPart, found := parts["xl/workbook.xml"]
	relationsPart, foundRelations := parts["xl/_rels/workbook.xml.rels"]
	if !found || !foundRelations || decodePart(workbookPart, &workbook) != nil || decodePart(relationsPart, &relations) != nil ||
		len(workbook.Sheets) == 0 {
		return usual
	}
	for _, rel := range relations.Relations {
		if rel.ID == workbook.Sheets[0].RelationID {
			if strings.HasPrefix(rel.Target, "/") {
				return strings.TrimPrefix(rel.Target, "/")
			}
			return path.Join("xl", rel.Target)
		}
	}
	return usual
}

// decodePart decodes the XML part of an .xlsx file into v, unpacking at most maxUnzippedBytes.
func decodePart(part *zip.File, v any) error {
	reader, err := part.Open()
	if err != nil {
		return errUnreadable
	}
	defer reader.Close()
	if err := xml.NewDecoder(io.LimitReader(reader, maxUnzippedBytes)).Decode(v); err != nil {
		return errUnreadable
	}
	return nil
}

// columnOf returns the column of a cell reference, from 0: "A12" → 0, "B3" → 1, "AA1" → 26.
func columnOf(ref string) int {
	column := 0
	for _, r := range ref {
		if r < 'A' || r > 'Z' {
			break
		}
		column = column*26 + int(r-'A'+1)
	}
	return column - 1
}
