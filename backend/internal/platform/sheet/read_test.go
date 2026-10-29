package sheet

import (
	"archive/zip"
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

func written(tb testing.TB, wb Workbook) []byte {
	tb.Helper()
	var buf bytes.Buffer
	if err := WriteXLSX(&buf, wb); err != nil {
		tb.Fatalf("WriteXLSX: %v", err)
	}
	return buf.Bytes()
}

func zipOf(tb testing.TB, parts map[string]string) []byte {
	tb.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := z.Create(name)
		if err != nil {
			tb.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			tb.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		tb.Fatal(err)
	}
	return buf.Bytes()
}

// handMade is a workbook as Excel saves it: shared strings with runs, a custom date format, a 1904 date
// system, booleans, a formula's saved text and cells without a reference.
func handMade(tb testing.TB, sheetRows string) []byte {
	tb.Helper()
	return zipOf(tb, map[string]string{
		"xl/workbook.xml": `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
			`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><workbookPr date1904="1"/>` +
			`<sheets><sheet name="Activities" sheetId="1" r:id="rId7"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="rId7" Target="/xl/worksheets/data.xml"/></Relationships>`,
		"xl/sharedStrings.xml": `<sst><si><t>Launch film</t></si><si><r><t>Trade </t></r><r><t>fair</t></r>` +
			`<rPh><t>x</t></rPh></si></sst>`,
		"xl/styles.xml": `<styleSheet><numFmts><numFmt numFmtId="170" formatCode="[Red]&quot;on &quot;d/mm/yyyy"/>` +
			`<numFmt numFmtId="171" formatCode="0.00&quot; day&quot;"/></numFmts>` +
			`<cellXfs><xf numFmtId="0"/><xf numFmtId="170"/><xf numFmtId="171"/><xf numFmtId="14"/></cellXfs></styleSheet>`,
		"xl/worksheets/data.xml": `<worksheet><sheetData>` + sheetRows + `</sheetData></worksheet>`,
	})
}

func TestReadXLSX_SharedInlineNumbersDates_012FR028(t *testing.T) {
	t.Parallel()
	data := handMade(t, `<row r="1"><c r="A1" t="s"><v>0</v></c><c r="C1" t="s"><v>1</v></c></row>`+
		`<row r="2"><c r="A2" s="1"><v>44000</v></c><c s="2"><v>3.5</v></c><c r="D2" t="b"><v>1</v></c></row>`+
		`<row r="4"><c r="B4" t="str"><v>=A1</v></c><c r="C4" t="inlineStr"><is><t> padded </t></is></c>`+
		`<c r="E4"><v>1.2E+3</v></c><c r="F4"><v>0.30000000000000004</v></c></row>`+
		`<row r="5"><c r="A5" t="s"><v>99</v></c></row>`)

	tables, err := ReadXLSX(data, 10)
	if err != nil {
		t.Fatalf("ReadXLSX: %v", err)
	}
	want := []Table{{Name: "Activities", Rows: []Row{
		{Number: 1, Cells: []string{"Launch film", "", "Trade fair"}},
		{Number: 2, Cells: []string{"2024-06-19", "3.5", "", "TRUE"}},
		{Number: 4, Cells: []string{"", "=A1", "padded", "", "1200", "0.3"}},
	}}}
	if diff := cmp.Diff(want, tables); diff != "" {
		t.Errorf("tables (-want +got):\n%s", diff)
	}
}

func TestReadXLSX_RoundTripWithWriter_012FR027(t *testing.T) {
	t.Parallel()
	amount, _ := money.ParseAmount("12500.50")
	data := written(t, Workbook{Sheets: []Sheet{
		{Name: "Campaigns", Rows: [][]Cell{{Bold("ID"), Bold("Name")}, {Text("C-1001"), Text("=not a formula")}}},
		{Name: "Activities", Rows: [][]Cell{{Text("A-1001"), Money(amount), Date(date.New(2026, 3, 12)), Number(7)}}},
	}})

	tables, err := ReadXLSX(data, 10)
	if err != nil {
		t.Fatalf("ReadXLSX: %v", err)
	}
	want := []Table{
		{Name: "Campaigns", Rows: []Row{{Number: 1, Cells: []string{"ID", "Name"}}, {Number: 2, Cells: []string{"C-1001", "=not a formula"}}}},
		{Name: "Activities", Rows: []Row{{Number: 1, Cells: []string{"A-1001", "12500.50", "2026-03-12", "7"}}}},
	}
	if diff := cmp.Diff(want, tables); diff != "" {
		t.Errorf("tables (-want +got):\n%s", diff)
	}
}

func TestReadXLSX_Refusals_012FR028(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
		want error
	}{
		{name: "not a zip archive", data: []byte("Name,Owner\n"), want: ErrUnreadable},
		{name: "a zip archive without a workbook", data: zipOf(t, map[string]string{"word/document.xml": "<w/>"}), want: ErrUnreadable},
		{name: "a sheet that is not XML", data: handMade(t, `<row><c><v>1</v></row>`), want: ErrUnreadable},
		{name: "more rows than allowed", data: handMade(t, `<row><c t="b"><v>1</v></c></row><row><c t="b"><v>0</v></c></row>`+
			`<row><c t="b"><v>1</v></c></row>`), want: ErrTooManyRows},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ReadXLSX(tt.data, 2); !errors.Is(err, tt.want) {
				t.Errorf("ReadXLSX error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestReadCSV_BOMDelimiterQuotes_012FR028(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data string
		want []Row
	}{
		{name: "commas with a byte-order mark", data: byteOrderMark + "Name,Owner\n\"Trade, fair\",mei.lin@shell.com\n",
			want: []Row{{Number: 1, Cells: []string{"Name", "Owner"}}, {Number: 2, Cells: []string{"Trade, fair", "mei.lin@shell.com"}}}},
		{name: "semicolons, an empty row skipped", data: "Name;Cost\n;\n Launch ;12,50\n",
			want: []Row{{Number: 1, Cells: []string{"Name", "Cost"}}, {Number: 3, Cells: []string{"Launch", "12,50"}}}},
		{name: "tabs", data: "Name\tCost\n", want: []Row{{Number: 1, Cells: []string{"Name", "Cost"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			table, err := ReadCSV([]byte(tt.data), 5)
			if err != nil {
				t.Fatalf("ReadCSV: %v", err)
			}
			if diff := cmp.Diff(tt.want, table.Rows); diff != "" {
				t.Errorf("rows (-want +got):\n%s", diff)
			}
		})
	}
	if _, err := ReadCSV([]byte("a\nb\nc\n"), 2); !errors.Is(err, ErrTooManyRows) {
		t.Errorf("three rows with a limit of two: %v, want ErrTooManyRows", err)
	}
	if _, err := ReadCSV([]byte{0xff, 0xfe, 'a'}, 2); !errors.Is(err, ErrUnreadable) {
		t.Errorf("UTF-16 text: %v, want ErrUnreadable", err)
	}
}

func checkRead(t *testing.T, tables []Table) {
	t.Helper()
	for _, table := range tables {
		previous := 0
		for _, row := range table.Rows {
			if row.Number <= previous || len(row.Cells) > maxColumns || !hasText(row.Cells) {
				t.Fatalf("row %+v after row %d breaks the reader's promises", row, previous)
			}
			previous = row.Number
		}
	}
}

func FuzzReadXLSX(f *testing.F) {
	f.Add(written(f, Workbook{Sheets: []Sheet{{Name: "Activities", Rows: [][]Cell{{Text("Name"), Number(3)}}}}}))
	f.Add(handMade(f, `<row r="3"><c r="B3" t="s"><v>1</v></c></row>`))
	f.Add([]byte("PK\x03\x04"))
	f.Fuzz(func(t *testing.T, data []byte) {
		tables, err := ReadXLSX(data, 50)
		if err != nil {
			if !errors.Is(err, ErrUnreadable) && !errors.Is(err, ErrTooManyRows) {
				t.Fatalf("ReadXLSX error %v is neither ErrUnreadable nor ErrTooManyRows", err)
			}
			return
		}
		checkRead(t, tables)
	})
}

func FuzzReadCSV(f *testing.F) {
	f.Add([]byte(byteOrderMark + "Name;Owner\n\"a\"\"b\";c\n"))
	f.Add([]byte("\"unterminated\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		table, err := ReadCSV(data, 50)
		if err != nil {
			return
		}
		checkRead(t, []Table{table})
	})
}
