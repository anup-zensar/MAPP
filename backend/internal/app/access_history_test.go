package app_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/apitest"
)

// binItem is a recycle-bin entry as the recycle bin page lists it, with its access object.
type binItem struct {
	ID, RecordName string
	Access         *struct{ Restore, DeletePermanently availability }
}

// binOf returns the recycle bin as the person of client sees it.
func binOf(t *testing.T, client *apitest.Client) []binItem {
	t.Helper()
	var page struct{ Items []binItem }
	client.Do(t, http.MethodGet, "/api/recycle-bin", nil).Decode(t, &page)
	return page.Items
}

// Each entry of the recycle bin says what the person may do with it, with the refusal as the reason, and the
// restore preview follows the same rule (FR-011, D-238). The demo's deleted initiative "Trade shows" is in
// FY2026 Lubricants Europe: Priya (Planner) may restore it but not delete it permanently; Alex may do both;
// Sara (Viewer) sees it but may not restore it; Mei (Mobility China) does not see it, and its preview
// refuses her with message 4.
func TestRecycleBin_EntryAccess_FR011_D238(t *testing.T) {
	server := apitest.New(t)
	alex, priya, sara, mei := server.SignIn(t, alexsEmail), server.SignIn(t, priyasEmail), server.SignIn(t, sarasEmail),
		server.SignIn(t, meisEmail)

	priyas := binOf(t, priya)
	if len(priyas) != 1 || priyas[0].RecordName != "Trade shows" || priyas[0].Access == nil {
		t.Fatalf("Priya's recycle bin = %+v, want Trade shows with its access", priyas)
	}
	entry := priyas[0]
	purgeReason := "isn't part of your role (Planner). Ask an administrator if you need it."
	if !entry.Access.Restore.Available || entry.Access.DeletePermanently.Available ||
		!strings.HasSuffix(entry.Access.DeletePermanently.Reason, purgeReason) {
		t.Errorf("Priya's access = %+v, want restore only", *entry.Access)
	}

	alexs := binOf(t, alex)
	if len(alexs) != 1 || alexs[0].Access == nil || !alexs[0].Access.Restore.Available || !alexs[0].Access.DeletePermanently.Available {
		t.Errorf("Alex's recycle bin = %+v, want Trade shows with both actions", alexs)
	}
	saras := binOf(t, sara)
	viewOnly := "Deleting campaigns and initiatives isn't part of your role (Viewer). Ask an administrator if you need it."
	if len(saras) != 1 || saras[0].Access == nil || saras[0].Access.Restore.Available || saras[0].Access.Restore.Reason != viewOnly {
		t.Errorf("Sara's recycle bin = %+v, want Trade shows without restore, reason %q", saras, viewOnly)
	}
	if meis := binOf(t, mei); len(meis) != 0 {
		t.Errorf("Mei's recycle bin = %+v, want empty", meis)
	}

	preview := "/api/recycle-bin/" + entry.ID + "/restore-preview"
	refused(t, "Mei previewing its restore", mei.Do(t, http.MethodGet, preview, nil),
		"You don't have access to this initiative. Ask an administrator if you need it.")
	refused(t, "Sara previewing its restore", sara.Do(t, http.MethodGet, preview, nil), viewOnly)
	if answer := priya.Do(t, http.MethodGet, preview, nil); answer.Status != http.StatusOK {
		t.Errorf("Priya previewing its restore = %d %s, want 200", answer.Status, answer.Body)
	}
	refused(t, "Priya deleting it permanently", priya.Do(t, http.MethodDelete, "/api/recycle-bin/"+entry.ID, nil),
		"Deleting permanently isn't part of your role (Planner). Ask an administrator if you need it.")
}

// The activity log's summary and filter choices cover only the entries the person may see (FR-007,
// D-238): Mei's name only her market's plans and fewer entries than Alex's; Alex's name every plan of the
// demo history.
func TestLogSummary_OnlyWhatThePersonSees_FR007(t *testing.T) {
	server := apitest.New(t)
	alex, mei := server.SignIn(t, alexsEmail), server.SignIn(t, meisEmail)
	const path = "/api/activity-log/summary?todayStart=2026-03-13T00:00:00Z&weekStart=2026-03-09T00:00:00Z"
	type summary struct {
		Total int
		Plans []struct{ ID string }
	}
	planIDs := func(s summary) []string {
		ids := []string{}
		for _, p := range s.Plans {
			ids = append(ids, p.ID)
		}
		return ids
	}

	var alexs, meis summary
	alex.Do(t, http.MethodGet, path, nil).Decode(t, &alexs)
	mei.Do(t, http.MethodGet, path, nil).Decode(t, &meis)

	if !slices.Contains(planIDs(alexs), "P-1001") || meis.Total >= alexs.Total || alexs.Total == 0 {
		t.Errorf("Alex's summary = %+v, Mei's = %+v; want Mei's smaller", alexs, meis)
	}
	if slices.Contains(planIDs(meis), "P-1001") {
		t.Errorf("Mei's summary names the Lubricants plan: %q", planIDs(meis))
	}
	var log struct{ Total, UnfilteredTotal int }
	mei.Do(t, http.MethodGet, "/api/activity-log?limit=50", nil).Decode(t, &log)
	if log.UnfilteredTotal != meis.Total || log.Total != meis.Total {
		t.Errorf("Mei's log = %+v, want both totals %d (her summary's)", log, meis.Total)
	}
}
