package purchaserequest_test

import (
	"net/http"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/apitest"
	"mapp/internal/purchaserequest"
)

// lineItemForm is Stand build's form as stored, with another planned cost.
func lineItemForm(t *testing.T, c *apitest.Client, plannedCost string) map[string]any {
	t.Helper()
	var d struct {
		LineItem struct {
			Name, Description string
			Owner             *struct{ ID string }
			Spread            struct{ Method string }
			Attributes        map[string]any
			RequestSwbs       bool
			FinanceBudgetYear *int
			FinanceBudgetItem *struct{ ID string }
			Version           int
		}
	}
	c.Do(t, http.MethodGet, "/api/v1/line-items/"+standBuild, nil).Decode(t, &d)
	li := d.LineItem
	return map[string]any{"name": li.Name, "ownerId": li.Owner.ID, "description": li.Description, "currency": "EUR",
		"plannedCost": plannedCost, "spread": map[string]any{"method": li.Spread.Method, "months": []any{}},
		"attributes": li.Attributes, "contractedAmount": nil, "requestSwbs": li.RequestSwbs,
		"financeBudgetYear": li.FinanceBudgetYear, "financeBudgetItemId": li.FinanceBudgetItem.ID, "version": li.Version}
}

// saveBody is the input of a stored purchase request with its first line's amount changed and then.
func saveBody(pr purchaserequest.View, amount, then string) map[string]any {
	b := body(standBuild, "EUR", amount)
	l := b["lines"].([]map[string]any)[0]
	l["id"], l["overallLimit"] = pr.Lines[0].ID, amount
	b["version"], b["then"] = pr.Version, then
	return b
}

func TestModification_ApproveAndRestore_Contract_032_FR036(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	pr := approved(t, s)
	priya, daniel := s.SignIn(t, priyasEmail), s.SignIn(t, danielsEmail)

	saved := priya.Do(t, http.MethodPut, prs+"/"+pr.ID, saveBody(pr, "8000.00", "save"))
	changed := decoded[purchaserequest.View](t, priya.Do(t, http.MethodPut, prs+"/"+pr.ID, saveBody(pr, "8000.00", "submitChanges")),
		http.StatusOK)

	same(t, "032 FR-036 saving an approved PR without Submit changes", http.StatusBadRequest, saved.Status)
	same(t, "032 FR-036 change in review", []any{purchaserequest.StatusInReview, true, "9,200.00 EUR", true},
		[]any{changed.Status, changed.ChangePending, approvedAmount(changed), changed.Lines[0].Changed})
	at := decoded[purchaserequest.ApprovalView](t, priya.Do(t, http.MethodGet, prs+"/"+pr.ID+"/approval", nil), http.StatusOK)
	decide(t, daniel, "decline", at)
	restored := get(t, priya, pr.ID)
	same(t, "032 FR-037 declined change", []any{purchaserequest.StatusApproved, false, "9,200.00 EUR"},
		[]any{restored.Status, restored.ChangePending, restored.Lines[0].Amount.Format()})

	again := decoded[purchaserequest.View](t, priya.Do(t, http.MethodPut, prs+"/"+pr.ID, saveBody(restored, "8000.00", "submitChanges")),
		http.StatusOK)
	at = decoded[purchaserequest.ApprovalView](t, priya.Do(t, http.MethodGet, prs+"/"+pr.ID+"/approval", nil), http.StatusOK)
	decide(t, daniel, "approve", at)
	done := get(t, priya, pr.ID)
	same(t, "032 FR-037 approved change", []any{purchaserequest.StatusInReview, purchaserequest.StatusApproved, "8,000.00 EUR",
		purchaserequest.RoutingModification}, []any{again.Status, done.Status, done.Lines[0].Amount.Format(), done.Routings[0].Type})
}

func approvedAmount(v purchaserequest.View) string {
	if v.ApprovedVersion == nil {
		return ""
	}
	return v.ApprovedVersion.Lines[0].Amount.Format()
}

func TestClose_ReleasesWhatWasNotInvoiced_Contract_032_FR041(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	pr := approved(t, s)
	priya := s.SignIn(t, priyasEmail)

	preview := decoded[purchaserequest.ClosePreview](t, priya.Do(t, http.MethodGet, prs+"/"+pr.ID+"/close-preview", nil), http.StatusOK)
	closed := decoded[purchaserequest.View](t, priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/close", map[string]any{"version": pr.Version}),
		http.StatusOK)
	changeAfter := decoded[apiError](t, priya.Do(t, http.MethodPut, prs+"/"+pr.ID, saveBody(closed, "1.00", "submitChanges")),
		http.StatusConflict)
	resent := decoded[purchaserequest.View](t, priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/resend", map[string]any{"version": closed.Version}),
		http.StatusOK)

	same(t, "preview", []any{true, "9,200.00 EUR", "0.00 EUR"},
		[]any{preview.Available, preview.Lines[0].Released.Format(), preview.Lines[0].ClosedAmount.Format()})
	no := "Closed purchase requests cannot be changed."
	same(t, "closed", []any{purchaserequest.StatusClosed, "0.00 EUR", purchaserequest.RoutingClose, no, no},
		[]any{closed.Status, closed.Lines[0].Amount.Format(), closed.Routings[0].Type, closed.Notice, changeAfter.Error.Message})
	same(t, "032 FR-033 resend", []any{3, purchaserequest.RoutingResend}, []any{len(resent.Routings), resent.Routings[0].Type})
}

func TestDelete_ByStatus_AndRestore_Contract_032_FR042(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	approvedPR := approved(t, s)
	priya := s.SignIn(t, priyasEmail)

	refused := decoded[apiError](t, priya.Do(t, http.MethodDelete, prs+"/"+approvedPR.ID, nil), http.StatusConflict)
	created := create(t, priya, body(mediaBooking, "USD", "30000.00"))
	deleted := decoded[history.DeleteResult](t, priya.Do(t, http.MethodDelete, prs+"/"+created.ID, nil), http.StatusOK)
	taken := create(t, priya, body(mediaBooking, "USD", "1.00"))
	restore := decoded[apiError](t, priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+deleted.RecycleBinEntry.ID+"/undo", nil),
		http.StatusConflict)

	if refused.Error.Message != "Approved purchase requests cannot be deleted — close them instead." {
		t.Errorf("032 FR-040 = %+v", refused)
	}
	if deleted.RecycleBinEntry.RecordType != "purchaseRequest" || taken.Status != purchaserequest.StatusCreated {
		t.Errorf("032 FR-042 deleted %+v", deleted.RecycleBinEntry)
	}
	want := "Can't restore — line item L-1005 has only 29,999.00 USD left to request."
	if restore.Error.Code != "cannot_restore" || restore.Error.Message != want {
		t.Errorf("032 FR-042 restore = %+v", restore)
	}
}

func TestGuards_KeepWhatAPurchaseRequestDrawsOn_Contract_032_FR019(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	priya := s.SignIn(t, priyasEmail)
	pr := create(t, priya, body(standBuild, "EUR", "5000.00"))

	item := priya.Do(t, http.MethodDelete, "/api/v1/line-items/"+standBuild, nil)
	act := priya.Do(t, http.MethodDelete, "/api/v1/activities/A-1003", nil)
	lower := decoded[apiError](t, priya.Do(t, http.MethodPatch, "/api/v1/line-items/"+standBuild,
		lineItemForm(t, priya, "4000.00")), http.StatusBadRequest)

	for name, r := range map[string]*apitest.Response{"line item": item, "activity": act} {
		body := decoded[apiError](t, r, http.StatusConflict)
		if body.Error.Code != "in_use" || body.Error.Message != "Can't delete — purchase request "+pr.ID+
			" draws on line item L-1006. Delete or close the purchase request first." {
			t.Errorf("delete %s = %+v", name, body)
		}
	}
	want := "Purchase requests " + pr.ID + " already request 5,000.00 EUR of this line item; the planned cost cannot be lower."
	if len(lower.Error.Fields) != 1 || lower.Error.Fields[0].Field != "plannedCost" || lower.Error.Fields[0].Message != want {
		t.Errorf("lower planned cost = %+v", lower)
	}
}

func TestGuards_CampaignKeepsItsPlan_Contract_032_FR019(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	s.GrantRole(t, priyasEmail, "globalPlanner", "all")
	priya := s.SignIn(t, priyasEmail)
	pr := create(t, priya, body(standBuild, "EUR", "100.00"))

	moved := decoded[apiError](t, priya.Do(t, http.MethodPost, "/api/v1/campaigns/C-1001/move",
		map[string]any{"planId": "P-1002", "version": 1}), http.StatusConflict)

	if moved.Error.Code != "in_use" || moved.Error.Message != "Can't move — purchase request "+pr.ID+" draws on a line item of this campaign." {
		t.Errorf("move = %+v", moved)
	}
}
