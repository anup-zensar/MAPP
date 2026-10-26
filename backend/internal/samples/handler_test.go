package samples_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/apitest"
)

const (
	priyasEmail  = "priya.raman@shell.com"
	danielsEmail = "daniel.okoro@shell.com"
)

// Every test starts without the demo history, so the bin entries (BIN-1001, …) are its own.
const (
	emptyPageBody = `{"items":[],"total":0}`
	ideasBody     = `{"id":"SF-1001","name":"Ideas","version":1,` +
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},` +
		`"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1002","name":"Priya Raman"},"notes":[]}`
	firstIdeaBody = `{"id":"SN-1001","folderId":"SF-1001","title":"First idea","version":1,` +
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},` +
		`"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1002","name":"Priya Raman"}}`
	folderGoneBody = `{"error":{"code":"not_found",` +
		`"message":"This sample folder no longer exists — someone may have deleted it.","fields":[]}}`
	noteGoneBody = `{"error":{"code":"not_found",` +
		`"message":"This sample note no longer exists — someone may have deleted it.","fields":[]}}`
)

func TestListFolders_WithTheirNotes(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/v1/sample-folders", nil), http.StatusOK, emptyPageBody)
	createFolder(t, priya, "Plans")
	createFolder(t, priya, "Ideas")
	addNote(t, priya, "SF-1002", "First idea")

	response := priya.Do(t, http.MethodGet, "/api/v1/sample-folders", nil)

	assertAnswer(t, response, http.StatusOK, `{"items":[`+
		`{"id":"SF-1002","name":"Ideas","version":1,`+
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},`+
		`"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1002","name":"Priya Raman"},"notes":[`+
		`{"id":"SN-1001","folderId":"SF-1002","title":"First idea","version":1,`+
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},`+
		`"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1002","name":"Priya Raman"}}]},`+
		`{"id":"SF-1001","name":"Plans","version":1,`+
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},`+
		`"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1002","name":"Priya Raman"},"notes":[]}],"total":2}`)
}

func TestCreateFolder_CreatedOrRefused_FR020(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)

	created := priya.Do(t, http.MethodPost, "/api/v1/sample-folders", map[string]string{"name": "  Ideas "})
	missing := priya.Do(t, http.MethodPost, "/api/v1/sample-folders", map[string]string{"name": " "})

	assertAnswer(t, created, http.StatusCreated, ideasBody)
	assertAnswer(t, missing, http.StatusBadRequest, validationBody(`{"field":"name","message":"Enter a name."}`))
}

// Daniel may rename Priya's folder: the sample records have no permission checks.
func TestUpdateFolder_Renamed_FR029(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	createFolder(t, server.SignIn(t, priyasEmail), "Ideas")
	daniel := server.SignIn(t, danielsEmail)
	server.Advance(time.Hour)

	response := daniel.Do(t, http.MethodPatch, "/api/v1/sample-folders/SF-1001", map[string]any{"name": "Ideas 2", "version": 1})

	assertAnswer(t, response, http.StatusOK, `{"id":"SF-1001","name":"Ideas 2","version":2,`+
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},`+
		`"updatedAt":"2026-03-12T10:00:00Z","updatedBy":{"id":"U-1003","name":"Daniel Okoro"},"notes":[]}`)
}

func TestUpdateFolder_Refused_FR020_FR029(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	createFolder(t, priya, "Ideas")
	server.Advance(5*time.Hour + 32*time.Minute)
	renamed := daniel.Do(t, http.MethodPatch, "/api/v1/sample-folders/SF-1001", map[string]any{"name": "Daniel's ideas", "version": 1})
	assertAnswer(t, renamed, http.StatusOK, `{"id":"SF-1001","name":"Daniel's ideas","version":2,`+
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},`+
		`"updatedAt":"2026-03-12T14:32:00Z","updatedBy":{"id":"U-1003","name":"Daniel Okoro"},"notes":[]}`)

	invalid := priya.Do(t, http.MethodPatch, "/api/v1/sample-folders/SF-1001", map[string]any{"name": " ", "version": 0})
	gone := priya.Do(t, http.MethodPatch, "/api/v1/sample-folders/SF-9999", map[string]any{"name": "Ideas", "version": 1})
	outOfDate := priya.Do(t, http.MethodPatch, "/api/v1/sample-folders/SF-1001", map[string]any{"name": "Priya's ideas", "version": 1})

	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(
		`{"field":"name","message":"Enter a name."},{"field":"version","message":"Reload the page and try again."}`))
	assertAnswer(t, gone, http.StatusNotFound, folderGoneBody)
	assertAnswer(t, outOfDate, http.StatusConflict, `{"error":{"code":"version_conflict",`+
		`"message":"Not saved — Daniel Okoro changed this sample folder","fields":[],`+
		`"conflict":{"changedBy":"Daniel Okoro","changedAt":"2026-03-12T14:32:00Z"}}}`)
}

// The answer is the bin entry, so the web app can offer Undo.
func TestDeleteFolder_MovesItToTheRecycleBin_FR024(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	createFolder(t, priya, "Plans")
	addNote(t, priya, "SF-1002", "Budget")
	addNote(t, priya, "SF-1002", "Dates")

	empty := priya.Do(t, http.MethodDelete, "/api/v1/sample-folders/SF-1001", nil)
	withNotes := priya.Do(t, http.MethodDelete, "/api/v1/sample-folders/SF-1002", nil)
	gone := priya.Do(t, http.MethodDelete, "/api/v1/sample-folders/SF-1001", nil)

	ideasEntry := `{"id":"BIN-1001","recordType":"sampleFolder","recordTypeLabel":"sample folder",` +
		`"recordId":"SF-1001","recordName":"Ideas","path":[],"parent":null,` +
		`"deletedAt":"2026-03-12T09:00:00Z","deletedBy":{"id":"U-1002","name":"Priya Raman"},"contents":[]}`
	plansEntry := `{"id":"BIN-1002","recordType":"sampleFolder","recordTypeLabel":"sample folder",` +
		`"recordId":"SF-1002","recordName":"Plans","path":[],"parent":null,` +
		`"deletedAt":"2026-03-12T09:00:00Z","deletedBy":{"id":"U-1002","name":"Priya Raman"},` +
		`"contents":[{"recordType":"sampleNote","label":"sample notes","count":2}]}`
	// Priya's Planner role may restore an entry but not delete it permanently.
	plannerAccess := `"access":{"restore":{"available":true,"reason":""},"deletePermanently":{"available":false,` +
		`"reason":"Deleting permanently isn't part of your role (Planner). Ask an administrator if you need it."}}`
	ideasRow := strings.TrimSuffix(ideasEntry, "}") + `,"recordTypePlural":"sample folders","restoreState":"restorable",` +
		`"missingParent":null,"parentEntryIds":[],"insideEntryIds":[],"records":[` +
		`{"recordType":"sampleFolder","recordTypeLabel":"sample folder","recordId":"SF-1001","recordName":"Ideas","parentId":""}],` +
		plannerAccess + `}`
	plansRow := strings.TrimSuffix(plansEntry, "}") + `,"recordTypePlural":"sample folders","restoreState":"restorable",` +
		`"missingParent":null,"parentEntryIds":[],"insideEntryIds":[],"records":[` +
		`{"recordType":"sampleFolder","recordTypeLabel":"sample folder","recordId":"SF-1002","recordName":"Plans","parentId":""},` +
		`{"recordType":"sampleNote","recordTypeLabel":"sample note","recordId":"SN-1001","recordName":"Budget","parentId":"SF-1002"},` +
		`{"recordType":"sampleNote","recordTypeLabel":"sample note","recordId":"SN-1002","recordName":"Dates","parentId":"SF-1002"}],` +
		plannerAccess + `}`
	assertAnswer(t, empty, http.StatusOK, `{"recycleBinEntry":`+ideasEntry+`}`)
	assertAnswer(t, withNotes, http.StatusOK, `{"recycleBinEntry":`+plansEntry+`}`)
	assertAnswer(t, gone, http.StatusNotFound, folderGoneBody)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil), http.StatusOK,
		`{"items":[`+plansRow+`,`+ideasRow+`],"total":2}`)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/v1/sample-folders", nil), http.StatusOK, emptyPageBody)
}

func TestCreateNote_AddedOrRefused_FR020(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")

	added := priya.Do(t, http.MethodPost, "/api/v1/sample-folders/SF-1001/notes", map[string]string{"title": " First idea "})
	missing := priya.Do(t, http.MethodPost, "/api/v1/sample-folders/SF-1001/notes", map[string]string{"title": ""})
	gone := priya.Do(t, http.MethodPost, "/api/v1/sample-folders/SF-9999/notes", map[string]string{"title": "Lost idea"})

	assertAnswer(t, added, http.StatusCreated, firstIdeaBody)
	assertAnswer(t, missing, http.StatusBadRequest, validationBody(`{"field":"title","message":"Enter a title."}`))
	assertAnswer(t, gone, http.StatusNotFound, folderGoneBody)
}

func TestDeleteNote_MovesItToTheRecycleBin_FR024(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	addNote(t, priya, "SF-1001", "First idea")

	deleted := priya.Do(t, http.MethodDelete, "/api/v1/sample-notes/SN-1001", nil)
	gone := priya.Do(t, http.MethodDelete, "/api/v1/sample-notes/SN-1001", nil)

	assertAnswer(t, deleted, http.StatusOK, `{"recycleBinEntry":{"id":"BIN-1001","recordType":"sampleNote",`+
		`"recordTypeLabel":"sample note","recordId":"SN-1001","recordName":"First idea",`+
		`"path":[{"type":"sampleFolder","id":"SF-1001","name":"Ideas"}],"parent":{"type":"sampleFolder","id":"SF-1001","name":"Ideas"},`+
		`"deletedAt":"2026-03-12T09:00:00Z","deletedBy":{"id":"U-1002","name":"Priya Raman"},"contents":[]}}`)
	assertAnswer(t, gone, http.StatusNotFound, noteGoneBody)
}

func TestUndo_BringsTheFolderBackExactly_FR026(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	addNote(t, priya, "SF-1001", "First idea")
	before := priya.Do(t, http.MethodGet, "/api/v1/sample-folders", nil)
	deleteFolder(t, priya, "SF-1001")
	server.Advance(2 * time.Minute) // the last moment the server accepts Undo

	response := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/undo", nil)

	assertAnswer(t, response, http.StatusOK, `{"recordType":"sampleFolder","recordTypeLabel":"sample folder",`+
		`"recordId":"SF-1001","recordName":"Ideas","details":""}`)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/v1/sample-folders", nil), http.StatusOK, string(before.Body))
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil), http.StatusOK, emptyPageBody)
}

func TestUndo_TooLate_FR026(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	deleteFolder(t, priya, "SF-1001")
	server.Advance(2*time.Minute + time.Millisecond)

	response := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/undo", nil)

	assertAnswer(t, response, http.StatusConflict, `{"error":{"code":"undo_expired",`+
		`"message":"Undo is no longer available — restore it from the recycle bin","fields":[]}}`)
}

func TestRestore_ParentFirst_FR025(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	addNote(t, priya, "SF-1001", "First idea")
	deleteNote(t, priya, "SN-1001")
	deleteFolder(t, priya, "SF-1001")

	response := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/restore", nil)

	assertAnswer(t, response, http.StatusConflict, `{"error":{"code":"restore_parent_first","message":"Restore 'Ideas' first",`+
		`"fields":[],"parent":{"type":"sampleFolder","id":"SF-1001","name":"Ideas"}}}`)
}

func TestUndo_AlreadyUndone_FR025(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	deleteFolder(t, priya, "SF-1001")
	server.Advance(time.Minute)
	assertAnswer(t, priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/undo", nil), http.StatusOK,
		`{"recordType":"sampleFolder","recordTypeLabel":"sample folder","recordId":"SF-1001","recordName":"Ideas","details":""}`)

	again := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/undo", nil)
	restore := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/restore", nil)

	alreadyUndone := `{"error":{"code":"already_resolved","message":"'Ideas' is no longer in the recycle bin.","fields":[],` +
		`"resolution":{"how":"undone","by":{"id":"U-1002","name":"Priya Raman"},"at":"2026-03-12T09:01:00Z"}}}`
	assertAnswer(t, again, http.StatusConflict, alreadyUndone)
	assertAnswer(t, restore, http.StatusConflict, alreadyUndone)
}

func createFolder(t *testing.T, c *apitest.Client, name string) {
	t.Helper()
	response := c.Do(t, http.MethodPost, "/api/v1/sample-folders", map[string]string{"name": name})
	if response.Status != http.StatusCreated {
		t.Fatalf("creating the sample folder %q: %d %s", name, response.Status, response.Body)
	}
}

func addNote(t *testing.T, c *apitest.Client, folderID, title string) {
	t.Helper()
	response := c.Do(t, http.MethodPost, "/api/v1/sample-folders/"+folderID+"/notes", map[string]string{"title": title})
	if response.Status != http.StatusCreated {
		t.Fatalf("adding the note %q: %d %s", title, response.Status, response.Body)
	}
}

func deleteFolder(t *testing.T, c *apitest.Client, id string) {
	t.Helper()
	if response := c.Do(t, http.MethodDelete, "/api/v1/sample-folders/"+id, nil); response.Status != http.StatusOK {
		t.Fatalf("deleting the sample folder %s: %d %s", id, response.Status, response.Body)
	}
}

func deleteNote(t *testing.T, c *apitest.Client, id string) {
	t.Helper()
	if response := c.Do(t, http.MethodDelete, "/api/v1/sample-notes/"+id, nil); response.Status != http.StatusOK {
		t.Fatalf("deleting the sample note %s: %d %s", id, response.Status, response.Body)
	}
}

func validationBody(fields string) string {
	return `{"error":{"code":"validation_failed","message":"Some fields need your attention.","fields":[` + fields + `]}}`
}

func assertAnswer(t *testing.T, response *apitest.Response, status int, body string) {
	t.Helper()
	if response.Status != status || string(response.Body) != body {
		t.Errorf("answer = %d %s\nwant %d %s", response.Status, response.Body, status, body)
	}
}
