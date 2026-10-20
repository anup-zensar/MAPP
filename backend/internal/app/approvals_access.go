package app

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"northstar/internal/approvals"
	"northstar/internal/people"
	"northstar/internal/settings"
)

// approvalsAccess implements approvals.Access (017 plan, Ports; D-290): what approvals needs to know about
// people's roles. It is a stand-in until feature 014's internal/access is merged: it gives the demo people
// the roles of 014's seed (Alex Morgan Administrator; Carlos Mendes Global planner; Daniel Okoro Planner and
// Approver; Priya Raman and Mei Lin Planner; Sara Novak Viewer; anyone else Viewer, D-234) with the approval
// permissions of 014 Table B, and keeps the access rule "Nobody approves an activity they own or submitted"
// on. When 014 is merged, this file is rewritten over 014's hooks (CanApprove, RoleHolders, roles with their
// permissions, the Rules tab, authz grants) without changing approvals.
type approvalsAccess struct {
	people people.Repository
}

// The compiler checks here that approvalsAccess implements approvals.Access.
var _ approvals.Access = approvalsAccess{}

// demoRole is one of 014's five seeded roles, with the approval permissions of Table B and its default
// scope: all markets, or the person's own teams.
type demoRole struct {
	ref         approvals.RoleRef
	permissions []approvals.Permission
	allMarkets  bool
}

// demoRoles are 014's seeded roles RL-1001 … RL-1005, by name.
var demoRoles = []demoRole{
	{approvals.RoleRef{ID: "RL-1001", Name: "Administrator", BuiltIn: "administrator"},
		[]approvals.Permission{approvals.PermissionDecide, approvals.PermissionDecideOnBehalf, approvals.PermissionManageWorkflows}, true},
	{approvals.RoleRef{ID: "RL-1004", Name: "Approver", BuiltIn: "approver"}, []approvals.Permission{approvals.PermissionDecide}, false},
	{approvals.RoleRef{ID: "RL-1002", Name: "Global planner", BuiltIn: "globalPlanner"}, []approvals.Permission{approvals.PermissionDecide}, true},
	{approvals.RoleRef{ID: "RL-1003", Name: "Planner", BuiltIn: "planner"}, nil, false},
	{approvals.RoleRef{ID: "RL-1005", Name: "Viewer", BuiltIn: "viewer"}, nil, false},
}

// demoAssignments are the role ids of 014's seed by the demo people's e-mail.
var demoAssignments = map[string][]string{
	"alex.morgan@northstar.demo":   {"RL-1001"},
	"carlos.mendes@northstar.demo": {"RL-1002"},
	"daniel.okoro@northstar.demo":  {"RL-1003", "RL-1004"},
	"priya.raman@northstar.demo":   {"RL-1003"},
	"mei.lin@northstar.demo":       {"RL-1003"},
	"sara.novak@northstar.demo":    {"RL-1005"},
}

// The refusal nouns of 014 Table A for the permissions approvals asks about.
var refusalNouns = map[approvals.Permission]string{
	approvals.PermissionDecide:          "Approving activities",
	approvals.PermissionDecideOnBehalf:  "Deciding on behalf of other approvers",
	approvals.PermissionManageWorkflows: "Changing approval workflows",
}

// rolesOf returns the roles of the person p, in the order of demoRoles.
func rolesOf(p people.Person) []demoRole {
	ids, assigned := demoAssignments[strings.ToLower(p.Email)]
	if !assigned {
		ids = []string{"RL-1005"} // new people get the Viewer role (014 D-234)
	}
	var roles []demoRole
	for _, r := range demoRoles {
		if slices.Contains(ids, r.ref.ID) {
			roles = append(roles, r)
		}
	}
	return roles
}

// covers reports whether the role r of the person personID covers the plan: all markets, or one of the
// person's teams is a team of the plan.
func (x approvalsAccess) covers(ctx context.Context, r demoRole, personID string, plan *approvals.PlanFacts) (bool, error) {
	if plan == nil || r.allMarkets {
		return true, nil
	}
	memberships, err := x.people.MembershipsOfPerson(ctx, personID)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(memberships, func(m people.Membership) bool { return slices.Contains(plan.TeamIDs, m.TeamID) }), nil
}

// CanApprove reports, for each of personIDs, whether one of their roles includes approvals.decide.
func (x approvalsAccess) CanApprove(ctx context.Context, personIDs []string) (map[string]bool, error) {
	can := make(map[string]bool, len(personIDs))
	for _, id := range personIDs {
		grant, err := x.Check(ctx, id, approvals.PermissionDecide, nil)
		if err != nil {
			return nil, err
		}
		can[id] = grant.Allowed
	}
	return can, nil
}

// Check reports whether the person personID holds p (in a scope that covers plan, when given), and 014's
// refusal message when not: "<Noun> isn't part of your role (<roles>). Ask an administrator if you need it."
func (x approvalsAccess) Check(ctx context.Context, personID string, p approvals.Permission, plan *approvals.PlanFacts) (approvals.Grant, error) {
	person, found, err := x.people.Get(ctx, personID)
	if err != nil || !found {
		return approvals.Grant{Refusal: fmt.Sprintf("%s needs a role, and you don't have one yet. Ask an administrator.", refusalNouns[p])}, err
	}
	roles := rolesOf(person)
	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, r.ref.Name)
		if !slices.Contains(r.permissions, p) {
			continue
		}
		covered, err := x.covers(ctx, r, personID, plan)
		if err != nil {
			return approvals.Grant{}, err
		}
		if covered {
			return approvals.Grant{Allowed: true}, nil
		}
	}
	yours := "your role"
	if len(names) > 1 {
		yours = "your roles"
	}
	return approvals.Grant{Refusal: fmt.Sprintf("%s isn't part of %s (%s). Ask an administrator if you need it.",
		refusalNouns[p], yours, strings.Join(names, ", "))}, nil
}

// NoSelfApproval is 014's access rule, on in the demo data.
func (approvalsAccess) NoSelfApproval(context.Context) (bool, error) {
	return true, nil
}

// Roles returns the five roles by name, with whether each includes approvals.decide.
func (approvalsAccess) Roles(context.Context) ([]approvals.RoleInfo, error) {
	roles := make([]approvals.RoleInfo, 0, len(demoRoles))
	for _, r := range demoRoles {
		roles = append(roles, approvals.RoleInfo{RoleRef: r.ref, Approves: slices.Contains(r.permissions, approvals.PermissionDecide)})
	}
	slices.SortFunc(roles, func(a, b approvals.RoleInfo) int { return strings.Compare(a.Name, b.Name) })
	return roles, nil
}

// RoleHolders returns the active people who hold the role roleID with a scope that covers plan, by name.
func (x approvalsAccess) RoleHolders(ctx context.Context, roleID string, plan approvals.PlanFacts) ([]string, error) {
	everyone, err := x.people.List(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(everyone, func(a, b people.Person) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	holders := []string{}
	for _, p := range everyone {
		if !p.Active {
			continue
		}
		for _, r := range rolesOf(p) {
			if r.ref.ID != roleID {
				continue
			}
			covered, err := x.covers(ctx, r, p.ID, &plan)
			if err != nil {
				return nil, err
			}
			if covered {
				holders = append(holders, p.ID)
			}
		}
	}
	return holders, nil
}

// approvalRuleChoices implements approvals.RuleChoices over feature 002's readers: every activity type and
// every option of the plan-level Business unit attribute, active or not, as Settings sorts them.
type approvalRuleChoices struct {
	settings *settings.Service
}

// The compiler checks here that approvalRuleChoices implements approvals.RuleChoices.
var _ approvals.RuleChoices = approvalRuleChoices{}

// ActivityTypes returns every activity type by name.
func (c approvalRuleChoices) ActivityTypes(ctx context.Context) ([]approvals.Choice, error) {
	types, err := c.settings.ActivityTypes(ctx)
	if err != nil {
		return nil, err
	}
	choices := make([]approvals.Choice, 0, len(types))
	for _, t := range types {
		choices = append(choices, approvals.Choice{ID: t.ID, Label: t.Name, Active: t.Active})
	}
	return choices, nil
}

// BusinessUnits returns the options of the Business unit attribute, in display order; none without it.
func (c approvalRuleChoices) BusinessUnits(ctx context.Context) ([]approvals.Choice, error) {
	attributes, err := c.settings.Attributes(ctx, settings.LevelPlan)
	if err != nil {
		return nil, err
	}
	choices := []approvals.Choice{}
	for _, a := range attributes {
		if a.Role != settings.RoleBusinessUnit {
			continue
		}
		options, err := c.settings.Options(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		for _, o := range options {
			choices = append(choices, approvals.Choice{ID: o.ID, Label: o.Label, Active: o.Active})
		}
	}
	return choices, nil
}
