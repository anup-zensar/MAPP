package financials_test

import (
	"bytes"
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

const meisEmail = "mei.lin@shell.com"

func sameBody(t *testing.T, probe string, hidden, missing *apitest.Response) {
	t.Helper()
	if hidden.Status != missing.Status || !bytes.Equal(hidden.Body, missing.Body) {
		t.Errorf("%s: hidden %d %s\nmissing %d %s", probe, hidden.Status, hidden.Body, missing.Status, missing.Body)
	}
}

// Mei, a planner without import, gets the answers of a missing transaction for a not-linked one and one of
// another market; with a role that links in her teams, a line item of another market answers as missing.
func TestLinkScopeParity_038_FR007_H5(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex, mei := server.SignIn(t, alexsEmail), server.SignIn(t, meisEmail)
	const base = "/api/v1/financials/transactions/"
	link := func(id, lineItemID string) *apitest.Response {
		return mei.Do(t, http.MethodPost, base+id+"/link", map[string]any{"lineItemId": lineItemID, "version": 1})
	}

	missing := mei.Do(t, http.MethodGet, base+"FT-9999", nil)
	refused(t, missing, http.StatusNotFound, "not_found")
	sameBody(t, "GET not linked", mei.Do(t, http.MethodGet, base+"FT-1004", nil), missing)
	sameBody(t, "GET other market", mei.Do(t, http.MethodGet, base+"FT-1001", nil), missing)
	sameBody(t, "link not linked", link("FT-1004", "L-1007"), link("FT-9999", "L-1007"))
	var page struct{ Counts map[string]int }
	mei.Do(t, http.MethodGet, "/api/v1/financials/transactions", nil).Decode(t, &page)
	if _, shown := page.Counts["notLinked"]; shown {
		t.Errorf("Mei's counts = %v, want no notLinked", page.Counts)
	}

	role := alex.Do(t, http.MethodPost, "/api/v1/roles", map[string]any{"name": "Linker", "defaultScope": "myTeams",
		"permissions": []string{"plans.view", "finance.view", "finance.import", "finance.link"}})
	var created struct{ ID string }
	role.Decode(t, &created)
	given := alex.Do(t, http.MethodPut, "/api/v1/people/U-1004/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1003", "scope": map[string]any{"type": "myTeams"}},
		{"roleId": created.ID, "scope": map[string]any{"type": "myTeams"}},
	}})
	if given.Status != http.StatusOK {
		t.Fatalf("giving Mei the Linker role = %d %s", given.Status, given.Body)
	}

	otherMarket, unknown := link("FT-1004", "L-1003"), link("FT-1004", "L-9999")
	fieldRefused(t, otherMarket, "lineItemId", "No line item L-1003.")
	fieldRefused(t, unknown, "lineItemId", "No line item L-9999.")
	unlink := func(id string) *apitest.Response {
		return mei.Do(t, http.MethodPost, base+id+"/unlink", map[string]any{"version": 1})
	}
	sameBody(t, "unlink other market", unlink("FT-1001"), unlink("FT-9999"))
}

// Without "Link financial transactions", a transaction outside the person's scope answers exactly as a missing
// one and one inside it is refused for permission; with it, one outside still answers as missing (038 FR-007, H-5).
func TestLinkWithoutPermission_ScopeBeforePermission_038_FR007_H5(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex, mei, priya := server.SignIn(t, alexsEmail), server.SignIn(t, meisEmail), server.SignIn(t, priyasEmail)
	post := func(who *apitest.Client, id, verb string) *apitest.Response {
		body := map[string]any{"version": 1}
		if verb == "link" {
			body["lineItemId"] = "L-1007"
		}
		return who.Do(t, http.MethodPost, "/api/v1/financials/transactions/"+id+"/"+verb, body)
	}

	for _, verb := range []string{"link", "unlink"} {
		missing := post(mei, "FT-9999", verb)
		refused(t, missing, http.StatusNotFound, "not_found")
		sameBody(t, verb+" other market", post(mei, "FT-1001", verb), missing)
		refused(t, post(priya, "FT-1001", verb), http.StatusForbidden, "forbidden")
	}

	role := alex.Do(t, http.MethodPost, "/api/v1/roles", map[string]any{"name": "Linker", "defaultScope": "myTeams",
		"permissions": []string{"plans.view", "finance.view", "finance.import", "finance.link"}})
	var created struct{ ID string }
	role.Decode(t, &created)
	given := alex.Do(t, http.MethodPut, "/api/v1/people/U-1004/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1003", "scope": map[string]any{"type": "myTeams"}},
		{"roleId": created.ID, "scope": map[string]any{"type": "myTeams"}},
	}})
	if given.Status != http.StatusOK {
		t.Fatalf("giving Mei the Linker role = %d %s", given.Status, given.Body)
	}
	missing := post(mei, "FT-9999", "link")
	refused(t, missing, http.StatusNotFound, "not_found")
	sameBody(t, "link other market with link", post(mei, "FT-1001", "link"), missing)
}
