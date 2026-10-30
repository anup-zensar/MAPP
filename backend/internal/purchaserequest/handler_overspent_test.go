package purchaserequest_test

import (
	"net/http"
	"testing"

	"mapp/internal/lineitem"
	"mapp/internal/platform/apitest"
	"mapp/internal/purchaserequest"
)

func bigLineItem() map[string]any {
	return map[string]any{
		"name": "Extra stands", "ownerId": "U-1002", "description": "", "currency": "USD", "plannedCost": "100000.00",
		"spread": map[string]any{"method": "evenly", "months": []any{}}, "attributes": map[string]any{},
		"contractedAmount": nil, "requestSwbs": false, "financeBudgetYear": 2026, "financeBudgetItemId": nil,
	}
}

func submissionMessages(t *testing.T, c *apitest.Client, id string) []string {
	t.Helper()
	check := decoded[purchaserequest.SubmissionView](t, c.Do(t, http.MethodGet, prs+"/"+id+"/approval/submission", nil),
		http.StatusOK)
	messages := []string{}
	for _, p := range check.Problems {
		messages = append(messages, p.Message)
	}
	return messages
}

func TestSubmit_RefusedWhileLineItemsExceedThePlannedCost_Contract_037_FR019(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	receiveSwbsCodes(t, s)
	priya := s.SignIn(t, priyasEmail)
	pr := create(t, priya, body(standBuild, "EUR", "9200.00"))
	stand := decoded[lineitem.Details](t, priya.Do(t, http.MethodGet, "/api/v1/line-items/"+standBuild, nil), http.StatusOK)
	extra := decoded[lineitem.View](t, priya.Do(t, http.MethodPost, "/api/v1/activities/"+stand.Activity.ID+"/line-items",
		bigLineItem()), http.StatusCreated)
	after := decoded[lineitem.Details](t, priya.Do(t, http.MethodGet, "/api/v1/line-items/"+standBuild, nil), http.StatusOK)
	over := after.Balance.Balance.Amount.Neg()

	refused := decoded[apiError](t, priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/approval/submit",
		map[string]any{"version": pr.Version}), http.StatusConflict)

	same(t, "037 FR-019 the overage is named with the other problems", []string{
		"Activity 'Billboard winter wave': line items exceed its planned cost by " + over.Format() + " " +
			after.Balance.Balance.Currency + " — lower them, or raise the activity's planned cost and get it approved again.",
	}, submissionMessages(t, priya, pr.ID))
	if refused.Error.Code != "not_ready" || get(t, priya, pr.ID).Status != purchaserequest.StatusCreated {
		t.Errorf("submit = %+v, want not_ready and the purchase request still Created", refused)
	}

	if r := priya.Do(t, http.MethodDelete, "/api/v1/line-items/"+extra.ID, nil); r.Status != http.StatusOK {
		t.Fatalf("delete the extra line item = %d %s", r.Status, r.Body)
	}
	same(t, "037 FR-019 back within the planned cost, the gate no longer stops it", []string{},
		submissionMessages(t, priya, pr.ID))
}

func TestApprove_OverPlannedCost_Contract_038_FR019_FR020_M18(t *testing.T) {
	t.Parallel()
	for _, round := range []string{"purchase request", "change"} {
		t.Run(round, func(t *testing.T) {
			t.Parallel()
			s := apitest.New(t)
			priya, daniel, alex := s.SignIn(t, priyasEmail), s.SignIn(t, danielsEmail), s.SignIn(t, alexsEmail)
			pr := submittedRound(t, s, round)
			decoded[lineitem.View](t, priya.Do(t, http.MethodPost, "/api/v1/activities/A-1003/line-items", bigLineItem()),
				http.StatusCreated)

			warned := decoded[purchaserequest.ApprovalView](t, daniel.Do(t, http.MethodGet, prs+"/"+pr.ID+"/approval", nil),
				http.StatusOK)
			decoded[map[string]any](t, alex.Do(t, http.MethodPut, "/api/v1/overspend-configuration",
				map[string]any{"version": 1, "defaultMode": "warn", "approvalOverPlannedCost": "refuse"}), http.StatusOK)
			refusedView := decoded[purchaserequest.ApprovalView](t, daniel.Do(t, http.MethodGet, prs+"/"+pr.ID+"/approval", nil),
				http.StatusOK)
			at := openRound(t, refusedView)
			refused := decoded[apiError](t, daniel.Do(t, http.MethodPost, prs+"/"+pr.ID+"/approval/approve",
				map[string]any{"roundId": at.ID, "level": at.CurrentLevel}), http.StatusConflict)

			if len(warned.OverPlannedCost) != 1 || warned.OverPlannedCost[0].Activity.ID != "A-1003" {
				t.Errorf("038 FR-019 overages = %+v, want A-1003", warned.OverPlannedCost)
			}
			message := "Line items of activity 'Billboard winter wave' are " + warned.OverPlannedCost[0].OverBy.Format() +
				" above its planned cost — ask the owner to lower them, or decline."
			same(t, "038 FR-020 refused", []string{"not_allowed", message, message},
				[]string{refused.Error.Code, refused.Error.Message, refusedView.Sections[0].Actions.Approve.Reason})
			decide(t, daniel, "decline", refusedView)
		})
	}
}

// submittedRound is a purchase request on Stand build waiting for Daniel: its first round, or a change of it.
func submittedRound(t *testing.T, s *apitest.Server, round string) purchaserequest.View {
	t.Helper()
	priya := s.SignIn(t, priyasEmail)
	if round == "change" {
		pr := approved(t, s)
		return decoded[purchaserequest.View](t, priya.Do(t, http.MethodPut, prs+"/"+pr.ID, saveBody(pr, "8000.00", "submitChanges")),
			http.StatusOK)
	}
	receiveSwbsCodes(t, s)
	pr := create(t, priya, body(standBuild, "EUR", "9200.00"))
	decoded[purchaserequest.ApprovalView](t, priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/approval/submit",
		map[string]any{"version": pr.Version}), http.StatusOK)
	return pr
}
