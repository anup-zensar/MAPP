package activity_test

import (
	"slices"
	"strings"
	"testing"

	"mapp/internal/activity"
	"mapp/internal/platform/errs"
)

// RestartApproval sends a Submitted or Approved activity back to Created for changes made elsewhere (feature
// 007), with one status entry naming the reasons; other statuses are left alone (FR-027, D-196).
func TestRestartApproval_SendsBackAndLogs_FR027(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	err := f.service.RestartApproval(t.Context(), daniel, "A-1002", []string{"Line item 'Banner' (L-1003) added: 1,000.00 EUR."})
	again := f.service.RestartApproval(t.Context(), daniel, "A-1002", []string{"again"})

	if err != nil || again != nil {
		t.Fatalf("RestartApproval: %v, %v", err, again)
	}
	stored, _ := f.stored(t, "A-1002")
	entries := f.entries(t)
	if stored.Status != activity.StatusCreated || stored.Version != 2 || stored.UpdatedBy != daniel || len(entries) != 1 ||
		entries[0].Details != "Line item 'Banner' (L-1003) added: 1,000.00 EUR." || len(f.hooks.restarted) != 1 {
		t.Errorf("stored %s v%d, entries %+v, hooks %q", stored.Status, stored.Version, entries, f.hooks.restarted)
	}
	assertError(t, f.service.RestartApproval(t.Context(), daniel, "A-9999", nil),
		errs.NotFound("This activity no longer exists — someone may have deleted it."))
}

// A confirmed re-approval change tells feature 008 the changes of the save's "changed" entry, with the
// labels and texts people read (008 FR-022).
func TestReapprovalChange_CallsApprovalHook_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	f.mustEdit(t, "A-1004", func(u *activity.Update) {
		u.PlannedCost = usdInput("65000.00")
		u.RestartApproval = true
	})

	stored, _ := f.stored(t, "A-1004")
	if !slices.Equal(f.hooks.restarted, []string{"A-1004: submitted → created"}) || stored.Status != activity.StatusCreated {
		t.Errorf("approval hooks %q, stored %s; want one restart of the Submitted activity", f.hooks.restarted, stored.Status)
	}
	if len(f.hooks.changes) != 1 || !strings.HasPrefix(f.hooks.changes[0], "Planned cost: ") || !strings.HasSuffix(f.hooks.changes[0], " → 65,000.00 USD") {
		t.Errorf("changes = %q, want the planned cost with its old and new text", f.hooks.changes)
	}
}

// A line-item change that restarts approval (feature 007, through RestartApproval) tells feature 008 each
// reason as one change, a ready-made sentence without old and new texts (008 FR-022, plan Reconciliation).
func TestLineItemReapprovalChange_CallsApprovalHook_FR022(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	reasons := []string{"Planned cost of line item 'Media booking' (L-1002): 18,000.00 USD → 18,000.00 EUR"}

	if err := f.service.RestartApproval(t.Context(), daniel, "A-1004", reasons); err != nil {
		t.Fatalf("RestartApproval: %v", err)
	}

	if !slices.Equal(f.hooks.restarted, []string{"A-1004: submitted → created"}) ||
		!slices.Equal(f.hooks.changes, []string{reasons[0] + ":  → "}) {
		t.Errorf("approval hooks %q with changes %q, want one restart with the reason", f.hooks.restarted, f.hooks.changes)
	}
}
