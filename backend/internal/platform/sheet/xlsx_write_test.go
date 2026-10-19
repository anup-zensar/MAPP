package sheet

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
)

// unzip writes wb as .xlsx and returns its parts by name; every XML part must be well-formed.
func unzip(t *testing.T, wb Workbook) map[string]string {
	t.Helper()
	var buf bytes.Buffer
	if err := WriteXLSX(&buf, wb); err != nil {
		t.Fatalf("WriteXLSX: %v", err)
	}
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("not a zip archive: %v", err)
	}
	parts := map[string]string{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		decoder := xml.NewDecoder(bytes.NewReader(body))
		for {
			if _, err := decoder.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("%s is not well-formed XML: %v\n%s", f.Name, err, body)
			}
		}
		parts[f.Name] = string(body)
	}
	return parts
}

// An .xlsx file is a zip archive with the content types, the package and workbook relationships, the
// workbook naming its sheets in order, the styles and one worksheet per sheet holding its text as inline
// strings (D-120).
func TestXLSX_OpensAsZipWithWorkbookSheetsStrings(t *testing.T) {
	parts := unzip(t, Workbook{Sheets: []Sheet{
		{Name: "About this export", Rows: [][]Cell{{Text("View"), Text("Grid")}}},
		{Name: "Grid", Rows: [][]Cell{{Bold("Name")}, {Text("Launch film")}}},
	}})

	var names []string
	for name := range parts {
		names = append(names, name)
	}
	slices.Sort(names)
	want := []string{"[Content_Types].xml", "_rels/.rels", "xl/_rels/workbook.xml.rels", "xl/styles.xml",
		"xl/workbook.xml", "xl/worksheets/sheet1.xml", "xl/worksheets/sheet2.xml"}
	if !slices.Equal(names, want) {
		t.Fatalf("parts = %q", names)
	}
	if !strings.Contains(parts["xl/workbook.xml"], `<sheet name="About this export" sheetId="1" r:id="rId1"/><sheet name="Grid" sheetId="2" r:id="rId2"/>`) {
		t.Errorf("workbook = %s", parts["xl/workbook.xml"])
	}
	if !strings.Contains(parts["xl/_rels/workbook.xml.rels"], `Target="worksheets/sheet2.xml"`) ||
		!strings.Contains(parts["xl/_rels/workbook.xml.rels"], `Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles"`) {
		t.Errorf("relationships = %s", parts["xl/_rels/workbook.xml.rels"])
	}
	if !strings.Contains(parts["[Content_Types].xml"], `/xl/worksheets/sheet2.xml`) {
		t.Errorf("content types = %s", parts["[Content_Types].xml"])
	}
	if !strings.Contains(parts["xl/worksheets/sheet2.xml"], `<row r="2"><c r="A2" s="0" t="inlineStr"><is><t xml:space="preserve">Launch film</t></is></c></row>`) {
		t.Errorf("sheet 2 = %s", parts["xl/worksheets/sheet2.xml"])
	}
}

// Numbers, money and dates are numbers in the file — money with the format #,##0.00, dates as day numbers
// with dd mmm yyyy, moments with dd mmm yyyy hh:mm — headings are bold, and empty cells are left out
// (D-120, FR-033).
func TestXLSX_NumbersDatesBoldHeadings(t *testing.T) {
	amount, _ := money.ParseAmount("12500.5")
	parts := unzip(t, Workbook{Sheets: []Sheet{{Name: "Grid", Rows: [][]Cell{
		{Bold("Planned cost (USD)"), Bold("Start date"), Bold("INQ"), {}, Bold("Created")},
		{Money(amount), Date(date.MustParse("2026-03-01")), Number(250000), Text(""), DateTime(time.Date(2026, 3, 12, 6, 0, 0, 0, time.UTC))},
		{Percent(80), Date(date.Date{}), Money(amount.Neg())},
	}}}})

	sheet := parts["xl/worksheets/sheet1.xml"]
	for _, want := range []string{
		`<c r="A1" s="1" t="inlineStr"><is><t xml:space="preserve">Planned cost (USD)</t></is></c>`,
		`<c r="A2" s="2"><v>12500.50</v></c>`,
		`<c r="B2" s="3"><v>46082</v></c>`, // 1 Mar 2026
		`<c r="C2"><v>250000</v></c>`,
		`<c r="E2" s="4"><v>46093.25000000</v></c>`, // 12 Mar 2026 06:00
		`<row r="3"><c r="A3"><v>80</v></c><c r="C3" s="2"><v>-12500.50</v></c></row>`,
	} {
		if !strings.Contains(sheet, want) {
			t.Errorf("sheet lacks %s\n%s", want, sheet)
		}
	}
	if strings.Contains(sheet, `r="D1"`) || strings.Contains(sheet, `r="D2"`) {
		t.Errorf("empty cells are written: %s", sheet)
	}
	styles := parts["xl/styles.xml"]
	for _, want := range []string{`formatCode="#,##0.00"`, `formatCode="dd mmm yyyy"`, `<b/>`} {
		if !strings.Contains(styles, want) {
			t.Errorf("styles lack %s", want)
		}
	}
	if columnName(0) != "A" || columnName(25) != "Z" || columnName(26) != "AA" || columnName(27) != "AB" || columnName(701) != "ZZ" ||
		columnName(702) != "AAA" {
		t.Errorf("column names: %s %s %s %s %s %s", columnName(0), columnName(25), columnName(26), columnName(27), columnName(701), columnName(702))
	}
}

// Text is escaped for XML, and sheet names lose the characters Excel refuses and are cut to 31 characters
// (D-120): "Budget & cost" keeps its name.
func TestXLSX_EscapesTextAndSheetNames(t *testing.T) {
	parts := unzip(t, Workbook{Sheets: []Sheet{
		{Name: "Budget & cost", Rows: [][]Cell{{Text(`Fish & chips <"quoted"> 'x'`)}}},
		{Name: "A/B: [test]? * with a very long name indeed", Rows: [][]Cell{}},
		{Name: "///", Rows: nil},
	}})

	if !strings.Contains(parts["xl/worksheets/sheet1.xml"], `Fish &amp; chips &lt;&#34;quoted&#34;&gt; &#39;x&#39;`) {
		t.Errorf("text = %s", parts["xl/worksheets/sheet1.xml"])
	}
	workbook := parts["xl/workbook.xml"]
	for _, want := range []string{`name="Budget &amp; cost"`, `name="AB test  with a very long name "`, `name="Sheet3"`} {
		if !strings.Contains(workbook, want) {
			t.Errorf("workbook lacks %s\n%s", want, workbook)
		}
	}
}
