package app_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/apitest"
)

// GBP is the plan currency of P-1004, so it cannot be deleted; Set inactive is offered.
func TestCurrencyUsage_CountsPlans_FR010(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")

	gbp := usageAt(t, priya, "/api/v1/currencies/GBP")
	refused := server.SignIn(t, "alex.morgan@shell.com").Do(t, http.MethodDelete, "/api/v1/currencies/GBP", nil)

	if !gbp.InUse || !strings.Contains(gbp.Summary, "Plan currency of 1 plan") ||
		!slices.Contains(gbp.Records, recordRef{Type: "plan", ID: "P-1004", Name: "FY2027 Lubricants UK & Ireland"}) {
		t.Errorf("GBP usage = %+v, want the plan P-1004 counted", gbp)
	}
	if refused.Status != http.StatusConflict || !strings.Contains(string(refused.Body), "Plan currency of 1 plan") {
		t.Errorf("DELETE GBP = %d %s, want 409 naming the plan", refused.Status, refused.Body)
	}
}
