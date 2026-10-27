package history_test

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/apitest"
)

const (
	priyasEmail  = "priya.raman@shell.com"
	danielsEmail = "daniel.okoro@shell.com"
)

// Every test here starts the API with apitest.WithoutDemoHistory: the activity log and the recycle bin start
// empty, so the entries and IDs are the test's own. The seed's demo history has its own tests
// (seed/history_test.go).

// The bodies of the API's answers, as the web app receives them.
const (
	emptyLogBody       = `{"items":[],"total":0,"unfilteredTotal":0,"record":null}`
	emptyBinBody       = `{"items":[],"total":0}`
	signInRequiredBody = `{"error":{"code":"sign_in_required","message":"Please sign in to continue.","fields":[]}}`
	unknownEntryBody   = `{"error":{"code":"not_found","message":"This item is not in the recycle bin.","fields":[]}}`
	invalidLimitBody   = `{"error":{"code":"validation_failed","message":"Some fields need your attention.",` +
		`"fields":[{"field":"limit","message":"Choose a number from 1 to 200."}]}}`
)

// An entry carries who did what to which record and when, where the record was and where it is now, and
// each changed field with readable values, exactly as the contract describes it (FR-022, 011 FR-008).
func TestListActivityLog_EntriesAsTheContractSays_FR022(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas") // SF-1001
	renameFolder(t, priya, "SF-1001", "Ideas 2")
	addNote(t, priya, "SF-1001", "First idea") // SN-1001

	response := priya.Do(t, http.MethodGet, "/api/v1/activity-log?limit=2", nil)

	assertAnswer(t, response, http.StatusOK, `{"items":[`+
		`{"id":"LOG-1003","seq":1003,"at":"2026-03-12T09:00:00Z","actor":{"id":"U-1002","name":"Priya Raman"},`+
		`"action":"created","verb":"created","recordType":"sampleNote","recordTypeLabel":"sample note",`+
		`"recordId":"SN-1001","recordName":"First idea","path":[{"type":"sampleFolder","id":"SF-1001","name":"Ideas 2"}],`+
		`"changes":[],"comment":"","details":"","items":[],`+
		`"recordNow":{"state":"exists","name":"First idea","path":[{"type":"sampleFolder","id":"SF-1001","name":"Ideas 2"}],"binEntryId":""}},`+
		`{"id":"LOG-1002","seq":1002,"at":"2026-03-12T09:00:00Z","actor":{"id":"U-1002","name":"Priya Raman"},`+
		`"action":"changed","verb":"changed","recordType":"sampleFolder","recordTypeLabel":"sample folder",`+
		`"recordId":"SF-1001","recordName":"Ideas 2","path":[],`+
		`"changes":[{"field":"name","label":"Name","old":"Ideas","new":"Ideas 2","oldText":"Ideas","newText":"Ideas 2"}],`+
		`"comment":"","details":"","items":[],"recordNow":{"state":"exists","name":"Ideas 2","path":[],"binEntryId":""}}],`+
		`"total":3,"unfilteredTotal":3,"record":null}`)
}

// Without a limit, the log answers the newest 50 entries (FR-022, contract default).
func TestListActivityLog_DefaultLimitIs50_FR022(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	for i := range 51 {
		createFolder(t, priya, fmt.Sprintf("Folder %d", i+1)) // LOG-1001 to LOG-1051
	}

	page := getLog(t, priya, "")

	if len(page.Items) != 50 || page.Total != 51 || page.Items[0].ID != "LOG-1051" || page.Items[49].ID != "LOG-1002" {
		t.Errorf("GET /api/v1/activity-log = %d entries from %s (total %d), want the 50 from LOG-1051 to LOG-1002 (total 51)",
			len(page.Items), firstID(page.Items), page.Total)
	}
}

// A limit that is not a whole number from 1 to 200 is refused next to the field (FR-020); 1 and 200 are
// accepted.
func TestListActivityLog_InvalidLimit_FR020(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)

	for _, limit := range []string{"0", "201", "-5", "ten", "1.5", ""} {
		response := priya.Do(t, http.MethodGet, "/api/v1/activity-log?limit="+limit, nil,
			apitest.SkipRequestValidation()) // the contract itself allows only 1 to 200

		assertAnswer(t, response, http.StatusBadRequest, invalidLimitBody)
	}
	for _, limit := range []string{"1", "200"} {
		response := priya.Do(t, http.MethodGet, "/api/v1/activity-log?limit="+limit, nil)

		assertAnswer(t, response, http.StatusOK, emptyLogBody)
	}
}

// Every filter of the Activity log page is a query parameter, applied to the whole log on the server:
// person, record type, actions, plan, record with or without the items inside it, date range, search, and
// Load more with beforeSeq; the answer counts the matching entries and the whole log, and names the record
// of recordId (011 FR-003, FR-009, FR-013).
func TestListActivityLog_Filters_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	createFolder(t, priya, "Ideas")             // SF-1001, LOG-1001
	addNote(t, daniel, "SF-1001", "First idea") // SN-1001, LOG-1002
	createFolder(t, daniel, "Plans")            // SF-1002, LOG-1003
	server.Advance(time.Hour)
	renameFolder(t, priya, "SF-1002", "Plans 2") // LOG-1004
	deleteNote(t, priya, "SN-1001")              // LOG-1005
	from := url.QueryEscape("2026-03-12T09:30:00Z")
	tests := []struct {
		query string
		want  []string
		total int
	}{
		{"?actorId=U-1003", []string{"LOG-1003", "LOG-1002"}, 2},
		{"?recordType=sampleNote", []string{"LOG-1005", "LOG-1002"}, 2},
		{"?action=changed&action=deleted", []string{"LOG-1005", "LOG-1004"}, 2},
		{"?planId=SF-1001", []string{"LOG-1005", "LOG-1002", "LOG-1001"}, 3},
		{"?recordId=SF-1001", []string{"LOG-1001"}, 1},
		{"?recordId=SF-1001&includeInside=true", []string{"LOG-1005", "LOG-1002", "LOG-1001"}, 3},
		{"?recordId=SF-1001&includeInside=false", []string{"LOG-1001"}, 1},
		{"?from=" + from, []string{"LOG-1005", "LOG-1004"}, 2},
		{"?to=" + from, []string{"LOG-1003", "LOG-1002", "LOG-1001"}, 3},
		{"?q=PLANS", []string{"LOG-1004", "LOG-1003"}, 2},
		{"?q=okoro&limit=1", []string{"LOG-1003"}, 2},
		{"?q=okoro&beforeSeq=1003", []string{"LOG-1002"}, 2},
		{"?actorId=U-1002&recordType=sampleFolder&action=created", []string{"LOG-1001"}, 1},
	}

	for _, tt := range tests {
		page := getLog(t, priya, tt.query)

		if got := entryIDs(page.Items); !slices.Equal(got, tt.want) || page.Total != tt.total || page.UnfilteredTotal != 5 {
			t.Errorf("GET /api/v1/activity-log%s = %v (%d of %d), want %v (%d of 5)",
				tt.query, got, page.Total, page.UnfilteredTotal, tt.want, tt.total)
		}
	}

	page := getLog(t, priya, "?recordId=SF-1002")
	if want := (history.RecordRef{Type: "sampleFolder", ID: "SF-1002", Name: "Plans 2"}); page.Record == nil || *page.Record != want {
		t.Errorf("record = %+v, want %+v", page.Record, want)
	}
	note := getLog(t, priya, "?recordId=SN-1001").Items[0].RecordNow
	if note.State != history.RecordInRecycleBin || note.BinEntryID != "BIN-1001" || note.Name != "First idea" {
		t.Errorf("recordNow of the deleted note = %+v, want in the recycle bin BIN-1001", note)
	}
}

// Every problem with the parameters is answered at once, next to its parameter, in the order of the
// parameters (011 FR-003).
func TestListActivityLog_InvalidParams_400(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	tests := []struct {
		query string
		body  string
	}{
		{
			"?action=renamed&includeInside=yes&from=yesterday&to=2026-03-12&beforeSeq=0&limit=ten",
			`{"field":"action","message":"Choose an action from the list."},` +
				`{"field":"includeInside","message":"Reload the page and try again."},` +
				`{"field":"from","message":"Choose a valid date."},` +
				`{"field":"to","message":"Choose a valid date."},` +
				`{"field":"beforeSeq","message":"Reload the page and try again."},` +
				`{"field":"limit","message":"Choose a number from 1 to 200."}`,
		},
		{
			"?from=2026-03-12T00:00:00Z&to=2026-03-12T00:00:00Z&beforeSeq=x",
			`{"field":"to","message":"The end date must be on or after the start date."},` +
				`{"field":"beforeSeq","message":"Reload the page and try again."}`,
		},
		{"?action=emptiedRecycleBin", `{"field":"action","message":"Choose an action from the list."}`},
	}

	for _, tt := range tests {
		response := priya.Do(t, http.MethodGet, "/api/v1/activity-log"+tt.query, nil, apitest.SkipRequestValidation())

		assertAnswer(t, response, http.StatusBadRequest,
			`{"error":{"code":"validation_failed","message":"Some fields need your attention.","fields":[`+tt.body+`]}}`)
	}
}

// The summary counts the whole log for the boundaries the browser sends (todayStart, weekStart), and offers
// the choices of the Person, Record type and Plan filters (011 FR-002, FR-003).
func TestActivityLogSummary_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	daniel := server.SignIn(t, danielsEmail)
	createFolder(t, daniel, "Ideas")           // yesterday
	addNote(t, daniel, "SF-1001", "Old idea")  // yesterday
	server.Advance(24 * time.Hour)             // 13 Mar 2026 09:00, a Friday
	priya := server.SignIn(t, priyasEmail)     // a session of today: yesterday's would have expired
	addNote(t, priya, "SF-1001", "First idea") // today
	deleteNote(t, priya, "SN-1002")            // today

	response := priya.Do(t, http.MethodGet,
		"/api/v1/activity-log/summary?todayStart=2026-03-13T00:00:00Z&weekStart=2026-03-09T00:00:00Z", nil)

	assertAnswer(t, response, http.StatusOK, `{"changesToday":2,"changesThisWeek":4,"peopleActiveThisWeek":2,`+
		`"deletionsThisWeek":1,"total":4,`+
		`"people":[{"id":"U-1003","name":"Daniel Okoro"},{"id":"U-1002","name":"Priya Raman"}],`+
		`"recordTypes":[{"type":"sampleFolder","label":"sample folder","count":1},{"type":"sampleNote","label":"sample note","count":3}],`+
		`"plans":[]}`)
}

// The boundaries are required and must be moments, and the week cannot start after today (011 FR-002).
func TestActivityLogSummary_MissingParams_400(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	reload := `"message":"Reload the page and try again."`
	tests := []struct {
		query  string
		fields string
	}{
		{"", `{"field":"todayStart",` + reload + `},{"field":"weekStart",` + reload + `}`},
		{"?todayStart=today&weekStart=2026-03-09", `{"field":"todayStart","message":"Choose a valid date."},` +
			`{"field":"weekStart","message":"Choose a valid date."}`},
		{"?todayStart=2026-03-09T00:00:00Z&weekStart=2026-03-12T00:00:00Z", `{"field":"weekStart",` + reload + `}`},
	}

	for _, tt := range tests {
		response := priya.Do(t, http.MethodGet, "/api/v1/activity-log/summary"+tt.query, nil, apitest.SkipRequestValidation())

		assertAnswer(t, response, http.StatusBadRequest,
			`{"error":{"code":"validation_failed","message":"Some fields need your attention.","fields":[`+tt.fields+`]}}`)
	}
}

// Every history operation needs a signed-in person (FR-005).
func TestHistoryOperations_WithoutSession_FR005(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	calls := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/activity-log"},
		{http.MethodGet, "/api/v1/activity-log/summary"},
		{http.MethodGet, "/api/v1/recycle-bin"},
		{http.MethodGet, "/api/v1/recycle-bin/BIN-1001/restore-preview"},
		{http.MethodPost, "/api/v1/recycle-bin/BIN-1001/undo"},
		{http.MethodPost, "/api/v1/recycle-bin/BIN-1001/restore"},
		{http.MethodDelete, "/api/v1/recycle-bin/BIN-1001"},
		{http.MethodPost, "/api/v1/recycle-bin/empty"},
	}

	for _, call := range calls {
		response := server.Do(t, call.method, call.path, nil, apitest.SkipRequestValidation())

		assertAnswer(t, response, http.StatusUnauthorized, signInRequiredBody)
	}
}
