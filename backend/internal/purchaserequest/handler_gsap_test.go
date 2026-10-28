package purchaserequest_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
	"mapp/internal/purchaserequest"
)

const gsapRun = "/api/v1/simulated-gsap/purchase-requests/"

type figures struct {
	Figures struct {
		Committed struct{ Amount, Currency string }
	}
}

type transactions struct {
	Items []struct {
		ID, Name string
		Amount   struct{ Amount string }
	}
}

func poTransactions(t *testing.T, c *apitest.Client) []string {
	t.Helper()
	list := decoded[transactions](t, c.Do(t, http.MethodGet, "/api/v1/financials/transactions?pageSize=200", nil), http.StatusOK)
	var names []string
	for _, tr := range list.Items {
		if tr.Name == "PO SIM-4500001001 / 10" {
			names = append(names, tr.Name+" "+tr.Amount.Amount)
		}
	}
	return names
}

func TestSimulatedGSAPOrder_EndToEnd_Contract_033_FR005(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	pr := approved(t, s)
	lea, priya, alex := s.SignIn(t, leasEmail), s.SignIn(t, priyasEmail), s.SignIn(t, alexsEmail)

	refused := decoded[apiError](t, priya.Do(t, http.MethodPost, gsapRun+pr.ID+"/purchase-order", map[string]any{"version": pr.Version}),
		http.StatusForbidden)
	ordered := decoded[purchaserequest.View](t, lea.Do(t, http.MethodPost, gsapRun+pr.ID+"/purchase-order",
		map[string]any{"version": pr.Version}), http.StatusOK)
	again := decoded[apiError](t, lea.Do(t, http.MethodPost, gsapRun+pr.ID+"/purchase-order",
		map[string]any{"version": ordered.Version}), http.StatusConflict)
	record := decoded[figures](t, priya.Do(t, http.MethodGet, "/api/v1/financials/records/"+standBuild, nil), http.StatusOK)
	waiting := decoded[purchaserequest.ListPage](t, lea.Do(t, http.MethodGet, prs+"?waitingForGsap=true", nil), http.StatusOK)

	same(t, "033 FR-010 FR-006 refusals", []string{
		"Importing GSAP extracts isn't part of your role (Planner). Ask an administrator if you need it.",
		"The simulated GSAP already created the PO for this purchase request. Run it again after an approved change of its amounts or lines.",
	}, []string{refused.Error.Message, again.Error.Message})
	l := ordered.Lines[0]
	same(t, "033 FR-002 FR-003 FR-004 ordered", []any{"SIM-0010001001", "simulatedGsap", false, "SIM-4500001001", "10", "Released",
		"9,200.00 EUR", false, false}, []any{ordered.GSAPPRNumber, ordered.GSAPSource, ordered.WaitingForGSAP, l.PONumber, l.POLine,
		l.POLineStatus, l.POValue.Format(), l.GSAPPending, ordered.Access.GSAPOrder.Available})
	// Committed: the demo PO's 10,652.17 USD and the new 9,200.00 EUR PO line.
	same(t, "033 FR-005 SC-002 committed, waiting, transactions", []any{"20652.17", 0, []string{"PO SIM-4500001001 / 10 9200.00"}},
		[]any{record.Figures.Committed.Amount, waiting.Total, poTransactions(t, alex)})
}

func TestSimulatedGSAPOrder_TopUp_Contract_033_FR007(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	pr := approved(t, s)
	lea, priya, daniel, alex := s.SignIn(t, leasEmail), s.SignIn(t, priyasEmail), s.SignIn(t, danielsEmail), s.SignIn(t, alexsEmail)
	ordered := decoded[purchaserequest.View](t, lea.Do(t, http.MethodPost, gsapRun+pr.ID+"/purchase-order",
		map[string]any{"version": pr.Version}), http.StatusOK)

	changing := decoded[purchaserequest.View](t, priya.Do(t, http.MethodPut, prs+"/"+pr.ID, saveBody(ordered, "8000.00", "submitChanges")),
		http.StatusOK)
	inReview := decoded[apiError](t, lea.Do(t, http.MethodPost, gsapRun+pr.ID+"/purchase-order",
		map[string]any{"version": changing.Version}), http.StatusConflict)
	at := decoded[purchaserequest.ApprovalView](t, priya.Do(t, http.MethodGet, prs+"/"+pr.ID+"/approval", nil), http.StatusOK)
	decide(t, daniel, "approve", at)
	changed := get(t, lea, pr.ID)
	topUp := decoded[purchaserequest.View](t, lea.Do(t, http.MethodPost, gsapRun+pr.ID+"/purchase-order",
		map[string]any{"version": changed.Version}), http.StatusOK)

	if inReview.Error.Message != "Wait until the change of this purchase request is approved or declined." || !changed.WaitingForGSAP {
		t.Errorf("033 US2: in review %+v, waiting %v", inReview, changed.WaitingForGSAP)
	}
	l := topUp.Lines[0]
	if topUp.GSAPPRNumber != "SIM-0010001001" || l.PONumber != "SIM-4500001001" || l.POLine != "10" || l.POValue.Format() != "8,000.00 EUR" {
		t.Errorf("033 FR-007 top-up = %+v", l)
	}
	if got := poTransactions(t, alex); len(got) != 1 || got[0] != "PO SIM-4500001001 / 10 8000.00" {
		t.Errorf("033 SC-002 one transaction, updated: %v", got)
	}
}

func TestSimulatedGSAPOrder_NotRouted_Contract_033_FR010(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	lea := s.SignIn(t, leasEmail)
	pr := get(t, lea, "PR-1001")

	refused := decoded[apiError](t, lea.Do(t, http.MethodPost, gsapRun+"PR-1001/purchase-order", map[string]any{"version": pr.Version}),
		http.StatusConflict)
	missing := lea.Do(t, http.MethodPost, gsapRun+"PR-9999/purchase-order", map[string]any{"version": 1})

	if refused.Error.Message != "The simulated GSAP processes approved purchase requests that were routed to the requisitioner." ||
		pr.Access.GSAPOrder.Reason != refused.Error.Message {
		t.Errorf("033 FR-010 = %+v, access %+v", refused, pr.Access.GSAPOrder)
	}
	if missing.Status != http.StatusNotFound {
		t.Errorf("a missing PR = %d", missing.Status)
	}
}
