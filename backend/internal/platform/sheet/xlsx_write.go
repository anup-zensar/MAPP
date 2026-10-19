package sheet

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// The .xlsx file (Office Open XML SpreadsheetML) as a zip archive of XML parts: the content types, the
// package relationships, the workbook with its relationships, the styles and one worksheet per sheet. Text is
// written as inline strings, so no shared-string table is needed.

// The styles of xl/styles.xml, by their index in cellXfs.
const (
	styleNormal   = 0
	styleBold     = 1
	styleMoney    = 2 // #,##0.00
	styleDate     = 3 // dd mmm yyyy
	styleDateTime = 4 // dd mmm yyyy hh:mm
)

// maxSheetName is the longest sheet name Excel accepts.
const maxSheetName = 31

// excelEpoch is day 0 of Excel's date numbers (for dates from 1 March 1900 on).
var excelEpoch = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)

// WriteXLSX writes wb to w as an .xlsx file.
func WriteXLSX(w io.Writer, wb Workbook) error {
	z := zip.NewWriter(w)
	names := sheetNames(wb.Sheets)
	parts := []struct {
		name string
		body []byte
	}{
		{"[Content_Types].xml", contentTypes(len(wb.Sheets))},
		{"_rels/.rels", []byte(xmlHeader + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
			`</Relationships>`)},
		{"xl/workbook.xml", workbookXML(names)},
		{"xl/_rels/workbook.xml.rels", workbookRels(len(wb.Sheets))},
		{"xl/styles.xml", []byte(stylesXML)},
	}
	for i, s := range wb.Sheets {
		parts = append(parts, struct {
			name string
			body []byte
		}{fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1), worksheetXML(s.Rows)})
	}
	for _, p := range parts {
		f, err := z.CreateHeader(&zip.FileHeader{Name: p.name, Method: zip.Deflate})
		if err != nil {
			return err
		}
		if _, err := f.Write(p.body); err != nil {
			return err
		}
	}
	return z.Close()
}

// xmlHeader starts every XML part.
const xmlHeader = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"

// contentTypes returns [Content_Types].xml for n worksheets.
func contentTypes(n int) []byte {
	var b strings.Builder
	b.WriteString(xmlHeader + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
		`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i)
	}
	b.WriteString(`</Types>`)
	return []byte(b.String())
}

// workbookXML returns xl/workbook.xml with the sheets named names.
func workbookXML(names []string) []byte {
	var b strings.Builder
	b.WriteString(xmlHeader + `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	for i, name := range names {
		fmt.Fprintf(&b, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, escape(name), i+1, i+1)
	}
	b.WriteString(`</sheets></workbook>`)
	return []byte(b.String())
}

// workbookRels returns xl/_rels/workbook.xml.rels: rId1 … rIdN the worksheets, then the styles.
func workbookRels(n int) []byte {
	var b strings.Builder
	b.WriteString(xmlHeader + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i, i)
	}
	fmt.Fprintf(&b, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, n+1)
	b.WriteString(`</Relationships>`)
	return []byte(b.String())
}

// stylesXML is xl/styles.xml: the number formats, a bold font and the cell styles of the style constants.
const stylesXML = xmlHeader + `<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
	`<numFmts count="3"><numFmt numFmtId="164" formatCode="#,##0.00"/><numFmt numFmtId="165" formatCode="dd mmm yyyy"/>` +
	`<numFmt numFmtId="166" formatCode="dd mmm yyyy hh:mm"/></numFmts>` +
	`<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts>` +
	`<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>` +
	`<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>` +
	`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
	`<cellXfs count="5"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>` +
	`<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/>` +
	`<xf numFmtId="164" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/>` +
	`<xf numFmtId="165" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/>` +
	`<xf numFmtId="166" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/></cellXfs>` +
	`<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`

// worksheetXML returns a worksheet with rows; empty cells are left out.
func worksheetXML(rows [][]Cell) []byte {
	var b strings.Builder
	b.WriteString(xmlHeader + `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for r, row := range rows {
		fmt.Fprintf(&b, `<row r="%d">`, r+1)
		for c, cell := range row {
			if cell.kind == kindEmpty {
				continue
			}
			ref := columnName(c) + strconv.Itoa(r+1)
			switch cell.kind {
			case kindText:
				style := styleNormal
				if cell.bold {
					style = styleBold
				}
				fmt.Fprintf(&b, `<c r="%s" s="%d" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, style, escape(cell.text))
			case kindNumber:
				fmt.Fprintf(&b, `<c r="%s"><v>%s</v></c>`, ref, cell.text)
			case kindMoney:
				fmt.Fprintf(&b, `<c r="%s" s="%d"><v>%s</v></c>`, ref, styleMoney, cell.text)
			case kindDate:
				days := int64(cell.date.Time().Sub(excelEpoch) / (24 * time.Hour))
				fmt.Fprintf(&b, `<c r="%s" s="%d"><v>%d</v></c>`, ref, styleDate, days)
			case kindDateTime:
				serial := float64(cell.moment.Sub(excelEpoch)) / float64(24*time.Hour)
				fmt.Fprintf(&b, `<c r="%s" s="%d"><v>%s</v></c>`, ref, styleDateTime, strconv.FormatFloat(serial, 'f', 8, 64))
			}
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return []byte(b.String())
}

// columnName returns the letters of the column with index i: 0 → A, 25 → Z, 26 → AA.
func columnName(i int) string {
	name := ""
	for i++; i > 0; i = (i - 1) / 26 {
		name = string(rune('A'+(i-1)%26)) + name
	}
	return name
}

// sheetNames returns names Excel accepts: without : \ / ? * [ ], at most 31 characters, never empty.
func sheetNames(sheets []Sheet) []string {
	names := make([]string, len(sheets))
	for i, s := range sheets {
		name := strings.Map(func(r rune) rune {
			if strings.ContainsRune(`:\/?*[]`, r) {
				return -1
			}
			return r
		}, s.Name)
		if runes := []rune(name); len(runes) > maxSheetName {
			name = string(runes[:maxSheetName])
		}
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("Sheet%d", i+1)
		}
		names[i] = name
	}
	return names
}

// escape returns s with the characters XML gives a meaning escaped, and those XML cannot hold replaced.
func escape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s)) // writing to a bytes.Buffer cannot fail
	return b.String()
}
