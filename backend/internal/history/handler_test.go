package history_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"northstar/internal/history"
	"northstar/internal/platform/apitest"
)

const priyasEmail = "priya.raman@northstar.demo"

// The bodies of the API's answers, as the web app receives them.
const (
	emptyPageBody      = `{"items":[],"total":0}`
	signInRequiredBody = `{"code":"sign_in_required","message":"Please sign in to continue.","fields":[]}`
	unknownEntryBody   = `{"code":"not_found","message":"This item is not in the recycle bin.","fields":[]}`
	invalidLimitBody   = `{"code":"validation_failed","message":"Some fields need your attention.",` +
		`"fields":[{"field":"limit","message":"Choose a number from 1 to 200."}]}`
)

// Before anything has changed, the activity log and the recycle bin answer empty lists, never null
// (FR-022, FR-024, research R10).
func TestEmptyLogAndBin_AnswerEmptyLists_FR022_FR024(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	for _, path := range []string{"/api/activity-log", "/api/recycle-bin"} {
		response := priya.Do(t, http.MethodGet, path, nil)

		assertAnswer(t, response, http.StatusOK, emptyPageBody)
	}
}

// An entry carries who did what to which record and when, where the record is, and each changed field
// with readable values, exactly as the contract describes it (FR-022).
func TestListActivityLog_EntriesAsTheContractSays_FR022(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas") // SF-1001
	renameFolder(t, priya, "SF-1001", "Ideas 2")
	addNote(t, priya, "SF-1001", "First idea") // SN-1001

	response := priya.Do(t, http.MethodGet, "/api/activity-log?limit=2", nil)

	assertAnswer(t, response, http.StatusOK, `{"items":[`+
		`{"id":"LOG-1003","seq":1003,"at":"2026-03-12T09:00:00Z","actor":{"id":"U-1002","name":"Priya Raman"},`+
		`"action":"created","verb":"created","recordType":"sampleNote","recordTypeLabel":"sample note",`+
		`"recordId":"SN-1001","recordName":"First idea","path":[{"type":"sampleFolder","id":"SF-1001","name":"Ideas 2"}],`+
		`"changes":[],"comment":"","details":""},`+
		`{"id":"LOG-1002","seq":1002,"at":"2026-03-12T09:00:00Z","actor":{"id":"U-1002","name":"Priya Raman"},`+
		`"action":"changed","verb":"changed","recordType":"sampleFolder","recordTypeLabel":"sample folder",`+
		`"recordId":"SF-1001","recordName":"Ideas 2","path":[],`+
		`"changes":[{"field":"name","label":"Name","old":"Ideas","new":"Ideas 2","oldText":"Ideas","newText":"Ideas 2"}],`+
		`"comment":"","details":""}],"total":3}`)
}

// The log answers the newest matching entries first, filtered by record type and by record, at most limit
// of them, with the number of all matching entries (FR-022).
func TestListActivityLog_FiltersNewestFirst_FR022(t *testing.T) {
	server := apitest.New(t)
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
		var page history.LogPage
		priya.Do(t, http.MethodGet, "/api/activity-log"+tt.query, nil).Decode(t, &page)

		if got := entryIDs(page.Items); !slices.Equal(got, tt.want) || page.Total != tt.total {
			t.Errorf("GET /api/activity-log%s = %v (total %d), want %v (total %d)", tt.query, got, page.Total, tt.want, tt.total)
		}
	}
}

// Without a limit, the log answers the newest 50 entries (FR-022, contract default).
func TestListActivityLog_DefaultLimitIs50_FR022(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	for i := range 51 {
		createFolder(t, priya, fmt.Sprintf("Folder %d", i+1)) // LOG-1001 to LOG-1051
	}

	var page history.LogPage
	priya.Do(t, http.MethodGet, "/api/activity-log", nil).Decode(t, &page)

	if len(page.Items) != 50 || page.Total != 51 || page.Items[0].ID != "LOG-1051" || page.Items[49].ID != "LOG-1002" {
		t.Errorf("GET /api/activity-log = %d entries from %s (total %d), want the 50 from LOG-1051 to LOG-1002 (total 51)",
			len(page.Items), firstID(page.Items), page.Total)
	}
}

// A limit that is not a whole number from 1 to 200 is refused next to the field (FR-020); 1 and 200 are
// accepted.
func TestListActivityLog_InvalidLimit_FR020(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	for _, limit := range []string{"0", "201", "-5", "ten", "1.5", ""} {
		response := priya.Do(t, http.MethodGet, "/api/activity-log?limit="+limit, nil,
			apitest.SkipRequestValidation()) // the contract itself allows only 1 to 200

		assertAnswer(t, response, http.StatusBadRequest, invalidLimitBody)
	}
	for _, limit := range []string{"1", "200"} {
		response := priya.Do(t, http.MethodGet, "/api/activity-log?limit="+limit, nil)

		assertAnswer(t, response, http.StatusOK, emptyPageBody)
	}
}

// Undo and Restore of an entry the recycle bin never had answer not found (FR-025).
func TestUndoAndRestore_UnknownEntry_FR025(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	for _, action := range []string{"undo", "restore"} {
		response := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-9999/"+action, nil)

		assertAnswer(t, response, http.StatusNotFound, unknownEntryBody)
	}
}

// Every history operation needs a signed-in person (FR-005).
func TestHistoryOperations_WithoutSession_FR005(t *testing.T) {
	server := apitest.New(t)
	calls := []struct{ method, path string }{
		{http.MethodGet, "/api/activity-log"},
		{http.MethodGet, "/api/recycle-bin"},
		{http.MethodPost, "/api/recycle-bin/BIN-1001/undo"},
		{http.MethodPost, "/api/recycle-bin/BIN-1001/restore"},
	}

	for _, call := range calls {
		response := server.Do(t, call.method, call.path, nil)

		assertAnswer(t, response, http.StatusUnauthorized, signInRequiredBody)
	}
}

// The helpers below make changes through the developer sample records, the record type the API offers in
// feature 001, so that the log has entries.

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

func entryIDs(entries []history.Entry) []string {
	list := make([]string, 0, len(entries))
	for _, e := range entries {
		list = append(list, e.ID)
	}
	return list
}

// firstID returns the id of the first entry, or "nothing" when there is none, for failure messages.
func firstID(entries []history.Entry) string {
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
