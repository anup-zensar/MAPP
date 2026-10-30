package purchaserequest_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
	"mapp/internal/purchaserequest"
)

func TestList_GroupsSearchesAndFilters_Contract_032_FR044(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	priya, lea := s.SignIn(t, priyasEmail), s.SignIn(t, leasEmail)
	create(t, priya, body(standBuild, "EUR", "100.00"))
	page := func(query string) purchaserequest.ListPage {
		return decoded[purchaserequest.ListPage](t, lea.Do(t, http.MethodGet, prs+query, nil), http.StatusOK)
	}

	all := page("")
	byHX := page("?q=hx1006")
	byStatus := page("?status=approved")
	byVendor := page("?vendor=brightside%20media%20bv&sort=-created")
	bad := lea.Do(t, http.MethodGet, prs+"?sort=colour&pageSize=7", nil, apitest.SkipRequestValidation())
	planner := priya.Do(t, http.MethodGet, prs, nil)

	same(t, "032 FR-044 page", []any{2, []string{"Brightside Media BV", "Northlight Events GmbH"}, "PR-1001"},
		[]any{all.Total, labels(all), all.Items[0].ID})
	same(t, "032 FR-045 search, status, vendor", []any{1, "HX1006", 0, 1},
		[]any{byHX.Total, byHX.Items[0].Lines[0].HXKey, byStatus.Total, byVendor.Total})
	same(t, "bad query, and 032 FR-049 a planner sees no Purchase requests page", []int{2, http.StatusForbidden},
		[]int{len(decoded[apiError](t, bad, http.StatusBadRequest).Error.Fields), planner.Status})
}

func labels(page purchaserequest.ListPage) []string {
	var got []string
	for _, v := range page.Vendors {
		got = append(got, v.Label)
	}
	return got
}

func TestOptions_PickerReasons_Contract_032_FR010(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	priya := s.SignIn(t, priyasEmail)
	pr := create(t, priya, body(standBuild, "EUR", "100.00"))

	found := decoded[purchaserequest.OptionPage](t, priya.Do(t, http.MethodGet, prs+"/line-item-options?purchaseRequestId="+pr.ID, nil),
		http.StatusOK)

	reasons := map[string]string{}
	for _, o := range found.Items {
		reasons[o.LineItem.ID] = o.Reason
	}
	if reasons[standBuild] != "Already on this purchase request" ||
		reasons[mediaBooking] != "One purchase request cannot charge two company codes (GB01 and NL02)." {
		t.Errorf("032 FR-010 reasons = %v", reasons)
	}
	if _, otherPlan := reasons["L-1007"]; otherPlan {
		t.Errorf("032 FR-002 a line item of another plan is offered")
	}
}

func TestConfiguration_SwitchedOff_Contract_032_FR004(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	alex, priya := s.SignIn(t, alexsEmail), s.SignIn(t, priyasEmail)
	before := decoded[purchaserequest.ConfigurationView](t, priya.Do(t, http.MethodGet, "/api/v1/purchase-request-configuration", nil),
		http.StatusOK)
	refused := priya.Do(t, http.MethodPut, "/api/v1/purchase-request-configuration", map[string]any{"version": before.Version,
		"purchaseSupport": false, "linesOnApprovedPurchaseRequests": "notAllowed"})

	saved := decoded[purchaserequest.ConfigurationView](t, alex.Do(t, http.MethodPut, "/api/v1/purchase-request-configuration",
		map[string]any{"version": before.Version, "purchaseSupport": false, "linesOnApprovedPurchaseRequests": "notAllowed"}), http.StatusOK)
	created := decoded[apiError](t, priya.Do(t, http.MethodPost, prs, body(standBuild, "EUR", "100.00")), http.StatusConflict)
	seen := get(t, priya, "PR-1001")

	same(t, "before", []any{false, true, purchaserequest.WithModificationApproval},
		[]any{before.CanChange.Available, before.PurchaseSupport, before.LinesOnApprovedPurchaseRequests})
	same(t, "refused and saved", []any{http.StatusForbidden, false, before.Version + 1}, []any{refused.Status, saved.PurchaseSupport, saved.Version})
	off := "Purchase requests are switched off for your organisation."
	same(t, "switched off", []any{off, false, off}, []any{created.Error.Message, seen.Access.Edit.Available, seen.Notice})
}

func TestConfiguration_NeverChanged_HasNoLastChange_Contract_038_FR057_L16(t *testing.T) {
	t.Parallel()
	s := apitest.New(t)
	alex := s.SignIn(t, alexsEmail)
	type lastChange struct {
		Version   int
		UpdatedAt *string
		UpdatedBy *struct{ Name string }
	}
	fresh := decoded[lastChange](t, alex.Do(t, http.MethodGet, "/api/v1/purchase-request-configuration", nil), http.StatusOK)
	saved := decoded[lastChange](t, alex.Do(t, http.MethodPut, "/api/v1/purchase-request-configuration",
		map[string]any{"version": fresh.Version, "purchaseSupport": true, "linesOnApprovedPurchaseRequests": "notAllowed"}),
		http.StatusOK)

	same(t, "038 FR-057 never changed", true, fresh.UpdatedAt == nil && fresh.UpdatedBy == nil)
	same(t, "038 FR-057 changed by Alex", true, saved.UpdatedAt != nil && saved.UpdatedBy != nil && saved.UpdatedBy.Name == "Alex Morgan")
}
