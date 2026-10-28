package purchaserequest_test

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/apitest"
	"mapp/internal/purchaserequest"
)

const (
	alexsEmail   = "alex.morgan@shell.com"
	priyasEmail  = "priya.raman@shell.com"
	danielsEmail = "daniel.okoro@shell.com"
	leasEmail    = "lea.santos@shell.com"
	sarasEmail   = "sara.novak@shell.com"
	meisEmail    = "mei.lin@shell.com"
	prs          = "/api/v1/purchase-requests"
	standBuild   = "L-1006"
	mediaBooking = "L-1005"
	deLubes      = "SO-1170"
)

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Fields  []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"fields"`
		Missing []struct {
			Record struct {
				Type, ID string
			} `json:"record"`
			Fields []struct {
				Reason string `json:"reason"`
			} `json:"fields"`
		} `json:"missing"`
	} `json:"error"`
}

func same[T any](t *testing.T, what string, want, got T) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("%s (-want +got):\n%s", what, diff)
	}
}

func decoded[T any](t *testing.T, r *apitest.Response, status int) T {
	t.Helper()
	if r.Status != status {
		t.Fatalf("status %d %s, want %d", r.Status, r.Body, status)
	}
	var v T
	r.Decode(t, &v)
	return v
}

// body is a complete purchase request on one line, every field needed to submit filled in.
func body(lineItemID, currency, amount string) map[string]any {
	return map[string]any{
		"name": "PR for the demo", "description": "", "ownerId": "U-1002", "moaApproverId": "U-1003",
		"approvalWorkflowId": "WF-1004", "modificationWorkflowId": "WF-1005",
		"attributes": map[string]any{
			"SA-1036": []string{"SO-1164"}, "SA-1037": []string{"SO-1165"}, "SA-1038": "Stand for the winter wave",
			"SA-1041": "Daniel Okoro", "SA-1042": "Northlight Events GmbH", "SA-1043": "V-100101", "SA-1050": deLubes,
		},
		"lines": []map[string]any{{"lineItemId": lineItemID, "currency": currency, "amount": amount,
			"deliveryDate": "2026-11-30", "overallLimit": amount, "description": "", "attributes": map[string]any{}}},
		"then": "save",
	}
}

func create(t *testing.T, c *apitest.Client, b map[string]any) purchaserequest.View {
	t.Helper()
	return decoded[purchaserequest.View](t, c.Do(t, http.MethodPost, prs, b), http.StatusCreated)
}

func get(t *testing.T, c *apitest.Client, id string) purchaserequest.View {
	t.Helper()
	return decoded[purchaserequest.View](t, c.Do(t, http.MethodGet, prs+"/"+id, nil), http.StatusOK)
}

// receiveSwbsCodes runs the simulated GSAP's SWBS batch as Lea, the requisitioner (feature 030).
func receiveSwbsCodes(t *testing.T, s *apitest.Server) {
	t.Helper()
	lea := s.SignIn(t, leasEmail)
	if r := lea.Do(t, http.MethodPost, "/api/v1/simulated-gsap/swbs-runs", map[string]any{}); r.Status != http.StatusCreated {
		t.Fatalf("SWBS run = %d %s", r.Status, r.Body)
	}
}

// approved creates a purchase request on Stand build, submits it as Priya and approves it as Daniel.
func approved(t *testing.T, s *apitest.Server) purchaserequest.View {
	t.Helper()
	receiveSwbsCodes(t, s)
	priya, daniel := s.SignIn(t, priyasEmail), s.SignIn(t, danielsEmail)
	pr := create(t, priya, body(standBuild, "EUR", "9200.00"))
	sub := decoded[purchaserequest.ApprovalView](t, priya.Do(t, http.MethodPost, prs+"/"+pr.ID+"/approval/submit",
		map[string]any{"version": pr.Version}), http.StatusOK)
	decide(t, daniel, "approve", sub)
	return get(t, priya, pr.ID)
}

func decide(t *testing.T, c *apitest.Client, action string, at purchaserequest.ApprovalView) purchaserequest.ApprovalView {
	t.Helper()
	round := openRound(t, at)
	return decoded[purchaserequest.ApprovalView](t, c.Do(t, http.MethodPost, prs+"/"+at.PurchaseRequestID+"/approval/"+action,
		map[string]any{"roundId": round.ID, "level": round.CurrentLevel, "comment": "Wrong vendor"}), http.StatusOK)
}

type roundRef struct {
	ID           string
	CurrentLevel int
}

func openRound(t *testing.T, at purchaserequest.ApprovalView) roundRef {
	t.Helper()
	for _, s := range at.Sections {
		for _, r := range s.Rounds {
			if r.Outcome == "open" {
				return roundRef{ID: r.ID, CurrentLevel: r.CurrentLevel}
			}
		}
	}
	t.Fatalf("no open round in %+v", at)
	return roundRef{}
}
