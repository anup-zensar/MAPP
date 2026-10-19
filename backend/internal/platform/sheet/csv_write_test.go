package sheet

import (
	"bytes"
	"testing"
	"time"

	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
)

// A CSV file starts with the UTF-8 byte-order mark and the notes as label,value lines, then an empty line
// and the table; numbers have no thousands separators, dates are YYYY-MM-DD, and a text a spreadsheet would
// take for a formula starts with an apostrophe (D-120, FR-033, FR-034).
func TestCSV_BOMNotesThenTable(t *testing.T) {
	amount, _ := money.ParseAmount("12500")
	var buf bytes.Buffer

	err := WriteCSV(&buf, [][2]string{{"View", "Grid"}, {"Filters", "Owner: Daniel Okoro"}, {"", "Activity status: Declined"}},
		[][]Cell{
			{Bold("Name"), Bold("Planned cost (USD)"), Bold("Start date"), Bold("Created"), Bold("INQ")},
			{Text("Launch film, Europe"), Money(amount), Date(date.MustParse("2026-03-01")),
				DateTime(time.Date(2026, 3, 12, 9, 5, 0, 0, time.UTC)), Number(250000)},
			{Text("=HYPERLINK(\"x\")"), Money(amount.Neg()), {}, {}, Text("@sum")},
		})

	if err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	want := "\xEF\xBB\xBFView,Grid\nFilters,Owner: Daniel Okoro\n,Activity status: Declined\n\n" +
		"Name,Planned cost (USD),Start date,Created,INQ\n" +
		"\"Launch film, Europe\",12500.00,2026-03-01,2026-03-12 09:05,250000\n" +
		"\"'=HYPERLINK(\"\"x\"\")\",-12500.00,,,'@sum\n"
	if got := buf.String(); got != want {
		t.Errorf("csv =\n%q\nwant\n%q", got, want)
	}
}
