package app

import (
	"context"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/authz"
)

// withPermissions returns a context of a person holding perms for All markets, through one role.
func withPermissions(perms ...authz.Permission) context.Context {
	role := authz.GrantedRole{ID: "RL-1010", Name: "Custom", Scope: authz.ScopeAll}
	g := authz.NewGrants("U-1006", []authz.GrantedRole{role}, nil, map[string][]authz.Permission{"RL-1010": perms}, nil, authz.Rules{})
	return authz.With(context.Background(), g)
}

// The log entries of the kinds features 014-017 add are seen as D-304 says: GSAP imports and transactions
// with "Import GSAP extracts" or "See the whole activity log"; delegations, exchange rates, rate imports,
// field help, roles and the access rules only with "See the whole activity log"; the sample records by
// everyone; an internal call sees everything (FR-007, FR-038, D-304).
func TestHistoryAccess_KindsOfLaterFeatures_D304(t *testing.T) {
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

// Restoring an administration record needs the permission that manages it (D-238): an exchange rate needs
// "Manage currencies and exchange rates", which an auditor does not hold.
func TestHistoryAccess_RestoreNeedsTheManagingPermission_D238(t *testing.T) {
	h := historyAccess{}
	e := history.BinEntry{RecordType: "exchangeRate", RecordTypeLabel: "exchange rate", RecordID: "XR-1001"}

	if err := h.CanRestore(withPermissions(authz.ActivityLogViewAll), e); err == nil {
		t.Error("an auditor restored an exchange rate")
	}
	if err := h.CanRestore(withPermissions(authz.ActivityLogViewAll, authz.ExchangeRatesManage), e); err != nil {
		t.Errorf("a rate manager who sees the log: %v", err)
	}
}
