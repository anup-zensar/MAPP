package allwork

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"io"
	"reflect"
	"strings"
	"testing"

	"mapp/internal/platform/actor"
)

// exportOf exports the view of q over the year of b on 18 Oct 2026 as Priya, in US dollars unless cur says
// otherwise.
func exportOf(t *testing.T, b *yearBuilder, q Query, format string, unassigned bool, cur ...Currency) File {
	t.Helper()
	c := usd
	if len(cur) > 0 {
		c = cur[0]
	}
	data := b.data()
	built := viewOf(data, q, c)
	in := input{query: built.query, year: 2026, today: fixtureToday, now: fixtureNow, currency: c, data: data,
		choices: fixtureChoices(priya.ID), catalogue: catalogue(fixtureAttributes)}
	file, err := exportFile(in, built, actor.Actor{ID: priya.ID, Name: priya.Name}, format, unassigned)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	return file
}

// csvLines reads a CSV export: its notes, then its table (heading, rows, totals).
func csvLines(t *testing.T, f File) (notes, table [][]string) {
	t.Helper()
	body := string(f.Body)
	if !strings.HasPrefix(body, "\xEF\xBB\xBF") {
		t.Fatalf("no byte-order mark: %q", body[:min(len(body), 20)])
	}
	before, after, found := strings.Cut(strings.TrimPrefix(body, "\xEF\xBB\xBF"), "\n\n") // the empty line after the notes
	if !found {
		t.Fatalf("no empty line after the notes: %q", body)
	}
	read := func(text string) [][]string {
		r := csv.NewReader(strings.NewReader(text))
		r.FieldsPerRecord = -1
		records, err := r.ReadAll()
		if err != nil {
			t.Fatalf("not CSV: %v", err)
		}
		return records
	}
	return read(before), read(after)
}

// rowNamed returns the table row whose Name (third value) is name.
func rowNamed(t *testing.T, table [][]string, name string) []string {
	t.Helper()
	for _, row := range table {
		if len(row) > 2 && row[2] == name {
			return row
		}
	}
	t.Fatalf("no row %q", name)
	return nil
}

// The file holds every matching and context row — collapsed branches too — with Level, Path and Name, the
// view's columns in order and the totals row; context rows are marked (FR-032, US7 scenario 1).
func TestExport_GridHasEveryRowLevelPathColumnsTotals_FR032(t *testing.T) {
	b := seedYear()
	f := exportOf(t, b, Query{}, FormatCSV, true)

	_, table := csvLines(t, f)

	heading := []string{"Level", "Path", "Name", "ID", "Status", "Owner", "Start date", "End date", "Plan budget (USD)",
		"Planned cost (USD)", "Line items (USD)", "Unassigned planned cost (USD)", "Committed (USD)", "Invoiced (USD)"}
	if !reflect.DeepEqual(table[0], heading) {
		t.Fatalf("heading = %q", table[0])
	}
	if rows := len(view(b, Query{}).Rows); len(table) != rows+2 {
		t.Errorf("%d lines, want the heading, %d rows and the totals", len(table), rows)
	}
	want := map[string][]string{
		"Digital & social": {"Initiative", "FY2026 Lubricants Europe › Helix launch", "Digital & social", "I-1001", "In progress",
			"Priya Raman", "2026-01-01", "2026-06-30", "300000.00", "200000.00", "125000.00", "75000.00", "0.00", "0.00"},
		"Film production": {"Line item", "FY2026 Lubricants Europe › Helix launch › Digital & social › Launch film", "Film production",
			"L-1002", "All filled in", "Priya Raman", "", "", "", "50000.00", "", "", "0.00", "0.00"},
		"Partner co-marketing fund": {"Activity", "FY2026 Lubricants Europe › Rimula fleet › Events", "Partner co-marketing fund",
			"A-1005", "In progress", "Priya Raman", "2026-03-01", "Ongoing", "", "100000.00", "", "100000.00", "0.00", "0.00"},
		"FY2026 Lubricants Europe": {"Plan", "", "FY2026 Lubricants Europe", "P-1001", "Published", "Priya Raman", "2026-01-01",
			"2026-12-31", "1200000.00", "460000.00", "165000.00", "295000.00", "0.00", "0.00"},
	}
	for name, line := range want {
		if got := rowNamed(t, table, name); !reflect.DeepEqual(got, line) {
			t.Errorf("%s = %q\nwant %q", name, got, line)
		}
	}
	unassigned := table[slicesIndex(table, "Launch film")+3]
	if !reflect.DeepEqual(unassigned, []string{"Unassigned planned cost", "FY2026 Lubricants Europe › Helix launch › Digital & social › Launch film",
		"Unassigned planned cost (Sys_Gen_Balance)", "", "", "", "", "", "", "20000.00", "", "20000.00", "", ""}) {
		t.Errorf("unassigned row = %q", unassigned)
	}
	totals := table[len(table)-1]
	if !reflect.DeepEqual(totals, []string{"", "", "Total · 10 activities", "", "", "", "", "", "2000000.00", "650000.00", "205500.00",
		"445000.00", "0.00", "0.00"}) {
		t.Errorf("totals = %q", totals)
	}

	daniels := exportOf(t, b, Query{Owners: []string{daniel.ID}}, FormatCSV, true)
	_, table = csvLines(t, daniels)
	if table[0][3] != "Shown for context" || rowNamed(t, table, "FY2026 Lubricants Europe")[3] != "Yes" ||
		rowNamed(t, table, "Rimula fleet")[3] != "" || table[len(table)-1][2] != "Total · 1 activity" {
		t.Errorf("with context rows: %q", table)
	}
}

// slicesIndex returns the index of the table row named name.
func slicesIndex(table [][]string, name string) int {
	for i, row := range table {
		if len(row) > 2 && row[2] == name {
			return i
		}
	}
	return -1
}

// The Calendar exports each record's ID, status, dates, owner and planned cost; Budget & cost its Plan
// budget, its periods and its total (FR-032, US7 scenario 2).
func TestExport_CalendarAndBudgetColumns_FR032(t *testing.T) {
	b := seedYear()

	_, calendar := csvLines(t, exportOf(t, b, Query{View: ViewCalendar}, FormatCSV, true))
	_, budget := csvLines(t, exportOf(t, b, Query{View: ViewBudget, Period: PeriodQuarter}, FormatCSV, true))

	if !reflect.DeepEqual(calendar[0], []string{"Level", "Path", "Name", "ID", "Status", "Start date", "End date", "Owner", "Planned cost (USD)"}) {
		t.Errorf("calendar heading = %q", calendar[0])
	}
	if got := rowNamed(t, calendar, "Partner co-marketing fund")[3:]; !reflect.DeepEqual(got, []string{"A-1005", "In progress", "2026-03-01", "Ongoing", "Priya Raman", "100000.00"}) {
		t.Errorf("calendar row = %q", got)
	}
	if got := calendar[len(calendar)-1][2:]; !reflect.DeepEqual(got, []string{"Total · 10 activities", "", "", "", "", "", "650000.00"}) {
		t.Errorf("calendar totals = %q", got)
	}
	for _, row := range calendar {
		if row[0] == "Line item" || row[0] == "Unassigned planned cost" {
			t.Errorf("the calendar has no %s rows", row[0])
		}
	}
	if !reflect.DeepEqual(budget[0], []string{"Level", "Path", "Name", "ID", "Plan budget (USD)", "Q1 (USD)", "Q2 (USD)", "Q3 (USD)", "Q4 (USD)", "Total (USD)"}) {
		t.Errorf("budget heading = %q", budget[0])
	}
	if got := rowNamed(t, budget, "Rimula fleet")[3:]; !reflect.DeepEqual(got, []string{"C-1002", "450000.00", "100000.00", "", "60000.00", "25000.00", "185000.00"}) {
		t.Errorf("budget row = %q", got)
	}
}

// Money is exported as numbers in the display currency, with the currency in the heading; a local amount
// has a Currency column beside it; dates are dates and statuses words (FR-033).
func TestExport_MoneyNumbersCurrencyHeadingsDatesStatusWords_FR033(t *testing.T) {
	eur := Currency{Code: "EUR", Rate: mustRate("0.92")}
	q := Query{Columns: []string{"plannedCost", "startDate", "status", "lineItem.localPlannedCost", "budgetUsed"}}
	f := exportOf(t, seedYear(), q, FormatXLSX, true, eur)

	r, err := zip.NewReader(bytes.NewReader(f.Body), int64(len(f.Body)))
	if err != nil {
		t.Fatalf("not an .xlsx file: %v", err)
	}
	var grid string
	for _, part := range r.File {
		if part.Name == "xl/worksheets/sheet2.xml" {
			rc, _ := part.Open()
			body, _ := io.ReadAll(rc)
			_ = rc.Close()
			grid = string(body)
		}
	}
	for _, want := range []string{
		`<t xml:space="preserve">Planned cost (EUR)</t>`,
		`<t xml:space="preserve">Planned cost in local currency</t>`,
		`<t xml:space="preserve">Currency</t>`,
		`<t xml:space="preserve">Budget used</t>`,
		`s="2"><v>110400.00</v>`, // Launch film's 120,000.00 USD in EUR
		`s="3"><v>46082</v>`,     // 1 Mar 2026
		`<t xml:space="preserve">Created</t>`,
		`s="2"><v>46000.00</v></c><c r="H`, // Film production's 46,000.00 EUR, then its currency
		`<t xml:space="preserve">EUR</t>`,
	} {
		if !strings.Contains(grid, want) {
			t.Errorf("the grid sheet lacks %s", want)
		}
	}
	_, table := csvLines(t, exportOf(t, seedYear(), q, FormatCSV, true, eur))
	if got := rowNamed(t, table, "Film production")[3:]; !reflect.DeepEqual(got, []string{"46000.00", "", "All filled in", "46000.00", "EUR", ""}) {
		t.Errorf("Film production = %q", got)
	}
	if got := rowNamed(t, table, "FY2026 Lubricants Europe")[3:]; !reflect.DeepEqual(got, []string{"423200.00", "2026-01-01", "Published", "", "", "38"}) {
		t.Errorf("plan = %q", got)
	}
}

// Every export states, before the data, the view, year, filters, search, Show levels, display currency,
// exporter and time — on an "About this export" sheet in a spreadsheet — and its name gives the view, year
// and date (FR-034).
func TestExport_NotesBeforeData_AboutSheet_FileName_FR034(t *testing.T) {
	b := seedYear()
	q := Query{ActivityStatuses: []string{"declined"}, Owners: []string{priya.ID}, Strategic: StrategicExclude,
		Search: "fleet", ShowLevels: ShowCompleteBranches}

	notes, table := csvLines(t, exportOf(t, b, q, FormatCSV, true))

	want := [][]string{
		{"View", "Grid"}, {"Year", "2026"}, {"Filters", "Activity status: Declined"}, {"", "Owner: Priya Raman"},
		{"", "Strategic / AOP plans: Exclude"}, {"Financial figures", "No GSAP data loaded yet"}, {"Search", "fleet"},
		{"Show levels", "Complete branches only"},
		{"Display currency", "USD"}, {"Exported by", "Priya Raman"}, {"Exported at", "18 Oct 2026, 14:05 UTC"},
	}
	if !reflect.DeepEqual(notes, want) {
		t.Errorf("notes = %q\nwant    %q", notes, want)
	}
	if rowNamed(t, table, "Fleet roadshow") == nil {
		t.Errorf("the data follows the notes")
	}
	plain, _ := csvLines(t, exportOf(t, b, Query{}, FormatCSV, true))
	if plain[2][1] != "None" || plain[4][1] != "None" {
		t.Errorf("no filters and no search read None: %q", plain)
	}

	xlsx := exportOf(t, b, Query{View: ViewBudget}, FormatXLSX, true)
	r, err := zip.NewReader(bytes.NewReader(xlsx.Body), int64(len(xlsx.Body)))
	if err != nil {
		t.Fatalf("not an .xlsx file: %v", err)
	}
	for _, part := range r.File {
		if part.Name == "xl/workbook.xml" {
			rc, _ := part.Open()
			body, _ := io.ReadAll(rc)
			_ = rc.Close()
			if !strings.Contains(string(body), `<sheet name="About this export" sheetId="1" r:id="rId1"/><sheet name="Budget &amp; cost" sheetId="2"`) {
				t.Errorf("workbook = %s", body)
			}
		}
	}
	if xlsx.Name != "MAPP All work – Budget & cost – 2026 – 18 Oct 2026.xlsx" || xlsx.ContentType != contentTypeXLSX {
		t.Errorf("file = %q, %s", xlsx.Name, xlsx.ContentType)
	}
	if got := ContentDisposition("MAPP All work – Grid – 2026 – 18 Oct 2026.csv"); got !=
		`attachment; filename="MAPP All work - Grid - 2026 - 18 Oct 2026.csv"; `+
			`filename*=UTF-8''MAPP%20All%20work%20%E2%80%93%20Grid%20%E2%80%93%202026%20%E2%80%93%2018%20Oct%202026.csv` {
		t.Errorf("Content-Disposition = %s", got)
	}
}

// Unassigned planned cost rows are exported as the screen shows them: with the switch on, not with it off
// (FR-012).
func TestExport_UnassignedFollowsSwitch_FR012(t *testing.T) {
	count := func(unassigned bool) int {
		_, table := csvLines(t, exportOf(t, seedYear(), Query{}, FormatCSV, unassigned))
		n := 0
		for _, row := range table {
			if row[0] == "Unassigned planned cost" {
				n++
			}
		}
		return n
	}

	if on, off := count(true), count(false); on != 8 || off != 0 {
		t.Errorf("unassigned rows: %d with the switch on, %d off; want 8 and 0", on, off)
	}
}
