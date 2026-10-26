package settings

import (
	"archive/zip"
	"bytes"
	"errors"
	"reflect"
	"testing"
)

// A CSV file is read in UTF-8 without its byte-order mark, separated by commas — or by semicolons when the
// first line has more of them — with quoted cells and the file's line numbers; blank rows are left out
// (FR-031).
func TestReadRows_CSVCommaSemicolonBOM(t *testing.T) {
	t.Parallel()
	comma := "\xEF\xBB\xBFOption,Parent option\r\nA,X\r\n\"B, quoted\",Y\r\n\r\n,\r\n  ,Z\r\nC\r\n"
	semicolon := "Option;Parent option\nA;X, Y\n\"B; quoted\";Z;extra\n"

	gotComma, err1 := readRows([]byte(comma))
	gotSemicolon, err2 := readRows([]byte(semicolon))
	_, notUTF8 := readRows([]byte("Option\n\xff\xfe\n"))

	wantComma := []importRow{{1, "Option", "Parent option"}, {2, "A", "X"}, {3, "B, quoted", "Y"}, {6, "  ", "Z"}, {7, "C", ""}}
	if err1 != nil || !reflect.DeepEqual(gotComma, wantComma) {
		t.Errorf("comma = %+v, %v\nwant %+v", gotComma, err1, wantComma)
	}
	wantSemicolon := []importRow{{1, "Option", "Parent option"}, {2, "A", "X, Y"}, {3, "B; quoted", "Z"}}
	if err2 != nil || !reflect.DeepEqual(gotSemicolon, wantSemicolon) {
		t.Errorf("semicolon = %+v, %v\nwant %+v", gotSemicolon, err2, wantSemicolon)
	}
	if !errors.Is(notUTF8, errUnreadable) {
		t.Errorf("a file that is not UTF-8: %v, want errUnreadable", notUTF8)
	}
}

// An .xlsx file is read from its first worksheet — as the workbook orders them — columns A and B, with
// shared strings (also rich text), inline strings, numbers and the row numbers of the file (FR-031).
func TestReadRows_XLSXSharedAndInlineStrings(t *testing.T) {
	t.Parallel()
	content := xlsxFile(t, map[string]string{
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
			`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>` +
			`<sheet name="Options" sheetId="2" r:id="rId7"/><sheet name="Notes" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
			`<Relationship Id="rId7" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/>` +
			`</Relationships>`,
		"xl/sharedStrings.xml": `<?xml version="1.0" encoding="UTF-8"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
			`<si><t>Option</t></si><si><t>Parent option</t></si><si><r><t>CN04</t></r><r><rPr><b/></rPr><t>000220</t></r></si>` +
			`<si><t xml:space="preserve"> Retail – China </t></si></sst>`,
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>` +
			`<row r="1"><c r="A1" t="inlineStr"><is><t>Not this sheet</t></is></c></row></sheetData></worksheet>`,
		"xl/worksheets/sheet2.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>` +
			`<row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>` +
			`<row r="3"><c r="A3" t="s"><v>2</v></c><c r="B3" t="s"><v>3</v></c><c r="C3" t="inlineStr"><is><t>ignored</t></is></c></row>` +
			`<row r="4"><c r="A4"><v>7250080</v></c><c r="B4" t="inlineStr"><is><r><t>Spend </t></r><r><t>type</t></r></is></c></row>` +
			`<row r="5"><c r="B5" s="1"/></row>` +
			`<row><c t="inlineStr"><is><t>No references</t></is></c><c t="inlineStr"><is><t>Second</t></is></c></row>` +
			`</sheetData></worksheet>`,
	})

	got, err := readRows(content)

	want := []importRow{{1, "Option", "Parent option"}, {3, "CN04000220", " Retail – China "}, {4, "7250080", "Spend type"}, {6, "No references", "Second"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %+v, %v\nwant %+v", got, err, want)
	}
	_, badIndex := readRows(xlsxFile(t, map[string]string{
		"xl/worksheets/sheet1.xml": `<worksheet><sheetData><row r="1"><c r="A1" t="s"><v>9</v></c></row></sheetData></worksheet>`,
	}))
	if !errors.Is(badIndex, errUnreadable) {
		t.Errorf("a shared string that does not exist: %v, want errUnreadable", badIndex)
	}
}

// xlsxFile returns a zip archive holding parts, as an .xlsx file.
func xlsxFile(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, text := range parts {
		w, err := archive.Create(name)
		if err != nil {
			t.Fatalf("zip: %v", err)
		}
		if _, err := w.Write([]byte(text)); err != nil {
			t.Fatalf("zip: %v", err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("zip: %v", err)
	}
	return buffer.Bytes()
}

// An Excel 97–2003 .xls file is refused with its own reason; a damaged .xlsx file and an .xlsx file
// without a worksheet cannot be read (FR-031).
func TestReadRows_RefusesXLS(t *testing.T) {
	t.Parallel()
	_, xls := readRows(append([]byte("\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1"), make([]byte, 512)...))
	_, damaged := readRows([]byte("PK\x03\x04 this is not a zip archive"))
	_, noSheet := readRows(xlsxFile(t, map[string]string{"docProps/app.xml": "<Properties/>"}))

	if !errors.Is(xls, errOldExcel) {
		t.Errorf(".xls: %v, want errOldExcel", xls)
	}
	if !errors.Is(damaged, errUnreadable) || !errors.Is(noSheet, errUnreadable) {
		t.Errorf("damaged: %v; no sheet: %v; want errUnreadable", damaged, noSheet)
	}
}
