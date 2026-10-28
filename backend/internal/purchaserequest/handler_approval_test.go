package purchaserequest_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/apitest"
	"mapp/internal/purchaserequest"
)

func TestSubmit_Gates_Contract_032_FR021(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	priya := s.SignIn(t, priyasEmail)
	pr := create(t, priya, body(mediaBooking, "USD", "1000.00"))

	check := decoded[purchaserequest.SubmissionView](t, priya.Do(t, http.MethodGet, prs+"/"+pr.ID+"/approval/submission", nil),
		http.StatusOK)
	refused := decoded[apiError](t, priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/approval/submit",
		map[string]any{"version": pr.Version}), http.StatusConflict)

	var messages []string
	for _, p := range check.Problems {
		messages = append(messages, p.Message)
	}
	want := []string{"Activity 'Trade magazine ads' must be approved before this purchase request can be submitted.",
		"Line item L-1005 'Media booking' has no SWBS code yet (Waiting for approval)."}
	if check.Ready || !slices.Equal(messages, want) || check.Kind != "submit" || check.RoundKind != purchaserequest.KindPurchaseRequest {
		t.Errorf("submission = %+v", check)
	}
	if refused.Error.Code != "not_ready" || len(refused.Error.Missing) != 2 {
		t.Errorf("submit = %+v", refused)
	}
	if again := get(t, priya, pr.ID); again.Status != purchaserequest.StatusCreated {
		t.Errorf("status after a refused submission = %s", again.Status)
	}
}

func TestApprove_RoutesToTheRequisitioner_Contract_032_FR030(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	pr := approved(t, s)
	page := decoded[purchaserequest.ListPage](t, s.SignIn(t, leasEmail).Do(t, http.MethodGet, prs+"?waitingForGsap=true", nil),
		http.StatusOK)

	got := []string{string(pr.Status), fmt.Sprint(pr.WaitingForGSAP), fmt.Sprint(pr.Lines[0].SentOn)}
	for _, r := range pr.Routings {
		got = append(got, fmt.Sprint(r.Type, " by ", r.RoutedBy.Name, " to ", r.Recipients))
	}
	for _, row := range page.Items {
		got = append(got, "waiting: "+row.ID)
	}
	same(t, "032 FR-030 FR-032 approved, routed once and waiting for GSAP",
		[]string{"approved", "true", "2026-03-12", "original by Daniel Okoro to [DE_LUBES]", "waiting: " + pr.ID}, got)
}

func TestApprove_PRForm_Contract_032_FR031(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	pr := approved(t, s)
	lea := s.SignIn(t, leasEmail)
	form := prs + "/" + pr.ID + "/routings/" + pr.Routings[0].ID + "/form?format="

	csv := lea.Do(t, http.MethodGet, form+"csv", nil)
	xlsx := lea.Do(t, http.MethodGet, form+"xlsx", nil)

	for _, want := range []string{"HX1006,Stand for the winter wave,Northlight Events GmbH,V-100101,GB01000105,SIM-GB01-0001",
		"9200.00,EUR,2026-11-30,Daniel Okoro,daniel.okoro@shell.com,Original,No", "Routed by MAPP — no supporting documents are attached"} {
		if !strings.Contains(string(csv.Body), want) {
			t.Errorf("csv lacks %q:\n%s", want, csv.Body)
		}
	}
	same(t, "answers", []any{http.StatusOK, http.StatusOK, true}, []any{csv.Status, xlsx.Status, strings.HasPrefix(string(xlsx.Body), "PK")})
}

func TestDecline_KeepsTheReservation_ThenResubmit_Contract_032_FR025(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	receiveSwbsCodes(t, s)
	priya, daniel := s.SignIn(t, priyasEmail), s.SignIn(t, danielsEmail)
	pr := create(t, priya, body(standBuild, "EUR", "9200.00"))
	sub := decoded[purchaserequest.ApprovalView](t, priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/approval/submit",
		map[string]any{"version": pr.Version, "comment": "Please"}), http.StatusOK)

	declined := decide(t, daniel, "decline", sub)
	section := decoded[purchaserequest.LineItemSection](t, priya.Do(t, http.MethodGet, "/api/v1/line-items/"+standBuild+
		"/purchase-requests", nil), http.StatusOK)
	again := decoded[purchaserequest.ApprovalView](t, priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/approval/submit",
		map[string]any{"version": get(t, priya, pr.ID).Version}), http.StatusOK)
	recalled := decoded[purchaserequest.ApprovalView](t, priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/approval/recall",
		map[string]any{"roundId": openRound(t, again).ID}), http.StatusOK)

	if declined.Status != purchaserequest.StatusDeclined || section.AvailableToRequest.Format() != "0.00 EUR" {
		t.Errorf("032 FR-017 declined %s, available %s", declined.Status, section.AvailableToRequest.Format())
	}
	if again.Status != purchaserequest.StatusSubmitted || recalled.Status != purchaserequest.StatusCreated {
		t.Errorf("032 FR-023 FR-024 re-submitted %s, recalled %s", again.Status, recalled.Status)
	}
}

func TestSelfApproval_Accepted_Contract_032_FR027(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	receiveSwbsCodes(t, s)
	s.GrantRole(t, priyasEmail, "approver", "myTeams")
	priya := s.SignIn(t, priyasEmail)
	b := body(standBuild, "EUR", "9200.00")
	b["moaApproverId"] = "U-1002"
	pr := create(t, priya, b)
	sub := decoded[purchaserequest.ApprovalView](t, priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/approval/submit",
		map[string]any{"version": pr.Version}), http.StatusOK)

	if done := decide(t, priya, "approve", sub); done.Status != purchaserequest.StatusApproved {
		t.Errorf("self-approval = %+v", done)
	}
}

func TestApprovalsInbox_ListsThePurchaseRequest_Contract_032_FR029(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	receiveSwbsCodes(t, s)
	priya, daniel := s.SignIn(t, priyasEmail), s.SignIn(t, danielsEmail)
	pr := create(t, priya, body(standBuild, "EUR", "9200.00"))
	priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/approval/submit", map[string]any{"version": pr.Version})

	inbox := daniel.Do(t, http.MethodGet, "/api/v1/approvals?tab=open", nil)
	preview := decoded[purchaserequest.PreviewView](t, daniel.Do(t, http.MethodGet, "/api/v1/approvals/purchase-requests/"+pr.ID, nil),
		http.StatusOK)

	if inbox.Status != http.StatusOK || !strings.Contains(string(inbox.Body), pr.ID) {
		t.Errorf("inbox = %d %s", inbox.Status, inbox.Body)
	}
	if preview.PurchaseRequest.ID != pr.ID || !preview.Approval.Sections[0].Actions.Approve.Available {
		t.Errorf("preview = %+v", preview)
	}
}

func TestApprovalSection_SubmitFollowsThePurchaseRequest_Contract_032_FR022(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	pr := approved(t, s)
	priya := s.SignIn(t, priyasEmail)
	created := create(t, priya, body(mediaBooking, "USD", "1000.00"))

	var got []string
	for who, c := range map[string]*apitest.Client{"lea": s.SignIn(t, leasEmail), "priya": priya} {
		view := decoded[purchaserequest.ApprovalView](t, c.Do(t, http.MethodGet, prs+"/"+pr.ID+"/approval", nil), http.StatusOK)
		for _, sec := range view.Sections {
			got = append(got, fmt.Sprint(who, " ", sec.Kind, " submit ", sec.Actions.Submit.Available, " resubmit ",
				sec.Actions.Resubmit.Available))
		}
	}
	draft := decoded[purchaserequest.ApprovalView](t, priya.Do(t, http.MethodGet, prs+"/"+created.ID+"/approval", nil),
		http.StatusOK)
	got = append(got, fmt.Sprint("draft submit ", draft.Sections[0].Actions.Submit.Available))
	slices.Sort(got)
	same(t, "032 FR-022 only the purchase request's own rules offer Submit", []string{
		"draft submit true",
		"lea modifiedPurchaseRequest submit false resubmit false",
		"lea purchaseRequest submit false resubmit false",
		"priya modifiedPurchaseRequest submit false resubmit false",
		"priya purchaseRequest submit false resubmit false",
	}, got)
}
