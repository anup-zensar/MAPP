package financials_test

import (
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"mapp/internal/financials"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
	"mapp/internal/platform/web"
)

// The activity Social always-on (A-1002) answers its figures, its line items as the breakdown, its latest
// transactions and the data's source; a line item and a plan answer their own shapes.
func TestRecord_Contract_FR015(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	activity := get[financials.RecordView](t, alex, "/api/v1/financials/records/A-1002")
	f := activity.Figures
	if f.Committed != dollars(t, "25000.00") || f.PlannedCost == nil || *f.PlannedCost != dollars(t, "80000.00") ||
		f.NotYetCommitted == nil || *f.NotYetCommitted != dollars(t, "55000.00") || f.CommittedPercent == nil ||
		*f.CommittedPercent != 31 || f.TransactionCount != 7 {
		t.Errorf("Social always-on = %+v", f)
	}
	if activity.Record.Name != "Social always-on" || len(activity.Breakdown) != 2 || len(activity.Latest) != 5 ||
		activity.Source.FeedName == nil || activity.Source.DataAsOf == nil {
		t.Errorf("record %+v, breakdown %+v, latest %d, source %+v", activity.Record, activity.Breakdown, len(activity.Latest),
			activity.Source)
	}
	standBuild := get[financials.RecordView](t, alex, "/api/v1/financials/records/L-1006")
	if standBuild.BreakdownLevel != nil || len(standBuild.SwbsCodes) != 1 || standBuild.SwbsCodes[0] != "SIM-GB01-0001" {
		t.Errorf("Stand build breakdown %v, SWBS codes %v", standBuild.BreakdownLevel, standBuild.SwbsCodes)
	}
	plan := get[financials.RecordView](t, alex, "/api/v1/financials/records/P-1001")
	if p := plan.Figures; p.FirstLineItemAbovePlanned == nil || p.FirstLineItemAbovePlanned.ID != "L-1006" || len(plan.Breakdown) != 2 {
		t.Errorf("plan = %+v, breakdown %+v", p, plan.Breakdown)
	}
}

// A record that does not exist answers 404 with the level's sentence; amounts follow the reader's display
// currency, each figure converted once (FR-011, D-061).
func TestRecord_NotFoundAndDisplayCurrency_FR011(t *testing.T) {
	t.Parallel()
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

// The Financials page on the seed: the 2026 tab counts and paging by default, and the type, sort, record and
// page-size parameters.
func TestTransactions_Contract_FR021(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	all := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions")
	one := 1
	want := financials.TypeCounts{All: 11, PurchaseOrder: 4, Invoice: 3, Payment: 2, Accrual: 2, NotLinked: &one}
	if !reflect.DeepEqual(all.Counts, want) || all.Total != 11 || len(all.Items) != 11 || all.Year != 2026 || all.Page != 1 || all.PageSize != 50 {
		t.Errorf("counts %+v, total %d, %d items, year %d, page %d of %d", all.Counts, all.Total, len(all.Items), all.Year,
			all.Page, all.PageSize)
	}
	pos := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions?type=purchaseOrder&sort=amount:desc")
	if pos.Total != 4 || pos.Items[0].ID != "FT-1001" || !reflect.DeepEqual(pos.Counts, want) || pos.Totals.Invoiced != dollars(t, "33152.17") {
		t.Errorf("purchase orders: total %d, first %s, counts %+v", pos.Total, pos.Items[0].ID, pos.Counts)
	}
	speakerFees := get[financials.TransactionPage](t, alex, "/api/v1/financials/transactions?recordId=L-1004&pageSize=100")
	if speakerFees.Total != 4 || speakerFees.RecordFilter == nil || speakerFees.RecordFilter.Name != "Speaker fees" ||
		*speakerFees.Counts.NotLinked != 0 || speakerFees.PageSize != 100 {
		t.Errorf("Speaker fees: total %d, record %+v, counts %+v", speakerFees.Total, speakerFees.RecordFilter, speakerFees.Counts)
	}
}

// Every wrong parameter is refused with its name as the field (FR-021, as D-219).
func TestTransactions_ParameterRefusals_FR021(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

// Link a not-linked transaction by hand and undo it, with every refusal (FR-026, D-253).
func TestLinkAndUnlink_Contract_FR026(t *testing.T) {
	t.Parallel()
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

func TestOverview_Contract_FR025(t *testing.T) {
	t.Parallel()
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

func TestImportsAndTemplate_Contract_FR008(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	page := get[financials.ImportPage](t, alex, "/api/v1/financials/imports")
	want := financials.ImportCounts{Rows: 11, New: 11, NotLinked: 1}
	if page.Total != 1 || page.Items[0].ID != "FIM-1001" || page.Items[0].Source != financials.SourceDemo ||
		page.Items[0].Counts != want || page.Items[0].ImportedBy.Name != "SP&A One" {
		t.Errorf("imports = %+v", page)
	}

	response := alex.Do(t, http.MethodGet, "/api/v1/financials/imports/template", nil)
	if response.Status != http.StatusOK || response.Header.Get("Content-Type") != "text/csv" ||
		response.Header.Get("Content-Disposition") != `attachment; filename="spa-one-gsap-extract-sample.csv"` {
		t.Fatalf("template = %d %v", response.Status, response.Header)
	}
	lines := strings.Split(strings.TrimSpace(string(response.Body)), "\r\n")
	if len(lines) != 4 || lines[0] != extractHeader || !strings.Contains(lines[1], "HX1001") {
		t.Errorf("template = %q", response.Body)
	}
}

func TestImport_Contract_FR004(t *testing.T) {
	t.Parallel()
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

	fieldRefused(t, alex.Do(t, http.MethodPost, "/api/v1/financials/imports/check", csvFile("gsap.txt", extractHeader)), "fileName",
		"Choose a .csv file — GSAP extracts are imported as CSV.")
	fieldRefused(t, alex.Do(t, http.MethodPost, "/api/v1/financials/imports", csvFile("gsap.csv", "type,amount")), "content",
		"The first row must name the columns. Missing: document_number, posting_date, currency.")
}

func TestOperations_TransactionsAreReadOnly_FR002(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	financials.NewHandler(nil).RegisterRoutes(engine.Group("/api/v1"))
	operations := web.Operations(engine.Routes())

	changing := []string{}
	for _, operation := range operations {
		if !strings.HasPrefix(operation, http.MethodGet+" ") {
			changing = append(changing, operation)
		}
	}
	slices.Sort(changing)
	want := []string{
		"POST /api/v1/financials/imports",
		"POST /api/v1/financials/imports/check",
		"POST /api/v1/financials/transactions/{transactionId}/link",
		"POST /api/v1/financials/transactions/{transactionId}/unlink",
	}
	if len(operations) != 10 || !slices.Equal(changing, want) {
		t.Errorf("operations = %q\nchanging = %q, want %q", operations, changing, want)
	}
}
