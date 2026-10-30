package history_test

import (
	"net/http"
	"slices"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/apitest"
)

func TestUndoAndRestore_UnknownEntry_FR025(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)

	for _, action := range []string{"undo", "restore"} {
		response := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-9999/"+action, nil)

		assertAnswer(t, response, http.StatusNotFound, unknownEntryBody)
	}
}

// Priya may restore the entries but not delete them permanently: her Planner role does not include it (014 FR-011).
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

// binWithParent: SF-1001 "Ideas" with SN-1001 and SN-1002; BIN-1001 holds SN-1001, BIN-1002 the folder with
// SN-1002; then SF-1002 takes the name "Ideas".
func binWithParent(t *testing.T, priya *apitest.Client) {
	t.Helper()
	createFolder(t, priya, "Ideas")
	addNote(t, priya, "First idea")
	addNote(t, priya, "Second idea")
	deleteNote(t, priya, "SN-1001")
	deleteFolder(t, priya, "SF-1001")
	createFolder(t, priya, "Ideas")
}

func TestListRecycleBin_Items_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)

	response := priya.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil)

	assertAnswer(t, response, http.StatusOK, `{"items":[`+folderBinItem+`,`+noteBinItem+`],"total":2}`)
}

// Nothing is saved: the bin and the log are unchanged.
func TestRestorePreview_Contract_038_FR049_L23(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)
	logBefore := getLog(t, priya, "")

	response := priya.Do(t, http.MethodGet, "/api/v1/recycle-bin/BIN-1001/restore-preview", nil)
	unknown := priya.Do(t, http.MethodGet, "/api/v1/recycle-bin/BIN-9999/restore-preview", nil)

	assertAnswer(t, response, http.StatusOK, `{"entry":`+noteBinItem+`,`+
		`"destination":[{"type":"sampleFolder","id":"SF-1001","name":"Ideas (restored)"}],`+
		`"parentEntries":[`+folderBinItem+`],`+
		`"changes":[],`+ // 038 FR-049 (L-23): the note states the rename
		`"notes":["Renamed to 'Ideas (restored)' because the name was taken."]}`)
	assertAnswer(t, unknown, http.StatusNotFound, unknownEntryBody)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil), http.StatusOK,
		`{"items":[`+folderBinItem+`,`+noteBinItem+`],"total":2}`)
	if logAfter := getLog(t, priya, ""); logAfter.UnfilteredTotal != logBefore.UnfilteredTotal {
		t.Errorf("the log holds %d entries after the preview, want still %d", logAfter.UnfilteredTotal, logBefore.UnfilteredTotal)
	}
}

func TestRestore_WithParents_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)

	invalid := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/restore?withParents=yes", nil, apitest.SkipRequestValidation())
	response := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/restore?withParents=true", nil)

	assertAnswer(t, invalid, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"withParents","message":"Reload the page and try again."}]}}`)
	assertAnswer(t, response, http.StatusOK, `{"recordType":"sampleNote","recordTypeLabel":"sample note",`+
		`"recordId":"SN-1001","recordName":"First idea","details":""}`)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil), http.StatusOK, emptyBinBody)
	page := getLog(t, priya, "?action=restored")
	if got := []string{page.Items[1].RecordName, page.Items[0].RecordName}; page.Total != 2 || !slices.Equal(got, []string{"Ideas (restored)", "First idea"}) {
		t.Errorf("restored entries = %d %v, want the folder, then the note", page.Total, got)
	}
}

func TestRestore_ParentCodes_409(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	server.GrantRole(t, priyasEmail, "administrator", "all") // 014: deleting permanently is an administrator's
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)
	parent := `"parent":{"type":"sampleFolder","id":"SF-1001","name":"Ideas"}`

	first := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/restore", nil)
	if purge := priya.Do(t, http.MethodDelete, "/api/v1/recycle-bin/BIN-1002", nil); purge.Status != http.StatusOK {
		t.Fatalf("deleting BIN-1002 permanently: %d %s", purge.Status, purge.Body)
	}
	gone := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/restore?withParents=true", nil)

	assertAnswer(t, first, http.StatusConflict, `{"error":{"code":"restore_parent_first","message":"Restore 'Ideas' first","fields":[],`+parent+`}}`)
	assertAnswer(t, gone, http.StatusConflict, `{"error":{"code":"parent_deleted_permanently","message":"This sample note can't be restored `+
		`because its sample folder 'Ideas' was permanently deleted.","fields":[],`+parent+`}}`)
}

// The entry waiting for the deleted one stays in the bin, no longer restorable.
func TestPurge_Contract(t *testing.T) { //nolint:gocyclo // 11: one purge, then the bin and the new entry checked field by field
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	server.GrantRole(t, priyasEmail, "administrator", "all")
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)

	response := priya.Do(t, http.MethodDelete, "/api/v1/recycle-bin/BIN-1002", nil)

	assertAnswer(t, response, http.StatusOK,
		`{"recordType":"sampleFolder","recordTypeLabel":"sample folder","recordId":"SF-1001","recordName":"Ideas"}`)
	var bin history.BinPage
	priya.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil).Decode(t, &bin)
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

func TestPurge_404_409(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	server.GrantRole(t, priyasEmail, "administrator", "all")
	priya := server.SignIn(t, priyasEmail)
	createFolder(t, priya, "Ideas")
	deleteFolder(t, priya, "SF-1001")
	if response := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/restore", nil); response.Status != http.StatusOK {
		t.Fatalf("restoring BIN-1001: %d %s", response.Status, response.Body)
	}

	assertAnswer(t, priya.Do(t, http.MethodDelete, "/api/v1/recycle-bin/BIN-9999", nil), http.StatusNotFound, unknownEntryBody)
	assertAnswer(t, priya.Do(t, http.MethodDelete, "/api/v1/recycle-bin/BIN-1001", nil), http.StatusConflict,
		`{"error":{"code":"already_resolved","message":"'Ideas' is no longer in the recycle bin.","fields":[],`+
			`"resolution":{"how":"restored","by":{"id":"U-1002","name":"Priya Raman"},"at":"2026-03-12T09:00:00Z"}}}`)
}

// When nothing listed is left in the bin, emptying answers 0 and writes nothing.
func TestEmptyRecycleBin_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	server.GrantRole(t, priyasEmail, "administrator", "all")
	priya := server.SignIn(t, priyasEmail)
	binWithParent(t, priya)
	body := map[string][]string{"entryIds": {"BIN-1002", "BIN-1001"}}

	response := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/empty", body)
	again := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/empty", body)

	assertAnswer(t, response, http.StatusOK, `{"count":2}`)
	assertAnswer(t, again, http.StatusOK, `{"count":0}`)
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil), http.StatusOK, emptyBinBody)
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
