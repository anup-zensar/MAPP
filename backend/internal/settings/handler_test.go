package settings_test

import (
	"encoding/base64"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/apitest"
)

// The handler tests run the real API on the demo data (002 data model §13) and check every request and
// answer against the contract. Seeded Settings may be used by the seeded plans once features 004–007
// report their records, so the tests assert the usage of records they create themselves only.

const (
	alexsEmail   = "alex.morgan@shell.com"
	danielsEmail = "daniel.okoro@shell.com"
)

// Bodies that several tests expect.
const (
	modelGoneBody = `{"code":"not_found","message":"This model no longer exists — someone may have deleted it.","fields":[]}`
	unusedBody    = `{"inUse":false,"summary":"Not used yet","parts":[],"records":[]}`
	alexAtStart   = `"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1001","name":"Alex Morgan"},` +
		`"updatedAt":"2026-03-12T09:00:00Z","updatedBy":{"id":"U-1001","name":"Alex Morgan"}`
)

// The Settings page counts the active records of each section of the demo data (FR-001).
func TestOverview_API_FR001(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	response := alex.Do(t, http.MethodGet, "/api/settings/overview", nil)

	assertAnswer(t, response, http.StatusOK, `{"models":2,"activityTypes":6,"shellAttributes":35,"currencies":5,"financeBudgetItems":6}`)
}

// A new model answers 201 with its values, its allowed types by name and its usage; the list has it in name
// order; a model that is gone answers not found; every problem of a new model comes at once (FR-009).
func TestModelsAPI_ListCreateGet_FR009(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	created := alex.Do(t, http.MethodPost, "/api/models", lubricants2)
	invalid := alex.Do(t, http.MethodPost, "/api/models", map[string]any{
		"name": " ", "inqToMqlRate": "0", "mqlToDealRate": "5", "averageDealSize": nil, "activityTypeIds": []string{},
	})
	got := alex.Do(t, http.MethodGet, "/api/models/MOD-1003", nil)
	gone := alex.Do(t, http.MethodGet, "/api/models/MOD-9999", nil)
	var list struct {
		Items []struct{ Name string } `json:"items"`
		Total int                     `json:"total"`
	}
	alex.Do(t, http.MethodGet, "/api/models", nil).Decode(t, &list)

	assertAnswer(t, created, http.StatusCreated, lubricants2Body(1, alexAtStart))
	assertAnswer(t, got, http.StatusOK, lubricants2Body(1, alexAtStart))
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(
		`{"field":"name","message":"Enter a name."},`+
			`{"field":"inqToMqlRate","message":"Enter a rate above 0 % and at most 100 %, with at most two decimals."},`+
			`{"field":"averageDealSize","message":"Enter an average deal size above 0 USD, with at most two decimals."},`+
			`{"field":"activityTypeIds","message":"Tick at least one activity type."}`))
	assertAnswer(t, gone, http.StatusNotFound, modelGoneBody)
	var names []string
	for _, m := range list.Items {
		names = append(names, m.Name)
	}
	if !slices.Equal(names, []string{"Lubricants 1.0", "Lubricants 2.0", "Mobility 1.0"}) || list.Total != 3 {
		t.Errorf("models = %q (total %d)", names, list.Total)
	}
}

// lubricants2 is a new model allowing Event (AT-1002) and Digital media (AT-1003).
var lubricants2 = map[string]any{
	"name": "Lubricants 2.0", "inqToMqlRate": "2", "mqlToDealRate": "5",
	"averageDealSize": map[string]string{"amount": "10000", "currency": "USD"}, "activityTypeIds": []string{"AT-1002", "AT-1003"},
}

// lubricants2Body is the answer for the model lubricants2 at version, with its bookkeeping meta.
func lubricants2Body(version int, meta string) string {
	return `{"id":"MOD-1003","name":"Lubricants 2.0","inqToMqlRate":"2.00","mqlToDealRate":"5.00",` +
		`"averageDealSize":{"amount":"10000.00","currency":"USD"},"activityTypes":[` +
		`{"id":"AT-1003","name":"Digital media","active":true},{"id":"AT-1002","name":"Event","active":true}],` +
		`"active":true,"usage":` + unusedBody + `,"version":` + itoa(version) + `,` + meta + `}`
}

// A save from an out-of-date copy is refused with who changed the model and when (FR-007).
func TestModelsAPI_UpdateConflict_FR007(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	daniel := server.SignIn(t, danielsEmail)
	server.GrantRole(t, danielsEmail, "administrator", "all") // 014: changing Settings
	mustCreate(t, alex, "/api/models", lubricants2)
	server.Advance(90 * time.Minute)
	daniels := daniel.Do(t, http.MethodPatch, "/api/models/MOD-1003", map[string]any{"mqlToDealRate": "6", "version": 1})

	alexs := alex.Do(t, http.MethodPatch, "/api/models/MOD-1003", map[string]any{"name": "Alex's model", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/models/MOD-9999", map[string]any{"name": "X", "version": 1})

	if daniels.Status != http.StatusOK {
		t.Fatalf("Daniel's update = %d %s", daniels.Status, daniels.Body)
	}
	assertAnswer(t, alexs, http.StatusConflict, `{"code":"version_conflict","message":"Not saved — Daniel Okoro changed this model",`+
		`"fields":[],"conflict":{"changedBy":"Daniel Okoro","changedAt":"2026-03-12T10:30:00Z"}}`)
	assertAnswer(t, gone, http.StatusNotFound, modelGoneBody)
}

// Everyone reads Settings, and a person whose role allows "Change Settings lists" changes them (FR-008 as
// feature 014 rewrote it, D-240): Daniel, a Planner and Approver, is refused with 403 and nothing changes;
// given the Administrator role, he edits Alex's model.
func TestSettings_ChangedWithSettingsManage_FR008(t *testing.T) {
	server := apitest.New(t)
	mustCreate(t, server.SignIn(t, alexsEmail), "/api/models", lubricants2)
	daniel := server.SignIn(t, danielsEmail)
	server.Advance(time.Hour)
	change := map[string]any{"name": "Lubricants 2.0", "inqToMqlRate": "2.5", "version": 1}

	refused := daniel.Do(t, http.MethodPatch, "/api/models/MOD-1003", change)
	read := daniel.Do(t, http.MethodGet, "/api/models/MOD-1003", nil)
	server.GrantRole(t, danielsEmail, "administrator", "all")
	response := daniel.Do(t, http.MethodPatch, "/api/models/MOD-1003", change)

	assertAnswer(t, refused, http.StatusForbidden, `{"code":"forbidden","message":"Changing Settings isn't part of your roles `+
		`(Approver, Planner). Ask an administrator if you need it.","fields":[]}`)
	if read.Status != http.StatusOK {
		t.Errorf("Daniel reading the model = %d %s, want 200", read.Status, read.Body)
	}

	body := lubricants2Body(2, `"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1001","name":"Alex Morgan"},`+
		`"updatedAt":"2026-03-12T10:00:00Z","updatedBy":{"id":"U-1003","name":"Daniel Okoro"}`)
	assertAnswer(t, response, http.StatusOK, replace(body, `"inqToMqlRate":"2.00"`, `"inqToMqlRate":"2.50"`))
}

// The impact check validates like the update and answers the warnings, saving nothing (FR-011, D-044).
func TestModelsAPI_Impact_FR011(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/models", lubricants2)

	unused := alex.Do(t, http.MethodPost, "/api/models/MOD-1003/impact", map[string]any{"inqToMqlRate": "3", "version": 1})
	invalid := alex.Do(t, http.MethodPost, "/api/models/MOD-1003/impact", map[string]any{"activityTypeIds": []string{}, "version": 1})
	gone := alex.Do(t, http.MethodPost, "/api/models/MOD-9999/impact", map[string]any{"version": 1})

	assertAnswer(t, unused, http.StatusOK, `{"warnings":[]}`)
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"activityTypeIds","message":"Tick at least one activity type."}`))
	assertAnswer(t, gone, http.StatusNotFound, modelGoneBody)
	assertAnswer(t, alex.Do(t, http.MethodGet, "/api/models/MOD-1003", nil), http.StatusOK, lubricants2Body(1, alexAtStart))
}

// Set inactive and Reactivate answer the model with its new status; an out-of-date version is refused
// (FR-004).
func TestModelsAPI_Status_FR004(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/models", lubricants2)

	inactive := alex.Do(t, http.MethodPost, "/api/models/MOD-1003/status", map[string]any{"active": false, "version": 1})
	stale := alex.Do(t, http.MethodPost, "/api/models/MOD-1003/status", map[string]any{"active": true, "version": 1})
	active := alex.Do(t, http.MethodPost, "/api/models/MOD-1003/status", map[string]any{"active": true, "version": 2})

	assertAnswer(t, inactive, http.StatusOK, replace(lubricants2Body(2, alexAtStart), `"active":true,"usage"`, `"active":false,"usage"`))
	if stale.Status != http.StatusConflict || active.Status != http.StatusOK {
		t.Errorf("stale %d %s; reactivate %d %s", stale.Status, stale.Body, active.Status, active.Body)
	}
}

// An unused model moves to the recycle bin, which lists it as a model, and Undo brings it back (FR-004,
// FR-006).
func TestModelsAPI_DeleteAndUndo_FR004(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory()) // BIN-1001 is the first entry of an empty recycle bin
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/models", lubricants2)

	deleted := alex.Do(t, http.MethodDelete, "/api/models/MOD-1003", nil)
	afterDelete := alex.Do(t, http.MethodGet, "/api/models/MOD-1003", nil)
	undone := alex.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/undo", nil)
	gone := alex.Do(t, http.MethodDelete, "/api/models/MOD-9999", nil)

	assertAnswer(t, deleted, http.StatusOK, `{"recycleBinEntry":{"id":"BIN-1001","recordType":"model","recordTypeLabel":"model",`+
		`"recordId":"MOD-1003","recordName":"Lubricants 2.0","path":[],"parent":null,"deletedAt":"2026-03-12T09:00:00Z",`+
		`"deletedBy":{"id":"U-1001","name":"Alex Morgan"},"contents":[]}}`)
	assertAnswer(t, afterDelete, http.StatusNotFound, modelGoneBody)
	assertAnswer(t, undone, http.StatusOK, `{"recordType":"model","recordTypeLabel":"model","recordId":"MOD-1003",`+
		`"recordName":"Lubricants 2.0","details":""}`)
	assertAnswer(t, alex.Do(t, http.MethodGet, "/api/models/MOD-1003", nil), http.StatusOK, lubricants2Body(1, alexAtStart))
	assertAnswer(t, gone, http.StatusNotFound, modelGoneBody)
}

// The activity types are listed by name with the models allowing them; one is read with its default
// planned cost; a new one answers 201, allowed in the models ticked, which list it from then on; every
// problem of a new one comes at once (FR-013, US5-1).
func TestActivityTypesAPI_ListCreateGet_FR013(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	var list struct {
		Items []struct{ Name string } `json:"items"`
	}
	alex.Do(t, http.MethodGet, "/api/activity-types", nil).Decode(t, &list)
	var event struct {
		Name               string
		DefaultPlannedCost struct{ Amount, Currency string }
		Models             []struct{ Name string }
	}

	alex.Do(t, http.MethodGet, "/api/activity-types/AT-1002", nil).Decode(t, &event)
	gone := alex.Do(t, http.MethodGet, "/api/activity-types/AT-9999", nil)
	created := alex.Do(t, http.MethodPost, "/api/activity-types", webinarBody("MOD-1001"))
	got := alex.Do(t, http.MethodGet, "/api/activity-types/AT-1007", nil)
	invalid := alex.Do(t, http.MethodPost, "/api/activity-types", map[string]any{
		"name": " event ", "defaultPlannedCost": map[string]string{"amount": "-5", "currency": "USD"}, "modelIds": []string{"MOD-9999"},
	})
	var lubricants struct{ ActivityTypes []struct{ Name string } }
	alex.Do(t, http.MethodGet, "/api/models/MOD-1001", nil).Decode(t, &lubricants)

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
	assertAnswer(t, created, http.StatusCreated, webinarAnswer(1, lubricantsRef, allowedInOneModel, alexAtStart))
	assertAnswer(t, got, http.StatusOK, webinarAnswer(1, lubricantsRef, allowedInOneModel, alexAtStart))
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
	activityTypeGoneBody = `{"code":"not_found","message":"This activity type no longer exists — someone may have deleted it.","fields":[]}`
	lubricantsRef        = `{"id":"MOD-1001","name":"Lubricants 1.0","active":true}`
	mobilityRef          = `{"id":"MOD-1002","name":"Mobility 1.0","active":true}`
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
// allowing it (JSON objects, comma separated), its usage and its bookkeeping meta.
func webinarAnswer(version int, models, usage, meta string) string {
	return `{"id":"AT-1007","name":"Webinar","defaultPlannedCost":{"amount":"5000.00","currency":"USD"},"models":[` + models +
		`],"active":true,"usage":` + usage + `,"version":` + itoa(version) + `,` + meta + `}`
}

// A save from an out-of-date copy is refused with who changed the activity type and when (FR-007).
func TestActivityTypesAPI_UpdateConflict_FR007(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	daniel := server.SignIn(t, danielsEmail)
	server.GrantRole(t, danielsEmail, "administrator", "all") // 014: changing Settings
	mustCreate(t, alex, "/api/activity-types", webinarBody())
	server.Advance(90 * time.Minute)
	daniels := daniel.Do(t, http.MethodPatch, "/api/activity-types/AT-1007", map[string]any{"defaultPlannedCost": nil, "version": 1})

	alexs := alex.Do(t, http.MethodPatch, "/api/activity-types/AT-1007", map[string]any{"name": "Webinars", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/activity-types/AT-9999", map[string]any{"name": "X", "version": 1})

	assertAnswer(t, daniels, http.StatusOK, `{"id":"AT-1007","name":"Webinar","defaultPlannedCost":null,"models":[],"active":true,"usage":`+
		unusedBody+`,"version":2,"createdAt":"2026-03-12T09:00:00Z","createdBy":{"id":"U-1001","name":"Alex Morgan"},`+
		`"updatedAt":"2026-03-12T10:30:00Z","updatedBy":{"id":"U-1003","name":"Daniel Okoro"}}`)
	assertAnswer(t, alexs, http.StatusConflict, `{"code":"version_conflict","message":"Not saved — Daniel Okoro changed this activity type",`+
		`"fields":[],"conflict":{"changedBy":"Daniel Okoro","changedAt":"2026-03-12T10:30:00Z"}}`)
	assertAnswer(t, gone, http.StatusNotFound, activityTypeGoneBody)
}

// The impact check validates like the update and answers, per model the type is removed from, the
// activities of its plans that keep the type; it saves nothing (FR-012, D-044).
func TestActivityTypesAPI_Impact_FR012(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/activity-types", webinarBody("MOD-1001"))
	var event struct {
		Warnings []struct {
			Kind, Message string
			Count         int
		}
	}

	unused := alex.Do(t, http.MethodPost, "/api/activity-types/AT-1007/impact", map[string]any{"modelIds": []string{}, "version": 1})
	alex.Do(t, http.MethodPost, "/api/activity-types/AT-1002/impact", map[string]any{"modelIds": []string{"MOD-1002"}, "version": 1}).Decode(t, &event)
	invalid := alex.Do(t, http.MethodPost, "/api/activity-types/AT-1007/impact", map[string]any{"modelIds": []string{"MOD-9999"}, "version": 1})
	gone := alex.Do(t, http.MethodPost, "/api/activity-types/AT-9999/impact", map[string]any{"version": 1})

	assertAnswer(t, unused, http.StatusOK, `{"warnings":[]}`)
	if len(event.Warnings) != 1 || event.Warnings[0].Kind != "activityTypeRemoved" || event.Warnings[0].Count < 1 ||
		!strings.HasPrefix(event.Warnings[0].Message, "In plans using 'Lubricants 1.0', ") {
		t.Errorf("removing Event from Lubricants 1.0 = %+v", event)
	}
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"modelIds","message":"A model you ticked no longer exists — reload the page and try again."}`))
	assertAnswer(t, gone, http.StatusNotFound, activityTypeGoneBody)
	assertAnswer(t, alex.Do(t, http.MethodGet, "/api/activity-types/AT-1007", nil), http.StatusOK,
		webinarAnswer(1, lubricantsRef, allowedInOneModel, alexAtStart))
}

// Set inactive and Reactivate answer the activity type with its new status; an out-of-date version is
// refused (FR-004).
func TestActivityTypesAPI_Status_FR004(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/activity-types", webinarBody())

	inactive := alex.Do(t, http.MethodPost, "/api/activity-types/AT-1007/status", map[string]any{"active": false, "version": 1})
	stale := alex.Do(t, http.MethodPost, "/api/activity-types/AT-1007/status", map[string]any{"active": true, "version": 1})
	active := alex.Do(t, http.MethodPost, "/api/activity-types/AT-1007/status", map[string]any{"active": true, "version": 2})

	assertAnswer(t, inactive, http.StatusOK, replace(webinarAnswer(2, "", unusedBody, alexAtStart), `"active":true`, `"active":false`))
	if stale.Status != http.StatusConflict || active.Status != http.StatusOK {
		t.Errorf("stale %d %s; reactivate %d %s", stale.Status, stale.Body, active.Status, active.Body)
	}
}

// An unused activity type moves to the recycle bin and Undo brings it back; a type in use is refused with
// in_use (FR-004, US5-5).
func TestActivityTypesAPI_DeleteAndUndo_FR004(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/activity-types", webinarBody())

	deleted := alex.Do(t, http.MethodDelete, "/api/activity-types/AT-1007", nil)
	afterDelete := alex.Do(t, http.MethodGet, "/api/activity-types/AT-1007", nil)
	undone := alex.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/undo", nil)
	inUse := alex.Do(t, http.MethodDelete, "/api/activity-types/AT-1002", nil)
	gone := alex.Do(t, http.MethodDelete, "/api/activity-types/AT-9999", nil)
	var refused struct{ Code, Message string }
	inUse.Decode(t, &refused)

	assertAnswer(t, deleted, http.StatusOK, `{"recycleBinEntry":{"id":"BIN-1001","recordType":"activityType","recordTypeLabel":"activity type",`+
		`"recordId":"AT-1007","recordName":"Webinar","path":[],"parent":null,"deletedAt":"2026-03-12T09:00:00Z",`+
		`"deletedBy":{"id":"U-1001","name":"Alex Morgan"},"contents":[]}}`)
	assertAnswer(t, afterDelete, http.StatusNotFound, activityTypeGoneBody)
	assertAnswer(t, undone, http.StatusOK, `{"recordType":"activityType","recordTypeLabel":"activity type","recordId":"AT-1007",`+
		`"recordName":"Webinar","details":""}`)
	assertAnswer(t, alex.Do(t, http.MethodGet, "/api/activity-types/AT-1007", nil), http.StatusOK, webinarAnswer(1, "", unusedBody, alexAtStart))
	if inUse.Status != http.StatusConflict || refused.Code != "in_use" || !strings.HasPrefix(refused.Message, "'Event' is in use (used by ") ||
		!strings.HasSuffix(refused.Message, " · Allowed in 2 models) — set it inactive instead.") {
		t.Errorf("deleting Event = %d %s", inUse.Status, inUse.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, activityTypeGoneBody)
}

// The models allowing a type are edited from the type: the models left out lose it and the models added
// gain it; a model is never left without a type (FR-013, FR-009).
func TestActivityTypesAPI_ModelsEditedFromType_FR013(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/activity-types", webinarBody("MOD-1001"))

	moved := alex.Do(t, http.MethodPatch, "/api/activity-types/AT-1007", map[string]any{"modelIds": []string{"MOD-1002"}, "version": 1})
	var lubricants, mobility struct{ ActivityTypes []struct{ ID string } }
	alex.Do(t, http.MethodGet, "/api/models/MOD-1001", nil).Decode(t, &lubricants)
	alex.Do(t, http.MethodGet, "/api/models/MOD-1002", nil).Decode(t, &mobility)
	mustCreate(t, alex, "/api/models", map[string]any{"name": "Webinars only", "inqToMqlRate": "2", "mqlToDealRate": "5",
		"averageDealSize": map[string]string{"amount": "10000", "currency": "USD"}, "activityTypeIds": []string{"AT-1007"}})
	lastType := alex.Do(t, http.MethodPatch, "/api/activity-types/AT-1007", map[string]any{"modelIds": []string{}, "version": 1})

	assertAnswer(t, moved, http.StatusOK, webinarAnswer(1, mobilityRef, allowedInOneModel, alexAtStart))
	has := func(types []struct{ ID string }) bool {
		return slices.ContainsFunc(types, func(t struct{ ID string }) bool { return t.ID == "AT-1007" })
	}
	if has(lubricants.ActivityTypes) || !has(mobility.ActivityTypes) {
		t.Errorf("Lubricants 1.0 = %+v, Mobility 1.0 = %+v", lubricants, mobility)
	}
	assertAnswer(t, lastType, http.StatusBadRequest, validationBody(
		`{"field":"modelIds","message":"'Webinars only' needs at least one activity type, so 'Webinar' can't be removed from it."}`))
}

// The attributes of a level come in display order — sections in the order of their first attribute — and
// every level's without the filter; an unknown level is refused (FR-019).
func TestShellAttributesAPI_ListByLevel_FR019(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	activity := attributeList(t, alex, "?level=activity")
	all := attributeList(t, alex, "")
	unknown := alex.Do(t, http.MethodGet, "/api/shell-attributes?level=program", nil, apitest.SkipRequestValidation())

	if len(activity) != 20 || activity[0].Label != "Channel / audience" || activity[10].Section != "Lubricants attributes" ||
		activity[15].Section != "Mobility attributes" {
		t.Errorf("activity attributes = %+v", activity)
	}
	if len(all) != 35 || all[0].Label != "Business unit" || all[34].Label != "Contracted spend" {
		t.Errorf("%d attributes, first %+v", len(all), all[0])
	}
	assertAnswer(t, unknown, http.StatusBadRequest, `{"code":"bad_request","message":"MAPP couldn't read this request. Reload the page and try again.","fields":[]}`)
}

// listedAttribute is the part of an attribute these tests read.
type listedAttribute struct {
	ID, Label, Section string
	BusinessUnit       bool
	ParentAttribute    *struct{ ID, Name string }
	Locks              []struct{ Field, Reason string }
	OptionCount        int
}

// attributeList lists the attributes with the query.
func attributeList(t *testing.T, c *apitest.Client, query string) []listedAttribute {
	t.Helper()
	var page struct{ Items []listedAttribute }
	c.Do(t, http.MethodGet, "/api/shell-attributes"+query, nil).Decode(t, &page)
	return page.Items
}

// findAttribute returns the attribute label of the level.
func findAttribute(t *testing.T, c *apitest.Client, level, label string) listedAttribute {
	t.Helper()
	for _, a := range attributeList(t, c, "?level="+level) {
		if a.Label == label {
			return a
		}
	}
	t.Fatalf("no %s attribute %q", level, label)
	return listedAttribute{}
}

// One attribute answers with its parent, its options counted and its locks: the Business unit attribute
// is locked with the reason (FR-020, FR-023).
func TestShellAttributesAPI_Get_FR020(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	region := findAttribute(t, alex, "campaign", "Region")
	var businessUnit, got listedAttribute

	alex.Do(t, http.MethodGet, "/api/shell-attributes/SA-1001", nil).Decode(t, &businessUnit)
	alex.Do(t, http.MethodGet, "/api/shell-attributes/"+region.ID, nil).Decode(t, &got)
	gone := alex.Do(t, http.MethodGet, "/api/shell-attributes/SA-9999", nil)

	if !businessUnit.BusinessUnit || len(businessUnit.Locks) != 8 || businessUnit.Locks[0].Reason != "Business unit can't be deleted, "+
		"set inactive, moved or given another type, because other attributes are shown or hidden by it." {
		t.Errorf("Business unit = %+v", businessUnit)
	}
	if got.ParentAttribute == nil || got.ParentAttribute.Name != "Cluster" || got.OptionCount != 7 {
		t.Errorf("Region = %+v", got)
	}
	assertAnswer(t, gone, http.StatusNotFound, `{"code":"not_found","message":"This Shell attribute no longer exists — someone may have deleted it.","fields":[]}`)
}

// Options are searched by any part of the label and paged; the total counts every match; wrong paging is
// refused with a message (SC-007, FR-018).
func TestOptionsAPI_ListSearch_SC007(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	costCentre := findAttribute(t, alex, "activity", "Cost centre")
	path := "/api/shell-attributes/" + costCentre.ID + "/options"
	type page struct {
		Items []struct{ Label string }
		Total int
	}
	var found, paged page

	alex.Do(t, http.MethodGet, path+"?q=%200220%20", nil).Decode(t, &found)
	alex.Do(t, http.MethodGet, path+"?limit=2&offset=1&status=active", nil).Decode(t, &paged)
	invalid := alex.Do(t, http.MethodGet, path+"?limit=0&offset=-1&status=some", nil, apitest.SkipRequestValidation())
	gone := alex.Do(t, http.MethodGet, "/api/shell-attributes/SA-9999/options", nil)

	if found.Total != 1 || len(found.Items) != 1 || found.Items[0].Label != "CN04000220" {
		t.Errorf("search 0220 = %+v", found)
	}
	if paged.Total != 8 || len(paged.Items) != 2 || paged.Items[0].Label != "CN04000231" {
		t.Errorf("page = %+v", paged)
	}
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"limit","message":"Choose a number from 1 to 500."},`+
		`{"field":"offset","message":"Choose 0 or more."},{"field":"status","message":"Choose active, inactive or all."}`))
	assertAnswer(t, gone, http.StatusNotFound, `{"code":"not_found","message":"This Shell attribute no longer exists — someone may have deleted it.","fields":[]}`)
}

// formView is the part of a form these tests read.
type formView struct {
	Sections []struct {
		Name        string
		Conditional bool
		Attributes  []struct {
			ID, Label string
			Options   []struct{ ID, Label string }
		}
	}
	BusinessUnitAttributeID string
	BusinessUnitOptions     []struct{ ID, Label string }
}

// ids returns the ids of the form's attributes by label, and of their options by "<attribute>/<option>".
func (f formView) ids() map[string]string {
	ids := map[string]string{}
	for _, s := range f.Sections {
		for _, a := range s.Attributes {
			ids[a.Label] = a.ID
			for _, o := range a.Options {
				ids[a.Label+"/"+o.Label] = o.ID
			}
		}
	}
	for _, o := range f.BusinessUnitOptions {
		ids["Business unit/"+o.Label] = o.ID
	}
	return ids
}

// getForm reads the form of level.
func getForm(t *testing.T, c *apitest.Client, level string) formView {
	t.Helper()
	var form formView
	c.Do(t, http.MethodGet, "/api/shell-attribute-forms/"+level, nil).Decode(t, &form)
	return form
}

// A level's form has its sections in display order, conditional ones marked, and the Business unit options
// for the condition notes (FR-024).
func TestFormsAPI_Get_FR024(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	form := getForm(t, alex, "activity")
	unknown := alex.Do(t, http.MethodGet, "/api/shell-attribute-forms/program", nil, apitest.SkipRequestValidation())

	var sections []string
	for _, s := range form.Sections {
		sections = append(sections, s.Name)
	}
	if !slices.Equal(sections, []string{"Shell attributes", "Lubricants attributes", "Mobility attributes"}) ||
		form.Sections[0].Conditional || !form.Sections[1].Conditional || form.BusinessUnitAttributeID != "SA-1001" || len(form.BusinessUnitOptions) != 2 {
		t.Errorf("form = %+v", form)
	}
	if unknown.Status != http.StatusBadRequest {
		t.Errorf("unknown level = %d %s", unknown.Status, unknown.Body)
	}
}

// The check answers what a save would say: a Region that no longer fits a changed Cluster is cleared, and
// a hidden Lubricants attribute is not missing for a Mobility plan (FR-025, FR-026, FR-020).
func TestFormsAPI_Check_FR025(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	campaign := getForm(t, alex, "campaign").ids()
	activity := getForm(t, alex, "activity").ids()
	cluster, region := campaign["Cluster"], campaign["Region"]
	europe, china := campaign["Cluster/Lubricants – Europe"], campaign["Cluster/Retail – China"]
	dach := campaign["Region/LUBES – DACH"]

	changed := alex.Do(t, http.MethodPost, "/api/shell-attribute-forms/campaign/check", map[string]any{
		"values":         map[string]any{cluster: []string{china}, region: []string{dach}},
		"previousValues": map[string]any{cluster: []string{europe}, region: []string{dach}},
	})
	mobility := alex.Do(t, http.MethodPost, "/api/shell-attribute-forms/activity/check", map[string]any{
		"values": map[string]any{}, "businessUnitOptionIds": []string{activity["Business unit/Mobility"]},
	})

	assertAnswer(t, changed, http.StatusOK, `{"problems":[],"missing":[{"field":"attributes.`+region+`","label":"Region"}],`+
		`"cleared":[{"attributeId":"`+region+`","label":"Region","optionIds":["`+dach+`"],"optionLabels":["LUBES – DACH"]}],`+
		`"values":{"`+cluster+`":["`+china+`"]}}`)
	var result struct{ Missing []struct{ Label string } }
	mobility.Decode(t, &result)
	for _, m := range result.Missing {
		if m.Label == "Brands" || m.Label == "Sector" {
			t.Errorf("hidden Lubricants attribute %s missing for a Mobility plan", m.Label)
		}
	}
	if len(result.Missing) != 9 { // the nine needed attributes of section Shell attributes
		t.Errorf("missing = %+v", result.Missing)
	}
}

// The currencies come USD first, then by code, with the rate both ways and their locks; a new currency
// answers 201 and is listed; every problem of a new currency comes at once (FR-028, US4-1, US4-5).
func TestCurrenciesAPI_ListCreateGet_FR028(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	gone := alex.Do(t, http.MethodGet, "/api/currencies/JPY", nil)
	created := mustCreate(t, alex, "/api/currencies", jpyBody)
	invalid := alex.Do(t, http.MethodPost, "/api/currencies", map[string]any{"code": "EURO", "name": "Euro", "rate": "0"})
	var list struct {
		Items []struct{ Code string }
	}
	alex.Do(t, http.MethodGet, "/api/currencies", nil).Decode(t, &list)
	var euro, dollar struct {
		Rate, InverseRate string
		Locks             []struct{ Field, Reason string }
	}

	alex.Do(t, http.MethodGet, "/api/currencies/EUR", nil).Decode(t, &euro)
	alex.Do(t, http.MethodGet, "/api/currencies/USD", nil).Decode(t, &dollar)

	assertAnswer(t, created, http.StatusCreated, jpyAnswer(1, `"rateChangedAt":"2026-03-12T09:00:00Z","rateChangedBy":{"id":"U-1001","name":"Alex Morgan"}`, true))
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"code","message":"Enter a three-letter code such as JPY."},`+
		`{"field":"name","message":"A currency called 'Euro' already exists."},`+
		`{"field":"rate","message":"Enter a rate above 0, with at most six decimals."}`))
	var codes []string
	for _, c := range list.Items {
		codes = append(codes, c.Code)
	}
	if !slices.Equal(codes, []string{"USD", "CNY", "EUR", "GBP", "INR", "JPY"}) {
		t.Errorf("currencies = %q", codes)
	}
	if euro.Rate != "0.92" || euro.InverseRate != "1.086957" || len(euro.Locks) != 1 || euro.Locks[0].Field != "code" {
		t.Errorf("EUR = %+v", euro)
	}
	if len(dollar.Locks) != 4 || dollar.Locks[1].Reason != "USD is MAPP's base currency; its rate is always 1." {
		t.Errorf("USD = %+v", dollar)
	}
	assertAnswer(t, gone, http.StatusNotFound, `{"code":"not_found","message":"This currency no longer exists — someone may have deleted it.","fields":[]}`)
}

// The finance budgets of the current year come by name with their items; the years run from the current
// year − 1 to + 5; one item answers with its budget and its year locked (FR-014, FR-015).
func TestFinanceAPI_ListYear_FR015(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	type budgetPage struct {
		Items []struct {
			Name            string
			ActiveItemCount int
			Items           []struct{ Name string }
		}
		Total, Year int
		Years       []int
	}
	var current, next budgetPage
	var item struct {
		BudgetName string
		Year       int
		Locks      []struct{ Field string }
	}

	alex.Do(t, http.MethodGet, "/api/finance-budgets", nil).Decode(t, &current)
	alex.Do(t, http.MethodGet, "/api/finance-budgets?year=2027", nil).Decode(t, &next)
	alex.Do(t, http.MethodGet, "/api/finance-budget-items/FBI-1002", nil).Decode(t, &item)
	notANumber := alex.Do(t, http.MethodGet, "/api/finance-budgets?year=next", nil, apitest.SkipRequestValidation())
	gone := alex.Do(t, http.MethodGet, "/api/finance-budget-items/FBI-9999", nil)

	if current.Year != 2026 || current.Total != 2 || current.Items[0].Name != "2026 Lubricants Budget" ||
		current.Items[0].ActiveItemCount != 3 || current.Items[0].Items[0].Name != "Europe – Brand" ||
		!slices.Equal(current.Years, []int{2025, 2026, 2027, 2028, 2029, 2030, 2031}) {
		t.Errorf("2026 = %+v", current)
	}
	if next.Year != 2027 || next.Total != 0 || next.Items == nil {
		t.Errorf("2027 = %+v", next)
	}
	if item.BudgetName != "2026 Lubricants Budget" || item.Year != 2026 || len(item.Locks) != 1 || item.Locks[0].Field != "year" {
		t.Errorf("item = %+v", item)
	}
	if notANumber.Status != http.StatusBadRequest {
		t.Errorf("year=next = %d %s", notANumber.Status, notANumber.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, itemGoneBody)
}

// Bodies of the finance tests.
const (
	itemGoneBody   = `{"code":"not_found","message":"This finance budget item no longer exists — someone may have deleted it.","fields":[]}`
	budgetGoneBody = `{"code":"not_found","message":"This finance budget no longer exists — someone may have deleted it.","fields":[]}`
	yearLockBody   = `[{"field":"year","reason":"The year can't change once the item is saved — create a new item for another year."}]`
)

// retailBody is the new item of US5-3: "Europe – Retail" in "2026 Lubricants Budget", code "LUB-EU-RET".
var retailBody = map[string]any{"year": 2026, "budgetName": "2026 Lubricants Budget", "name": "Europe – Retail", "code": "LUB-EU-RET"}

// retailAnswer is the answer for the item retailBody (FBI-1007) in the budget budget (id and name) at
// version, with its code, status and bookkeeping meta.
func retailAnswer(budgetID, budgetName, code string, version int, active bool, meta string) string {
	return `{"id":"FBI-1007","budgetId":"` + budgetID + `","budgetName":"` + budgetName + `","year":2026,"name":"Europe – Retail","code":"` + code +
		`","active":` + strconv.FormatBool(active) + `,"usage":` + unusedBody + `,"locks":` + yearLockBody + `,"version":` + itoa(version) + `,` + meta + `}`
}

// A new item answers 201 in the budget of its year and name and is read back; a new budget name moves it to
// a new budget of the same year; every problem comes at once (FR-014, US5-3, US5-4).
func TestFinanceAPI_CreateGetUpdate_FR014(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	created := alex.Do(t, http.MethodPost, "/api/finance-budget-items", retailBody)
	got := alex.Do(t, http.MethodGet, "/api/finance-budget-items/FBI-1007", nil)
	moved := alex.Do(t, http.MethodPatch, "/api/finance-budget-items/FBI-1007", map[string]any{"budgetName": "2026 Retail Budget", "code": "", "version": 1})
	taken := alex.Do(t, http.MethodPost, "/api/finance-budget-items", map[string]any{
		"year": 2026, "budgetName": " 2026 lubricants budget", "name": "europe – digital", "code": "lub-eu-dig"})
	invalid := alex.Do(t, http.MethodPost, "/api/finance-budget-items", map[string]any{"year": 2040, "budgetName": " ", "name": ""})
	stale := alex.Do(t, http.MethodPatch, "/api/finance-budget-items/FBI-1007", map[string]any{"name": "X", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/finance-budget-items/FBI-9999", map[string]any{"name": "X", "version": 1})

	assertAnswer(t, created, http.StatusCreated, retailAnswer("FB-1001", "2026 Lubricants Budget", "LUB-EU-RET", 1, true, alexAtStart))
	assertAnswer(t, got, http.StatusOK, retailAnswer("FB-1001", "2026 Lubricants Budget", "LUB-EU-RET", 1, true, alexAtStart))
	assertAnswer(t, moved, http.StatusOK, retailAnswer("FB-1003", "2026 Retail Budget", "", 2, true, alexAtStart))
	assertAnswer(t, taken, http.StatusBadRequest, validationBody(
		`{"field":"name","message":"The budget '2026 Lubricants Budget' already has an item called 'Europe – Digital'."},`+
			`{"field":"code","message":"The code 'lub-eu-dig' is already used in 2026 by 'Europe – Digital'."}`))
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"year","message":"Choose a year from 2025 to 2031."},`+
		`{"field":"budgetName","message":"Enter a budget name."},{"field":"name","message":"Enter a name."}`))
	if stale.Status != http.StatusConflict {
		t.Errorf("stale update = %d %s", stale.Status, stale.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, itemGoneBody)
}

// Rename budget answers the budget with all its items under the new name; the name must be free in its
// year; an out-of-date version is refused (FR-015, US5-4).
func TestFinanceAPI_Rename_FR015(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	var renamed struct {
		Name    string
		Version int
		Items   []struct{ BudgetName string }
		Usage   struct{ Summary string }
	}
	var item struct{ BudgetName string }

	alex.Do(t, http.MethodPatch, "/api/finance-budgets/FB-1002", map[string]any{"name": "2026 Mobility Budget (China)", "version": 1}).Decode(t, &renamed)
	alex.Do(t, http.MethodGet, "/api/finance-budget-items/FBI-1004", nil).Decode(t, &item)
	taken := alex.Do(t, http.MethodPatch, "/api/finance-budgets/FB-1002", map[string]any{"name": "2026 lubricants budget", "version": 2})
	stale := alex.Do(t, http.MethodPatch, "/api/finance-budgets/FB-1002", map[string]any{"name": "Other", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/finance-budgets/FB-9999", map[string]any{"name": "Other", "version": 1})

	if renamed.Name != "2026 Mobility Budget (China)" || renamed.Version != 2 || len(renamed.Items) != 3 ||
		renamed.Items[0].BudgetName != "2026 Mobility Budget (China)" || !strings.HasPrefix(renamed.Usage.Summary, "Default finance budget of ") {
		t.Errorf("renamed = %+v", renamed)
	}
	if item.BudgetName != "2026 Mobility Budget (China)" {
		t.Errorf("China – Retail is in %q", item.BudgetName)
	}
	assertAnswer(t, taken, http.StatusBadRequest, validationBody(`{"field":"name","message":"2026 already has a budget called '2026 Lubricants Budget'."}`))
	assertAnswer(t, stale, http.StatusConflict, `{"code":"version_conflict","message":"Not saved — Alex Morgan changed this finance budget",`+
		`"fields":[],"conflict":{"changedBy":"Alex Morgan","changedAt":"2026-03-12T09:00:00Z"}}`)
	assertAnswer(t, gone, http.StatusNotFound, budgetGoneBody)
}

// Set inactive and Reactivate answer the item with its new status; an out-of-date version is refused
// (FR-004).
func TestFinanceAPI_Status_FR004(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/finance-budget-items", retailBody)

	inactive := alex.Do(t, http.MethodPost, "/api/finance-budget-items/FBI-1007/status", map[string]any{"active": false, "version": 1})
	stale := alex.Do(t, http.MethodPost, "/api/finance-budget-items/FBI-1007/status", map[string]any{"active": true, "version": 1})
	active := alex.Do(t, http.MethodPost, "/api/finance-budget-items/FBI-1007/status", map[string]any{"active": true, "version": 2})
	gone := alex.Do(t, http.MethodPost, "/api/finance-budget-items/FBI-9999/status", map[string]any{"active": true, "version": 1})

	assertAnswer(t, inactive, http.StatusOK, retailAnswer("FB-1001", "2026 Lubricants Budget", "LUB-EU-RET", 2, false, alexAtStart))
	if stale.Status != http.StatusConflict || active.Status != http.StatusOK {
		t.Errorf("stale %d %s; reactivate %d %s", stale.Status, stale.Body, active.Status, active.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, itemGoneBody)
}

// An unused item moves to the recycle bin under its budget and Undo brings it back; an item linked to line
// items is refused with in_use (FR-004, US5-5).
func TestFinanceAPI_Delete_FR004(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/finance-budget-items", retailBody)

	deleted := alex.Do(t, http.MethodDelete, "/api/finance-budget-items/FBI-1007", nil)
	afterDelete := alex.Do(t, http.MethodGet, "/api/finance-budget-items/FBI-1007", nil)
	undone := alex.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/undo", nil)
	inUse := alex.Do(t, http.MethodDelete, "/api/finance-budget-items/FBI-1001", nil)
	gone := alex.Do(t, http.MethodDelete, "/api/finance-budget-items/FBI-9999", nil)
	var refused struct{ Code, Message string }
	inUse.Decode(t, &refused)

	budget := `{"type":"financeBudget","id":"FB-1001","name":"2026 Lubricants Budget"}`
	assertAnswer(t, deleted, http.StatusOK, `{"recycleBinEntry":{"id":"BIN-1001","recordType":"financeBudgetItem","recordTypeLabel":"finance budget item",`+
		`"recordId":"FBI-1007","recordName":"Europe – Retail","path":[`+budget+`],"parent":`+budget+`,"deletedAt":"2026-03-12T09:00:00Z",`+
		`"deletedBy":{"id":"U-1001","name":"Alex Morgan"},"contents":[]}}`)
	assertAnswer(t, afterDelete, http.StatusNotFound, itemGoneBody)
	assertAnswer(t, undone, http.StatusOK, `{"recordType":"financeBudgetItem","recordTypeLabel":"finance budget item","recordId":"FBI-1007",`+
		`"recordName":"Europe – Retail","details":""}`)
	assertAnswer(t, alex.Do(t, http.MethodGet, "/api/finance-budget-items/FBI-1007", nil), http.StatusOK,
		retailAnswer("FB-1001", "2026 Lubricants Budget", "LUB-EU-RET", 1, true, alexAtStart))
	if inUse.Status != http.StatusConflict || refused.Code != "in_use" ||
		!strings.HasPrefix(refused.Message, "'Europe – Digital' is in use (linked to ") {
		t.Errorf("deleting Europe – Digital = %d %s", inUse.Status, inUse.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, itemGoneBody)
}

// Bodies of the Shell attribute tests.
const (
	attributeGoneBody = `{"code":"not_found","message":"This Shell attribute no longer exists — someone may have deleted it.","fields":[]}`
	optionGoneBody    = `{"code":"not_found","message":"This option no longer exists — someone may have deleted it.","fields":[]}`
)

// mediaChannelBody is the new activity attribute of US2-1: a single choice with four options, needed to
// submit.
var mediaChannelBody = map[string]any{
	"label": "Media channel", "level": "activity", "type": "singleChoice", "marking": "neededToSubmit", "section": "Shell attributes",
	"options": []map[string]any{{"label": "TV"}, {"label": "Radio"}, {"label": "Print"}, {"label": "Online"}},
}

// savedAttribute is the part of an attribute answer these tests read.
type savedAttribute struct {
	ID, Label, Section, Type, Marking string
	Position, Version, OptionCount    int
	Active                            bool
	ParentAttribute                   *struct{ ID, Name string }
}

// savedOption is the part of an option answer these tests read.
type savedOption struct {
	ID, AttributeID, Label string
	Position, Version      int
	Active                 bool
}

// createMediaChannel creates Media channel and returns it with its options.
func createMediaChannel(t *testing.T, c *apitest.Client) (savedAttribute, []savedOption) {
	t.Helper()
	var created savedAttribute
	mustCreate(t, c, "/api/shell-attributes", mediaChannelBody).Decode(t, &created)
	var page struct{ Items []savedOption }
	c.Do(t, http.MethodGet, "/api/shell-attributes/"+created.ID+"/options", nil).Decode(t, &page)
	return created, page.Items
}

// A new attribute answers 201, last in its section with its options counted; every problem comes at once
// (FR-016, FR-018, US2-1, US2-2).
func TestShellAttributesAPI_Create_FR016(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	before := attributeList(t, alex, "?level=activity")

	created, options := createMediaChannel(t, alex)
	invalid := alex.Do(t, http.MethodPost, "/api/shell-attributes", map[string]any{
		"label": "Channel / audience", "level": "activity", "type": "singleChoice", "marking": "neededToPublish",
		"options": []map[string]any{{"label": "TV"}, {"label": "tv"}},
	})
	after := attributeList(t, alex, "?level=activity")

	if created.Label != "Media channel" || created.OptionCount != 4 || !created.Active || created.Version != 1 || len(options) != 4 {
		t.Errorf("created = %+v with %+v", created, options)
	}
	if len(after) != len(before)+1 {
		t.Fatalf("%d activity attributes after, %d before", len(after), len(before))
	}
	for i, a := range after {
		if a.ID == created.ID && (i+1 < len(after) && after[i+1].Section == "Shell attributes" || a.Section != "Shell attributes") {
			t.Errorf("Media channel is not last in its section: %+v", after)
		}
	}
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(
		`{"field":"marking","message":"At activity level the marking is Optional or Needed to submit."},`+
			`{"field":"label","message":"The Activity form already has a Shell attribute called 'Channel / audience'."},`+
			`{"field":"options[1].label","message":"'TV' appears twice."}`))
}

// A change answers the attribute at its new version; an out-of-date version is a conflict; a locked field
// is refused with its reason (FR-021, FR-007, FR-023).
func TestShellAttributesAPI_Update_FR021(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, _ := createMediaChannel(t, alex)
	path := "/api/shell-attributes/" + created.ID
	var renamed savedAttribute

	alex.Do(t, http.MethodPatch, path, map[string]any{"label": "Media", "helpText": "Where it runs", "version": 1}).Decode(t, &renamed)
	stale := alex.Do(t, http.MethodPatch, path, map[string]any{"label": "Media channel", "version": 1})
	locked := alex.Do(t, http.MethodPatch, "/api/shell-attributes/SA-1001", map[string]any{"type": "singleChoice", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/shell-attributes/SA-9999", map[string]any{"label": "X", "version": 1})

	if renamed.Label != "Media" || renamed.Version != 2 {
		t.Errorf("renamed = %+v", renamed)
	}
	if stale.Status != http.StatusConflict {
		t.Errorf("stale = %d %s", stale.Status, stale.Body)
	}
	assertAnswer(t, locked, http.StatusBadRequest, validationBody(`{"field":"type","message":"Business unit can't be deleted, set inactive, `+
		`moved or given another type, because other attributes are shown or hidden by it."}`))
	assertAnswer(t, gone, http.StatusNotFound, attributeGoneBody)
}

// The impact check answers the warnings and saves nothing (FR-021, D-044).
func TestShellAttributesAPI_Impact_FR021(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, _ := createMediaChannel(t, alex)
	path := "/api/shell-attributes/" + created.ID

	removed := alex.Do(t, http.MethodPost, path+"/impact", map[string]any{"type": "text", "version": 1})
	invalid := alex.Do(t, http.MethodPost, path+"/impact", map[string]any{"marking": "neededToPublish", "version": 1})
	gone := alex.Do(t, http.MethodPost, "/api/shell-attributes/SA-9999/impact", map[string]any{"version": 1})

	assertAnswer(t, removed, http.StatusOK, `{"warnings":[{"kind":"optionsRemoved","message":"Its 4 options will be removed: TV, Radio, Print, Online.","count":4}]}`)
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"marking","message":"At activity level the marking is Optional or Needed to submit."}`))
	assertAnswer(t, gone, http.StatusNotFound, attributeGoneBody)
	if got := findAttribute(t, alex, "activity", "Media channel"); got.OptionCount != 4 {
		t.Errorf("the impact check changed the attribute: %+v", got)
	}
}

// Set inactive and Reactivate answer the attribute in its new status; the Business unit attribute is
// refused with its reason (FR-004, FR-023).
func TestShellAttributesAPI_Status_FR023(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, _ := createMediaChannel(t, alex)
	path := "/api/shell-attributes/" + created.ID + "/status"
	var inactive, active savedAttribute

	alex.Do(t, http.MethodPost, path, map[string]any{"active": false, "version": 1}).Decode(t, &inactive)
	alex.Do(t, http.MethodPost, path, map[string]any{"active": true, "version": 2}).Decode(t, &active)
	unit := alex.Do(t, http.MethodPost, "/api/shell-attributes/SA-1001/status", map[string]any{"active": false, "version": 1})
	stale := alex.Do(t, http.MethodPost, path, map[string]any{"active": false, "version": 1})

	if inactive.Active || inactive.Version != 2 || !active.Active || active.Version != 3 {
		t.Errorf("inactive %+v, active %+v", inactive, active)
	}
	assertAnswer(t, unit, http.StatusBadRequest, validationBody(`{"field":"active","message":"Business unit can't be deleted, set inactive, `+
		`moved or given another type, because other attributes are shown or hidden by it."}`))
	if stale.Status != http.StatusConflict {
		t.Errorf("stale = %d %s", stale.Status, stale.Body)
	}
}

// A move answers the level's attributes in their new order; a move past the edge of the section is refused
// (FR-019, US2-6).
func TestShellAttributesAPI_Move_FR019(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, _ := createMediaChannel(t, alex)
	path := "/api/shell-attributes/" + created.ID + "/move"
	var page struct {
		Items []listedAttribute
		Total int
	}

	alex.Do(t, http.MethodPost, path, map[string]any{"what": "attribute", "direction": "up", "version": 1}).Decode(t, &page)
	last := alex.Do(t, http.MethodPost, path, map[string]any{"what": "attribute", "direction": "down", "version": 2})
	again := alex.Do(t, http.MethodPost, path, map[string]any{"what": "attribute", "direction": "down", "version": 3})

	i := slices.IndexFunc(page.Items, func(a listedAttribute) bool { return a.ID == created.ID })
	if i < 1 || page.Total != len(page.Items) || page.Items[i+1].Section != "Shell attributes" {
		t.Errorf("after the move: %+v", page.Items)
	}
	if last.Status != http.StatusOK {
		t.Errorf("moving back = %d %s", last.Status, last.Body)
	}
	assertAnswer(t, again, http.StatusBadRequest, validationBody(`{"field":"direction","message":"'Media channel' is already last in its section."}`))
}

// An unused attribute moves to the recycle bin with its options and Undo brings it back; one in use is
// refused (FR-004, US2-7).
func TestShellAttributesAPI_Delete_FR004(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, _ := createMediaChannel(t, alex)
	var deleted struct {
		RecycleBinEntry struct {
			ID, RecordType string
			Contents       []struct{ Count int }
		}
	}

	alex.Do(t, http.MethodDelete, "/api/shell-attributes/"+created.ID, nil).Decode(t, &deleted)
	afterDelete := alex.Do(t, http.MethodGet, "/api/shell-attributes/"+created.ID, nil)
	undone := alex.Do(t, http.MethodPost, "/api/recycle-bin/"+deleted.RecycleBinEntry.ID+"/undo", nil)
	unit := alex.Do(t, http.MethodDelete, "/api/shell-attributes/SA-1001", nil)
	gone := alex.Do(t, http.MethodDelete, "/api/shell-attributes/SA-9999", nil)

	if deleted.RecycleBinEntry.RecordType != "shellAttribute" || len(deleted.RecycleBinEntry.Contents) != 1 || deleted.RecycleBinEntry.Contents[0].Count != 4 {
		t.Errorf("deleted = %+v", deleted)
	}
	assertAnswer(t, afterDelete, http.StatusNotFound, attributeGoneBody)
	if undone.Status != http.StatusOK || findAttribute(t, alex, "activity", "Media channel").OptionCount != 4 {
		t.Errorf("undo = %d %s", undone.Status, undone.Body)
	}
	assertAnswer(t, unit, http.StatusConflict, `{"code":"in_use","message":"Business unit can't be deleted, set inactive, moved or given `+
		`another type, because other attributes are shown or hidden by it.","fields":[]}`)
	assertAnswer(t, gone, http.StatusNotFound, attributeGoneBody)
}

// A new option answers 201, last in its attribute's list; a label already used is refused (FR-018).
func TestOptionsAPI_Create_FR018(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, _ := createMediaChannel(t, alex)
	path := "/api/shell-attributes/" + created.ID + "/options"
	var added savedOption

	mustCreate(t, alex, path, map[string]any{"label": "Cinema"}).Decode(t, &added)
	taken := alex.Do(t, http.MethodPost, path, map[string]any{"label": "tv"})
	gone := alex.Do(t, http.MethodPost, "/api/shell-attributes/SA-9999/options", map[string]any{"label": "Cinema"})

	if added.Label != "Cinema" || added.Position != 5 || added.AttributeID != created.ID || !added.Active {
		t.Errorf("added = %+v", added)
	}
	assertAnswer(t, taken, http.StatusBadRequest, validationBody(`{"field":"label","message":"'TV' is already an option of Media channel."}`))
	assertAnswer(t, gone, http.StatusNotFound, attributeGoneBody)
}

// A renamed option answers at its new version; an out-of-date version is a conflict (FR-018, FR-007).
func TestOptionsAPI_Update_FR018(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	_, options := createMediaChannel(t, alex)
	path := "/api/shell-attribute-options/" + options[0].ID
	var renamed savedOption

	alex.Do(t, http.MethodPatch, path, map[string]any{"label": "Television", "version": 1}).Decode(t, &renamed)
	stale := alex.Do(t, http.MethodPatch, path, map[string]any{"label": "TV", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/shell-attribute-options/SO-99999", map[string]any{"label": "TV", "version": 1})

	if renamed.Label != "Television" || renamed.Version != 2 {
		t.Errorf("renamed = %+v", renamed)
	}
	if stale.Status != http.StatusConflict {
		t.Errorf("stale = %d %s", stale.Status, stale.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, optionGoneBody)
}

// Set inactive and Reactivate answer the option in its new status (FR-018).
func TestOptionsAPI_Status_FR018(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	_, options := createMediaChannel(t, alex)
	path := "/api/shell-attribute-options/" + options[1].ID + "/status"
	var inactive, active savedOption

	alex.Do(t, http.MethodPost, path, map[string]any{"active": false, "version": 1}).Decode(t, &inactive)
	alex.Do(t, http.MethodPost, path, map[string]any{"active": true, "version": 2}).Decode(t, &active)
	invalid := alex.Do(t, http.MethodPost, path, map[string]any{"version": 3}, apitest.SkipRequestValidation())

	if inactive.Active || inactive.Version != 2 || !active.Active || active.Version != 3 {
		t.Errorf("inactive %+v, active %+v", inactive, active)
	}
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"active","message":"Reload the page and try again."}`))
}

// An option moves one place and answers in its new place; the first goes no further up (FR-018).
func TestOptionsAPI_Move_FR018(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	_, options := createMediaChannel(t, alex)
	var moved savedOption

	alex.Do(t, http.MethodPost, "/api/shell-attribute-options/"+options[1].ID+"/move", map[string]any{"direction": "up", "version": 1}).Decode(t, &moved)
	first := alex.Do(t, http.MethodPost, "/api/shell-attribute-options/"+options[1].ID+"/move", map[string]any{"direction": "up", "version": 2})

	if moved.Label != "Radio" || moved.Position != 1 || moved.Version != 2 {
		t.Errorf("moved = %+v", moved)
	}
	assertAnswer(t, first, http.StatusBadRequest, validationBody(`{"field":"direction","message":"'Radio' is already first."}`))
}

// GET answers one option as stored now — with no locks while its attribute has other options, with the
// delete lock once it is the last one — and 404 for an option that is gone (D-211). Both answers follow
// the contract.
func TestOptionsAPI_Get_D211(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	_, options := createMediaChannel(t, alex)
	type lock struct{ Field, Reason string }
	var first struct {
		ID, Label string
		Locks     []lock
	}

	alex.Do(t, http.MethodGet, "/api/shell-attribute-options/"+options[0].ID, nil).Decode(t, &first)
	for _, o := range options[1:] {
		alex.Do(t, http.MethodDelete, "/api/shell-attribute-options/"+o.ID, nil)
	}
	var last struct{ Locks []lock }
	alex.Do(t, http.MethodGet, "/api/shell-attribute-options/"+options[0].ID, nil).Decode(t, &last)
	gone := alex.Do(t, http.MethodGet, "/api/shell-attribute-options/SO-99999", nil)

	if first.ID != options[0].ID || first.Label != "TV" || first.Locks == nil || len(first.Locks) != 0 {
		t.Errorf("first = %+v; want TV without locks", first)
	}
	want := []lock{{Field: "delete", Reason: "'Media channel' needs at least one option, so 'TV' can't be deleted."}}
	if !reflect.DeepEqual(last.Locks, want) {
		t.Errorf("last option's locks = %+v, want %+v", last.Locks, want)
	}
	assertAnswer(t, gone, http.StatusNotFound,
		`{"code":"not_found","message":"This option no longer exists — someone may have deleted it.","fields":[]}`)
}

// An unused option moves to the recycle bin and Undo brings it back; the last option of an attribute is
// refused (FR-004, FR-018).
func TestOptionsAPI_Delete_FR004(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, options := createMediaChannel(t, alex)
	var deleted struct {
		RecycleBinEntry struct {
			ID, RecordType string
			Parent         *struct{ ID string }
		}
	}

	alex.Do(t, http.MethodDelete, "/api/shell-attribute-options/"+options[3].ID, nil).Decode(t, &deleted)
	undone := alex.Do(t, http.MethodPost, "/api/recycle-bin/"+deleted.RecycleBinEntry.ID+"/undo", nil)
	for _, o := range options[1:] {
		alex.Do(t, http.MethodDelete, "/api/shell-attribute-options/"+o.ID, nil)
	}
	last := alex.Do(t, http.MethodDelete, "/api/shell-attribute-options/"+options[0].ID, nil)
	gone := alex.Do(t, http.MethodDelete, "/api/shell-attribute-options/SO-99999", nil)

	if deleted.RecycleBinEntry.RecordType != "shellAttributeOption" || deleted.RecycleBinEntry.Parent == nil ||
		deleted.RecycleBinEntry.Parent.ID != created.ID || undone.Status != http.StatusOK {
		t.Errorf("deleted = %+v; undo %d %s", deleted, undone.Status, undone.Body)
	}
	assertAnswer(t, last, http.StatusConflict, `{"code":"in_use","message":"'Media channel' needs at least one option, so 'TV' can't be deleted.","fields":[]}`)
	assertAnswer(t, gone, http.StatusNotFound, optionGoneBody)
}

// The import check answers every row's outcome and adds nothing; the import adds the new valid rows and
// answers the counts; a file with nothing new, a file over 700 KB and an attribute that is gone are refused
// (FR-031, US6-1, US6-2).
func TestImportAPI_CheckAndImport_FR031(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	var created savedAttribute
	mustCreate(t, alex, "/api/shell-attributes", map[string]any{"label": "Cost centre", "level": "lineItem", "type": "singleChoice",
		"marking": "optional", "section": "Finance coding", "options": []map[string]any{{"label": "CN04000001"}}}).Decode(t, &created)
	file := func(text string) map[string]string {
		return map[string]string{"fileName": "cost-centres.csv", "content": base64.StdEncoding.EncodeToString([]byte(text))}
	}
	path := "/api/shell-attributes/" + created.ID + "/options/"
	csv := "Option\nCN04000001\nCN04000002\n,see the note\n"

	checked := alex.Do(t, http.MethodPost, path+"import-check", file(csv))
	imported := alex.Do(t, http.MethodPost, path+"import", file(csv))
	again := alex.Do(t, http.MethodPost, path+"import", file(csv))
	tooLarge := alex.Do(t, http.MethodPost, path+"import-check", file(strings.Repeat("a", 700*1024+10)))
	gone := alex.Do(t, http.MethodPost, "/api/shell-attributes/SA-9999/options/import", file(csv))
	var options struct{ Items []savedOption }
	alex.Do(t, http.MethodGet, path[:len(path)-1], nil).Decode(t, &options)

	assertAnswer(t, checked, http.StatusOK, `{"totalRows":3,"toAdd":[{"row":3,"label":"CN04000002","parentLabel":"","message":""}],`+
		`"alreadyPresent":[{"row":2,"label":"CN04000001","parentLabel":"","message":"Already an option — skipped."}],`+
		`"problems":[{"row":4,"label":"","parentLabel":"","message":"Empty — enter an option."}]}`)
	assertAnswer(t, imported, http.StatusOK, `{"added":1,"alreadyPresent":1,"problems":1}`)
	assertAnswer(t, again, http.StatusBadRequest, validationBody(`{"field":"content","message":"Nothing to add — every row is already an option or has a problem."}`))
	assertAnswer(t, tooLarge, http.StatusBadRequest, validationBody(`{"field":"content","message":"The file is larger than 700 KB — split it into smaller files."}`))
	assertAnswer(t, gone, http.StatusNotFound, attributeGoneBody)
	if len(options.Items) != 2 || options.Items[1].Label != "CN04000002" || options.Items[1].Position != 2 || !options.Items[1].Active {
		t.Errorf("options = %+v", options.Items)
	}
}

// Bodies of the currency tests.
const (
	currencyGoneBody = `{"code":"not_found","message":"This currency no longer exists — someone may have deleted it.","fields":[]}`
	usdLockedReason  = "USD is MAPP's base currency; its rate is always 1."
)

// jpyBody is the new currency of US4-1.
var jpyBody = map[string]any{"code": "jpy", "name": "Japanese yen", "symbol": "¥", "rate": "150"}

// jpyAnswer is the answer for the currency jpyBody at version, with who changed its rate and its status;
// Alex created it at the start.
func jpyAnswer(version int, rateChanged string, active bool) string {
	return `{"code":"JPY","name":"Japanese yen","symbol":"¥","rate":"150","inverseRate":"0.006667",` + rateChanged +
		`,"rateDate":"2026-03-12","rateSource":"manual","rateImportFileName":null,"active":` + strconv.FormatBool(active) + `,"usage":` + unusedBody +
		`,"locks":[{"field":"code","reason":"The code can't change once the currency is saved."}],"version":` + itoa(version) + `,` + alexAtStart + `}`
}

// A save from an out-of-date copy is refused with who changed the currency and when; a new rate marks who
// changed it (FR-007, FR-028).
func TestCurrenciesAPI_UpdateConflict_FR007(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	daniel := server.SignIn(t, danielsEmail)
	server.GrantRole(t, danielsEmail, "administrator", "all") // 014: changing Settings
	mustCreate(t, alex, "/api/currencies", jpyBody)
	server.Advance(90 * time.Minute)
	var daniels struct {
		Rate          string
		RateChangedBy struct{ Name string }
		Version       int
	}

	daniel.Do(t, http.MethodPatch, "/api/currencies/JPY", map[string]any{"rate": "151", "version": 1}).Decode(t, &daniels)
	alexs := alex.Do(t, http.MethodPatch, "/api/currencies/JPY", map[string]any{"name": "Yen", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/currencies/XXX", map[string]any{"name": "X", "version": 1})

	if daniels.Rate != "151" || daniels.RateChangedBy.Name != "Daniel Okoro" || daniels.Version != 2 {
		t.Errorf("Daniel's update = %+v", daniels)
	}
	assertAnswer(t, alexs, http.StatusConflict, `{"code":"version_conflict","message":"Not saved — Daniel Okoro changed this currency",`+
		`"fields":[],"conflict":{"changedBy":"Daniel Okoro","changedAt":"2026-03-12T10:30:00Z"}}`)
	assertAnswer(t, gone, http.StatusNotFound, currencyGoneBody)
}

// The impact check validates like the update and answers the warnings, saving nothing (FR-030, D-044).
func TestCurrenciesAPI_Impact_FR030(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/currencies", jpyBody)

	unused := alex.Do(t, http.MethodPost, "/api/currencies/JPY/impact", map[string]any{"rate": "151", "version": 1})
	invalid := alex.Do(t, http.MethodPost, "/api/currencies/JPY/impact", map[string]any{"rate": "0", "version": 1})
	gone := alex.Do(t, http.MethodPost, "/api/currencies/XXX/impact", map[string]any{"version": 1})

	assertAnswer(t, unused, http.StatusOK, `{"warnings":[]}`)
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"rate","message":"Enter a rate above 0, with at most six decimals."}`))
	assertAnswer(t, gone, http.StatusNotFound, currencyGoneBody)
	assertAnswer(t, alex.Do(t, http.MethodGet, "/api/currencies/JPY", nil), http.StatusOK,
		jpyAnswer(1, `"rateChangedAt":"2026-03-12T09:00:00Z","rateChangedBy":{"id":"U-1001","name":"Alex Morgan"}`, true))
}

// Set inactive and Reactivate answer the currency with its new status; an out-of-date version is refused
// (FR-004).
func TestCurrenciesAPI_Status_FR004(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/currencies", jpyBody)
	rateChanged := `"rateChangedAt":"2026-03-12T09:00:00Z","rateChangedBy":{"id":"U-1001","name":"Alex Morgan"}`

	inactive := alex.Do(t, http.MethodPost, "/api/currencies/JPY/status", map[string]any{"active": false, "version": 1})
	stale := alex.Do(t, http.MethodPost, "/api/currencies/JPY/status", map[string]any{"active": true, "version": 1})
	active := alex.Do(t, http.MethodPost, "/api/currencies/JPY/status", map[string]any{"active": true, "version": 2})

	assertAnswer(t, inactive, http.StatusOK, jpyAnswer(2, rateChanged, false))
	if stale.Status != http.StatusConflict || active.Status != http.StatusOK {
		t.Errorf("stale %d %s; reactivate %d %s", stale.Status, stale.Body, active.Status, active.Body)
	}
}

// An unused currency moves to the recycle bin, listed by its code, and Undo brings it back (FR-004).
func TestCurrenciesAPI_DeleteAndUndo_FR004(t *testing.T) {
	server := apitest.New(t, apitest.WithoutDemoHistory())
	alex := server.SignIn(t, alexsEmail)
	mustCreate(t, alex, "/api/currencies", jpyBody)

	deleted := alex.Do(t, http.MethodDelete, "/api/currencies/JPY", nil)
	afterDelete := alex.Do(t, http.MethodGet, "/api/currencies/JPY", nil)
	undone := alex.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/undo", nil)
	gone := alex.Do(t, http.MethodDelete, "/api/currencies/XXX", nil)

	assertAnswer(t, deleted, http.StatusOK, `{"recycleBinEntry":{"id":"BIN-1001","recordType":"currency","recordTypeLabel":"currency",`+
		`"recordId":"JPY","recordName":"JPY","path":[],"parent":null,"deletedAt":"2026-03-12T09:00:00Z",`+
		`"deletedBy":{"id":"U-1001","name":"Alex Morgan"},"contents":[{"recordType":"exchangeRate","label":"exchange rate","count":1}]}}`)
	assertAnswer(t, afterDelete, http.StatusNotFound, currencyGoneBody)
	assertAnswer(t, undone, http.StatusOK, `{"recordType":"currency","recordTypeLabel":"currency","recordId":"JPY","recordName":"JPY","details":""}`)
	assertAnswer(t, gone, http.StatusNotFound, currencyGoneBody)
}

// USD's rate, status and delete are refused with the reason; its name may change (FR-028, US4-4).
func TestCurrenciesAPI_USDLocked_FR028(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	rate := alex.Do(t, http.MethodPatch, "/api/currencies/USD", map[string]any{"rate": "2", "version": 1})
	status := alex.Do(t, http.MethodPost, "/api/currencies/USD/status", map[string]any{"active": false, "version": 1})
	deleted := alex.Do(t, http.MethodDelete, "/api/currencies/USD", nil)
	renamed := alex.Do(t, http.MethodPatch, "/api/currencies/USD", map[string]any{"name": "United States dollar", "version": 1})

	assertAnswer(t, rate, http.StatusBadRequest, validationBody(`{"field":"rate","message":"`+usdLockedReason+`"}`))
	assertAnswer(t, status, http.StatusBadRequest, validationBody(`{"field":"active","message":"`+usdLockedReason+`"}`))
	assertAnswer(t, deleted, http.StatusConflict, `{"code":"in_use","message":"`+usdLockedReason+`","fields":[]}`)
	if renamed.Status != http.StatusOK {
		t.Errorf("rename USD = %d %s", renamed.Status, renamed.Body)
	}
}

// mustCreate posts body to path and stops the test unless the answer is 201.
func mustCreate(t *testing.T, c *apitest.Client, path string, body any) *apitest.Response {
	t.Helper()
	response := c.Do(t, http.MethodPost, path, body)
	if response.Status != http.StatusCreated {
		t.Fatalf("POST %s = %d %s", path, response.Status, response.Body)
	}
	return response
}

// validationBody is the body of a validation_failed answer listing fields, which are written as JSON.
func validationBody(fields string) string {
	return `{"code":"validation_failed","message":"Some fields need your attention.","fields":[` + fields + `]}`
}

// assertAnswer checks the status and the exact body of an answer.
func assertAnswer(t *testing.T, response *apitest.Response, status int, body string) {
	t.Helper()
	if response.Status != status || string(response.Body) != body {
		t.Errorf("answer = %d %s\nwant     %d %s", response.Status, response.Body, status, body)
	}
}

// replace returns s with the first old replaced by new, and fails loudly when old is not there.
func replace(s, old, new string) string {
	i := index(s, old)
	if i < 0 {
		panic("replace: " + old + " not in " + s)
	}
	return s[:i] + new + s[i+len(old):]
}

// index returns the index of the first sub in s, or -1.
func index(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// itoa writes n.
func itoa(n int) string {
	return string(rune('0' + n))
}
