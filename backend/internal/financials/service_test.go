package financials

import (
	"net/http"
	"testing"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// User story 6 — Home's card (FR-025).

// The overview of the current year: Priya's team plan, everything, and the activities above planned cost
// (FR-025, data model §6.1).
func TestOverview_MyTeamsAndEverything_FR025(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	mine, err := f.service.Overview(t.Context(), priya, "myTeams")
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if mine.Year != 2026 || mine.Figures.Committed != usd(t, "35652.17") || *mine.Figures.PlannedCost != usd(t, "460000.00") ||
		*mine.Figures.CommittedPercent != 8 || mine.ActivitiesAbovePlanned != 1 || mine.NotLinked != 1 ||
		mine.Figures.FirstLineItemAbovePlanned.ID != "L-1006" {
		t.Errorf("my teams = %+v", mine)
	}
	everything, err := f.service.Overview(t.Context(), alex, "everything")
	if err != nil || *everything.Figures.PlannedCost != usd(t, "500000.00") || everything.Scope != ScopeEverything {
		t.Errorf("everything = %+v, %v", everything, err)
	}
	none, err := f.service.Overview(t.Context(), alex, "myTeams")
	if err != nil || none.Figures.PlannedCost != nil || none.Figures.Committed != usd(t, "0.00") {
		t.Errorf("Alex's teams have no plan: %+v, %v", none.Figures, err)
	}
	_, err = f.service.Overview(t.Context(), alex, "all")
	assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed)
}

// User story 7 — link a transaction by hand (FR-026, FR-027).

// A manual link is logged with the line item before and after; unlinking matches again and is logged too
// (FR-026, FR-027).
func TestLinkAndUnlink_Logged_FR027(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	view, err := f.service.Link(t.Context(), alex, "FT-1004", LinkRequest{LineItemID: "L-1007", Version: version(1)})
	if err != nil {
		t.Fatalf("Link: %v", err)
	}
	if view.Link.State != StateLinked || *view.Link.MatchedBy != MatchedByManual || !view.Link.ManualAt.Equal(start) ||
		!view.CanUnlink || view.CanLink {
		t.Errorf("view = %+v", view)
	}
	if _, err := f.service.Unlink(t.Context(), alex, "FT-1004", UnlinkRequest{Version: version(2)}); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	got := f.stored(t, "FT-1004")
	if got.LineItemID != "" || got.ManualLink != nil || got.NotLinkedReason != "No line item for HX key HX1099 — check the tracking key in GSAP." {
		t.Errorf("after unlink = %+v", got)
	}
	entries := f.entries(t)
	if len(entries) != 2 {
		t.Fatalf("%d entries, want 2", len(entries))
	}
	linked, unlinked := entries[0], entries[1]
	if linked.Verb != "linked" || linked.RecordType != "financialTransaction" || linked.RecordName != "PO 4500001004 / 10" ||
		len(linked.Changes) != 1 || linked.Changes[0].Label != "Line item" || linked.Changes[0].OldText != "Not linked" ||
		linked.Changes[0].NewText != "Station signage (L-1007)" {
		t.Errorf("linked entry = %+v", linked)
	}
	if unlinked.Verb != "unlinked" || unlinked.Changes[0].OldText != "Station signage (L-1007)" || unlinked.Changes[0].NewText != "Not linked" {
		t.Errorf("unlinked entry = %+v", unlinked)
	}
}

// Linking refuses a line item in the recycle bin, an unknown one and a wrong ID form, with the field's
// message (FR-026).
func TestLink_RefusesLineItemsNotLive_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.planning.moveToBin("L-1005")
	cases := map[string]string{
		"L-1005": "Line item L-1005 is in the recycle bin — restore it first.",
		"L-9999": "No line item L-9999.",
		"A-1002": "Enter a line item ID like L-1007.",
	}
	for id, message := range cases {
		_, err := f.service.Link(t.Context(), alex, "FT-1004", LinkRequest{LineItemID: id, Version: version(1)})
		e := assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed)
		if e.Fields[0] != (errs.FieldError{Field: "lineItemId", Message: message}) {
			t.Errorf("%s: fields = %+v", id, e.Fields)
		}
	}
	if got := f.stored(t, "FT-1004"); got.Version != 1 || got.LineItemID != "" {
		t.Errorf("FT-1004 changed: %+v", got)
	}
}

// Polish — the access seam (FR-028, D-252, D-301).

// Without finance.view every read is forbidden (FR-028).
func TestAccess_NoViewRefusesEveryRead_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.noView = true
	ctx := t.Context()
	_, err := f.service.Record(ctx, priya, "A-1002")
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Transactions(ctx, priya, Query{Type: filterAll, Link: filterAll, Page: 1, PageSize: 50})
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Transaction(ctx, priya, "FT-1001")
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Overview(ctx, priya, "myTeams")
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Imports(ctx, priya)
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
}

// Without finance.import a person cannot check, import or get the sample file, and does not see the
// transactions that are not linked: not in the lists, counts, totals or the overview (FR-028, D-301).
func TestAccess_NoImportHidesNotLinked_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.noImport = true
	ctx := t.Context()
	_, err := f.service.Check(ctx, priya, threeRows)
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Import(ctx, priya, threeRows)
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Template(ctx, priya)
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Transaction(ctx, priya, "FT-1004")
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)

	page := f.list(t, priya, Query{})
	if page.Counts.All != 10 || page.Counts.NotLinked != 0 || page.Totals.Committed != usd(t, "35652.17") {
		t.Errorf("counts %+v, totals %+v", page.Counts, page.Totals)
	}
	overview, err := f.service.Overview(ctx, priya, "everything")
	if err != nil || overview.NotLinked != 0 {
		t.Errorf("overview = %+v, %v", overview, err)
	}
}

// Without finance.link a person can neither link nor unlink, and the panel offers neither (FR-028).
func TestAccess_NoLinkRefusesLinking_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.noLink = true
	_, err := f.service.Link(t.Context(), priya, "FT-1004", LinkRequest{LineItemID: "L-1007", Version: version(1)})
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Unlink(t.Context(), priya, "FT-1004", UnlinkRequest{Version: version(1)})
	assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	view, err := f.service.Transaction(t.Context(), priya, "FT-1004")
	if err != nil || view.CanLink || view.CanUnlink {
		t.Errorf("view = %+v, %v", view, err)
	}
}

// A person who sees only some plans gets only their transactions; a record or transaction of another plan
// answers like a missing one, and a link into one is forbidden (FR-028, 014 FR-007, 022 FR-005).
func TestAccess_VisiblePlansScopeEverything_FR028_022_FR005(t *testing.T) { //nolint:gocyclo // 12: one scenario per access rule, kept together so the rules read as one table.
	f := newFixture(t)
	f.access.plans = []string{"P-1002"}
	ctx := t.Context()
	_, err := f.service.Record(ctx, priya, "A-1002")
	if e := assertCode(t, err, http.StatusNotFound, errs.CodeNotFound); e.Message != "This activity no longer exists — someone may have deleted it." {
		t.Errorf("record message = %q", e.Message)
	}
	_, err = f.service.Transaction(ctx, priya, "FT-1001")
	if e := assertCode(t, err, http.StatusNotFound, errs.CodeNotFound); e.Message != "This GSAP transaction doesn't exist." {
		t.Errorf("transaction message = %q", e.Message)
	}
	if _, err := f.service.Record(ctx, priya, "A-1008"); err != nil {
		t.Errorf("Record(A-1008): %v", err)
	}
	page := f.list(t, priya, Query{})
	if page.Counts.All != 1 || page.Items[0].ID != "FT-1004" || len(page.Choices.Plans) != 1 || page.Choices.Plans[0].ID != "P-1002" {
		t.Errorf("page = %+v", page)
	}
	overview, err := f.service.Overview(ctx, priya, "everything")
	if err != nil || overview.Figures.Committed != usd(t, "0.00") || *overview.Figures.PlannedCost != usd(t, "40000.00") ||
		overview.ActivitiesAbovePlanned != 0 {
		t.Errorf("overview = %+v, %v", overview.Figures, err)
	}
	_, err = f.service.Link(ctx, priya, "FT-1004", LinkRequest{LineItemID: "L-1003", Version: version(1)})
	_ = assertCode(t, err, http.StatusForbidden, errs.CodeForbidden)
	_, err = f.service.Transactions(ctx, priya, Query{Type: filterAll, Link: filterAll, Page: 1, PageSize: 50, PlanID: "P-1001"})
	_ = assertCode(t, err, http.StatusBadRequest, errs.CodeValidationFailed)
}

// FactsByLineItem gives All work the same figures as the cards (FR-023, SC-002).
func TestFactsByLineItem_SameAsTheCards_SC002(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	facts, err := f.service.FactsByLineItem(t.Context())
	if err != nil {
		t.Fatalf("FactsByLineItem: %v", err)
	}
	if len(facts) != 3 {
		t.Errorf("%d line items with facts, want 3", len(facts))
	}
	for _, id := range []string{"L-1003", "L-1004", "L-1006"} {
		card := f.record(t, alex, id).Figures
		got := facts[id]
		if money.USD(got.Committed) != card.Committed || money.USD(got.Invoiced) != card.Invoiced ||
			money.USD(got.Paid) != card.Paid || money.USD(got.Accrued) != card.Accrued || got.Count != card.TransactionCount {
			t.Errorf("%s facts %+v, card %+v", id, got, card)
		}
	}
	if facts["L-1004"].POCount != 1 || facts["L-1004"].InvoiceCount != 1 {
		t.Errorf("L-1004 counts = %+v", facts["L-1004"])
	}
}
