package overspend_test

import (
	"net/http"
	"testing"

	"mapp/internal/activity"
	"mapp/internal/history"
	"mapp/internal/platform/apitest"
)

const (
	events       = "I-1003" // Rimula fleet, outside Helix launch
	helixRefused = "campaign 'Helix launch'"
)

// rimulaActivity is an activity of 500.00 USD outside Helix launch, saved before Helix refuses overspending.
func rimulaActivity(t *testing.T, c *apitest.Client, name string) activity.View {
	t.Helper()
	return created(t, createActivity(t, c, events, newActivity(name, "500.00")))
}

func TestRefuse_ActivityMoveCopyAndSplit_Contract_037_US2(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	moving, copying, splitting := rimulaActivity(t, priya, "Moving"), rimulaActivity(t, priya, "Copying"),
		rimulaActivity(t, priya, "Splitting")
	refuseHelix(t, priya, "0.00")

	moved := priya.Do(t, http.MethodPost, "/api/v1/activities/"+moving.ID+"/move",
		map[string]any{"initiativeId": digital, "version": moving.Version})
	copied := priya.Do(t, http.MethodPost, "/api/v1/activities/"+copying.ID+"/copy", map[string]any{"initiativeId": digital})
	split := priya.Do(t, http.MethodPost, "/api/v1/activities/"+splitting.ID+"/split", map[string]any{
		"name": "Split part", "initiativeId": digital, "lineItemIds": []string{}, "plannedCost": "100.00",
		"restartApproval": true, "version": splitting.Version})
	within := priya.Do(t, http.MethodPost, "/api/v1/activities/"+moving.ID+"/copy", map[string]any{"initiativeId": events})

	for name, r := range map[string]*apitest.Response{"037 FR-009 move": moved, "copy": copied, "split": split} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			refusedNaming(t, r, helixRefused)
		})
	}
	decoded[activity.View](t, within, http.StatusCreated)
	if again := decoded[activity.View](t, priya.Do(t, http.MethodGet, "/api/v1/activities/"+moving.ID, nil),
		http.StatusOK); again.Initiative.ID != events || again.Version != moving.Version {
		t.Errorf("037 FR-009 a refused move changes nothing: %+v", again.Initiative)
	}
}

func TestRefuse_InitiativeMoveAndCopy_Contract_037_US2(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	rimulaActivity(t, priya, "Cost in events")
	refuseHelix(t, priya, "0.00")

	moved := priya.Do(t, http.MethodPost, "/api/v1/initiatives/"+events+"/move", map[string]any{"campaignId": helix, "version": 1})
	copied := priya.Do(t, http.MethodPost, "/api/v1/initiatives/"+events+"/copy",
		map[string]any{"campaignId": helix, "name": "Events again"})

	refusedNaming(t, moved, helixRefused)
	refusedNaming(t, copied, helixRefused)
}

func TestRefuse_RestoreAndUndoAddBackCost_DeleteAlwaysPasses_Contract_037_US2(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	first := created(t, createActivity(t, priya, digital, newActivity("Deleted first", "500.00")))
	second := created(t, createActivity(t, priya, digital, newActivity("Deleted second", "500.00")))
	undoEntry := decoded[history.DeleteResult](t, priya.Do(t, http.MethodDelete, "/api/v1/activities/"+first.ID, nil),
		http.StatusOK).RecycleBinEntry
	restoreEntry := decoded[history.DeleteResult](t, priya.Do(t, http.MethodDelete, "/api/v1/activities/"+second.ID, nil),
		http.StatusOK).RecycleBinEntry
	refuseHelix(t, priya, "0.00")

	undone := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+undoEntry.ID+"/undo", nil)
	restored := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+restoreEntry.ID+"/restore", nil)
	deleted := priya.Do(t, http.MethodDelete, "/api/v1/activities/A-1001", nil)

	refusedNaming(t, undone, helixRefused)
	refusedNaming(t, restored, helixRefused)
	if deleted.Status != http.StatusOK {
		t.Errorf("037 FR-011 deleting cost under a Refuse budget = %d %s", deleted.Status, deleted.Body)
	}
}

func TestRefuse_LineItemsAreNeverRefused_Contract_037_FR012(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	a := created(t, createActivity(t, priya, digital, newActivity("With line items", "100.00")))
	refuseHelix(t, priya, "0.00")

	r := priya.Do(t, http.MethodPost, "/api/v1/activities/"+a.ID+"/line-items", map[string]any{
		"name": "Much more than planned", "ownerId": nil, "description": "", "currency": "USD", "plannedCost": "90000.00",
		"spread": map[string]any{"method": "evenly", "months": []any{}}, "attributes": map[string]any{},
		"contractedAmount": nil, "requestSwbs": false, "financeBudgetYear": nil, "financeBudgetItemId": nil})

	if r.Status != http.StatusCreated {
		t.Errorf("037 FR-012 line item = %d %s, want it saved", r.Status, r.Body)
	}
}
