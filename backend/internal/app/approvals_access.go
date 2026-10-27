package app

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/access"
	"mapp/internal/approval"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/settings"
)

// approvalsAccess implements approval.Access (017 plan, Ports; D-290) over feature 014's access service: who
// may approve, decide on behalf and manage workflows, the access rule "Nobody approves an activity they own
// or submitted", the roles and their holders, all as Settings > Roles and access stores them now.
type approvalsAccess struct {
	access *access.Service
}

// The compiler checks here that approvalsAccess implements approval.Access.
var _ approval.Access = approvalsAccess{}

// CanApprove reports, for each of personIDs, whether one of their roles includes approvals.decide.
func (x approvalsAccess) CanApprove(ctx context.Context, personIDs []string) (map[string]bool, error) {
	return x.access.CanApprove(ctx, personIDs)
}

// Check reports whether the person personID holds p (in a scope that covers plan, when given), and 014's
// refusal message when not.
func (x approvalsAccess) Check(ctx context.Context, personID string, p approval.Permission, plan *approval.PlanFacts) (approval.Grant, error) {
	g, err := x.access.GrantsFor(ctx, personID)
	if err != nil {
		return approval.Grant{}, err
	}
	perm := authz.Permission(p)
	if plan == nil {
		if g.Has(perm) {
			return approval.Grant{Allowed: true}, nil
		}
		return approval.Grant{Refusal: g.Refusal(perm, nil)}, nil
	}
	facts := authz.PlanFacts{ID: plan.ID, TeamIDs: plan.TeamIDs, OwnerID: plan.OwnerID, CreatedByID: plan.CreatedByID}
	if g.InScope(perm, facts) {
		return approval.Grant{Allowed: true}, nil
	}
	return approval.Grant{Refusal: g.Refusal(perm, &facts)}, nil
}

// NoSelfApproval reports whether 014's access rule "Nobody approves an activity they own or submitted" is on.
func (x approvalsAccess) NoSelfApproval(ctx context.Context) (bool, error) {
	rules, err := x.access.GetRules(ctx, actor.Actor{})
	return rules.NoSelfApproval, err
}

// Roles returns every role by name, with whether each includes approvals.decide.
func (x approvalsAccess) Roles(ctx context.Context) ([]approval.RoleInfo, error) {
	views, err := x.access.ListRoles(ctx, actor.Actor{})
	if err != nil {
		return nil, err
	}
	roles := make([]approval.RoleInfo, 0, len(views))
	for _, r := range views {
		approves := r.BuiltIn == authz.BuiltInAdministrator || slices.Contains(r.Permissions, authz.ApprovalsDecide)
		roles = append(roles, approval.RoleInfo{RoleRef: approval.RoleRef{ID: r.ID, Name: r.Name, BuiltIn: r.BuiltIn}, Approves: approves})
	}
	slices.SortStableFunc(roles, func(a, b approval.RoleInfo) int { return strings.Compare(a.Name, b.Name) })
	return roles, nil
}

// RoleHolders returns the active people who hold the role roleID with a scope that covers plan, by name.
func (x approvalsAccess) RoleHolders(ctx context.Context, roleID string, plan approval.PlanFacts) ([]string, error) {
	return x.access.RoleHolders(ctx, roleID, authz.PlanFacts{ID: plan.ID, TeamIDs: plan.TeamIDs, OwnerID: plan.OwnerID,
		CreatedByID: plan.CreatedByID})
}

// approvalRuleChoices implements approval.RuleChoices over feature 002's readers: every activity type and
// every option of the plan-level Business unit attribute, active or not, as Settings sorts them.
type approvalRuleChoices struct {
	settings *settings.Service
}

// The compiler checks here that approvalRuleChoices implements approval.RuleChoices.
var _ approval.RuleChoices = approvalRuleChoices{}

// ActivityTypes returns every activity type by name.
func (c approvalRuleChoices) ActivityTypes(ctx context.Context) ([]approval.Choice, error) {
	types, err := c.settings.ActivityTypes(ctx)
	if err != nil {
		return nil, err
	}
	choices := make([]approval.Choice, 0, len(types))
	for _, t := range types {
		choices = append(choices, approval.Choice{ID: t.ID, Label: t.Name, Active: t.Active})
	}
	return choices, nil
}

// BusinessUnits returns the options of the Business unit attribute, in display order; none without it.
func (c approvalRuleChoices) BusinessUnits(ctx context.Context) ([]approval.Choice, error) {
	attributes, err := c.settings.Attributes(ctx, settings.LevelPlan)
	if err != nil {
		return nil, err
	}
	choices := []approval.Choice{}
	for _, a := range attributes {
		if a.Role != settings.RoleBusinessUnit {
			continue
		}
		options, err := c.settings.Options(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		for _, o := range options {
			choices = append(choices, approval.Choice{ID: o.ID, Label: o.Label, Active: o.Active})
		}
	}
	return choices, nil
}
