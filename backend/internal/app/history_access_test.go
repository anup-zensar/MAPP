package app

import (
	"context"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/authz"
)

func withPermissions(perms ...authz.Permission) context.Context {
	role := authz.GrantedRole{ID: "RL-1010", Name: "Custom", Scope: authz.ScopeAll}
	g := authz.NewGrants("U-1006", []authz.GrantedRole{role}, nil, map[string][]authz.Permission{"RL-1010": perms}, nil, authz.Rules{})
	return authz.With(context.Background(), g)
}

// D-304: GSAP imports and transactions need Import GSAP extracts or the whole log, the administration kinds
// the whole log; everyone sees the sample records, and an internal call sees everything.
func TestHistoryAccess_KindsOfLaterFeatures_D304(t *testing.T) {
	t.Parallel()
	h := historyAccess{}
	importer, auditor, planner := withPermissions(authz.FinanceImport), withPermissions(authz.ActivityLogViewAll),
		withPermissions(authz.PlansView, authz.ActivitiesEdit)

	for _, c := range []struct {
		kind                    string
		importer, auditor, plan bool
	}{
		{"financialImport", true, true, false},
		{"financialTransaction", true, true, false},
		{"approvalDelegation", false, true, false},
		{"exchangeRate", false, true, false},
		{"rateImport", false, true, false},
		{"fieldHelp", false, true, false},
		{"role", false, true, false},
		{"accessRules", false, true, false},
		{"aKindNobodyKnows", false, true, false},
		{"sampleFolder", true, true, true},
	} {
		e := history.Entry{RecordType: c.kind, RecordID: "X-1001"}
		got := []bool{h.CanSeeEntry(importer, e), h.CanSeeEntry(auditor, e), h.CanSeeEntry(planner, e)}
		if want := []bool{c.importer, c.auditor, c.plan}; got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
			t.Errorf("%s: importer, auditor, planner see it = %v, want %v", c.kind, got, want)
		}
		if !h.CanSeeEntry(context.Background(), e) {
			t.Errorf("%s: an internal call does not see it", c.kind)
		}
	}
}

func TestHistoryAccess_RestoreNeedsTheManagingPermission_D238(t *testing.T) {
	t.Parallel()
	h := historyAccess{}
	e := history.BinEntry{RecordType: "exchangeRate", RecordTypeLabel: "exchange rate", RecordID: "XR-1001"}

	if err := h.CanRestore(withPermissions(authz.ActivityLogViewAll), e); err == nil {
		t.Error("an auditor restored an exchange rate")
	}
	if err := h.CanRestore(withPermissions(authz.ActivityLogViewAll, authz.ExchangeRatesManage), e); err != nil {
		t.Errorf("a rate manager who sees the log: %v", err)
	}
}
