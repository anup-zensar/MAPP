package financials

import (
	"context"
	"encoding/base64"
	"net/http"
	"strconv"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

// User story 4 — load GSAP data (FR-004 – FR-008, SC-003).

// importFile imports req as a, stopping the test on an error.
func (f *fixture) importFile(t *testing.T, a actor.Actor, req ImportRequest) ImportResult {
	t.Helper()
	result, err := f.service.Import(t.Context(), a, req)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	return result
}

// threeRows is the file of US4: a new PO for HX1007, an invoice for it, and a row with a currency not in
// Settings.
var threeRows = file("gsap-2026-10-19.csv", header,
	"PO,4500001010,10,,,2026-10-01,,3000.00,USD,V-1,Signs Ltd,Signage,,HX1007,",
	"INVOICE,5100002010,,4500001010,,2026-10-05,,1000.00,USD,V-1,Signs Ltd,Signage,,,",
	"PO,4500001011,10,,,2026-10-01,,500.00,XYZ,V-1,Signs Ltd,Signage,,HX1007,",
)

// A check says what the import does and saves nothing: no transaction, import, ID or log entry (FR-004,
// SC-003, D-177).
func TestCheck_SavesNothing_FR004(t *testing.T) { //nolint:gocyclo // 14: one check, then the store and the log, then the import it foretold
	t.Parallel()
	f := newFixture(t)
	check, err := f.service.Check(t.Context(), alex, threeRows)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if check.Counts != (ImportCounts{Rows: 3, New: 2, Refused: 1}) || check.ProblemsTotal != 1 || check.NotLinkedTotal != 0 ||
		check.FileName != "gsap-2026-10-19.csv" {
		t.Errorf("check = %+v", check)
	}
	if n := len(f.transactions(t)); n != 11 {
		t.Errorf("%d transactions after the check, want 11", n)
	}
	if entries := f.entries(t); len(entries) != 0 {
		t.Errorf("log entries after the check: %+v", entries)
	}
	result := f.importFile(t, alex, threeRows)
	if result.Import.ID != "FIM-1002" || result.Import.Counts != check.Counts {
		t.Errorf("import %s %+v, want FIM-1002 with the check's counts", result.Import.ID, result.Import.Counts)
	}
	if got := f.stored(t, "FT-1012"); got.DocumentNumber != "4500001010" || got.LineItemID != "L-1007" {
		t.Errorf("FT-1012 = %+v", got)
	}
	if source := f.record(t, alex, "A-1008").Source; source.FileName == nil || *source.FileName != "gsap-2026-10-19.csv" {
		t.Errorf("Grand opening Shanghai's source = %+v, want the imported file", source)
	}
}

// Rows = new + updated + unchanged + refused, and the import says the same as its check; importing the same
// file twice changes nothing the second time (FR-006, SC-003).
func TestImport_CountsAddUpAndTwiceChangesNothing_FR006(t *testing.T) { //nolint:gocyclo // 14: the counts and stored rows of one import, then of the same file again
	t.Parallel()
	f := newFixture(t)
	req := file("gsap.csv", header,
		"PO,4500001001,10,,,2026-01-12,,15800.00,GBP,V-100101,Northlight Events GmbH,Venue hire Berlin,DE03-M26-0003,HX1003,",
		"PO,4500001002,10,,,2026-05-04,,5200.00,USD,V-100102,Keynote Speakers Ltd,Keynote speakers,,HX1004,",
		"PAYMENT,1500003010,,4500001002,,2026-10-01,,1000.00,USD,,,,,,",
		"PAYMENT,1500003011,,,,2026-10-01,,1.00,USD,,,,,,",
		"PAYMENT,1500003012,,,,not a date,,1.00,USD,,,,,,",
	)
	check, err := f.service.Check(t.Context(), alex, req)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	first := f.importFile(t, alex, req)
	c := first.Import.Counts
	if c != check.Counts || c != (ImportCounts{Rows: 5, New: 2, Updated: 1, Unchanged: 1, Refused: 1, NotLinked: 1}) ||
		c.Rows != c.New+c.Updated+c.Unchanged+c.Refused {
		t.Errorf("counts = %+v (check %+v)", c, check.Counts)
	}
	if got := f.stored(t, "FT-1002"); got.Amount != amountOf(t, "5200.00") || got.ImportID != "FIM-1002" || got.Version != 2 {
		t.Errorf("updated FT-1002 = %+v", got)
	}
	if got := f.stored(t, "FT-1001"); got.Version != 1 || got.ImportID != "FIM-1001" {
		t.Errorf("unchanged FT-1001 = %+v", got)
	}
	if first.NotLinked[0] != (NotLinkedRow{Row: 5, Name: "Payment 1500003011", Reason: reasonNoReference}) {
		t.Errorf("not linked = %+v", first.NotLinked)
	}

	before := f.transactions(t)
	second := f.importFile(t, alex, req)
	if second.Import.Counts != (ImportCounts{Rows: 5, Unchanged: 4, Refused: 1, NotLinked: 1}) {
		t.Errorf("second import = %+v", second.Import.Counts)
	}
	after := f.transactions(t)
	for i := range before {
		if before[i].Version != after[i].Version {
			t.Errorf("%s changed: version %d → %d", before[i].ID, before[i].Version, after[i].Version)
		}
	}
}

// Refused rows are skipped and listed with their row; a key repeated in the file refuses the later row; the
// rest are imported; lists stop at 200, the totals say how many there are (FR-005).
func TestImport_RefusedRowsAreSkipped_FR005(t *testing.T) { //nolint:gocyclo // 11: the problems and kept row of one import, then the capped list of a check
	t.Parallel()
	f := newFixture(t)
	result := f.importFile(t, alex, file("gsap.csv", header,
		"PO,4500009001,10,,,2026-10-01,,100.00,USD,,,,,HX1007,",
		"PO,4500009002,10,,,2026-10-01,,100.00,CHF,,,,,HX1007,",
		"PO,4500009001,10,,,2026-10-02,,200.00,USD,,,,,HX1007,",
	))
	wantProblems := []ImportProblem{
		{Row: 3, Column: "currency", Message: "Currency 'CHF' isn't set up in Settings — an administrator adds it under Currencies."},
		{Row: 4, Column: "document_number", Message: "Row 4 repeats PO 4500009001 / 10 from row 2 — keep one."},
	}
	if result.Import.Counts != (ImportCounts{Rows: 3, New: 1, Refused: 2}) || len(result.Problems) != 2 ||
		result.Problems[0] != wantProblems[0] || result.Problems[1] != wantProblems[1] {
		t.Errorf("result = %+v", result)
	}
	if got := f.stored(t, "FT-1012"); got.Amount != amountOf(t, "100.00") {
		t.Errorf("the first row was not kept: %+v", got)
	}

	lines := []string{header}
	for i := range 250 {
		lines = append(lines, "PO,46000"+strconv.Itoa(10000+i)+",10,,,2026-10-01,,1.00,XYZ,,,,,,")
	}
	check, err := f.service.Check(t.Context(), alex, file("many.csv", lines...))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if check.ProblemsTotal != 250 || len(check.Problems) != 200 || check.Counts.Refused != 250 {
		t.Errorf("problems %d of %d, refused %d", len(check.Problems), check.ProblemsTotal, check.Counts.Refused)
	}
}

// Purchase orders are applied first, so an invoice finds a PO of the same file and the PO gets the lower
// ID (FR-007, data model §4.1).
func TestImport_POsFirstAndInDeliveryReferences_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.importFile(t, alex, file("gsap.csv", header,
		"INVOICE,5100009001,,4500009001,,2026-10-02,,100.00,USD,,,,,,",
		"PO,4500009001,10,,,2026-10-01,,400.00,USD,,,,,HX1007,",
	))
	po, invoice := f.stored(t, "FT-1012"), f.stored(t, "FT-1013")
	if po.Type != TypePurchaseOrder || po.LineItemID != "L-1007" || po.MatchedBy != MatchedByTrackingKey {
		t.Errorf("FT-1012 = %+v", po)
	}
	if invoice.Type != TypeInvoice || invoice.LineItemID != "L-1007" || invoice.MatchedBy != MatchedByPONumber {
		t.Errorf("FT-1013 = %+v", invoice)
	}
}

// Every import matches the stored not-linked transactions again and counts those it links; the log entry
// says so (FR-007, FR-027, US4 scenario 8).
func TestImport_RetryRelinksAndLogs_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	first := f.importFile(t, alex, file("invoice.csv", header, "INVOICE,5100009002,,4500009002,,2026-10-02,,100.00,USD,,,,,,"))
	if first.NotLinked[0].Reason != "PO 4500009002 isn't linked to a line item yet." {
		t.Errorf("not linked = %+v", first.NotLinked)
	}
	f.now = f.now.Add(1)
	second := f.importFile(t, alex, file("po.csv", header, "PO,4500009002,10,,,2026-10-01,,400.00,USD,,,,,HX1007,"))
	if second.Import.Counts != (ImportCounts{Rows: 1, New: 1, Relinked: 1}) {
		t.Errorf("second import = %+v", second.Import.Counts)
	}
	if got := f.stored(t, "FT-1012"); got.LineItemID != "L-1007" || got.MatchedBy != MatchedByPONumber || got.NotLinkedReason != "" {
		t.Errorf("the invoice was not relinked: %+v", got)
	}

	entries := f.entries(t)
	if len(entries) != 2 {
		t.Fatalf("%d log entries, want one per import", len(entries))
	}
	checkRelinkEntry(t, entries[1])
	if entries[0].Details != "1 row: 1 new, 0 updated, 0 unchanged, 0 refused · 1 not linked" {
		t.Errorf("first entry = %q", entries[0].Details)
	}
}

// checkRelinkEntry checks the log entry of the import that linked an earlier transaction.
func checkRelinkEntry(t *testing.T, e history.Entry) {
	t.Helper()
	if e.RecordType != "financialImport" || e.Verb != "imported" || e.RecordID != "FIM-1003" || e.RecordName != "po.csv" ||
		e.Actor.Name != "Alex Morgan" || e.RecordTypeLabel != "GSAP extract" ||
		e.Details != "1 row: 1 new, 0 updated, 0 unchanged, 0 refused · 0 not linked · 1 earlier transaction is now linked" {
		t.Errorf("entry = %+v", e)
	}
}

// An import never changes a manual link, even when it updates the transaction (FR-007, FR-026).
func TestImport_KeepsManualLinks_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if _, err := f.service.Link(t.Context(), alex, "FT-1004", LinkRequest{LineItemID: "L-1007", Version: version(1)}); err != nil {
		t.Fatalf("Link: %v", err)
	}
	result := f.importFile(t, alex, file("gsap.csv", header,
		"PO,4500001004,10,,,2026-09-15,,12500.00,USD,V-100104,Brightside Media BV,Retail display production,,HX1099,"))
	if result.Import.Counts.Updated != 1 || result.Import.Counts.NotLinked != 0 {
		t.Errorf("counts = %+v", result.Import.Counts)
	}
	got := f.stored(t, "FT-1004")
	if got.LineItemID != "L-1007" || got.MatchedBy != MatchedByManual || got.ManualLink == nil || got.Amount != amountOf(t, "12500.00") {
		t.Errorf("FT-1004 = %+v", got)
	}
}

// A whole-file problem saves nothing; unknown columns are named in the check (FR-004).
func TestImport_WholeFileRefusalSavesNothing_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	_, err := f.service.Import(t.Context(), alex, ImportRequest{FileName: "gsap.csv", Content: "not base64!"})
	if e := assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed); e.Fields[0] != (errs.FieldError{Field: "content",
		Message: "The file couldn't be read — choose it again."}) {
		t.Errorf("fields = %+v", e.Fields)
	}
	_, err = f.service.Import(t.Context(), alex, file("gsap.csv", "type,amount", "PO,1"))
	_ = assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed)
	page, err := f.service.Imports(t.Context(), alex)
	if err != nil || page.Total != 1 || len(f.entries(t)) != 0 {
		t.Errorf("imports %+v, %v; log %d", page, err, len(f.entries(t)))
	}
	check, err := f.service.Check(t.Context(), alex, file("gsap.csv", header+",cost_centre", "PO,1,,,,2026-01-01,,1,USD,,,,,,,CC-1"))
	if err != nil || len(check.UnusedColumns) != 1 || check.UnusedColumns[0] != "cost_centre" {
		t.Errorf("unused = %v, %v", check.UnusedColumns, err)
	}
}

// The import history is newest first; GSAP data as of is the newest import (FR-008).
func TestImports_NewestFirstAndDataAsOf_FR008(t *testing.T) { //nolint:gocyclo // 12: the history and data as of before and after an import, then without data
	t.Parallel()
	f := newFixture(t)
	asOf, err := f.service.DataAsOf(t.Context())
	if err != nil || asOf == nil || !asOf.Equal(start.Add(-24*time.Hour)) {
		t.Errorf("data as of = %v, %v", asOf, err)
	}
	f.importFile(t, alex, threeRows)
	page, err := f.service.Imports(t.Context(), alex)
	if err != nil || page.Total != 2 || page.Items[0].ID != "FIM-1002" || page.Items[1].Source != SourceDemo {
		t.Errorf("imports = %+v, %v", page, err)
	}
	if asOf, _ = f.service.DataAsOf(t.Context()); !asOf.Equal(start) {
		t.Errorf("data as of = %v, want the import's time", asOf)
	}
	empty := newEmptyFixture(t)
	if asOf, _ := empty.service.DataAsOf(t.Context()); asOf != nil {
		t.Errorf("without data: %v", asOf)
	}
	if v := empty.record(t, alex, "A-1002"); v.Source.DataAsOf != nil || v.Figures.Committed != usd(t, "0.00") {
		t.Errorf("empty card = %+v", v)
	}
}

// fakeFeed is a second source of GSAP data, as a real GSAP feed later (SC-006).
type fakeFeed struct{ rows []Incoming }

func (fakeFeed) Name() string { return "GSAP PGSI 941" }

func (fakeFeed) Source() Source { return SourceFile }

func (f fakeFeed) Read(context.Context, Delivery) ([]Incoming, []RowProblem, []string, error) {
	return f.rows, nil, nil, nil
}

// Another feed replaces the simulated extract with no other change: the same checks, matching and figures
// (FR-003, SC-006).
func TestImport_AnotherFeed_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.service.UseFeed(fakeFeed{rows: []Incoming{{Row: 2, Fields: Fields{
		Type: TypePurchaseOrder, DocumentNumber: "4500007001", PONumber: "4500007001", PostingDate: date.MustParse("2026-10-01"),
		Amount: amountOf(t, "2000.00"), Currency: "USD", TrackingKey: "HX1007",
	}}}})
	result := f.importFile(t, alex, ImportRequest{FileName: "pgsi941.dat", Content: base64.StdEncoding.EncodeToString([]byte("x"))})
	if result.Import.FeedName != "GSAP PGSI 941" || result.Import.Counts.New != 1 {
		t.Errorf("import = %+v", result.Import)
	}
	if got := f.record(t, alex, "L-1007").Figures.Committed; got != usd(t, "2000.00") {
		t.Errorf("Station signage committed = %v", got)
	}
}

// The sample file has the header and three rows that import cleanly (FR-009).
func TestTemplate_ImportsCleanly_FR009(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	body, err := f.service.Template(t.Context(), alex)
	if err != nil {
		t.Fatalf("Template: %v", err)
	}
	check, err := f.service.Check(t.Context(), alex, ImportRequest{FileName: TemplateName, Content: base64.StdEncoding.EncodeToString(body)})
	if err != nil || check.Counts != (ImportCounts{Rows: 3, New: 3}) || check.ProblemsTotal != 0 {
		t.Errorf("check of the sample = %+v, %v", check, err)
	}
}
