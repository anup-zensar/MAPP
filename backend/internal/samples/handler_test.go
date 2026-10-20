package samples_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"northstar/internal/platform/apitest"
)

const (
	priyasEmail  = "priya.raman@northstar.demo"
	danielsEmail = "daniel.okoro@northstar.demo"
)

// Every test here starts the API with apitest.WithoutDemoHistory: the recycle bin starts empty, so its
// entries (BIN-1001, …) are the test's own.

// The bodies of the API's answers, as the web app receives them. Priya creates every record when the test
// starts, at 12 Mar 2026 09:00 UTC.
const (
	emptyPageBody = `{"items":[],"total":0}`
	ideasBody     = `{"id":"SF-1001","name":"Ideas","version":1,` +
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},` +
		`"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1002","name":"Priya Raman"},"notes":[]}`
	firstIdeaBody = `{"id":"SN-1001","folderId":"SF-1001","title":"First idea","version":1,` +
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},` +
		`"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1002","name":"Priya Raman"}}`
	folderGoneBody = `{"code":"not_found",` +
		`"message":"This sample folder no longer exists — someone may have deleted it.","fields":[]}`
	noteGoneBody = `{"code":"not_found",` +
		`"message":"This sample note no longer exists — someone may have deleted it.","fields":[]}`
)

// The developer page lists every sample folder with its notes; a folder without notes has an empty list
// (research R10).
func TestListFolders_WithTheirNotes(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/sample-folders", nil), http.StatusOK, emptyPageBody)
	createFolder(t, priya, "Plans") // SF-1001
	createFolder(t, priya, "Ideas") // SF-1002
	addNote(t, priya, "SF-1002", "First idea")

	response := priya.Do(t, http.MethodGet, "/api/sample-folders", nil)

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

// Creating a folder answers 201 with the new folder (FR-022 logs it); a missing or taken name is refused
// next to the field (FR-020).
func TestCreateFolder_CreatedOrRefused_FR020(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)

	created := priya.Do(t, http.MethodPost, "/api/sample-folders", map[string]string{"name": "  Ideas "})
	missing := priya.Do(t, http.MethodPost, "/api/sample-folders", map[string]string{"name": " "})
	taken := priya.Do(t, http.MethodPost, "/api/sample-folders", map[string]string{"name": "IDEAS"})

	assertAnswer(t, created, http.StatusCreated, ideasBody)
	assertAnswer(t, missing, http.StatusBadRequest, validationBody(`{"field":"name","message":"Enter a name."}`))
	assertAnswer(t, taken, http.StatusBadRequest, validationBody(
		`{"field":"name","message":"There is already a sample folder called 'Ideas'. Choose another name."}`))
}

// A rename from the current version answers the folder with its new name and version, and who changed it
// when; Daniel may rename Priya's folder, because there are no permission checks (FR-029, FR-008).
func TestUpdateFolder_Renamed_FR029(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	createFolder(t, server.SignIn(t, priyasEmail), "Ideas")
	daniel := server.SignIn(t, danielsEmail)
	server.Advance(time.Hour)

	response := daniel.Do(t, http.MethodPatch, "/api/sample-folders/SF-1001", map[string]any{"name": "Ideas 2", "version": 1})

	assertAnswer(t, response, http.StatusOK, `{"id":"SF-1001","name":"Ideas 2","version":2,`+
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},`+
		`"updatedAt":"2026-03-12T10:00:00Z","updatedBy":{"id":"U-1003","name":"Daniel Okoro"},"notes":[]}`)
}

// Every problem of a rename is reported at once, a folder that is gone answers not found, and a rename
// from an out-of-date copy is refused naming who changed the folder and when (FR-020, FR-029).
func TestUpdateFolder_Refused_FR020_FR029(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	createFolder(t, priya, "Ideas")
	server.Advance(5*time.Hour + 32*time.Minute)
	renamed := daniel.Do(t, http.MethodPatch, "/api/sample-folders/SF-1001", map[string]any{"name": "Daniel's ideas", "version": 1})
	assertAnswer(t, renamed, http.StatusOK, `{"id":"SF-1001","name":"Daniel's ideas","version":2,`+
		`"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1002","name":"Priya Raman"},`+
		`"updatedAt":"2026-03-12T14:32:00Z","updatedBy":{"id":"U-1003","name":"Daniel Okoro"},"notes":[]}`)

	invalid := priya.Do(t, http.MethodPatch, "/api/sample-folders/SF-1001", map[string]any{"name": " ", "version": 0})
	gone := priya.Do(t, http.MethodPatch, "/api/sample-folders/SF-9999", map[string]any{"name": "Ideas", "version": 1})
	outOfDate := priya.Do(t, http.MethodPatch, "/api/sample-folders/SF-1001", map[string]any{"name": "Priya's ideas", "version": 1})

	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(
		`{"field":"name","message":"Enter a name."},{"field":"version","message":"Reload the page and try again."}`))
	assertAnswer(t, gone, http.StatusNotFound, folderGoneBody)
	assertAnswer(t, outOfDate, http.StatusConflict, `{"code":"version_conflict",`+
		`"message":"Not saved — Daniel Okoro changed this sample folder","fields":[],`+
		`"conflict":{"changedBy":"Daniel Okoro","changedAt":"2026-03-12T14:32:00Z"}}`)
}

// Deleting a folder moves it with its notes to the recycle bin and answers the bin entry, so that the web
// app can offer Undo; the bin lists it, newest first, and the folders list no longer does (FR-024).
func TestDeleteFolder_MovesItToTheRecycleBin_FR024(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas") // SF-1001
	createFolder(t, priya, "Plans") // SF-1002
	addNote(t, priya, "SF-1002", "Budget")
	addNote(t, priya, "SF-1002", "Dates")

	empty := priya.Do(t, http.MethodDelete, "/api/sample-folders/SF-1001", nil)
	withNotes := priya.Do(t, http.MethodDelete, "/api/sample-folders/SF-1002", nil)
	gone := priya.Do(t, http.MethodDelete, "/api/sample-folders/SF-1001", nil)

	ideasEntry := `{"id":"BIN-1001","recordType":"sampleFolder","recordTypeLabel":"sample folder",` +
		`"recordId":"SF-1001","recordName":"Ideas","path":[],"parent":null,` +
		`"deletedAt":"2026-03-12T09:00:00Z","deletedBy":{"id":"U-1002","name":"Priya Raman"},"contents":[]}`
	plansEntry := `{"id":"BIN-1002","recordType":"sampleFolder","recordTypeLabel":"sample folder",` +
		`"recordId":"SF-1002","recordName":"Plans","path":[],"parent":null,` +
		`"deletedAt":"2026-03-12T09:00:00Z","deletedBy":{"id":"U-1002","name":"Priya Raman"},` +
		`"contents":[{"recordType":"sampleNote","label":"sample notes","count":2}]}`
	// The recycle bin lists each entry with its restore state and the records it holds (011 FR-015 – FR-017).
	ideasRow := strings.TrimSuffix(ideasEntry, "}") + `,"recordTypePlural":"sample folders","restoreState":"restorable",` +
		`"missingParent":null,"parentEntryIds":[],"insideEntryIds":[],"records":[` +
		`{"recordType":"sampleFolder","recordTypeLabel":"sample folder","recordId":"SF-1001","recordName":"Ideas","parentId":""}]}`
	plansRow := strings.TrimSuffix(plansEntry, "}") + `,"recordTypePlural":"sample folders","restoreState":"restorable",` +
		`"missingParent":null,"parentEntryIds":[],"insideEntryIds":[],"records":[` +
		`{"recordType":"sampleFolder","recordTypeLabel":"sample folder","recordId":"SF-1002","recordName":"Plans","parentId":""},` +
		`{"recordType":"sampleNote","recordTypeLabel":"sample note","recordId":"SN-1001","recordName":"Budget","parentId":"SF-1002"},` +
		`{"recordType":"sampleNote","recordTypeLabel":"sample note","recordId":"SN-1002","recordName":"Dates","parentId":"SF-1002"}]}`
	assertAnswer(t, empty, http.StatusOK, `{"recycleBinEntry":`+ideasEntry+`}`)
	assertAnswer(t, withNotes, http.StatusOK, `{"recycleBinEntry":`+plansEntry+`}`)
	assertAnswer(t, gone, http.StatusNotFound, folderGoneBody)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/recycle-bin", nil), http.StatusOK,
		`{"items":[`+plansRow+`,`+ideasRow+`],"total":2}`)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/sample-folders", nil), http.StatusOK, emptyPageBody)
}

// Adding a note answers 201 with the note; a missing title is refused next to the field, and a folder
// that is gone answers not found (FR-020).
func TestCreateNote_AddedOrRefused_FR020(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")

	added := priya.Do(t, http.MethodPost, "/api/sample-folders/SF-1001/notes", map[string]string{"title": " First idea "})
	missing := priya.Do(t, http.MethodPost, "/api/sample-folders/SF-1001/notes", map[string]string{"title": ""})
	gone := priya.Do(t, http.MethodPost, "/api/sample-folders/SF-9999/notes", map[string]string{"title": "Lost idea"})

	assertAnswer(t, added, http.StatusCreated, firstIdeaBody)
	assertAnswer(t, missing, http.StatusBadRequest, validationBody(`{"field":"title","message":"Enter a title."}`))
	assertAnswer(t, gone, http.StatusNotFound, folderGoneBody)
}

// Deleting a note moves it to the recycle bin; the entry says which folder it was in (FR-024).
func TestDeleteNote_MovesItToTheRecycleBin_FR024(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	addNote(t, priya, "SF-1001", "First idea")

	deleted := priya.Do(t, http.MethodDelete, "/api/sample-notes/SN-1001", nil)
	gone := priya.Do(t, http.MethodDelete, "/api/sample-notes/SN-1001", nil)

	assertAnswer(t, deleted, http.StatusOK, `{"recycleBinEntry":{"id":"BIN-1001","recordType":"sampleNote",`+
		`"recordTypeLabel":"sample note","recordId":"SN-1001","recordName":"First idea",`+
		`"path":[{"type":"sampleFolder","id":"SF-1001","name":"Ideas"}],"parent":{"type":"sampleFolder","id":"SF-1001","name":"Ideas"},`+
		`"deletedAt":"2026-03-12T09:00:00Z","deletedBy":{"id":"U-1002","name":"Priya Raman"},"contents":[]}}`)
	assertAnswer(t, gone, http.StatusNotFound, noteGoneBody)
}

// Undo right after a delete brings the folder and its notes back exactly as they were, and the bin is
// empty again (FR-026, US5 scenario 3).
func TestUndo_BringsTheFolderBackExactly_FR026(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	addNote(t, priya, "SF-1001", "First idea")
	before := priya.Do(t, http.MethodGet, "/api/sample-folders", nil)
	deleteFolder(t, priya, "SF-1001")
	server.Advance(2 * time.Minute) // the last moment the server accepts Undo

	response := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/undo", nil)

	assertAnswer(t, response, http.StatusOK, `{"recordType":"sampleFolder","recordTypeLabel":"sample folder",`+
		`"recordId":"SF-1001","recordName":"Ideas","details":""}`)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/sample-folders", nil), http.StatusOK, string(before.Body))
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/recycle-bin", nil), http.StatusOK, emptyPageBody)
}

// The server accepts Undo for 2 minutes after the delete; later, the item must be restored from the
// recycle bin (FR-026, D-037).
func TestUndo_TooLate_FR026(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	deleteFolder(t, priya, "SF-1001")
	server.Advance(2*time.Minute + time.Millisecond)

	response := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/undo", nil)

	assertAnswer(t, response, http.StatusConflict, `{"code":"undo_expired",`+
		`"message":"Undo is no longer available — restore it from the recycle bin","fields":[]}`)
}

// A note whose folder is also in the recycle bin cannot come back on its own: the refusal names the folder
// to restore first (FR-025, US5 scenario 4).
func TestRestore_ParentFirst_FR025(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	addNote(t, priya, "SF-1001", "First idea")
	deleteNote(t, priya, "SN-1001")   // BIN-1001
	deleteFolder(t, priya, "SF-1001") // BIN-1002

	response := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/restore", nil)

	assertAnswer(t, response, http.StatusConflict, `{"code":"restore_parent_first","message":"Restore 'Ideas' first",`+
		`"fields":[],"parent":{"type":"sampleFolder","id":"SF-1001","name":"Ideas"}}`)
}

// A delete that was already undone cannot be undone or restored again; the refusal says who did it and
// when (FR-025).
func TestUndo_AlreadyUndone_FR025(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	deleteFolder(t, priya, "SF-1001")
	server.Advance(time.Minute)
	assertAnswer(t, priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/undo", nil), http.StatusOK,
		`{"recordType":"sampleFolder","recordTypeLabel":"sample folder","recordId":"SF-1001","recordName":"Ideas","details":""}`)

	again := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/undo", nil)
	restore := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/restore", nil)

	alreadyUndone := `{"code":"already_resolved","message":"'Ideas' is no longer in the recycle bin.","fields":[],` +
		`"resolution":{"how":"undone","by":{"id":"U-1002","name":"Priya Raman"},"at":"2026-03-12T09:01:00Z"}}`
	assertAnswer(t, again, http.StatusConflict, alreadyUndone)
	assertAnswer(t, restore, http.StatusConflict, alreadyUndone)
}

// When someone has meanwhile created a folder with the same name, Undo is refused and nothing changes
// (spec edge case, D-037).
func TestUndo_NameTaken_FR026(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	deleteFolder(t, priya, "SF-1001")
	createFolder(t, server.SignIn(t, danielsEmail), "IDEAS")

	response := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/undo", nil)

	assertAnswer(t, response, http.StatusConflict, `{"code":"name_taken","message":"Can't undo — there is now `+
		`another sample folder called 'IDEAS'. Restore it from the recycle bin instead.","fields":[]}`)
}

// Restore from the recycle bin brings the folder back even when its name is taken, with the suffix
// " (restored)", and says so (FR-025, spec edge case).
func TestRestore_RenamesWhenTheNameIsTaken_FR025(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	deleteFolder(t, priya, "SF-1001")
	createFolder(t, priya, "Ideas")

	response := priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/restore", nil)

	assertAnswer(t, response, http.StatusOK, `{"recordType":"sampleFolder","recordTypeLabel":"sample folder",`+
		`"recordId":"SF-1001","recordName":"Ideas (restored)","details":"Renamed to 'Ideas (restored)' because the name was taken."}`)
}

// createFolder creates a sample folder and stops the test when that fails.
func createFolder(t *testing.T, c *apitest.Client, name string) {
	t.Helper()
	response := c.Do(t, http.MethodPost, "/api/sample-folders", map[string]string{"name": name})
	if response.Status != http.StatusCreated {
		t.Fatalf("creating the sample folder %q: %d %s", name, response.Status, response.Body)
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

// deleteFolder deletes a sample folder and stops the test when that fails.
func deleteFolder(t *testing.T, c *apitest.Client, id string) {
	t.Helper()
	if response := c.Do(t, http.MethodDelete, "/api/sample-folders/"+id, nil); response.Status != http.StatusOK {
		t.Fatalf("deleting the sample folder %s: %d %s", id, response.Status, response.Body)
	}
}

// deleteNote deletes a sample note and stops the test when that fails.
func deleteNote(t *testing.T, c *apitest.Client, id string) {
	t.Helper()
	if response := c.Do(t, http.MethodDelete, "/api/sample-notes/"+id, nil); response.Status != http.StatusOK {
		t.Fatalf("deleting the sample note %s: %d %s", id, response.Status, response.Body)
	}
}

// validationBody is the body of a validation_failed answer listing fields, which are written as JSON.
func validationBody(fields string) string {
	return `{"code":"validation_failed","message":"Some fields need your attention.","fields":[` + fields + `]}`
}

// assertAnswer checks the status and the exact body of an answer.
func assertAnswer(t *testing.T, response *apitest.Response, status int, body string) {
	t.Helper()
	if response.Status != status || string(response.Body) != body {
		t.Errorf("answer = %d %s\nwant %d %s", response.Status, response.Body, status, body)
	}
}
