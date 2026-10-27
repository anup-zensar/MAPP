package settings_test

import (
	"net/http"
	"slices"
	"testing"

	"mapp/internal/platform/apitest"
)

// The attributes of a level come in display order — sections in the order of their first attribute — and
// every level's without the filter; an unknown level is refused (FR-019).
func TestShellAttributesAPI_ListByLevel_FR019(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)

	activity := attributeList(t, alex, "?level=activity")
	all := attributeList(t, alex, "")
	unknown := alex.Do(t, http.MethodGet, "/api/v1/shell-attributes?level=program", nil, apitest.SkipRequestValidation())

	if len(activity) != 20 || activity[0].Label != "Channel / audience" || activity[10].Section != "Lubricants attributes" ||
		activity[15].Section != "Mobility attributes" {
		t.Errorf("activity attributes = %+v", activity)
	}
	if len(all) != 35 || all[0].Label != "Business unit" || all[34].Label != "Contracted spend" {
		t.Errorf("%d attributes, first %+v", len(all), all[0])
	}
	assertAnswer(t, unknown, http.StatusBadRequest, `{"error":{"code":"bad_request","message":"MAPP couldn't read this request. Reload the page and try again.","fields":[]}}`)
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
	c.Do(t, http.MethodGet, "/api/v1/shell-attributes"+query, nil).Decode(t, &page)
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
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	region := findAttribute(t, alex, "campaign", "Region")
	var businessUnit, got listedAttribute

	alex.Do(t, http.MethodGet, "/api/v1/shell-attributes/SA-1001", nil).Decode(t, &businessUnit)
	alex.Do(t, http.MethodGet, "/api/v1/shell-attributes/"+region.ID, nil).Decode(t, &got)
	gone := alex.Do(t, http.MethodGet, "/api/v1/shell-attributes/SA-9999", nil)

	if !businessUnit.BusinessUnit || len(businessUnit.Locks) != 8 || businessUnit.Locks[0].Reason != "Business unit can't be deleted, "+
		"set inactive, moved or given another type, because other attributes are shown or hidden by it." {
		t.Errorf("Business unit = %+v", businessUnit)
	}
	if got.ParentAttribute == nil || got.ParentAttribute.Name != "Cluster" || got.OptionCount != 7 {
		t.Errorf("Region = %+v", got)
	}
	assertAnswer(t, gone, http.StatusNotFound, `{"error":{"code":"not_found","message":"This Shell attribute no longer exists — someone may have deleted it.","fields":[]}}`)
}

// Bodies of the Shell attribute tests.
const (
	attributeGoneBody = `{"error":{"code":"not_found","message":"This Shell attribute no longer exists — someone may have deleted it.","fields":[]}}`
	optionGoneBody    = `{"error":{"code":"not_found","message":"This option no longer exists — someone may have deleted it.","fields":[]}}`
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
	mustCreate(t, c, "/api/v1/shell-attributes", mediaChannelBody).Decode(t, &created)
	var page struct{ Items []savedOption }
	c.Do(t, http.MethodGet, "/api/v1/shell-attributes/"+created.ID+"/options", nil).Decode(t, &page)
	return created, page.Items
}

// A new attribute answers 201, last in its section with its options counted; every problem comes at once
// (FR-016, FR-018, US2-1, US2-2).
func TestShellAttributesAPI_Create_FR016(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	before := attributeList(t, alex, "?level=activity")

	created, options := createMediaChannel(t, alex)
	invalid := alex.Do(t, http.MethodPost, "/api/v1/shell-attributes", map[string]any{
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
	assertLastOfShellAttributes(t, after, created.ID)
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(
		`{"field":"marking","message":"At activity level the marking is Optional or Needed to submit."},`+
			`{"field":"label","message":"The Activity form already has a Shell attribute called 'Channel / audience'."},`+
			`{"field":"options[1].label","message":"'TV' appears twice."}`))
}

// A change answers the attribute at its new version; an out-of-date version is a conflict; a locked field
// is refused with its reason (FR-021, FR-007, FR-023).
func TestShellAttributesAPI_Update_FR021(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, _ := createMediaChannel(t, alex)
	path := "/api/v1/shell-attributes/" + created.ID
	var renamed savedAttribute

	alex.Do(t, http.MethodPatch, path, map[string]any{"label": "Media", "helpText": "Where it runs", "version": 1}).Decode(t, &renamed)
	stale := alex.Do(t, http.MethodPatch, path, map[string]any{"label": "Media channel", "version": 1})
	locked := alex.Do(t, http.MethodPatch, "/api/v1/shell-attributes/SA-1001", map[string]any{"type": "singleChoice", "version": 1})
	gone := alex.Do(t, http.MethodPatch, "/api/v1/shell-attributes/SA-9999", map[string]any{"label": "X", "version": 1})

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
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, _ := createMediaChannel(t, alex)
	path := "/api/v1/shell-attributes/" + created.ID

	removed := alex.Do(t, http.MethodPost, path+"/impact", map[string]any{"type": "text", "version": 1})
	invalid := alex.Do(t, http.MethodPost, path+"/impact", map[string]any{"marking": "neededToPublish", "version": 1})
	gone := alex.Do(t, http.MethodPost, "/api/v1/shell-attributes/SA-9999/impact", map[string]any{"version": 1})

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
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, _ := createMediaChannel(t, alex)
	path := "/api/v1/shell-attributes/" + created.ID + "/status"
	var inactive, active savedAttribute

	alex.Do(t, http.MethodPost, path, map[string]any{"active": false, "version": 1}).Decode(t, &inactive)
	alex.Do(t, http.MethodPost, path, map[string]any{"active": true, "version": 2}).Decode(t, &active)
	unit := alex.Do(t, http.MethodPost, "/api/v1/shell-attributes/SA-1001/status", map[string]any{"active": false, "version": 1})
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
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, _ := createMediaChannel(t, alex)
	path := "/api/v1/shell-attributes/" + created.ID + "/move"
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
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created, _ := createMediaChannel(t, alex)
	var deleted struct {
		RecycleBinEntry struct {
			ID, RecordType string
			Contents       []struct{ Count int }
		}
	}

	alex.Do(t, http.MethodDelete, "/api/v1/shell-attributes/"+created.ID, nil).Decode(t, &deleted)
	afterDelete := alex.Do(t, http.MethodGet, "/api/v1/shell-attributes/"+created.ID, nil)
	undone := alex.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+deleted.RecycleBinEntry.ID+"/undo", nil)
	unit := alex.Do(t, http.MethodDelete, "/api/v1/shell-attributes/SA-1001", nil)
	gone := alex.Do(t, http.MethodDelete, "/api/v1/shell-attributes/SA-9999", nil)

	if deleted.RecycleBinEntry.RecordType != "shellAttribute" || len(deleted.RecycleBinEntry.Contents) != 1 || deleted.RecycleBinEntry.Contents[0].Count != 4 {
		t.Errorf("deleted = %+v", deleted)
	}
	assertAnswer(t, afterDelete, http.StatusNotFound, attributeGoneBody)
	if undone.Status != http.StatusOK || findAttribute(t, alex, "activity", "Media channel").OptionCount != 4 {
		t.Errorf("undo = %d %s", undone.Status, undone.Body)
	}
	assertAnswer(t, unit, http.StatusConflict, `{"error":{"code":"in_use","message":"Business unit can't be deleted, set inactive, moved or given `+
		`another type, because other attributes are shown or hidden by it.","fields":[]}}`)
	assertAnswer(t, gone, http.StatusNotFound, attributeGoneBody)
}

// assertLastOfShellAttributes checks that the attribute id comes last in the section "Shell attributes" of list.
func assertLastOfShellAttributes(t *testing.T, list []listedAttribute, id string) {
	t.Helper()
	for i, a := range list {
		if a.ID == id && (i+1 < len(list) && list[i+1].Section == "Shell attributes" || a.Section != "Shell attributes") {
			t.Errorf("Media channel is not last in its section: %+v", list)
		}
	}
}
