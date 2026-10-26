package financials_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/financials"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
	"mapp/internal/platform/web"
)

// The contract tests of feature 015: every operation on the seed's demo data (data model §6), checked
// against backend/api/openapi.yaml by apitest. The test server's clock is 12 Mar 2026 09:00 UTC, so the
// current year is 2026.

// The demo people's e-mail addresses (demo-users-and-data.md).
const (
	alexsEmail  = "alex.morgan@shell.com"
	priyasEmail = "priya.raman@shell.com"
)

// get sends GET path as client, checks that it answers 200 and decodes the answer into a T.
func get[T any](t *testing.T, client *apitest.Client, path string) T {
	t.Helper()
	response := client.Do(t, http.MethodGet, path, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s = %d %s, want 200", path, response.Status, response.Body)
	}
	var v T
	response.Decode(t, &v)
	return v
}

// apiError is the inner object of an error answer.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Fields  []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"fields"`
}

// refused checks that the answer has status and code and returns its body.
func refused(t *testing.T, response *apitest.Response, status int, code string) apiError {
	t.Helper()
	if response.Status != status {
		t.Fatalf("status %d %s, want %d %s", response.Status, response.Body, status, code)
	}
	var body struct {
		Error apiError `json:"error"`
	}
	response.Decode(t, &body)
	if body.Error.Code != code {
		t.Fatalf("code %q (%s), want %q", body.Error.Code, body.Error.Message, code)
	}
	return body.Error
}

// fieldRefused checks a 400 validation_failed on field with message (any message when "").
func fieldRefused(t *testing.T, response *apitest.Response, field, message string) {
	t.Helper()
	body := refused(t, response, http.StatusBadRequest, "validation_failed")
	if len(body.Fields) != 1 || body.Fields[0].Field != field || (message != "" && body.Fields[0].Message != message) {
		t.Errorf("fields = %+v, want %s: %q", body.Fields, field, message)
	}
}

// dollars is text as US dollars.
func dollars(t *testing.T, text string) money.Money {
	t.Helper()
	amount, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return money.USD(amount)
}

// csvFile is a CSV file as the body of an import: lines joined with line breaks, as base64.
func csvFile(name string, lines ...string) map[string]string {
	content := strings.Join(lines, "\n") + "\n"
	return map[string]string{"fileName": name, "content": base64.StdEncoding.EncodeToString([]byte(content))}
}

// extractHeader is the header row of the simulated GSAP extract.
const extractHeader = "type,document_number,line,po_number,po_line,posting_date,accrual_month,amount,currency," +
	"vendor_code,vendor_name,description,swbs_code,tracking_key,line_item_id"

// The activity Social always-on (A-1002) shows its figures, its two line items as the breakdown and its
// latest five transactions; Stand build (L-1006) is committed above its planned cost and shows its SWBS
// code; the plan breaks down by campaign (FR-011, FR-015 – FR-018, data model §6.1).
func TestRecord_Contract_FR015(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	activity := get[financials.RecordView](t, alex, "/api/v1/financials/records/A-1002")
	f := activity.Figures
	if f.Committed != dollars(t, "25000.00") || f.Invoiced != dollars(t, "22500.00") || f.Paid != dollars(t, "20000.00") ||
		f.Accrued != dollars(t, "2500.00") || f.PlannedCost == nil || *f.PlannedCost != dollars(t, "80000.00") ||
		f.NotYetCommitted == nil || *f.NotYetCommitted != dollars(t, "55000.00") || f.CommittedPercent == nil ||
		*f.CommittedPercent != 31 || f.CommittedAbovePlanned || f.TransactionCount != 7 {
		t.Errorf("Social always-on = %+v", f)
	}
	if activity.Record.Name != "Social always-on" || activity.PlanID != "P-1001" || len(activity.Path) != 3 ||
		activity.BreakdownLevel == nil || *activity.BreakdownLevel != financials.LevelLineItem {
		t.Errorf("record %+v, plan %s, path %+v, breakdown level %v", activity.Record, activity.PlanID, activity.Path,
			activity.BreakdownLevel)
	}
	if len(activity.Breakdown) != 2 || activity.Breakdown[0].Record.ID != "L-1003" || activity.Breakdown[1].Record.ID != "L-1004" ||
		activity.Breakdown[1].Figures.Accrued != dollars(t, "2500.00") {
		t.Errorf("breakdown = %+v, want Venue hire then Speaker fees", activity.Breakdown)
	}
	if len(activity.Latest) != 5 || activity.Latest[0].ID != "FT-1011" || activity.Latest[0].Link.State != financials.StateLinked {
		t.Errorf("latest = %+v, want 5, the September accrual first", activity.Latest)
	}
	if activity.Source.FeedName == nil || *activity.Source.FeedName != "Demo data (simulated GSAP)" ||
		activity.Source.AccrualMonth == nil || *activity.Source.AccrualMonth != "2026-09" || activity.Source.DataAsOf == nil {
		t.Errorf("source = %+v", activity.Source)
	}

	standBuild := get[financials.RecordView](t, alex, "/api/v1/financials/records/L-1006")
	if s := standBuild.Figures; !s.CommittedAbovePlanned || s.CommittedPercent == nil || *s.CommittedPercent != 107 ||
		*s.NotYetCommitted != dollars(t, "-652.17") || s.LineItemsAbovePlanned != 0 {
		t.Errorf("Stand build = %+v", s)
	}
	if standBuild.BreakdownLevel != nil || len(standBuild.Breakdown) != 0 || len(standBuild.SwbsCodes) != 1 ||
		standBuild.SwbsCodes[0] != "GB01-M26-0006" {
		t.Errorf("Stand build breakdown %v %+v, SWBS codes %v", standBuild.BreakdownLevel, standBuild.Breakdown, standBuild.SwbsCodes)
	}

	plan := get[financials.RecordView](t, alex, "/api/v1/financials/records/P-1001")
	if p := plan.Figures; p.Committed != dollars(t, "35652.17") || *p.PlannedCost != dollars(t, "460000.00") ||
		*p.CommittedPercent != 8 || p.LineItemsAbovePlanned != 1 || p.FirstLineItemAbovePlanned == nil ||
		p.FirstLineItemAbovePlanned.ID != "L-1006" {
		t.Errorf("plan = %+v", p)
	}
	if *plan.BreakdownLevel != financials.LevelCampaign || len(plan.Breakdown) != 2 || plan.Breakdown[0].Record.Name != "Helix launch" ||
		plan.Breakdown[0].Figures.Committed != dollars(t, "35652.17") || len(plan.Path) != 0 {
		t.Errorf("plan breakdown = %+v", plan.Breakdown)
	}
}

// A record that does not exist answers 404 with the level's sentence; amounts follow the reader's display
// currency, each figure converted once (FR-011, D-061).
func TestRecord_NotFoundAndDisplayCurrency_FR011(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	body := refused(t, priya.Do(t, http.MethodGet, "/api/v1/financials/records/A-9999", nil), http.StatusNotFound, "not_found")
	if body.Message != "This activity no longer exists — someone may have deleted it." {
		t.Errorf("message = %q", body.Message)
	}
	if response := priya.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"displayCurrency": "EUR"}); response.Status != http.StatusOK {
		t.Fatalf("preferences = %d %s", response.Status, response.Body)
	}
	got := get[financials.RecordView](t, priya, "/api/v1/financials/records/L-1003")
	if eur := dollars(t, "18400.00").Amount; got.Figures.Committed != (money.Money{Amount: eur, Currency: "EUR"}) {
		t.Errorf("Venue hire committed = %+v, want 18,400.00 EUR", got.Figures.Committed)
	}
}

// The Financials page on the seed: the 2026 tab counts and totals, the type, link, record and search
// filters, and paging (FR-021, spec US3 scenario 1).
func TestTransactions_Contract_FR021(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	all := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions")
	want := financials.TypeCounts{All: 11, PurchaseOrder: 4, Invoice: 3, Payment: 2, Accrual: 2, NotLinked: 1}
	if all.Counts != want || all.Total != 11 || len(all.Items) != 11 || all.Year != 2026 || all.Page != 1 || all.PageSize != 50 {
		t.Errorf("counts %+v, total %d, %d items, year %d, page %d of %d", all.Counts, all.Total, len(all.Items), all.Year,
			all.Page, all.PageSize)
	}
	if all.Totals.Committed != dollars(t, "47652.17") || all.Totals.Invoiced != dollars(t, "33152.17") ||
		all.Totals.Paid != dollars(t, "30652.17") || all.Totals.Accrued != dollars(t, "2500.00") {
		t.Errorf("totals = %+v", all.Totals)
	}
	if all.Items[0].ID != "FT-1011" || all.Choices.CurrentYear != 2026 || len(all.Choices.Plans) == 0 {
		t.Errorf("first %s, choices %+v", all.Items[0].ID, all.Choices)
	}

	pos := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions?type=purchaseOrder&sort=amount:desc")
	if pos.Total != 4 || pos.Items[0].ID != "FT-1001" || pos.Counts != want || pos.Totals.Invoiced != dollars(t, "33152.17") {
		t.Errorf("purchase orders: total %d, first %s, counts %+v", pos.Total, pos.Items[0].ID, pos.Counts)
	}
	notLinked := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions?link=notLinked")
	if notLinked.Total != 1 || notLinked.Items[0].ID != "FT-1004" || notLinked.Items[0].Link.Reason == nil ||
		*notLinked.Items[0].Link.Reason != "No line item for HX key HX1099 — check the tracking key in GSAP." {
		t.Errorf("not linked = %+v", notLinked.Items)
	}
	speakerFees := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions?recordId=L-1004&pageSize=100")
	if speakerFees.Total != 4 || speakerFees.RecordFilter == nil || speakerFees.RecordFilter.Name != "Speaker fees" ||
		speakerFees.Counts.NotLinked != 0 || speakerFees.PageSize != 100 {
		t.Errorf("Speaker fees: total %d, record %+v, counts %+v", speakerFees.Total, speakerFees.RecordFilter, speakerFees.Counts)
	}
	search := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions?q=4500001003")
	if search.Total != 2 || search.Items[0].ID != "FT-1007" || search.Items[1].ID != "FT-1003" {
		t.Errorf("search 4500001003 = %d items", search.Total)
	}
	byActivity := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions?q=social%20ALWAYS")
	if byActivity.Total != 7 {
		t.Errorf("search by activity name = %d, want 7", byActivity.Total)
	}
	secondPage := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions?page=2")
	if len(secondPage.Items) != 0 || secondPage.Total != 11 || secondPage.Page != 2 {
		t.Errorf("page 2 = %d items of %d", len(secondPage.Items), secondPage.Total)
	}
	lastYear := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions?year=2025")
	if lastYear.Total != 0 || lastYear.Counts.All != 0 || lastYear.Totals.Committed != dollars(t, "0.00") {
		t.Errorf("2025 = %d, counts %+v", lastYear.Total, lastYear.Counts)
	}
}

// Every wrong parameter is refused with its name as the field (FR-021, as D-219).
func TestTransactions_ParameterRefusals_FR021(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	cases := map[string]string{
		"type=refund": "type", "year=1999": "year", "planId=X-1": "planId", "recordId=Q-1": "recordId",
		"link=maybe": "link", "sort=amount": "sort", "page=0": "page", "pageSize=7": "pageSize",
		"q=" + strings.Repeat("a", 201): "q",
	}
	for query, field := range cases {
		t.Run(field, func(t *testing.T) {
			fieldRefused(t, alex.Do(t, http.MethodGet, "/api/v1/financials/transactions?"+query, nil, apitest.SkipRequestValidation()),
				field, "")
		})
	}
	fieldRefused(t, alex.Do(t, http.MethodGet, "/api/v1/financials/transactions?recordId=A-9999", nil), "recordId",
		"This activity no longer exists — someone may have deleted it.")
	fieldRefused(t, alex.Do(t, http.MethodGet, "/api/v1/financials/transactions?planId=P-9999", nil), "planId", "Choose a plan of the list.")
}

// One transaction with every field, how it was linked and the import it came from; an unknown one is 404
// (FR-022).
func TestTransaction_Contract_FR022(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	got := get[financials.TransactionView](t, alex, "/api/v1/financials/transactions/FT-1003")
	if got.Name != "PO 4500001003 / 10" || got.TrackingKey != "HX1006" || got.Amount.Currency != "EUR" ||
		got.AmountUSD == nil || *got.AmountUSD != dollars(t, "10652.17") || got.Link.State != financials.StateLinked ||
		*got.Link.MatchedBy != financials.MatchedByTrackingKey || len(got.Link.Path) != 5 || got.Link.Path[4].ID != "L-1006" {
		t.Errorf("FT-1003 = %+v", got)
	}
	if got.Import.ID != "FIM-1001" || got.Import.FeedName != "Demo data (simulated GSAP)" || got.CanLink || got.CanUnlink ||
		got.Version != 1 {
		t.Errorf("import %+v, can link %v, can unlink %v, version %d", got.Import, got.CanLink, got.CanUnlink, got.Version)
	}
	notLinked := get[financials.TransactionView](t, alex, "/api/v1/financials/transactions/FT-1004")
	if !notLinked.CanLink || notLinked.Link.State != financials.StateNotLinked || len(notLinked.Link.Path) != 0 {
		t.Errorf("FT-1004 = %+v", notLinked)
	}
	body := refused(t, alex.Do(t, http.MethodGet, "/api/v1/financials/transactions/FT-9999", nil), http.StatusNotFound, "not_found")
	if body.Message != "This GSAP transaction doesn't exist." {
		t.Errorf("message = %q", body.Message)
	}
}

// Link a not-linked transaction by hand and undo it, with every refusal: a rule-based link, a manual link
// linked again, an unknown line item, a missing version, an out-of-date version, an unlink of a rule-based
// link (FR-026, D-253).
func TestLinkAndUnlink_Contract_FR026(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	link := func(id string, body any, opts ...apitest.RequestOption) *apitest.Response {
		return alex.Do(t, http.MethodPost, "/api/v1/financials/transactions/"+id+"/link", body, opts...)
	}
	unlink := func(id string, body any, opts ...apitest.RequestOption) *apitest.Response {
		return alex.Do(t, http.MethodPost, "/api/v1/financials/transactions/"+id+"/unlink", body, opts...)
	}

	byRule := refused(t, link("FT-1003", map[string]any{"lineItemId": "L-1007", "version": 1}), http.StatusConflict, "not_allowed")
	if byRule.Message != "Linked by GSAP data (HX key HX1006) — correct it in GSAP." {
		t.Errorf("message = %q", byRule.Message)
	}
	fieldRefused(t, link("FT-1004", map[string]any{"lineItemId": "L-9999", "version": 1}), "lineItemId", "No line item L-9999.")
	fieldRefused(t, link("FT-1004", map[string]any{"lineItemId": "L-1007"}, apitest.SkipRequestValidation()), "version", "")
	refused(t, link("FT-9999", map[string]any{"lineItemId": "L-1007", "version": 1}), http.StatusNotFound, "not_found")
	refused(t, link("FT-1004", map[string]any{"lineItemId": "L-1007", "version": 9}), http.StatusConflict, "version_conflict")

	response := link("FT-1004", map[string]any{"lineItemId": "L-1007", "version": 1})
	if response.Status != http.StatusOK {
		t.Fatalf("link = %d %s", response.Status, response.Body)
	}
	var linked financials.TransactionView
	response.Decode(t, &linked)
	if linked.Link.State != financials.StateLinked || *linked.Link.MatchedBy != financials.MatchedByManual ||
		linked.Link.ManualBy == nil || linked.Link.ManualBy.Name != "Alex Morgan" || !linked.CanUnlink || linked.Version != 2 {
		t.Errorf("linked = %+v", linked)
	}
	signage := get[financials.RecordView](t, alex, "/api/v1/financials/records/L-1007")
	if signage.Figures.Committed != dollars(t, "12000.00") || !signage.Figures.CommittedAbovePlanned {
		t.Errorf("Station signage = %+v", signage.Figures)
	}
	again := refused(t, link("FT-1004", map[string]any{"lineItemId": "L-1007", "version": 2}), http.StatusConflict, "not_allowed")
	if !strings.HasPrefix(again.Message, "Linked by Alex Morgan on ") {
		t.Errorf("message = %q", again.Message)
	}

	onlyManual := refused(t, unlink("FT-1003", map[string]any{"version": 1}), http.StatusConflict, "not_allowed")
	if onlyManual.Message != "Only links made by a person can be undone here." {
		t.Errorf("message = %q", onlyManual.Message)
	}
	refused(t, unlink("FT-1004", map[string]any{"version": 1}), http.StatusConflict, "version_conflict")
	fieldRefused(t, unlink("FT-1004", map[string]any{}, apitest.SkipRequestValidation()), "version", "")
	response = unlink("FT-1004", map[string]any{"version": 2})
	if response.Status != http.StatusOK {
		t.Fatalf("unlink = %d %s", response.Status, response.Body)
	}
	var unlinked financials.TransactionView
	response.Decode(t, &unlinked)
	if unlinked.Link.State != financials.StateNotLinked || unlinked.Link.Reason == nil || !unlinked.CanLink || unlinked.Version != 3 {
		t.Errorf("unlinked = %+v", unlinked)
	}
}

// Home's card for the current year: Priya's team plan and everything (FR-025, data model §6.1); a scope
// that is not listed is refused.
func TestOverview_Contract_FR025(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	mine := get[financials.Overview](t, priya, "/api/v1/financials/overview?scope=myTeams")
	f := mine.Figures
	if mine.Year != 2026 || mine.Scope != financials.ScopeMyTeams || f.Committed != dollars(t, "35652.17") ||
		*f.PlannedCost != dollars(t, "460000.00") || *f.CommittedPercent != 8 || mine.ActivitiesAbovePlanned != 1 ||
		mine.NotLinked != 0 { // not-linked transactions show only to people who may import (014: Priya may not)
		t.Errorf("my teams = %+v", mine)
	}
	everything := get[financials.Overview](t, priya, "/api/v1/financials/overview?scope=everything")
	if everything.Figures.Committed != dollars(t, "35652.17") || everything.Figures.PlannedCost.Amount.Cmp(f.PlannedCost.Amount) < 0 { // Priya sees her team's plans only (014)
		t.Errorf("everything = %+v", everything.Figures)
	}
	fieldRefused(t, priya.Do(t, http.MethodGet, "/api/v1/financials/overview?scope=all", nil, apitest.SkipRequestValidation()), "scope",
		"Choose My teams or Everything.")
}

// The import history lists the demo load; the sample file is a CSV attachment with the header and three
// rows (FR-008, FR-009).
func TestImportsAndTemplate_Contract_FR008(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	page := get[financials.ImportPage](t, alex, "/api/v1/financials/imports")
	want := financials.ImportCounts{Rows: 11, New: 11, NotLinked: 1}
	if page.Total != 1 || page.Items[0].ID != "FIM-1001" || page.Items[0].Source != financials.SourceDemo ||
		page.Items[0].Counts != want || page.Items[0].ImportedBy.Name != "MAPP" {
		t.Errorf("imports = %+v", page)
	}

	response := alex.Do(t, http.MethodGet, "/api/v1/financials/imports/template", nil)
	if response.Status != http.StatusOK || response.Header.Get("Content-Type") != "text/csv" ||
		response.Header.Get("Content-Disposition") != `attachment; filename="mapp-gsap-extract-sample.csv"` {
		t.Fatalf("template = %d %v", response.Status, response.Header)
	}
	lines := strings.Split(strings.TrimSpace(string(response.Body)), "\r\n")
	if len(lines) != 4 || lines[0] != extractHeader || !strings.Contains(lines[1], "HX1001") {
		t.Errorf("template = %q", response.Body)
	}
}

// Check a three-row file, which saves nothing; import it; import it again; whole-file refusals (FR-004 –
// FR-007, SC-003, spec US4).
func TestImport_Contract_FR004(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	body := csvFile("gsap-2026-10-19.csv", extractHeader,
		"PO,4500001010,10,,,2026-10-01,,3000.00,USD,V-1,Signs Ltd,Signage,,HX1007,",
		"INVOICE,5100002010,,4500001010,,2026-10-05,,1000.00,USD,V-1,Signs Ltd,Signage,,,",
		"PO,4500001011,10,,,2026-10-01,,500.00,XYZ,V-1,Signs Ltd,Signage,,HX1007,",
	)

	response := alex.Do(t, http.MethodPost, "/api/v1/financials/imports/check", body)
	if response.Status != http.StatusOK {
		t.Fatalf("check = %d %s", response.Status, response.Body)
	}
	var check financials.ImportCheck
	response.Decode(t, &check)
	if check.Counts != (financials.ImportCounts{Rows: 3, New: 2, Refused: 1}) || check.ProblemsTotal != 1 ||
		check.Problems[0] != (financials.ImportProblem{Row: 4, Column: "currency",
			Message: "Currency 'XYZ' isn't set up in Settings — an administrator adds it under Currencies."}) {
		t.Errorf("check = %+v", check)
	}
	if got := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions"); got.Counts.All != 11 {
		t.Errorf("after the check: %d transactions, want 11", got.Counts.All)
	}

	response = alex.Do(t, http.MethodPost, "/api/v1/financials/imports", body)
	if response.Status != http.StatusCreated {
		t.Fatalf("import = %d %s", response.Status, response.Body)
	}
	var result financials.ImportResult
	response.Decode(t, &result)
	if result.Import.Counts != check.Counts || result.Import.FileName != "gsap-2026-10-19.csv" || result.ProblemsTotal != 1 ||
		result.Import.Source != financials.SourceFile || result.Import.FeedName != "Simulated GSAP extract" {
		t.Errorf("import = %+v", result)
	}
	opening := get[financials.RecordView](t, alex, "/api/v1/financials/records/A-1008")
	if opening.Figures.Committed != dollars(t, "3000.00") || opening.Figures.Invoiced != dollars(t, "1000.00") {
		t.Errorf("Grand opening Shanghai = %+v", opening.Figures)
	}
	if source := opening.Source; source.FileName == nil || *source.FileName != "gsap-2026-10-19.csv" {
		t.Errorf("source = %+v", source)
	}

	response = alex.Do(t, http.MethodPost, "/api/v1/financials/imports", body)
	response.Decode(t, &result)
	if result.Import.Counts != (financials.ImportCounts{Rows: 3, Unchanged: 2, Refused: 1}) {
		t.Errorf("second import = %+v", result.Import.Counts)
	}

	fieldRefused(t, alex.Do(t, http.MethodPost, "/api/v1/financials/imports/check", csvFile("gsap.txt", extractHeader)), "fileName",
		"Choose a .csv file — GSAP extracts are imported as CSV.")
	fieldRefused(t, alex.Do(t, http.MethodPost, "/api/v1/financials/imports",
		map[string]string{"fileName": "gsap.csv", "content": "%%%"}), "content", "The file couldn't be read — choose it again.")
	fieldRefused(t, alex.Do(t, http.MethodPost, "/api/v1/financials/imports", csvFile("gsap.csv", "type,amount")), "content",
		"The first row must name the columns. Missing: document_number, posting_date, currency.")
	if got := get[financials.ImportPage](t, alex, "/api/v1/financials/imports"); got.Total != 3 || got.Items[0].Counts.Unchanged != 2 {
		t.Errorf("imports = %+v", got)
	}
}

// Every operation needs a signed-in person.
func TestOperations_NeedSignIn_FR028(t *testing.T) {
	server := apitest.New(t)
	for _, path := range []string{
		"/api/v1/financials/records/A-1002", "/api/v1/financials/transactions", "/api/v1/financials/transactions/FT-1001",
		"/api/v1/financials/overview?scope=myTeams", "/api/v1/financials/imports", "/api/v1/financials/imports/template",
	} {
		if response := server.Do(t, http.MethodGet, path, nil); response.Status != http.StatusUnauthorized {
			t.Errorf("GET %s = %d, want 401", path, response.Status)
		}
	}
	payload, _ := json.Marshal(map[string]any{"version": 1})
	if response := server.Do(t, http.MethodPost, "/api/v1/financials/transactions/FT-1004/unlink", nil, apitest.RawBody(payload)); response.Status != http.StatusUnauthorized {
		t.Errorf("unlink = %d, want 401", response.Status)
	}
}

// Transactions are read-only: of the ten operations, only an import, its check and a manual link or unlink
// change anything; none creates, edits or deletes a transaction (FR-002).
func TestOperations_TransactionsAreReadOnly_FR002(t *testing.T) {
	router := web.NewRouter()
	financials.NewHandler(nil).Routes(router, func(h http.HandlerFunc) http.HandlerFunc { return h })

	changing := []string{}
	for _, pattern := range router.Patterns() {
		if !strings.HasPrefix(pattern, http.MethodGet+" ") {
			changing = append(changing, pattern)
		}
	}
	want := []string{
		"POST /api/v1/financials/transactions/{transactionId}/link",
		"POST /api/v1/financials/transactions/{transactionId}/unlink",
		"POST /api/v1/financials/imports",
		"POST /api/v1/financials/imports/check",
	}
	if len(router.Patterns()) != 10 || !slices.Equal(changing, want) {
		t.Errorf("operations = %q\nchanging = %q, want %q", router.Patterns(), changing, want)
	}
}
