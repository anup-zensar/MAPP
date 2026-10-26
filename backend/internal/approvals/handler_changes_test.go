package approvals_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/activities"
	"mapp/internal/approvals"
	"mapp/internal/history"
	"mapp/internal/lineitems"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// User story 5 — keep approval meaningful when things change, through the real endpoints of features 006
// and 007 on the demo data.

// socialAlwaysOn is the seeded Approved Lubricants activity, approved by Daniel Okoro in round 1, with the
// line items Venue hire (L-1003, GBP) and Speaker fees (L-1004).
const socialAlwaysOn = "A-1002"

// Mei Lin recalls the Mobility activity waiting at level 2 with a reason: it is Created, its round is
// recalled at level 2 with the reason, and it leaves Alex Morgan's Waiting for me. Recalling it again
// explains what happened; an Approved activity can't be recalled (FR-021, FR-018).
func TestRecall_WaitingMobilityActivity_FR021(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	mei := server.SignIn(t, meisEmail)
	server.GrantRole(t, meisEmail, "globalPlanner", "all") // 014: every market's plans
	alex := server.SignIn(t, alexsEmail)
	recall := map[string]any{"roundId": "AR-1008", "reason": "Wrong workflow — use the fleet one."}

	response := mei.Do(t, http.MethodPost, "/api/v1/activities/"+fleetCardLaunch+"/approval/recall", recall)
	again := mei.Do(t, http.MethodPost, "/api/v1/activities/"+fleetCardLaunch+"/approval/recall", map[string]any{"roundId": "AR-1008"})

	approval := decodeApproval(t, response)
	round := approval.Rounds[0]
	if approval.Status != "created" || round.Outcome != approvals.OutcomeRecalled || round.Levels[1].State != "notDecided" {
		t.Errorf("approval %s, round %s, level 2 %s; want Created, recalled, level 2 not decided", approval.Status, round.Outcome, round.Levels[1].State)
	}
	if entry := round.History[0]; entry.Action != "recalled" || entry.Level != 2 || entry.Actor.Name != "Mei Lin" ||
		entry.Comment != "Wrong workflow — use the fleet one." {
		t.Errorf("latest history entry = %+v, want Mei Lin's recall at level 2 with the reason", entry)
	}
	if waiting := rowIDs(decodeRows(t, alex.Do(t, http.MethodGet, "/api/v1/approvals?tab=waiting", nil))); len(waiting) != 0 {
		t.Errorf("Alex's waiting = %q, want none", waiting)
	}
	assertError(t, again, http.StatusConflict, "version_conflict", "The submission was recalled by Mei Lin")

	approved := decodeApproval(t, mei.Do(t, http.MethodGet, "/api/v1/activities/"+socialAlwaysOn+"/approval", nil))
	refused := mei.Do(t, http.MethodPost, "/api/v1/activities/"+socialAlwaysOn+"/approval/recall", map[string]any{"roundId": approved.Rounds[0].ID})
	assertError(t, refused, http.StatusBadRequest, "bad_request",
		"Approved activities can't be recalled. Changing its planned cost, dates or line-item amounts sends it back for approval.")
	gone := mei.Do(t, http.MethodPost, "/api/v1/activities/A-9999/approval/recall", recall)
	assertError(t, gone, http.StatusNotFound, "not_found", "This activity no longer exists — someone may have deleted it.")
}

// Changing the planned cost through 006's edit, confirmed, returns a Submitted and an Approved activity to
// Created: the open round ends as restarted with the change; the approved round stays approved and gets the
// restart in its history (FR-022, US5 scenario 1).
func TestPlannedCostChangeRestartsApproval_FR022(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	alex := server.SignIn(t, alexsEmail)

	submitted := editActivity(t, priya, tradeMagazineAds, func(u *activities.Update) { u.PlannedCost = usd("35000.00") })
	approved := editActivity(t, alex, socialAlwaysOn, func(u *activities.Update) { u.PlannedCost = usd("90000.00") })

	if submitted.Status != activities.StatusCreated || approved.Status != activities.StatusCreated {
		t.Errorf("statuses %s, %s; want both Created", submitted.Status, approved.Status)
	}
	open := decodeApproval(t, priya.Do(t, http.MethodGet, "/api/v1/activities/"+tradeMagazineAds+"/approval", nil))
	if round := open.Rounds[0]; round.Number != 2 || round.Outcome != approvals.OutcomeRestarted || round.History[0].Action != "restarted" ||
		round.History[0].Actor.Name != "Priya Raman" || !slices.Equal(changeTexts(round.History[0]),
		[]string{"Planned cost changed from 30,000.00 USD to 35,000.00 USD"}) {
		t.Errorf("round 2 = %s, latest entry %+v; want restarted with the planned cost change", round.Outcome, round.History[0])
	}
	closed := decodeApproval(t, priya.Do(t, http.MethodGet, "/api/v1/activities/"+socialAlwaysOn+"/approval", nil))
	if round := closed.Rounds[0]; round.Outcome != approvals.OutcomeApproved || round.History[0].Action != "restarted" ||
		round.History[0].Level != 0 || !slices.Equal(changeTexts(round.History[0]), []string{"Planned cost changed from 80,000.00 USD to 90,000.00 USD"}) {
		t.Errorf("approved round = %s, latest entry %+v; want still approved, with the restart", round.Outcome, round.History[0])
	}
	if rows := rowIDs(decodeRows(t, priya.Do(t, http.MethodGet, "/api/v1/approvals?tab=open", nil))); slices.Contains(rows, tradeMagazineAds) {
		t.Errorf("open = %q, want %s gone", rows, tradeMagazineAds)
	}
}

// Changing a line item's currency through 007's edit, confirmed, returns the Approved activity to Created;
// its approval history holds 007's sentence about the change (FR-022, US5 scenario 2).
func TestLineItemCurrencyChangeRestartsApproval_FR022(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	form := lineItemForm(t, priya, "L-1003") // Venue hire, 15,800.00 GBP
	form["currency"], form["restartApproval"] = "EUR", true

	response := priya.Do(t, http.MethodPatch, "/api/v1/line-items/L-1003", form)

	if response.Status != http.StatusOK {
		t.Fatalf("PATCH L-1003 = %d %s, want 200", response.Status, response.Body)
	}
	approval := decodeApproval(t, priya.Do(t, http.MethodGet, "/api/v1/activities/"+socialAlwaysOn+"/approval", nil))
	latest := approval.Rounds[0].History[0]
	want := []string{"Planned cost of line item 'Venue hire' (L-1003): 15,800.00 GBP → 15,800.00 EUR"}
	if approval.Status != "created" || latest.Action != "restarted" || !slices.Equal(changeTexts(latest), want) {
		t.Errorf("approval %s, latest entry %+v; want Created with the restart %q", approval.Status, latest, want)
	}
}

// Deleting a Submitted activity takes it off the approvals page; Undo brings it back Submitted with the same
// open round (D-031); a later Restore from the recycle bin brings it back Created and ends that round as
// cancelled, dated at the delete and by the person who deleted it, and the restore says so (FR-023,
// 008-P3).
func TestDeleteSubmittedActivity_UndoKeepsRound_RestoreCancels_FR023(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	mei := server.SignIn(t, meisEmail)
	server.GrantRole(t, meisEmail, "globalPlanner", "all") // 014: every market's plans
	daniel := server.SignIn(t, danielsEmail)
	path := "/api/v1/activities/" + tradeMagazineAds

	undoEntry := deleteActivity(t, priya, tradeMagazineAds)
	whileDeleted := rowIDs(decodeRows(t, daniel.Do(t, http.MethodGet, "/api/v1/approvals?tab=open", nil)))
	undone := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+undoEntry+"/undo", nil)
	afterUndo := decodeApproval(t, daniel.Do(t, http.MethodGet, path+"/approval", nil))
	server.Advance(time.Hour)
	deletedAt := now.Add(time.Hour)
	restoreEntry := deleteActivity(t, mei, tradeMagazineAds)
	server.Advance(24 * time.Hour)
	priya, daniel = server.SignIn(t, priyasEmail), server.SignIn(t, danielsEmail) // the day ended their sessions
	restored := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+restoreEntry+"/restore", nil)
	afterRestore := decodeApproval(t, daniel.Do(t, http.MethodGet, path+"/approval", nil))

	if slices.Contains(whileDeleted, tradeMagazineAds) {
		t.Errorf("open while deleted = %q, want %s gone", whileDeleted, tradeMagazineAds)
	}
	if undone.Status != http.StatusOK || afterUndo.Status != "submitted" || afterUndo.Rounds[0].ID != "AR-1004" ||
		afterUndo.Rounds[0].Outcome != approvals.OutcomeOpen || !afterUndo.Viewer.IsCurrentApprover {
		t.Errorf("after Undo: %d, %s, round %s %s; want Submitted with AR-1004 still open", undone.Status, afterUndo.Status,
			afterUndo.Rounds[0].ID, afterUndo.Rounds[0].Outcome)
	}
	var result history.RestoreResult
	if restored.Status != http.StatusOK {
		t.Fatalf("restore = %d %s, want 200", restored.Status, restored.Body)
	}
	restored.Decode(t, &result)
	if !strings.HasSuffix(result.Details, "Approval round 2 was cancelled because the activity was deleted — submit it for approval again.") {
		t.Errorf("restore details = %q, want the note about round 2", result.Details)
	}
	round := afterRestore.Rounds[0]
	if afterRestore.Status != "created" || round.ID != "AR-1004" || round.Outcome != approvals.OutcomeCancelled ||
		round.EndedAt == nil || !round.EndedAt.Equal(deletedAt) || round.Levels[0].State != "notDecided" {
		t.Errorf("after Restore: %s, round %+v; want Created with AR-1004 cancelled at the delete", afterRestore.Status, round)
	}
	if entry := round.History[0]; entry.Action != "cancelled" || entry.Actor.Name != "Mei Lin" || !entry.At.Equal(deletedAt) {
		t.Errorf("latest history entry = %+v, want cancelled by Mei Lin at the delete", entry)
	}
	if rows := rowIDs(decodeRows(t, daniel.Do(t, http.MethodGet, "/api/v1/approvals?tab=waiting", nil))); len(rows) != 0 {
		t.Errorf("Daniel's waiting = %q, want none", rows)
	}
}

// editActivity saves the activity id through 006's PATCH as client, with change applied to its whole form
// and the re-approval confirmed, and returns the activity after the save.
func editActivity(t *testing.T, client *apitest.Client, id string, change func(u *activities.Update)) activities.View {
	t.Helper()
	form := activityForm(t, client, id)
	change(&form)
	form.RestartApproval = true
	response := client.Do(t, http.MethodPatch, "/api/v1/activities/"+id, form)
	if response.Status != http.StatusOK {
		t.Fatalf("PATCH %s = %d %s, want 200", id, response.Status, response.Body)
	}
	var view activities.View
	response.Decode(t, &view)
	return view
}

// activityForm reads the activity id and returns its whole editable state, as 006's edit form sends it.
func activityForm(t *testing.T, client *apitest.Client, id string) activities.Update {
	t.Helper()
	response := client.Do(t, http.MethodGet, "/api/v1/activities/"+id, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s = %d %s", id, response.Status, response.Body)
	}
	var v activities.View
	response.Decode(t, &v)
	form := activities.Update{
		Name: v.Name, ExternalName: v.ExternalName, ActivityTypeID: v.ActivityType.ID, Ongoing: v.Ongoing,
		Description: v.Description, Attributes: settings.AttributeValues{}, Version: &v.Version,
		Goals: activities.GoalsInput{INQ: v.Goals.INQ, SetLaterStagesMyself: v.Goals.SetLaterStagesMyself},
	}
	for attributeID, value := range v.Attributes {
		form.Attributes[attributeID] = value
	}
	if v.Owner != nil {
		form.OwnerID = &v.Owner.ID
	}
	if v.StartDate != nil {
		text := v.StartDate.String()
		form.StartDate = &text
	}
	if v.EndDate != nil {
		text := v.EndDate.String()
		form.EndDate = &text
	}
	if v.PlannedCost != nil {
		form.PlannedCost = usd(v.PlannedCost.Amount.String())
	}
	if v.ApprovalWorkflow != nil {
		form.ApprovalWorkflowID = &v.ApprovalWorkflow.ID
	}
	if v.Goals.SetLaterStagesMyself {
		form.Goals.MQL = v.Goals.MQL
		if v.Goals.Revenue != nil {
			form.Goals.Revenue = usd(v.Goals.Revenue.Amount.String())
		}
	}
	return form
}

// lineItemForm reads the line item id and returns its whole form, as 007's edit page sends it.
func lineItemForm(t *testing.T, client *apitest.Client, id string) map[string]any {
	t.Helper()
	response := client.Do(t, http.MethodGet, "/api/v1/line-items/"+id, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET %s = %d %s", id, response.Status, response.Body)
	}
	var details lineitems.Details
	response.Decode(t, &details)
	v := details.LineItem
	months := []any{}
	for _, m := range v.Spread.Months {
		if v.Spread.Method == lineitems.SpreadCustom {
			months = append(months, map[string]any{"month": m.Month, "amount": m.Amount.Amount.String()})
		}
	}
	form := map[string]any{
		"name": v.Name, "ownerId": nil, "description": v.Description, "currency": v.Currency.Code,
		"plannedCost": v.PlannedCost.Amount.String(), "spread": map[string]any{"method": v.Spread.Method, "months": months},
		"attributes": v.Attributes, "contractedAmount": nil, "requestSwbs": v.RequestSwbs,
		"financeBudgetYear": v.FinanceBudgetYear, "financeBudgetItemId": nil, "version": v.Version,
	}
	if v.Owner != nil {
		form["ownerId"] = v.Owner.ID
	}
	if v.ContractedAmount != nil {
		form["contractedAmount"] = v.ContractedAmount.Amount.String()
	}
	if v.FinanceBudgetItem != nil {
		form["financeBudgetItemId"] = v.FinanceBudgetItem.ID
	}
	return form
}

// deleteActivity deletes the activity id through 006's DELETE as client and returns its recycle-bin entry.
func deleteActivity(t *testing.T, client *apitest.Client, id string) string {
	t.Helper()
	response := client.Do(t, http.MethodDelete, "/api/v1/activities/"+id, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("DELETE %s = %d %s, want 200", id, response.Status, response.Body)
	}
	var result history.DeleteResult
	response.Decode(t, &result)
	return result.RecycleBinEntry.ID
}

// usd is an amount in US dollars as a request sends it.
func usd(amount string) *money.Input {
	return &money.Input{Amount: amount, Currency: "USD"}
}

// changeTexts lists the change sentences of a history entry.
func changeTexts(e approvals.HistoryEntryView) []string {
	texts := []string{}
	for _, c := range e.Changes {
		texts = append(texts, c.Text)
	}
	return texts
}
