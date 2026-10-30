package financials

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// mei is a planner of Mobility China (P-1002) who does not import GSAP extracts.
var mei = actor.Actor{ID: "U-1004", Name: "Mei Lin"}

func sameAnswer(t *testing.T, got, want error) {
	t.Helper()
	var g, w *errs.Error
	if !errors.As(got, &g) || !errors.As(want, &w) {
		t.Fatalf("answers %v and %v, want two API errors", got, want)
	}
	if g.Status != w.Status || g.Code != w.Code || g.Message != w.Message || !reflect.DeepEqual(g.Fields, w.Fields) {
		t.Errorf("answer = %d %s %q %+v\nwant     %d %s %q %+v", g.Status, g.Code, g.Message, g.Fields, w.Status, w.Code, w.Message, w.Fields)
	}
}

func TestLink_InvisibleTransactionIsMissing_038_FR007_H5(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.noImport = true
	request := LinkRequest{LineItemID: "L-1007", Version: version(1)}

	_, hidden := f.service.Link(t.Context(), mei, "FT-1004", request)
	_, missing := f.service.Link(t.Context(), mei, "FT-9999", request)

	assertCode(t, hidden, http.StatusNotFound, errs.CodeNotFound)
	sameAnswer(t, hidden, missing)
	if got := f.stored(t, "FT-1004"); got.Version != 1 || got.LineItemID != "" {
		t.Errorf("FT-1004 changed: %+v", got)
	}
}

// The scope is checked before the version: a stale unlink outside it answers as a missing transaction.
func TestUnlink_OutOfScopeIsMissingBeforeVersion_038_FR007_H5(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if _, err := f.service.Link(t.Context(), alex, "FT-1004", LinkRequest{LineItemID: "L-1003", Version: version(1)}); err != nil {
		t.Fatalf("Link: %v", err)
	}
	f.access.plans = []string{"P-1002"}

	_, hidden := f.service.Unlink(t.Context(), mei, "FT-1004", UnlinkRequest{Version: version(1)})
	_, missing := f.service.Unlink(t.Context(), mei, "FT-9999", UnlinkRequest{Version: version(1)})

	e := assertCode(t, hidden, http.StatusNotFound, errs.CodeNotFound)
	sameAnswer(t, hidden, missing)
	if strings.Contains(e.Message, alex.Name) {
		t.Errorf("message names a person: %q", e.Message)
	}
}

func TestLink_HiddenTargetEqualsMissing_038_FR007_H5(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.plans = []string{"P-1002"}

	_, hidden := f.service.Link(t.Context(), mei, "FT-1004", LinkRequest{LineItemID: "L-1003", Version: version(1)})

	e := assertCode(t, hidden, http.StatusBadRequest, errs.CodeValidationFailed)
	if len(e.Fields) != 1 || e.Fields[0] != (errs.FieldError{Field: "lineItemId", Message: "No line item L-1003."}) {
		t.Errorf("fields = %+v", e.Fields)
	}
}

// Without link, linking is refused for permission before version and state; without import, the transaction is not there.
func TestLink_NeedsImportAndLink_038_FR005_H4(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.noLink = true
	_, stale := f.service.Link(t.Context(), mei, "FT-1004", LinkRequest{LineItemID: "L-1007", Version: version(7)})
	assertCode(t, stale, http.StatusForbidden, errs.CodeForbidden)
	_, ruleLinked := f.service.Link(t.Context(), mei, "FT-1001", LinkRequest{LineItemID: "L-1007", Version: version(1)})
	assertCode(t, ruleLinked, http.StatusForbidden, errs.CodeForbidden)

	f.access.noLink, f.access.noImport = false, true
	_, noImport := f.service.Link(t.Context(), mei, "FT-1004", LinkRequest{LineItemID: "L-1007", Version: version(1)})
	assertCode(t, noImport, http.StatusNotFound, errs.CodeNotFound)

	f.access.noImport = false
	view, err := f.service.Transaction(t.Context(), mei, "FT-1004")
	if err != nil || !view.CanLink {
		t.Errorf("with import and link: view %+v, %v", view, err)
	}
}

// Unlinking needs link in the plan the transaction is linked to, checked before the version.
func TestUnlink_NeedsLinkInLinkedPlan_038_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	if _, err := f.service.Link(t.Context(), alex, "FT-1004", LinkRequest{LineItemID: "L-1007", Version: version(1)}); err != nil {
		t.Fatalf("Link: %v", err)
	}
	f.access.noLink = true

	_, stale := f.service.Unlink(t.Context(), mei, "FT-1004", UnlinkRequest{Version: version(1)})

	assertCode(t, stale, http.StatusForbidden, errs.CodeForbidden)
	f.access.noLink = false
	if _, err := f.service.Unlink(t.Context(), mei, "FT-1004", UnlinkRequest{Version: version(2)}); err != nil {
		t.Errorf("Unlink with link in P-1002: %v", err)
	}
}
