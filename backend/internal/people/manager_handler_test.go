package people_test

import (
	"net/http"
	"testing"

	"northstar/internal/people"
	"northstar/internal/platform/apitest"
)

// The line manager through the API (017 FR-016): a new person is created with one, the list shows each
// person's manager and how many people report to them, the details count "Line manager of <n> people", and
// a change that would make a loop answers 400 on managerId. Requests and answers follow the contract.
func TestPeople_ManagerField_FR016(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	const daniel = "U-1003"
	before := listPeople(t, alex)

	jane := createPerson(t, alex, map[string]any{"name": "Jane Doe", "email": janesEmail, "managerId": daniel})
	after := listPeople(t, alex)
	var danielsDetails people.PersonDetails
	alex.Do(t, http.MethodGet, "/api/people/"+daniel, nil).Decode(t, &danielsDetails)
	loop := alex.Do(t, http.MethodPatch, "/api/people/"+daniel, map[string]any{"managerId": jane.ID, "version": danielsDetails.Version})
	cleared := alex.Do(t, http.MethodPatch, "/api/people/"+jane.ID, map[string]any{"managerId": "", "version": jane.Version})

	if m := after[jane.ID].Manager; m == nil || m.ID != daniel || m.Name != "Daniel Okoro" {
		t.Errorf("Jane's manager = %+v, want Daniel Okoro", m)
	}
	if got, want := after[daniel].ReportCount, before[daniel].ReportCount+1; got != want {
		t.Errorf("Daniel's report count = %d, want %d", got, want)
	}
	last := danielsDetails.Usage[len(danielsDetails.Usage)-1]
	if last.Role != "manager" || last.RecordType != "person" || last.Count != after[daniel].ReportCount {
		t.Errorf("Daniel's last usage = %+v, want line manager of %d people", last, after[daniel].ReportCount)
	}
	assertAnswer(t, loop, http.StatusBadRequest, validationBody(
		`{"field":"managerId","message":"Jane Doe reports to Daniel Okoro, so Daniel Okoro can't report to Jane Doe."}`))
	if cleared.Status != http.StatusOK || listPeople(t, alex)[jane.ID].Manager != nil {
		t.Errorf("clearing Jane's manager = %d %s, want 200 and no manager", cleared.Status, cleared.Body)
	}
}

// listPeople reads GET /api/people as client, by person id.
func listPeople(t *testing.T, client *apitest.Client) map[string]people.PersonListItem {
	t.Helper()
	var page people.PersonPage
	response := client.Do(t, http.MethodGet, "/api/people", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/people = %d %s", response.Status, response.Body)
	}
	response.Decode(t, &page)
	byID := map[string]people.PersonListItem{}
	for _, p := range page.Items {
		byID[p.ID] = p
	}
	return byID
}
