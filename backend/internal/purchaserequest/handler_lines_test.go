package purchaserequest_test

import (
	"net/http"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/apitest"
	"mapp/internal/purchaserequest"
)

func linesOnApproved(t *testing.T, s *apitest.Server, setting string) {
	t.Helper()
	alex := s.SignIn(t, alexsEmail)
	before := decoded[purchaserequest.ConfigurationView](t, alex.Do(t, http.MethodGet, "/api/v1/purchase-request-configuration", nil),
		http.StatusOK)
	decoded[purchaserequest.ConfigurationView](t, alex.Do(t, http.MethodPut, "/api/v1/purchase-request-configuration",
		map[string]any{"version": before.Version, "purchaseSupport": true, "linesOnApprovedPurchaseRequests": setting}), http.StatusOK)
}

// secondStandBuild duplicates Stand build in its approved activity and gives the copy its SWBS code.
func secondStandBuild(t *testing.T, s *apitest.Server) string {
	t.Helper()
	var copied struct{ ID string }
	s.SignIn(t, priyasEmail).Do(t, http.MethodPost, "/api/v1/line-items/"+standBuild+"/duplicate", map[string]any{}).Decode(t, &copied)
	receiveSwbsCodes(t, s)
	return copied.ID
}

// withAddedLine is a save of pr that keeps its line and adds one of 100.00 EUR on lineItemID.
func withAddedLine(pr purchaserequest.View, lineItemID, then string) map[string]any {
	b := saveBody(pr, "9200.00", then)
	added := map[string]any{"lineItemId": lineItemID, "currency": "EUR", "amount": "100.00", "deliveryDate": "2026-11-30",
		"overallLimit": "100.00", "description": "", "attributes": map[string]any{}}
	b["lines"] = append(b["lines"].([]map[string]any), added)
	return b
}

func TestAddLine_FollowsTheSetting_Contract_032_FR039(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, setting, then string
		status              int
		want                string
	}{
		{name: "not allowed", setting: "notAllowed", then: "submitChanges", status: http.StatusBadRequest,
			want: "Your organisation does not allow lines to be added to approved purchase requests."},
		{name: "with modification approval needs Submit changes", setting: "withModificationApproval", then: "save",
			status: http.StatusBadRequest, want: "Choose Submit changes."},
		{name: "without approval stays Approved", setting: "withoutApproval", then: "save", status: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := apitest.New(t)
			pr := approved(t, s)
			linesOnApproved(t, s, tt.setting)
			second := secondStandBuild(t, s)

			r := s.SignIn(t, priyasEmail).Do(t, http.MethodPut, prs+"/"+pr.ID, withAddedLine(pr, second, tt.then))

			if tt.status != http.StatusOK {
				refused := decoded[apiError](t, r, tt.status)
				if len(refused.Error.Fields) != 1 || refused.Error.Fields[0].Message != tt.want {
					t.Errorf("refusal = %+v", refused)
				}
				return
			}
			saved := decoded[purchaserequest.View](t, r, http.StatusOK)
			if saved.Status != purchaserequest.StatusApproved || len(saved.Lines) != 2 ||
				saved.Routings[0].Type != purchaserequest.RoutingModification {
				t.Errorf("saved = %+v", saved)
			}
		})
	}
}

func TestRemoveLine_RefusedOnceApproved_Contract_032_FR039(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	pr := approved(t, s)
	b := saveBody(pr, "9200.00", "submitChanges")
	b["lines"] = []map[string]any{withAddedLine(pr, secondStandBuild(t, s), "save")["lines"].([]map[string]any)[1]}

	refused := decoded[apiError](t, s.SignIn(t, priyasEmail).Do(t, http.MethodPut, prs+"/"+pr.ID, b), http.StatusBadRequest)

	want := "Lines of approved purchase requests cannot be removed — close the purchase request instead."
	if len(refused.Error.Fields) == 0 || refused.Error.Fields[0].Message != want {
		t.Errorf("refusal = %+v", refused)
	}
}

func TestRestore_DeclinedComesBackCreated_Contract_032_FR042(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	receiveSwbsCodes(t, s)
	priya, daniel := s.SignIn(t, priyasEmail), s.SignIn(t, danielsEmail)
	pr := create(t, priya, body(standBuild, "EUR", "500.00"))
	sub := decoded[purchaserequest.ApprovalView](t, priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/approval/submit",
		map[string]any{"version": pr.Version}), http.StatusOK)
	decide(t, daniel, "decline", sub)

	deleted := decoded[history.DeleteResult](t, priya.Do(t, http.MethodDelete, prs+"/"+pr.ID, nil), http.StatusOK)
	restored := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+deleted.RecycleBinEntry.ID+"/restore", nil)

	if restored.Status != http.StatusOK {
		t.Fatalf("restore = %d %s", restored.Status, restored.Body)
	}
	if back := get(t, priya, pr.ID); back.Status != purchaserequest.StatusCreated || len(back.Lines) != 1 {
		t.Errorf("restored = %+v", back)
	}
}
