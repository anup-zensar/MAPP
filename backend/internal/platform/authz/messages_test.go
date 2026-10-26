package authz_test

import (
	"testing"

	"mapp/internal/platform/authz"
)

// The refusal messages 1–3 and 22 of the plan and the not-found answer for a record outside the person's
// access, word for word (FR-010, 022 FR-005).
func TestRefusal_MessagesWordForWord_FR010(t *testing.T) {
	sara := grants("U-1006", []string{lubricants, mobility}, viewer)
	priya := grants("U-1002", []string{lubricants}, planner)
	daniel := grants("U-1003", []string{lubricants}, planner, approver)
	nobody := grants("U-1007", []string{lubricants})
	newcomer := grants("U-1008", nil, planner)
	mobilityPlan := plan(mobility)

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"1 no role", nobody.Refusal(authz.ActivitiesEdit, nil),
			"Changing activities and line items needs a role, and you don't have one yet. Ask an administrator."},
		{"2 one role", sara.Refusal(authz.ActivitiesEdit, &mobilityPlan),
			"Changing activities and line items isn't part of your role (Viewer). Ask an administrator if you need it."},
		{"2 several roles", daniel.Refusal(authz.PlansDelete, nil),
			"Deleting plans isn't part of your roles (Planner, Approver). Ask an administrator if you need it."},
		{"3 other teams", priya.Refusal(authz.PlansEdit, &mobilityPlan),
			"Editing plans is allowed for you only in plans of Lubricants Europe. Ask an administrator if you need it here."},
		{"3 several teams, sorted", grants("U-1006", []string{mobility, lubricants}, planner).Refusal(authz.PlansEdit, &authz.PlanFacts{TeamIDs: []string{globalOps}}),
			"Editing plans is allowed for you only in plans of Lubricants Europe, Mobility China. Ask an administrator if you need it here."},
		{"22 no team", newcomer.Refusal(authz.PlansCreate, &mobilityPlan),
			"Plans belong to teams, and you are not in a team yet. Ask an administrator to add you to one."},
		{"not found", authz.NotFoundMessage("line item"),
			"This line item no longer exists — someone may have deleted it."},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s:\n got %q\nwant %q", tt.name, tt.got, tt.want)
		}
	}
}

// Segregation of duties: while the rule is on, nobody decides on an activity they own or submitted
// (FR-026, messages 6a and 6b); with the rule off, nothing stops them.
func TestSelfApprovalRefusal_OwnerAndSubmitter_FR026(t *testing.T) {
	daniel := grants("U-1003", []string{lubricants}, planner, approver)
	relaxed := daniel
	relaxed.Rules.NoSelfApproval = false

	if got := daniel.SelfApprovalRefusal("U-1003", "U-1002"); got != "You own this activity, so someone else must approve it." {
		t.Errorf("owner: %q", got)
	}
	if got := daniel.SelfApprovalRefusal("U-1002", "U-1003"); got != "You submitted this activity, so someone else must approve it." {
		t.Errorf("submitter: %q", got)
	}
	if got := daniel.SelfApprovalRefusal("U-1002", "U-1002"); got != "" {
		t.Errorf("someone else's activity: %q", got)
	}
	if got := relaxed.SelfApprovalRefusal("U-1003", "U-1003"); got != "" {
		t.Errorf("rule off: %q", got)
	}
}

// Table A has 25 permissions in five groups; Lookup finds them by key and refuses others (FR-001).
func TestCatalogue_TableA_FR001(t *testing.T) {
	catalogue := authz.Catalogue()

	if len(catalogue) != 25 {
		t.Fatalf("%d permissions, want 25", len(catalogue))
	}
	groups := map[string]int{}
	for _, p := range catalogue {
		groups[p.Group]++
		if p.Label == "" || p.Noun == "" {
			t.Errorf("%s has no label or noun", p.Key)
		}
	}
	if len(groups) != 5 || groups[authz.GroupAdministration] != 7 {
		t.Errorf("groups = %v", groups)
	}
	if info, found := authz.Lookup("plans.edit"); !found || info.Label != "Edit plan details" || !info.Scoped {
		t.Errorf("Lookup(plans.edit) = %+v, %v", info, found)
	}
	if _, found := authz.Lookup("plans.fly"); found {
		t.Error("Lookup found an unknown key")
	}
	if got := authz.InOrder([]authz.Permission{authz.RecycleBinPurge, authz.PlansView, authz.PlansView}); len(got) != 2 || got[0] != authz.PlansView {
		t.Errorf("InOrder = %v", got)
	}
}
