package campaigns_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"northstar/internal/platform/apitest"
)

// The move and copy operations of US6 on the demo data: FY2026 Lubricants Europe (P-1001) holds Helix
// launch (C-1001, 2026) with Digital & social (I-1001, 1 Jan – 30 Jun) and Print & outdoor (I-1002), and
// Rimula fleet (C-1002, 1 Feb – 31 Dec) with Events (I-1003, 1 Mar – 30 Sep) and Retail events (I-1004);
// FY2026 Mobility China (P-1002) is the other plan of 2026, with another model and business unit.

// destination is the part of a Destination answer these tests read.
type destination struct {
	ID                string  `json:"id"`
	Current           bool    `json:"current"`
	Available         bool    `json:"available"`
	UnavailableReason *string `json:"unavailableReason"`
	SuggestedName     *string `json:"suggestedName"`
	Consequences      *struct {
		ModelName string `json:"modelName"`
	} `json:"consequences"`
}

// destinationList is a DestinationList answer.
type destinationList struct {
	Items []destination `json:"items"`
	Total int           `json:"total"`
	Rule  string        `json:"rule"`
}

// ref is a record of an answer by its id.
type ref struct {
	ID string `json:"id"`
}

// counted is the part of a Campaign or Initiative answer with what is below it.
type counted struct {
	ID          string `json:"id"`
	PlanID      string `json:"planId"`
	CampaignID  string `json:"campaignId"`
	Name        string `json:"name"`
	Version     int    `json:"version"`
	Initiatives []ref  `json:"initiatives"`
	Counts      struct {
		Initiatives        int `json:"initiatives"`
		Activities         int `json:"activities"`
		ApprovedActivities int `json:"approvedActivities"`
		LineItems          int `json:"lineItems"`
	} `json:"counts"`
}

// A campaign of 2026 moves only to the other plan of 2026, with the consequences of its other model and
// business unit; a copy may also go into its own plan, first, with a suggested name; any other purpose is
// refused (FR-023, FR-024, FR-026).
func TestCampaignDestinations_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	var move, copies destinationList
	moveAnswer := priya.Do(t, http.MethodGet, "/api/campaigns/C-1001/destinations?purpose=move", nil)
	moveAnswer.Decode(t, &move)
	copyAnswer := priya.Do(t, http.MethodGet, "/api/campaigns/C-1001/destinations?purpose=copy", nil)
	copyAnswer.Decode(t, &copies)
	other := priya.Do(t, http.MethodGet, "/api/campaigns/C-1001/destinations?purpose=other", nil, apitest.SkipRequestValidation())
	gone := priya.Do(t, http.MethodGet, "/api/campaigns/C-9999/destinations?purpose=move", nil)

	assertStatus(t, moveAnswer, http.StatusOK)
	if move.Total != 1 || move.Items[0].ID != "P-1002" || !move.Items[0].Available || move.Items[0].Consequences == nil ||
		move.Items[0].Consequences.ModelName != "Mobility 1.0" || move.Rule != "Campaigns move only between plans of the same year" {
		t.Errorf("move destinations = %s", moveAnswer.Body)
	}
	if copies.Total != 2 || copies.Items[0].ID != "P-1001" || !copies.Items[0].Current || copies.Items[0].Consequences != nil ||
		*copies.Items[0].SuggestedName != "Helix launch (copy)" || copies.Items[1].ID != "P-1002" {
		t.Errorf("copy destinations = %s", copyAnswer.Body)
	}
	assertBody(t, other, http.StatusBadRequest, validationBody(`{"field":"purpose","message":"Choose move or copy."}`))
	assertBody(t, gone, http.StatusNotFound, campaignGoneBody)
}

// Moving Helix launch to FY2026 Mobility China answers the campaign there with a new version; its
// initiatives come along and the old plan's totals follow (FR-025).
func TestMoveCampaign_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPost, "/api/campaigns/C-1001/move", map[string]any{"planId": "P-1002", "version": 1})

	assertStatus(t, response, http.StatusOK)
	var moved counted
	response.Decode(t, &moved)
	if moved.PlanID != "P-1002" || moved.Version != 2 || moved.Counts.Initiatives != 2 {
		t.Errorf("moved = %s", response.Body)
	}
	var tree planCampaigns
	priya.Do(t, http.MethodGet, "/api/plans/P-1001/campaigns", nil).Decode(t, &tree)
	if tree.Totals.Campaigns != 1 || tree.Totals.Initiatives != 2 {
		t.Errorf("FY2026 Lubricants Europe now holds %d campaigns and %d initiatives, want 1 and 2", tree.Totals.Campaigns, tree.Totals.Initiatives)
	}
}

// A move to a plan of another year is refused on planId, a move of a campaign that is gone answers 404, and
// one from an out-of-date version 409 (FR-023, FR-025).
func TestMoveCampaign_Refused400(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.Advance(time.Hour)
	edit := helixForm(1)
	edit["description"] = "Premium launch"
	assertStatus(t, priya.Do(t, http.MethodPatch, "/api/campaigns/C-1001", edit), http.StatusOK)

	otherYear := priya.Do(t, http.MethodPost, "/api/campaigns/C-1001/move", map[string]any{"planId": "P-1003", "version": 2})
	gone := priya.Do(t, http.MethodPost, "/api/campaigns/C-9999/move", map[string]any{"planId": "P-1002", "version": 1})
	stale := priya.Do(t, http.MethodPost, "/api/campaigns/C-1001/move", map[string]any{"planId": "P-1002", "version": 1})

	assertBody(t, otherYear, http.StatusBadRequest,
		validationBody(`{"field":"planId","message":"Campaigns move only between plans of the same year (2026)."}`))
	assertBody(t, gone, http.StatusNotFound, campaignGoneBody)
	assertBody(t, stale, http.StatusConflict, `{"code":"version_conflict","message":"Not saved — Priya Raman changed this campaign",`+
		`"fields":[],"conflict":{"changedBy":"Priya Raman","changedAt":"2026-03-12T10:00:00Z"}}`)
}

// Copying Helix launch into its own plan answers 201 with a new campaign holding copies of its 2
// initiatives, 4 activities — all Created — and 6 line items; one log entry counts them (FR-027, R13, D-097).
func TestCopyCampaign_Contract201(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPost, "/api/campaigns/C-1001/copy", map[string]any{"planId": "P-1001", "name": "Helix launch (copy)"})
	taken := priya.Do(t, http.MethodPost, "/api/campaigns/C-1001/copy", map[string]any{"planId": "P-1001", "name": "rimula fleet"})

	assertStatus(t, response, http.StatusCreated)
	var copied counted
	response.Decode(t, &copied)
	if copied.ID == "C-1001" || copied.PlanID != "P-1001" || copied.Name != "Helix launch (copy)" || copied.Counts.Initiatives != 2 ||
		copied.Counts.Activities != 4 || copied.Counts.ApprovedActivities != 0 || copied.Counts.LineItems != 6 ||
		slices.ContainsFunc(copied.Initiatives, func(i ref) bool { return i.ID == "I-1001" || i.ID == "I-1002" }) {
		t.Errorf("copy = %s", response.Body)
	}
	assertBody(t, taken, http.StatusBadRequest,
		validationBody(`{"field":"name","message":"'FY2026 Lubricants Europe' already has a campaign called 'Rimula fleet'."}`))
	var log struct {
		Items []struct {
			Action  string `json:"action"`
			Details string `json:"details"`
		} `json:"items"`
	}
	priya.Do(t, http.MethodGet, "/api/activity-log?recordType=campaign&recordId="+copied.ID, nil).Decode(t, &log)
	if len(log.Items) != 1 || log.Items[0].Action != "copied" ||
		log.Items[0].Details != "Copied from 'Helix launch' (C-1001), with 2 initiatives, 4 activities and 6 line items" {
		t.Errorf("log of the copy = %+v", log.Items)
	}
}

// Digital & social (1 Jan – 30 Jun) does not fit in Rimula fleet (1 Feb – 31 Dec): a move lists it as
// unavailable with the reason; a copy lists its own campaign first with a suggested name (FR-023, FR-026).
func TestInitiativeDestinations_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	var move, copies destinationList
	moveAnswer := priya.Do(t, http.MethodGet, "/api/initiatives/I-1001/destinations?purpose=move", nil)
	moveAnswer.Decode(t, &move)
	copyAnswer := priya.Do(t, http.MethodGet, "/api/initiatives/I-1001/destinations?purpose=copy", nil)
	copyAnswer.Decode(t, &copies)
	gone := priya.Do(t, http.MethodGet, "/api/initiatives/I-9999/destinations?purpose=copy", nil)

	assertStatus(t, moveAnswer, http.StatusOK)
	if move.Total != 1 || move.Items[0].ID != "C-1002" || move.Items[0].Available ||
		*move.Items[0].UnavailableReason != "Dates don't fit (01 Feb – 31 Dec 2026) — change the initiative's dates first" {
		t.Errorf("move destinations = %s", moveAnswer.Body)
	}
	if copies.Total != 2 || copies.Items[0].ID != "C-1001" || !copies.Items[0].Current ||
		*copies.Items[0].SuggestedName != "Digital & social (copy)" || copies.Rule != "A copy can go into this campaign or another campaign of the same plan" {
		t.Errorf("copy destinations = %s", copyAnswer.Body)
	}
	assertBody(t, gone, http.StatusNotFound, initiativeGoneBody)
}

// Events moves from Rimula fleet to Helix launch, whose dates contain it; Digital & social does not fit in
// Rimula fleet and is refused on campaignId (FR-023, FR-025).
func TestMoveInitiative_Contract(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	moved := priya.Do(t, http.MethodPost, "/api/initiatives/I-1003/move", map[string]any{"campaignId": "C-1001", "version": 1})
	refused := priya.Do(t, http.MethodPost, "/api/initiatives/I-1001/move", map[string]any{"campaignId": "C-1002", "version": 1})

	assertStatus(t, moved, http.StatusOK)
	var events counted
	moved.Decode(t, &events)
	if events.CampaignID != "C-1001" || events.Version != 2 {
		t.Errorf("moved = %s", moved.Body)
	}
	assertBody(t, refused, http.StatusBadRequest, validationBody(
		`{"field":"campaignId","message":"Dates don't fit (01 Feb – 31 Dec 2026) — change the initiative's dates first."}`))
}

// Copying Digital & social into its own campaign answers 201 with a new initiative holding copies of its 2
// activities and 4 line items (FR-027).
func TestCopyInitiative_Contract201(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPost, "/api/initiatives/I-1001/copy", map[string]any{"campaignId": "C-1001", "name": "Digital & social (copy)"})
	gone := priya.Do(t, http.MethodPost, "/api/initiatives/I-1001/copy", map[string]any{"campaignId": "C-9999", "name": "Copy"})

	assertStatus(t, response, http.StatusCreated)
	var copied counted
	response.Decode(t, &copied)
	if copied.ID == "I-1001" || copied.CampaignID != "C-1001" || copied.Counts.Activities != 2 || copied.Counts.LineItems != 4 {
		t.Errorf("copy = %s", response.Body)
	}
	assertBody(t, gone, http.StatusBadRequest,
		validationBody(`{"field":"campaignId","message":"That campaign no longer exists — choose another one."}`))
}
