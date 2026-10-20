package history_test

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/platform/apitest"
)

const (
	priyasEmail  = "priya.raman@northstar.demo"
	danielsEmail = "daniel.okoro@northstar.demo"
)

// Every test here starts the API with apitest.WithoutDemoHistory: the activity log and the recycle bin start
// empty, so the entries and IDs are the test's own. The seed's demo history has its own tests
// (seed/history_test.go).

// The bodies of the API's answers, as the web app receives them.
const (
	emptyLogBody       = `{"items":[],"total":0,"unfilteredTotal":0,"record":null}`
	emptyBinBody       = `{"items":[],"total":0}`
	signInRequiredBody = `{"code":"sign_in_required","message":"Please sign in to continue.","fields":[]}`
	unknownEntryBody   = `{"code":"not_found","message":"This item is not in the recycle bin.","fields":[]}`
	invalidLimitBody   = `{"code":"validation_failed","message":"Some fields need your attention.",` +
		`"fields":[{"field":"limit","message":"Choose a number from 1 to 200."}]}`
)

// Before anything has changed, the activity log and the recycle bin answer empty lists, never null
// (FR-022, FR-024, research R10).
func TestEmptyLogAndBin_AnswerEmptyLists_FR022_FR024(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)

	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/activity-log", nil), http.StatusOK, emptyLogBody)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/recycle-bin", nil), http.StatusOK, emptyBinBody)
}

// An entry carries who did what to which record and when, where the record was and where it is now, and
// each changed field with readable values, exactly as the contract describes it (FR-022, 011 FR-008).
func TestListActivityLog_EntriesAsTheContractSays_FR022(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas") // SF-1001
	renameFolder(t, priya, "SF-1001", "Ideas 2")
	addNote(t, priya, "SF-1001", "First idea") // SN-1001

	response := priya.Do(t, http.MethodGet, "/api/activity-log?limit=2", nil)

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

// The log answers the newest matching entries first, filtered by record type and by record, at most limit
// of them, with the number of all matching entries (FR-022).
func TestListActivityLog_FiltersNewestFirst_FR022(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")              // SF-1001, LOG-1001
	createFolder(t, priya, "Plans")              // SF-1002, LOG-1002
	renameFolder(t, priya, "SF-1001", "Ideas 2") // LOG-1003
	tests := []struct {
		query string
		want  []string
		total int
	}{
		{"", []string{"LOG-1003", "LOG-1002", "LOG-1001"}, 3},
		{"?recordId=SF-1001", []string{"LOG-1003", "LOG-1001"}, 2},
		{"?recordType=sampleFolder&limit=1", []string{"LOG-1003"}, 3},
		{"?recordType=sampleNote", []string{}, 0},
	}

	for _, tt := range tests {
		page := getLog(t, priya, tt.query)

		if got := entryIDs(page.Items); !slices.Equal(got, tt.want) || page.Total != tt.total {
			t.Errorf("GET /api/activity-log%s = %v (total %d), want %v (total %d)", tt.query, got, page.Total, tt.want, tt.total)
		}
	}
}

// Without a limit, the log answers the newest 50 entries (FR-022, contract default).
func TestListActivityLog_DefaultLimitIs50_FR022(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	for i := range 51 {
		createFolder(t, priya, fmt.Sprintf("Folder %d", i+1)) // LOG-1001 to LOG-1051
	}

	page := getLog(t, priya, "")

	if len(page.Items) != 50 || page.Total != 51 || page.Items[0].ID != "LOG-1051" || page.Items[49].ID != "LOG-1002" {
		t.Errorf("GET /api/activity-log = %d entries from %s (total %d), want the 50 from LOG-1051 to LOG-1002 (total 51)",
			len(page.Items), firstID(page.Items), page.Total)
	}
}

// A limit that is not a whole number from 1 to 200 is refused next to the field (FR-020); 1 and 200 are
// accepted.
func TestListActivityLog_InvalidLimit_FR020(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)

	for _, limit := range []string{"0", "201", "-5", "ten", "1.5", ""} {
		response := priya.Do(t, http.MethodGet, "/api/activity-log?limit="+limit, nil,
			apitest.SkipRequestValidation()) // the contract itself allows only 1 to 200

		assertAnswer(t, response, http.StatusBadRequest, invalidLimitBody)
	}
	for _, limit := range []string{"1", "200"} {
		response := priya.Do(t, http.MethodGet, "/api/activity-log?limit="+limit, nil)

		assertAnswer(t, response, http.StatusOK, emptyLogBody)
	}
}

// Every filter of the Activity log page is a query parameter, applied to the whole log on the server:
// person, record type, actions, plan, record with or without the items inside it, date range, search, and
// Load more with beforeSeq; the answer counts the matching entries and the whole log, and names the record
// of recordId (011 FR-003, FR-009, FR-013).
func TestListActivityLog_Filters_Contract(t *testing.T) {
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
			t.Errorf("GET /api/activity-log%s = %v (%d of %d), want %v (%d of 5)",
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
		response := priya.Do(t, http.MethodGet, "/api/activity-log"+tt.query, nil, apitest.SkipRequestValidation())

		assertAnswer(t, response, http.StatusBadRequest,
			`{"code":"validation_failed","message":"Some fields need your attention.","fields":[`+tt.body+`]}`)
	}
}

// The summary counts the whole log for the boundaries the browser sends (todayStart, weekStart), and offers
// the choices of the Person, Record type and Plan filters (011 FR-002, FR-003).
func TestActivityLogSummary_Contract(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	daniel := server.SignIn(t, danielsEmail)
	createFolder(t, daniel, "Ideas")           // yesterday
	addNote(t, daniel, "SF-1001", "Old idea")  // yesterday
	server.Advance(24 * time.Hour)             // 13 Mar 2026 09:00, a Friday
	priya := server.SignIn(t, priyasEmail)     // a session of today: yesterday's would have expired
	addNote(t, priya, "SF-1001", "First idea") // today
	deleteNote(t, priya, "SN-1002")            // today

	response := priya.Do(t, http.MethodGet,
		"/api/activity-log/summary?todayStart=2026-03-13T00:00:00Z&weekStart=2026-03-09T00:00:00Z", nil)

	assertAnswer(t, response, http.StatusOK, `{"changesToday":2,"changesThisWeek":4,"peopleActiveThisWeek":2,`+
		`"deletionsThisWeek":1,"total":4,`+
		`"people":[{"id":"U-1003","name":"Daniel Okoro"},{"id":"U-1002","name":"Priya Raman"}],`+
		`"recordTypes":[{"type":"sampleFolder","label":"sample folder","count":1},{"type":"sampleNote","label":"sample note","count":3}],`+
		`"plans":[]}`)
}

// The boundaries are required and must be moments, and the week cannot start after today (011 FR-002).
func TestActivityLogSummary_MissingParams_400(t *testing.T) {
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
		response := priya.Do(t, http.MethodGet, "/api/activity-log/summary"+tt.query, nil, apitest.SkipRequestValidation())

		assertAnswer(t, response, http.StatusBadRequest,
			`{"code":"validation_failed","message":"Some fields need your attention.","fields":[`+tt.fields+`]}`)
	}
}

// Undo and Restore of an entry the recycle bin never had answer not found (FR-025).
func TestUndoAndRestore_UnknownEntry_FR025(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)

	for _, action := range []string{"undo", "restore"} {
		response := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-9999/"+action, nil)

		assertAnswer(t, response, http.StatusNotFound, unknownEntryBody)
	}
}

// The bin's two entries of binWithParent, as GET /api/recycle-bin answers them to Priya: the folder, which the
// note's entry waits for, and the note, whose folder is in the bin (011 FR-015 – FR-017, FR-028), each with what
// she may do with it — restore it, but not delete it permanently, which her Planner role does not include
// (014 FR-011).
const (
	plannerBinAccess = `"access":{"restore":{"available":true,"reason":""},"deletePermanently":{"available":false,` +
		`"reason":"Deleting permanently isn't part of your role (Planner). Ask an administrator if you need it."}}`
	folderBinItem = `{"id":"BIN-1002","recordType":"sampleFolder","recordTypeLabel":"sample folder","recordId":"SF-1001",` +
		`"recordName":"Ideas","path":[],"parent":null,"deletedAt":"2026-03-12T09:00:00Z",` +
		`"deletedBy":{"id":"U-1002","name":"Priya Raman"},"contents":[{"recordType":"sampleNote","label":"sample note","count":1}],` +
		`"recordTypePlural":"sample folders","restoreState":"restorable","missingParent":null,"parentEntryIds":[],` +
		`"insideEntryIds":["BIN-1001"],"records":[` +
		`{"recordType":"sampleFolder","recordTypeLabel":"sample folder","recordId":"SF-1001","recordName":"Ideas","parentId":""},` +
		`{"recordType":"sampleNote","recordTypeLabel":"sample note","recordId":"SN-1002","recordName":"Second idea","parentId":"SF-1001"}],` +
		plannerBinAccess + `}`
	noteBinItem = `{"id":"BIN-1001","recordType":"sampleNote","recordTypeLabel":"sample note","recordId":"SN-1001",` +
		`"recordName":"First idea","path":[{"type":"sampleFolder","id":"SF-1001","name":"Ideas"}],` +
		`"parent":{"type":"sampleFolder","id":"SF-1001","name":"Ideas"},"deletedAt":"2026-03-12T09:00:00Z",` +
		`"deletedBy":{"id":"U-1002","name":"Priya Raman"},"contents":[],"recordTypePlural":"sample notes",` +
		`"restoreState":"parentInRecycleBin",` +
		`"missingParent":{"type":"sampleFolder","typeLabel":"sample folder","id":"SF-1001","name":"Ideas"},` +
		`"parentEntryIds":["BIN-1002"],"insideEntryIds":[],"records":[` +
		`{"recordType":"sampleNote","recordTypeLabel":"sample note","recordId":"SN-1001","recordName":"First idea","parentId":"SF-1001"}],` +
		plannerBinAccess + `}`
)

// binWithParent fills the recycle bin as Priya: the note "First idea" (BIN-1001), then its folder "Ideas"
// with the note "Second idea" (BIN-1002); then a new folder takes the name "Ideas" (SF-1002).
func binWithParent(t *testing.T, priya *apitest.Client) {
	t.Helper()
	createFolder(t, priya, "Ideas")             // SF-1001
	addNote(t, priya, "SF-1001", "First idea")  // SN-1001
	addNote(t, priya, "SF-1001", "Second idea") // SN-1002
	deleteNote(t, priya, "SN-1001")             // BIN-1001
	deleteFolder(t, priya, "SF-1001")           // BIN-1002
	createFolder(t, priya, "Ideas")             // SF-1002
}

// The recycle bin lists its entries newest first with their restore state, the entries to restore first,
// the entries inside them and the records they hold, as the contract describes (011 FR-014 – FR-017,
// FR-028).
func TestListRecycleBin_Items_Contract(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)

	response := priya.Do(t, http.MethodGet, "/api/recycle-bin", nil)

	assertAnswer(t, response, http.StatusOK, `{"items":[`+folderBinItem+`,`+noteBinItem+`],"total":2}`)
}

// The restore preview of the note whose folder is in the bin: the folder comes back first, under a new
// name because "Ideas" is taken, and the note returns into it (011 FR-018, FR-022, FR-023). Nothing is
// saved: the bin and the log are unchanged.
func TestRestorePreview_Contract(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)
	logBefore := getLog(t, priya, "")

	response := priya.Do(t, http.MethodGet, "/api/recycle-bin/BIN-1001/restore-preview", nil)

	assertAnswer(t, response, http.StatusOK, `{"entry":`+noteBinItem+`,`+
		`"destination":[{"type":"sampleFolder","id":"SF-1001","name":"Ideas (restored)"}],`+
		`"parentEntries":[`+folderBinItem+`],`+
		`"changes":[{"recordType":"sampleFolder","recordTypeLabel":"sample folder","recordId":"SF-1001","recordName":"Ideas",`+
		`"restored":true,"field":"name","label":"Name","oldText":"Ideas","newText":"Ideas (restored)"}],`+
		`"notes":["Renamed to 'Ideas (restored)' because the name was taken."]}`)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/recycle-bin", nil), http.StatusOK,
		`{"items":[`+folderBinItem+`,`+noteBinItem+`],"total":2}`)
	if logAfter := getLog(t, priya, ""); logAfter.UnfilteredTotal != logBefore.UnfilteredTotal {
		t.Errorf("the log holds %d entries after the preview, want still %d", logAfter.UnfilteredTotal, logBefore.UnfilteredTotal)
	}
}

// The preview of an entry the bin never had answers not found; of an entry that already left the bin, the
// conflict already_resolved with how, by whom and when (011 FR-032).
func TestRestorePreview_UnknownAndResolved_404_409(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")   // SF-1001
	deleteFolder(t, priya, "SF-1001") // BIN-1001
	if response := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/restore", nil); response.Status != http.StatusOK {
		t.Fatalf("restoring BIN-1001: %d %s", response.Status, response.Body)
	}

	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/recycle-bin/BIN-9999/restore-preview", nil), http.StatusNotFound, unknownEntryBody)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/recycle-bin/BIN-1001/restore-preview", nil), http.StatusConflict,
		`{"code":"already_resolved","message":"'Ideas' is no longer in the recycle bin.","fields":[],`+
			`"resolution":{"how":"restored","by":{"id":"U-1002","name":"Priya Raman"},"at":"2026-03-12T09:00:00Z"}}`)
}

// Restore with withParents=true brings back the folder's entry first, then the note's, in one step (011
// FR-023); withParents must be true or false.
func TestRestore_WithParents_Contract(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)

	invalid := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/restore?withParents=yes", nil, apitest.SkipRequestValidation())
	response := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/restore?withParents=true", nil)

	assertAnswer(t, invalid, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"withParents","message":"Reload the page and try again."}]}`)
	assertAnswer(t, response, http.StatusOK, `{"recordType":"sampleNote","recordTypeLabel":"sample note",`+
		`"recordId":"SN-1001","recordName":"First idea","details":""}`)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/recycle-bin", nil), http.StatusOK, emptyBinBody)
	page := getLog(t, priya, "?action=restored")
	if got := []string{page.Items[1].RecordName, page.Items[0].RecordName}; page.Total != 2 || !slices.Equal(got, []string{"Ideas (restored)", "First idea"}) {
		t.Errorf("restored entries = %d %v, want the folder, then the note", page.Total, got)
	}
}

// Restore refuses with restore_parent_first while the parent is in the bin, and with
// parent_deleted_permanently once it was deleted permanently, also with withParents=true; both name the
// parent (011 FR-023, FR-024).
func TestRestore_ParentCodes_409(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	server.GrantRole(t, priyasEmail, "administrator", "all") // 014: deleting permanently is an administrator's
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)
	parent := `"parent":{"type":"sampleFolder","id":"SF-1001","name":"Ideas"}`

	first := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/restore", nil)
	if purge := priya.Do(t, http.MethodDelete, "/api/recycle-bin/BIN-1002", nil); purge.Status != http.StatusOK {
		t.Fatalf("deleting BIN-1002 permanently: %d %s", purge.Status, purge.Body)
	}
	gone := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/restore?withParents=true", nil)

	assertAnswer(t, first, http.StatusConflict, `{"code":"restore_parent_first","message":"Restore 'Ideas' first","fields":[],`+parent+`}`)
	assertAnswer(t, gone, http.StatusConflict, `{"code":"parent_deleted_permanently","message":"This sample note can't be restored `+
		`because its sample folder 'Ideas' was permanently deleted.","fields":[],`+parent+`}`)
}

// Delete permanently answers the record that was deleted for good, writes one "permanently deleted" entry
// with the record's place and contents at the time of the delete, and leaves the entry waiting for it in
// the bin, no longer restorable (011 FR-028, FR-030).
func TestPurge_Contract(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	server.GrantRole(t, priyasEmail, "administrator", "all") // 014: deleting permanently is an administrator's
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)

	response := priya.Do(t, http.MethodDelete, "/api/recycle-bin/BIN-1002", nil)

	assertAnswer(t, response, http.StatusOK,
		`{"recordType":"sampleFolder","recordTypeLabel":"sample folder","recordId":"SF-1001","recordName":"Ideas"}`)
	var bin history.BinPage
	priya.Do(t, http.MethodGet, "/api/recycle-bin", nil).Decode(t, &bin)
	if bin.Total != 1 || bin.Items[0].ID != "BIN-1001" || bin.Items[0].RestoreState != history.ParentDeletedPermanently {
		t.Errorf("recycle bin = %+v, want only BIN-1001, which can no longer be restored", bin)
	}
	page := getLog(t, priya, "?action=deletedPermanently")
	if page.Total != 1 {
		t.Fatalf("%d permanently deleted entries, want 1", page.Total)
	}
	got := page.Items[0]
	if got.Action != history.ActionDeletedPermanently || got.Verb != "permanently deleted" || got.RecordID != "SF-1001" ||
		got.Details != "with 1 sample note" || got.Actor.Name != "Priya Raman" || got.RecordNow.State != history.RecordDeletedPermanently {
		t.Errorf("entry = %+v, want Priya permanently deleted SF-1001 with 1 sample note", got)
	}
}

// Delete permanently of an entry the bin never had answers not found; of an entry that already left the
// bin, the conflict already_resolved with how, by whom and when (011 FR-032).
func TestPurge_404_409(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	server.GrantRole(t, priyasEmail, "administrator", "all") // 014: deleting permanently is an administrator's
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")   // SF-1001
	deleteFolder(t, priya, "SF-1001") // BIN-1001
	if response := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/restore", nil); response.Status != http.StatusOK {
		t.Fatalf("restoring BIN-1001: %d %s", response.Status, response.Body)
	}

	assertAnswer(t, priya.Do(t, http.MethodDelete, "/api/recycle-bin/BIN-9999", nil), http.StatusNotFound, unknownEntryBody)
	assertAnswer(t, priya.Do(t, http.MethodDelete, "/api/recycle-bin/BIN-1001", nil), http.StatusConflict,
		`{"code":"already_resolved","message":"'Ideas' is no longer in the recycle bin.","fields":[],`+
			`"resolution":{"how":"restored","by":{"id":"U-1002","name":"Priya Raman"},"at":"2026-03-12T09:00:00Z"}}`)
}

// Empty recycle bin deletes permanently the listed entries that are still in the bin, answers how many,
// and writes one "emptied the recycle bin" entry that lists them; when nothing listed is left it answers 0
// and writes nothing (011 FR-029, FR-030, FR-032).
func TestEmptyRecycleBin_Contract(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	server.GrantRole(t, priyasEmail, "administrator", "all") // 014: deleting permanently is an administrator's
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)
	body := map[string][]string{"entryIds": {"BIN-1002", "BIN-1001"}}

	response := priya.Do(t, http.MethodPost, "/api/recycle-bin/empty", body)
	again := priya.Do(t, http.MethodPost, "/api/recycle-bin/empty", body)

	assertAnswer(t, response, http.StatusOK, `{"count":2}`)
	assertAnswer(t, again, http.StatusOK, `{"count":0}`)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/recycle-bin", nil), http.StatusOK, emptyBinBody)
	page := getLog(t, priya, "?action=deletedPermanently")
	if page.Total != 1 {
		t.Fatalf("%d permanently deleted entries, want 1 for the emptied bin", page.Total)
	}
	got := page.Items[0]
	items := []string{}
	for _, item := range got.Items {
		items = append(items, item.RecordID+" "+item.Details)
	}
	if got.Action != history.ActionEmptiedRecycleBin || got.Verb != "emptied the recycle bin" || got.Details != "2 items" ||
		!slices.Equal(items, []string{"SF-1001 with 1 sample note", "SN-1001 "}) {
		t.Errorf("entry = %+v with items %q, want the emptied bin with SF-1001 and SN-1001", got, items)
	}
}

// Empty recycle bin needs at least one entry: an empty list is refused next to entryIds (011 FR-029).
func TestEmptyRecycleBin_EmptyList_400(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	server.GrantRole(t, priyasEmail, "administrator", "all") // 014: deleting permanently is an administrator's
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPost, "/api/recycle-bin/empty", map[string][]string{"entryIds": {}},
		apitest.SkipRequestValidation())

	assertAnswer(t, response, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"entryIds","message":"Reload the page and try again."}]}`)
}

// Every history operation needs a signed-in person (FR-005).
func TestHistoryOperations_WithoutSession_FR005(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	calls := []struct{ method, path string }{
		{http.MethodGet, "/api/activity-log"},
		{http.MethodGet, "/api/activity-log/summary"},
		{http.MethodGet, "/api/recycle-bin"},
		{http.MethodGet, "/api/recycle-bin/BIN-1001/restore-preview"},
		{http.MethodPost, "/api/recycle-bin/BIN-1001/undo"},
		{http.MethodPost, "/api/recycle-bin/BIN-1001/restore"},
		{http.MethodDelete, "/api/recycle-bin/BIN-1001"},
		{http.MethodPost, "/api/recycle-bin/empty"},
	}

	for _, call := range calls {
		response := server.Do(t, call.method, call.path, nil, apitest.SkipRequestValidation())

		assertAnswer(t, response, http.StatusUnauthorized, signInRequiredBody)
	}
}

// The helpers below make changes through the developer sample records, the record type the API offers in
// feature 001, so that the log and the recycle bin have entries.

// createFolder creates a sample folder and stops the test when that fails.
func createFolder(t *testing.T, c *apitest.Client, name string) {
	t.Helper()
	response := c.Do(t, http.MethodPost, "/api/sample-folders", map[string]string{"name": name})
	if response.Status != http.StatusCreated {
		t.Fatalf("creating the sample folder %q: %d %s", name, response.Status, response.Body)
	}
}

// renameFolder renames a sample folder still at version 1, and stops the test when that fails.
func renameFolder(t *testing.T, c *apitest.Client, id, name string) {
	t.Helper()
	response := c.Do(t, http.MethodPatch, "/api/sample-folders/"+id, map[string]any{"name": name, "version": 1})
	if response.Status != http.StatusOK {
		t.Fatalf("renaming the sample folder %s: %d %s", id, response.Status, response.Body)
	}
}

// addNote adds a note to a sample folder and stops the test when that fails.
func addNote(t *testing.T, c *apitest.Client, folderID, title string) {
	t.Helper()
	response := c.Do(t, http.MethodPost, "/api/sample-folders/"+folderID+"/notes", map[string]string{"title": title})
	if response.Status != http.StatusCreated {
		t.Fatalf("adding the note %q: %d %s", title, response.Status, response.Body)
	}
}

// deleteNote deletes a sample note, which moves it to the recycle bin, and stops the test when that fails.
func deleteNote(t *testing.T, c *apitest.Client, id string) {
	t.Helper()
	if response := c.Do(t, http.MethodDelete, "/api/sample-notes/"+id, nil); response.Status != http.StatusOK {
		t.Fatalf("deleting the note %s: %d %s", id, response.Status, response.Body)
	}
}

// deleteFolder deletes a sample folder with its notes and stops the test when that fails.
func deleteFolder(t *testing.T, c *apitest.Client, id string) {
	t.Helper()
	if response := c.Do(t, http.MethodDelete, "/api/sample-folders/"+id, nil); response.Status != http.StatusOK {
		t.Fatalf("deleting the folder %s: %d %s", id, response.Status, response.Body)
	}
}

// getLog answers GET /api/activity-log with query, and stops the test when it is not 200.
func getLog(t *testing.T, c *apitest.Client, query string) history.LogPage {
	t.Helper()
	response := c.Do(t, http.MethodGet, "/api/activity-log"+query, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/activity-log%s = %d %s", query, response.Status, response.Body)
	}
	var page history.LogPage
	response.Decode(t, &page)
	return page
}

func entryIDs(entries []history.EntryView) []string {
	list := make([]string, 0, len(entries))
	for _, e := range entries {
		list = append(list, e.ID)
	}
	return list
}

// firstID returns the id of the first entry, or "nothing" when there is none, for failure messages.
func firstID(entries []history.EntryView) string {
	if len(entries) == 0 {
		return "nothing"
	}
	return entries[0].ID
}

// assertAnswer checks the status and the exact body of an answer.
func assertAnswer(t *testing.T, response *apitest.Response, status int, body string) {
	t.Helper()
	if response.Status != status || string(response.Body) != body {
		t.Errorf("answer = %d %s\nwant %d %s", response.Status, response.Body, status, body)
	}
}
