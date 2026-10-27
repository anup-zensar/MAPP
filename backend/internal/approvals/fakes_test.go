package approvals

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/money"
)

// fakeActivities is the Activities of the service tests: activities in memory, a ready-to-submit check that
// lists the problems a test sets, and line items a test sets.
type fakeActivities struct {
	clock      *testClock
	facts      map[string]ActivityFacts
	problems   map[string][]SubmitProblem
	missing    map[string][]MissingFieldRef
	unassigned map[string]*money.Money
	lineItems  map[string][]PreviewLineItem
	order      []string // ids in the order they were added
}

func (f *fakeActivities) Activity(_ context.Context, id string) (ActivityFacts, bool, error) {
	facts, found := f.facts[id]
	return facts, found, nil
}

func (f *fakeActivities) ListOf(_ context.Context, activityIDs []string) ([]ActivityFacts, error) {
	list := []ActivityFacts{}
	for _, id := range f.order {
		if facts, found := f.facts[id]; found && slices.Contains(activityIDs, id) {
			list = append(list, facts)
		}
	}
	return list, nil
}

func (f *fakeActivities) WorkflowFields(context.Context) (map[string]string, error) {
	fields := map[string]string{}
	for id, facts := range f.facts {
		fields[id] = facts.WorkflowID
	}
	return fields, nil
}

// ListInScope applies the scope to the plan and the owner; the fake knows no round's approvers.
func (f *fakeActivities) ListInScope(ctx context.Context, scope authz.Scope, activityIDs []string) ([]ActivityFacts, error) {
	listed, err := f.ListOf(ctx, activityIDs)
	return slices.DeleteFunc(listed, func(facts ActivityFacts) bool {
		return !scope.SeesActivity(authz.ActivityFacts{Plan: facts.Access, OwnerID: facts.OwnerID})
	}), err
}

// SetStatus changes the status, and the version goes up by one, as the adapter's Touch does.
func (f *fakeActivities) SetStatus(_ context.Context, a actor.Actor, id, status string) (ActivityFacts, error) {
	facts, found := f.facts[id]
	if !found {
		return ActivityFacts{}, fmt.Errorf("fake: activity %s is not stored", id)
	}
	facts.Status, facts.DisplayStatus = status, status
	facts.Touch(a, f.clock.Now())
	f.facts[id] = facts
	return facts, nil
}

func (f *fakeActivities) ReadyCheck(_ context.Context, _ actor.Actor, id string) (ReadyCheck, error) {
	return ReadyCheck{Problems: f.problems[id], UnassignedPlannedCost: f.unassigned[id], Missing: f.missing[id]}, nil
}

func (f *fakeActivities) LineItems(_ context.Context, _ actor.Actor, activityID string) ([]PreviewLineItem, error) {
	if items, found := f.lineItems[activityID]; found {
		return items, nil
	}
	return []PreviewLineItem{}, nil
}

// The rule choices of the fixture (017): two activity types and two business units.
const (
	eventType       = "AT-1001" // Event
	sponsorshipType = "AT-1002" // Sponsorship
	lubricantsUnit  = "SO-1001" // Lubricants
	mobilityUnit    = "SO-1002" // Mobility
)

// fakeChoices is the RuleChoices of the service tests.
type fakeChoices struct{}

func (fakeChoices) ActivityTypes(context.Context) ([]Choice, error) {
	return []Choice{{ID: eventType, Label: "Event", Active: true}, {ID: sponsorshipType, Label: "Sponsorship", Active: true}}, nil
}

func (fakeChoices) BusinessUnits(context.Context) ([]Choice, error) {
	return []Choice{{ID: lubricantsUnit, Label: "Lubricants", Active: true}, {ID: mobilityUnit, Label: "Mobility", Active: true}}, nil
}

// The roles of the fixture's fake access (014 seed names).
var (
	approverRole = RoleInfo{RoleRef: RoleRef{ID: "RL-1004", Name: "Approver", BuiltIn: "approver"}, Approves: true}
	viewerRole   = RoleInfo{RoleRef: RoleRef{ID: "RL-1005", Name: "Viewer", BuiltIn: "viewer"}, Approves: false}
)

// fakeAccess is the Access of the service tests: everyone may approve except cannotApprove; everyone may
// decide on behalf and manage workflows unless the lists are set; the access rule is on unless a test turns
// it off; two roles and their holders by role id.
type fakeAccess struct {
	rule          bool
	cannotApprove map[string]bool
	onBehalf      []string // nil: everyone may decide on behalf
	managers      []string // nil: everyone holds workflows.manage
	holders       map[string][]string
}

func (f *fakeAccess) CanApprove(_ context.Context, personIDs []string) (map[string]bool, error) {
	can := map[string]bool{}
	for _, id := range personIDs {
		can[id] = !f.cannotApprove[id]
	}
	return can, nil
}

func (f *fakeAccess) Check(_ context.Context, personID string, p Permission, _ *PlanFacts) (Grant, error) {
	var list []string
	switch p {
	case PermissionDecide:
		if f.cannotApprove[personID] {
			return Grant{Refusal: "Approving activities isn't part of your role (Viewer). Ask an administrator if you need it."}, nil
		}
		return Grant{Allowed: true}, nil
	case PermissionDecideOnBehalf:
		list = f.onBehalf
	case PermissionManageWorkflows:
		list = f.managers
	}
	if list == nil || slices.Contains(list, personID) {
		return Grant{Allowed: true}, nil
	}
	return Grant{Refusal: "Changing approval workflows isn't part of your role (Planner). Ask an administrator if you need it."}, nil
}

func (f *fakeAccess) NoSelfApproval(context.Context) (bool, error) { return f.rule, nil }

func (f *fakeAccess) Roles(context.Context) ([]RoleInfo, error) {
	return []RoleInfo{approverRole, viewerRole}, nil
}

func (f *fakeAccess) RoleHolders(_ context.Context, roleID string, _ PlanFacts) ([]string, error) {
	return f.holders[roleID], nil
}
