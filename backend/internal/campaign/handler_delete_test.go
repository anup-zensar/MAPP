package campaign_test

import (
	"net/http"
	"reflect"
	"testing"

	"mapp/internal/platform/apitest"
)

// pathStep is one step of a recycle-bin entry's path.
type pathStep struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// deleteResult is the part of a DeleteResult answer these tests read.
type deleteResult struct {
	RecycleBinEntry struct {
		ID         string     `json:"id"`
		RecordType string     `json:"recordType"`
		RecordID   string     `json:"recordId"`
		RecordName string     `json:"recordName"`
		Path       []pathStep `json:"path"`
		Contents   []struct {
			RecordType string `json:"recordType"`
			Label      string `json:"label"`
			Count      int    `json:"count"`
		} `json:"contents"`
	} `json:"recycleBinEntry"`
}

// globalBrand is the plan FY2027 Global Brand as a path step (no activities below it in the demo data).
var globalBrand = pathStep{Type: "plan", ID: "P-1003", Name: "FY2027 Global Brand"}

// Deleting Brand platform answers its recycle-bin entry, with its 2 initiatives, so the web app can offer
// Undo; the campaign is gone from the plan until Undo puts it back as it was (FR-028, FR-029).
func TestDeleteCampaign_ContractAndUndo(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, "alex.morgan@shell.com")
	var before campaignAnswer
	alex.Do(t, http.MethodGet, "/api/v1/campaigns/C-1004", nil).Decode(t, &before)

	response := alex.Do(t, http.MethodDelete, "/api/v1/campaigns/C-1004", nil)

	assertStatus(t, response, http.StatusOK)
	var result deleteResult
	response.Decode(t, &result)
	entry := result.RecycleBinEntry
	if entry.RecordType != "campaign" || entry.RecordID != "C-1004" || entry.RecordName != "Brand platform" ||
		!reflect.DeepEqual(entry.Path, []pathStep{globalBrand}) || len(entry.Contents) != 1 ||
		entry.Contents[0].RecordType != "initiative" || entry.Contents[0].Label != "initiatives" || entry.Contents[0].Count != 2 {
		t.Errorf("recycle-bin entry = %+v", entry)
	}
	assertBody(t, alex.Do(t, http.MethodGet, "/api/v1/campaigns/C-1004", nil), http.StatusNotFound, campaignGoneBody)
	assertBody(t, alex.Do(t, http.MethodGet, "/api/v1/initiatives/I-1007", nil), http.StatusNotFound, initiativeGoneBody)
	var tree planCampaigns
	alex.Do(t, http.MethodGet, "/api/v1/plans/P-1003/campaigns", nil).Decode(t, &tree)
	if tree.Total != 0 || tree.Totals.Initiatives != 0 {
		t.Errorf("plan tree after the delete = %+v", tree)
	}

	assertStatus(t, alex.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+entry.ID+"/undo", nil), http.StatusOK)

	var after campaignAnswer
	alex.Do(t, http.MethodGet, "/api/v1/campaigns/C-1004", nil).Decode(t, &after)
	if !reflect.DeepEqual(after, before) {
		t.Errorf("after Undo =\n%+v\nwant\n%+v", after, before)
	}
}

// Deleting an initiative answers its recycle-bin entry with its path; one that no longer exists answers 404
// (FR-028).
func TestDeleteInitiative_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, "alex.morgan@shell.com")

	response := alex.Do(t, http.MethodDelete, "/api/v1/initiatives/I-1007", nil)

	assertStatus(t, response, http.StatusOK)
	var result deleteResult
	response.Decode(t, &result)
	entry := result.RecycleBinEntry
	wantPath := []pathStep{globalBrand, {Type: "campaign", ID: "C-1004", Name: "Brand platform"}}
	if entry.RecordType != "initiative" || entry.RecordID != "I-1007" || entry.RecordName != "Brand film" ||
		!reflect.DeepEqual(entry.Path, wantPath) || entry.Contents == nil || len(entry.Contents) != 0 {
		t.Errorf("recycle-bin entry = %+v", entry)
	}
	var brand campaignAnswer
	alex.Do(t, http.MethodGet, "/api/v1/campaigns/C-1004", nil).Decode(t, &brand)
	if brand.Counts.Initiatives != 1 {
		t.Errorf("Brand platform has %d initiatives after the delete, want 1", brand.Counts.Initiatives)
	}

	assertBody(t, alex.Do(t, http.MethodDelete, "/api/v1/initiatives/I-1007", nil), http.StatusNotFound, initiativeGoneBody)
	assertBody(t, alex.Do(t, http.MethodDelete, "/api/v1/campaigns/C-9999", nil), http.StatusNotFound, campaignGoneBody)
}
