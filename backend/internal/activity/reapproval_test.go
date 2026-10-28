package activity_test

import (
	"slices"
	"strings"
	"testing"

	"mapp/internal/activity"
)

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
