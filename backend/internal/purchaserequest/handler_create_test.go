package purchaserequest_test

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/apitest"
	"mapp/internal/purchaserequest"
)

func TestDefaults_Contract(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	priya := s.SignIn(t, priyasEmail)

	t.Run("032 FR-003 PR for Stand build, the acting owner, everything left, HX1006", func(t *testing.T) {
		t.Parallel()
		d := decoded[purchaserequest.Defaults](t, priya.Do(t, http.MethodGet, prs+"/new-defaults?lineItemId="+standBuild, nil), http.StatusOK)
		if d.Name != "PR for Stand build" || d.Owner.ID != "U-1002" || d.Line.HXKey != "HX1006" || d.CompanyCode != "GB01" ||
			d.Line.Amount.Format() != "9,200.00 EUR" || d.Line.OverallLimit.Format() != "9,200.00 EUR" {
			t.Errorf("defaults = %+v", d)
		}
	})
	t.Run("032 FR-050 a line item outside the person's plans answers 404", func(t *testing.T) {
		t.Parallel()
		mei := s.SignIn(t, meisEmail)
		body := decoded[apiError](t, mei.Do(t, http.MethodGet, prs+"/new-defaults?lineItemId="+standBuild, nil), http.StatusNotFound)
		if body.Error.Code != "not_found" {
			t.Errorf("answer = %+v", body)
		}
	})
	t.Run("032 FR-049 a viewer may not raise purchase requests", func(t *testing.T) {
		t.Parallel()
		sara := s.SignIn(t, sarasEmail)
		body := decoded[apiError](t, sara.Do(t, http.MethodPost, prs, body(standBuild, "EUR", "100.00")), http.StatusForbidden)
		if body.Error.Code != "forbidden" {
			t.Errorf("answer = %+v", body)
		}
	})
}

func TestCreate_ReservesAtOnce_Contract_032_FR017(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	priya := s.SignIn(t, priyasEmail)

	pr := create(t, priya, body(mediaBooking, "USD", "26000.00"))
	section := decoded[purchaserequest.LineItemSection](t, priya.Do(t, http.MethodGet, "/api/v1/line-items/"+mediaBooking+
		"/purchase-requests", nil), http.StatusOK)
	activity := decoded[purchaserequest.ActivitySection](t, priya.Do(t, http.MethodGet, "/api/v1/activities/A-1004/purchase-requests",
		nil), http.StatusOK)

	got := []any{pr.Status, pr.HXKeys, pr.Vendor, pr.CompanyCode, pr.Lines[0].LineItem.GLCode != "", pr.Notice,
		section.Requested.Format(), section.AvailableToRequest.Format(), len(section.PurchaseRequests), section.Create.Available,
		activity.Requested.Format(), len(activity.LineItems)}
	same(t, "created, line item and activity sections", []any{purchaserequest.StatusCreated, []string{"HX1005"},
		"Northlight Events GmbH", "NL02", true, "", "26,000.00 USD", "4,000.00 USD", 1, true, "26,000.00 USD", 1}, got)
	t.Run("032 FR-012 3,700.00 EUR is 4,021.74 USD at today's rate, above the 4,000.00 USD left", func(t *testing.T) {
		t.Parallel()
		refused := decoded[apiError](t, priya.Do(t, http.MethodPost, prs, body(mediaBooking, "EUR", "3700.00")), http.StatusBadRequest)
		want := "Only 4,000.00 USD of line item L-1005 is not yet requested. 3,700.00 EUR is 4,021.74 USD at today's rate."
		if len(refused.Error.Fields) != 1 || refused.Error.Fields[0].Message != want {
			t.Errorf("answer = %+v", refused)
		}
	})
}

func TestCreate_ValidationAllAtOnce_Contract_032_FR053(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)
	b := body(standBuild, "EUR", "1.234")
	b["name"], b["approvalWorkflowId"], b["modificationWorkflowId"] = " ", "", "WF-1001"
	b["attributes"].(map[string]any)["SA-1045"] = "abc"
	b["lines"].([]map[string]any)[0]["deliveryDate"] = ""

	refused := decoded[apiError](t, priya.Do(t, http.MethodPost, prs, b, apitest.SkipRequestValidation()), http.StatusBadRequest)

	var fields []string
	for _, f := range refused.Error.Fields {
		fields = append(fields, f.Field)
	}
	want := []string{"name", "approvalWorkflowId", "modificationWorkflowId", "attributes.SA-1045", "lines[0].amount",
		"lines[0].deliveryDate", "lines[0].overallLimit"}
	if diff := cmp.Diff(want, fields); diff != "" {
		t.Errorf("032 FR-005 FR-007 FR-011 every problem at once (-want +got):\n%s", diff)
	}
}

func TestCreate_LinesRules_Contract_032_FR013(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)
	b := body(standBuild, "EUR", "100.00")
	line := b["lines"].([]map[string]any)[0]
	other := map[string]any{"lineItemId": mediaBooking, "currency": "USD", "amount": "100.00", "deliveryDate": "2026-11-30",
		"overallLimit": nil, "description": "", "attributes": map[string]any{}}
	b["lines"] = []map[string]any{line, line, other}

	refused := decoded[apiError](t, priya.Do(t, http.MethodPost, prs, b), http.StatusBadRequest)

	want := map[string]string{
		"lines[1].lineItemId": "Line item L-1006 is already on this purchase request.",
		"lines[2].lineItemId": "One purchase request cannot charge two company codes (GB01 and NL02).",
	}
	got := map[string]string{}
	for _, f := range refused.Error.Fields {
		got[f.Field] = f.Message
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("032 FR-013 FR-014 (-want +got):\n%s", diff)
	}
}
