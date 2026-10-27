package settings_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/apitest"
)

// The activity types are listed by name with the models allowing them; one is read with its default
// planned cost; a new one answers 201, allowed in the models ticked, which list it from then on; every
// problem of a new one comes at once (FR-013, US5-1).
func TestActivityTypesAPI_ListCreateGet_FR013(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	var list struct {
		Items []struct{ Name string } `json:"items"`
	}
	alex.Do(t, http.MethodGet, "/api/v1/activity-types", nil).Decode(t, &list)
	var event struct {
		Name               string
		DefaultPlannedCost struct{ Amount, Currency string }
		Models             []struct{ Name string }
	}

	alex.Do(t, http.MethodGet, "/api/v1/activity-types/AT-1002", nil).Decode(t, &event)
	gone := alex.Do(t, http.MethodGet, "/api/v1/activity-types/AT-9999", nil)
	created := alex.Do(t, http.MethodPost, "/api/v1/activity-types", webinarBody("MOD-1001"))
	got := alex.Do(t, http.MethodGet, "/api/v1/activity-types/AT-1007", nil)
	invalid := alex.Do(t, http.MethodPost, "/api/v1/activity-types", map[string]any{
		"name": " event ", "defaultPlannedCost": map[string]string{"amount": "-5", "currency": "USD"}, "modelIds": []string{"MOD-9999"},
	})
	var lubricants struct{ ActivityTypes []struct{ Name string } }
	alex.Do(t, http.MethodGet, "/api/v1/models/MOD-1001", nil).Decode(t, &lubricants)

	var names []string
	for _, item := range list.Items {
		names = append(names, item.Name)
	}
	want := []string{"Agency fees – Creative", "Content production", "Digital media", "Event", "MOBILITY – Long-term contract", "Sponsorship"}
	if !slices.Equal(names, want) {
		t.Errorf("activity types = %q, want %q", names, want)
	}
	if event.Name != "Event" || event.DefaultPlannedCost.Amount != "25000.00" || len(event.Models) != 2 || event.Models[0].Name != "Lubricants 1.0" {
		t.Errorf("Event = %+v", event)
	}
	assertAnswer(t, gone, http.StatusNotFound, activityTypeGoneBody)
	assertAnswer(t, created, http.StatusCreated, webinarAnswer(1, lubricantsRef, allowedInOneModel))
	assertAnswer(t, got, http.StatusOK, webinarAnswer(1, lubricantsRef, allowedInOneModel))
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(
		`{"field":"name","message":"An activity type called 'Event' already exists."},`+
			`{"field":"defaultPlannedCost","message":"Enter an amount of 0 or more, with at most two decimals."},`+
			`{"field":"modelIds","message":"A model you ticked no longer exists — reload the page and try again."}`))
	if len(lubricants.ActivityTypes) != 6 || lubricants.ActivityTypes[5].Name != "Webinar" {
		t.Errorf("Lubricants 1.0 allows %+v, want Webinar last by name", lubricants.ActivityTypes)
	}
}

// Bodies of the activity-type tests.
const (
	activityTypeGoneBody = `{"error":{"code":"not_found","message":"This activity type no longer exists — someone may have deleted it.","fields":[]}}`
	lubricantsRef        = `{"id":"MOD-1001","name":"Lubricants 1.0","active":true}`
	allowedInOneModel    = `{"inUse":true,"summary":"Allowed in 1 model","parts":[{"kind":"model","text":"Allowed in 1 model","count":1,"inRecycleBin":0}],"records":[]}`
)

// webinarBody is the new activity type of US5-1: "Webinar", default planned cost 5,000 USD, allowed in the
// models.
func webinarBody(modelIDs ...string) map[string]any {
	if modelIDs == nil {
		modelIDs = []string{}
	}
	return map[string]any{"name": "Webinar", "defaultPlannedCost": map[string]string{"amount": "5000", "currency": "USD"}, "modelIds": modelIDs}
}

// webinarAnswer is the answer for the activity type webinarBody (AT-1007) at version, with the models
// allowing it (JSON objects, comma separated) and its usage, created by Alex.
func webinarAnswer(version int, models, usage string) string {
	return `{"id":"AT-1007","name":"Webinar","defaultPlannedCost":{"amount":"5000.00","currency":"USD"},"models":[` + models +
		`],"active":true,"usage":` + usage + `,"version":` + itoa(version) + `,` + alexAtStart + `}`
}

// A save from an out-of-date copy is refused with who changed the activity type and when (FR-007).
func TestActivityTypesAPI_UpdateConflict_FR007(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	daniel := server.SignIn(t, danielsEmail)
	server.GrantRole(t, danielsEmail, "administrator", "all") // 014: changing Settings
	mustCreate(t, alex, "/api/v1/activity-types", webinarBody())
	server.Advance(90 * time.Minute)
	daniels := daniel.Do(t, http.MethodPatch, "/api/v1/activity-types/AT-1007", map[string]any{"defaultPlannedCost": nil, "version": 1})

	alexs := alex.Do(t, http.MethodPatch, "/api/v1/activity-types/AT-1007", map[string]any{"name": "Webinars", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/v1/activity-types/AT-9999", map[string]any{"name": "X", "version": 1})

	assertAnswer(t, daniels, http.StatusOK, `{"id":"AT-1007","name":"Webinar","defaultPlannedCost":null,"models":[],"active":true,"usage":`+
		unusedBody+`,"version":2,"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1001","name":"Alex Morgan"},`+
		`"updatedAt":"2026-03-12T10:30:00Z","updatedBy":{"id":"U-1003","name":"Daniel Okoro"}}`)
	assertAnswer(t, alexs, http.StatusConflict, `{"error":{"code":"version_conflict","message":"Not saved — Daniel Okoro changed this activity type",`+
		`"fields":[],"conflict":{"changedBy":"Daniel Okoro","changedAt":"2026-03-12T10:30:00Z"}}}`)
	assertAnswer(t, gone, http.StatusNotFound, activityTypeGoneBody)
}

// The impact check validates like the update and answers, per model the type is removed from, the
// activities of its plans that keep the type; it saves nothing (FR-012, D-044).
func TestActivityTypesAPI_Impact_FR012(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/v1/activity-types", webinarBody("MOD-1001"))
	var event struct {
		Warnings []struct {
			Kind, Message string
			Count         int
		}
	}

	unused := alex.Do(t, http.MethodPost, "/api/v1/activity-types/AT-1007/impact", map[string]any{"modelIds": []string{}, "version": 1})
	alex.Do(t, http.MethodPost, "/api/v1/activity-types/AT-1002/impact", map[string]any{"modelIds": []string{"MOD-1002"}, "version": 1}).Decode(t, &event)
	invalid := alex.Do(t, http.MethodPost, "/api/v1/activity-types/AT-1007/impact", map[string]any{"modelIds": []string{"MOD-9999"}, "version": 1})
	gone := alex.Do(t, http.MethodPost, "/api/v1/activity-types/AT-9999/impact", map[string]any{"version": 1})

	assertAnswer(t, unused, http.StatusOK, `{"warnings":[]}`)
	if len(event.Warnings) != 1 || event.Warnings[0].Kind != "activityTypeRemoved" || event.Warnings[0].Count < 1 ||
		!strings.HasPrefix(event.Warnings[0].Message, "In plans using 'Lubricants 1.0', ") {
		t.Errorf("removing Event from Lubricants 1.0 = %+v", event)
	}
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"modelIds","message":"A model you ticked no longer exists — reload the page and try again."}`))
	assertAnswer(t, gone, http.StatusNotFound, activityTypeGoneBody)
	assertAnswer(t, alex.Do(t, http.MethodGet, "/api/v1/activity-types/AT-1007", nil), http.StatusOK,
		webinarAnswer(1, lubricantsRef, allowedInOneModel))
}

// Set inactive and Reactivate answer the activity type with its new status; an out-of-date version is
// refused (FR-004).
func TestActivityTypesAPI_Status_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/v1/activity-types", webinarBody())

	inactive := alex.Do(t, http.MethodPost, "/api/v1/activity-types/AT-1007/status", map[string]any{"active": false, "version": 1})
	stale := alex.Do(t, http.MethodPost, "/api/v1/activity-types/AT-1007/status", map[string]any{"active": true, "version": 1})
	active := alex.Do(t, http.MethodPost, "/api/v1/activity-types/AT-1007/status", map[string]any{"active": true, "version": 2})

	assertAnswer(t, inactive, http.StatusOK, replace(webinarAnswer(2, "", unusedBody), `"active":true`, `"active":false`))
	if stale.Status != http.StatusConflict || active.Status != http.StatusOK {
		t.Errorf("stale %d %s; reactivate %d %s", stale.Status, stale.Body, active.Status, active.Body)
	}
}

// An unused activity type moves to the recycle bin and Undo brings it back; a type in use is refused with
// in_use (FR-004, US5-5).
func TestActivityTypesAPI_DeleteAndUndo_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutDemoHistory())
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/v1/activity-types", webinarBody())

	deleted := alex.Do(t, http.MethodDelete, "/api/v1/activity-types/AT-1007", nil)
	afterDelete := alex.Do(t, http.MethodGet, "/api/v1/activity-types/AT-1007", nil)
	undone := alex.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/undo", nil)
	inUse := alex.Do(t, http.MethodDelete, "/api/v1/activity-types/AT-1002", nil)
	gone := alex.Do(t, http.MethodDelete, "/api/v1/activity-types/AT-9999", nil)
	var refused struct {
		Error struct{ Code, Message string }
	}
	inUse.Decode(t, &refused)

	assertAnswer(t, deleted, http.StatusOK, `{"recycleBinEntry":{"id":"BIN-1001","recordType":"activityType","recordTypeLabel":"activity type",`+
		`"recordId":"AT-1007","recordName":"Webinar","path":[],"parent":null,"deletedAt":"2026-03-12T09:00:00Z",`+
		`"deletedBy":{"id":"U-1001","name":"Alex Morgan"},"contents":[]}}`)
	assertAnswer(t, afterDelete, http.StatusNotFound, activityTypeGoneBody)
	assertAnswer(t, undone, http.StatusOK, `{"recordType":"activityType","recordTypeLabel":"activity type","recordId":"AT-1007",`+
		`"recordName":"Webinar","details":""}`)
	assertAnswer(t, alex.Do(t, http.MethodGet, "/api/v1/activity-types/AT-1007", nil), http.StatusOK, webinarAnswer(1, "", unusedBody))
	if inUse.Status != http.StatusConflict || refused.Error.Code != "in_use" || !strings.HasPrefix(refused.Error.Message, "'Event' is in use (used by ") ||
		!strings.HasSuffix(refused.Error.Message, " · Allowed in 2 models) — set it inactive instead.") {
		t.Errorf("deleting Event = %d %s", inUse.Status, inUse.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, activityTypeGoneBody)
}
