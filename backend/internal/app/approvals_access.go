package app

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/access"
	"mapp/internal/approvals"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/settings"
)

// approvalsAccess implements approvals.Access (017 plan, Ports; D-290) over feature 014's access service: who
// may approve, decide on behalf and manage workflows, the access rule "Nobody approves an activity they own
// or submitted", the roles and their holders, all as Settings > Roles and access stores them now.
type approvalsAccess struct {
	access *access.Service
}

// The compiler checks here that approvalsAccess implements approvals.Access.
var _ approvals.Access = approvalsAccess{}

// CanApprove reports, for each of personIDs, whether one of their roles includes approvals.decide.
func (x approvalsAccess) CanApprove(ctx context.Context, personIDs []string) (map[string]bool, error) {
	return x.access.CanApprove(ctx, personIDs)
}

// Check reports whether the person personID holds p (in a scope that covers plan, when given), and 014's
// refusal message when not.
func (x approvalsAccess) Check(ctx context.Context, personID string, p approvals.Permission, plan *approvals.PlanFacts) (approvals.Grant, error) {
	g, err := x.access.GrantsFor(ctx, personID)
	if err != nil {
		return approvals.Grant{}, err
	}
	perm := authz.Permission(p)
	if plan == nil {
		if g.Has(perm) {
			return approvals.Grant{Allowed: true}, nil
		}
		return approvals.Grant{Refusal: g.Refusal(perm, nil)}, nil
	}
	facts := authz.PlanFacts{ID: plan.ID, TeamIDs: plan.TeamIDs, OwnerID: plan.OwnerID, CreatedByID: plan.CreatedByID}
	if g.InScope(perm, facts) {
		return approvals.Grant{Allowed: true}, nil
	}
	return approvals.Grant{Refusal: g.Refusal(perm, &facts)}, nil
}

// NoSelfApproval reports whether 014's access rule "Nobody approves an activity they own or submitted" is on.
func (x approvalsAccess) NoSelfApproval(ctx context.Context) (bool, error) {
	rules, err := x.access.GetRules(ctx, actor.Actor{})
	return rules.NoSelfApproval, err
}

// Roles returns every role by name, with whether each includes approvals.decide.
func (x approvalsAccess) Roles(ctx context.Context) ([]approvals.RoleInfo, error) {
	views, err := x.access.ListRoles(ctx, actor.Actor{})
	if err != nil {
		return nil, err
	}
	roles := make([]approvals.RoleInfo, 0, len(views))
	for _, r := range views {
		approves := r.BuiltIn == authz.BuiltInAdministrator || slices.Contains(r.Permissions, authz.ApprovalsDecide)
		roles = append(roles, approvals.RoleInfo{RoleRef: approvals.RoleRef{ID: r.ID, Name: r.Name, BuiltIn: r.BuiltIn}, Approves: approves})
	}
	slices.SortStableFunc(roles, func(a, b approvals.RoleInfo) int { return strings.Compare(a.Name, b.Name) })
	return roles, nil
}

// RoleHolders returns the active people who hold the role roleID with a scope that covers plan, by name.
func (x approvalsAccess) RoleHolders(ctx context.Context, roleID string, plan approvals.PlanFacts) ([]string, error) {
	return x.access.RoleHolders(ctx, roleID, authz.PlanFacts{ID: plan.ID, TeamIDs: plan.TeamIDs, OwnerID: plan.OwnerID,
		CreatedByID: plan.CreatedByID})
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
