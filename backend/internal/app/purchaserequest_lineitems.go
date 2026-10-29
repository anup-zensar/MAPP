package app

import (
	"context"
	"slices"
	"strings"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/lineitem"
	"mapp/internal/plan"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/purchaserequest"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// prLineItems implements purchaserequest.LineItems over the planning stores (D-196): each line item with its
// activity, path, plan, SWBS status (030) and the finance roleCoding of the attribute roles (032 data-model §4).
type prLineItems struct {
	set       stores.Set
	settings  *settings.Service
	lineItems *lineitem.Service
}

var _ purchaserequest.LineItems = prLineItems{}

func (x prLineItems) Facts(ctx context.Context, ids []string) (map[string]purchaserequest.LineItemFacts, error) {
	items, err := x.set.LineItems.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return x.factsOf(ctx, items)
}

func (x prLineItems) Search(ctx context.Context, scope authz.Scope, q purchaserequest.LineItemSearch) ([]purchaserequest.LineItemFacts, error) {
	acts, err := x.set.Activities.ListInScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(acts))
	for _, a := range acts {
		ids = append(ids, a.ID)
	}
	items, err := x.set.LineItems.ListByActivities(ctx, ids)
	if err != nil {
		return nil, err
	}
	facts, err := x.factsOf(ctx, items)
	if err != nil {
		return nil, err
	}
	found := []purchaserequest.LineItemFacts{}
	for _, li := range items {
		if f := facts[li.ID]; searched(f, scope, q) {
			found = append(found, f)
		}
	}
	return found[:min(q.Limit, len(found))], nil
}

// searched reports whether f is in a plan of the scope (and of q.PlanID) and its name, ID or HX key holds q.Text.
func searched(f purchaserequest.LineItemFacts, scope authz.Scope, q purchaserequest.LineItemSearch) bool {
	text := strings.ToLower(f.Name + " " + f.ID + " " + purchaserequest.HXKey(f.ID))
	inPlan := q.PlanID == "" || f.Plan.Access.ID == q.PlanID
	return strings.Contains(text, strings.ToLower(q.Text)) && scope.Covers(f.Plan.Access) && inPlan
}

func (x prLineItems) Activity(ctx context.Context, activityID string) (purchaserequest.ActivityLineItems, bool, error) {
	act, found, err := x.set.Activities.Get(ctx, activityID)
	if err != nil || !found {
		return purchaserequest.ActivityLineItems{}, false, err
	}
	items, err := x.set.LineItems.ListByActivity(ctx, activityID)
	if err != nil {
		return purchaserequest.ActivityLineItems{}, false, err
	}
	facts, err := x.factsOf(ctx, items)
	if err != nil {
		return purchaserequest.ActivityLineItems{}, false, err
	}
	parents, err := x.parentsOf(ctx, []activity.Activity{act})
	if err != nil {
		return purchaserequest.ActivityLineItems{}, false, err
	}
	of := parents[act.ID]
	section := purchaserequest.ActivityLineItems{ID: act.ID, Name: act.Name, Plan: of.plan, Access: of.access,
		LineItems: make([]purchaserequest.LineItemFacts, 0, len(items))}
	for _, li := range items {
		section.LineItems = append(section.LineItems, facts[li.ID])
	}
	return section, true, nil
}

// factsOf puts the facts of items together, reading their activities, parents and Settings once for all.
func (x prLineItems) factsOf(ctx context.Context, items []lineitem.LineItem) (map[string]purchaserequest.LineItemFacts, error) {
	activityIDs := make([]string, 0, len(items))
	for _, li := range items {
		activityIDs = append(activityIDs, li.ActivityID)
	}
	acts, err := x.set.Activities.ListByIDs(ctx, activityIDs)
	if err != nil {
		return nil, err
	}
	c, err := x.roleCoding(ctx)
	if err != nil {
		return nil, err
	}
	parents, err := x.parentsOf(ctx, acts)
	if err != nil {
		return nil, err
	}
	overBy, err := x.overspentActivities(ctx, acts)
	if err != nil {
		return nil, err
	}
	facts := make(map[string]purchaserequest.LineItemFacts, len(items))
	for _, li := range items {
		if of, found := parents[li.ActivityID]; found {
			f := of.lineItem(li, c)
			f.ActivityOverBy = overBy[li.ActivityID]
			facts[li.ID] = f
		}
	}
	return facts, nil
}

// activityOf is what a line item's facts take from its activity and the levels above it.
type activityOf struct {
	act          activity.Activity
	plan         purchaserequest.PlanFacts
	access       authz.ActivityFacts
	path         []errs.RecordRef
	businessUnit []string
}

func (o activityOf) lineItem(li lineitem.LineItem, c roleCoding) purchaserequest.LineItemFacts {
	return purchaserequest.LineItemFacts{
		ID: li.ID, Name: li.Name, OwnerID: li.OwnerID, Currency: li.Currency, PlannedCost: li.PlannedCost,
		ActivityID: o.act.ID, ActivityName: o.act.Name, ActivityStatus: string(o.act.Status), Plan: o.plan,
		Access: o.access, Path: o.path, SwbsCode: li.SwbsCode,
		SwbsStatus:  string(lineitem.SwbsStatusOf(li, lineitem.ActivityFacts{Status: string(o.act.Status)})),
		CompanyCode: c.text(settings.RoleCompanyCode, o.act.Attributes), CostCentre: c.text(settings.RoleCostCentre, o.act.Attributes),
		GLCode: c.text(settings.RoleGLCode, li.Attributes), BusinessUnit: o.businessUnit,
	}
}

func (x prLineItems) parentsOf(ctx context.Context, acts []activity.Activity) (map[string]activityOf, error) {
	approvers, err := x.approvers(ctx)
	if err != nil {
		return nil, err
	}
	paths := lineItemActivities{activities: x.set.Activities, campaigns: x.set.Campaigns, plans: x.set.Plans, settings: x.settings}
	if err = paths.readParents(ctx, acts); err != nil {
		return nil, err
	}
	parents := make(map[string]activityOf, len(acts))
	for _, act := range acts {
		path, p, err := paths.pathOf(ctx, act.InitiativeID)
		if err != nil {
			return nil, err
		}
		unit, err := x.settings.BusinessUnit(ctx, p.Attributes)
		if err != nil {
			return nil, err
		}
		access := plan.AccessFacts(p)
		parents[act.ID] = activityOf{act: act, plan: purchaserequest.PlanFacts{Access: access, Name: p.Name, Currency: p.Currency},
			access: authz.ActivityFacts{Plan: access, OwnerID: act.OwnerID, ApproverIDs: approvers[act.ID]},
			path:   append(path, errs.RecordRef{Type: string(approval.KindActivity), ID: act.ID, Name: act.Name}), businessUnit: unit}
	}
	return parents, nil
}

// approvers are the people named in any level of any round of each activity (authz.CanViewActivity).
func (x prLineItems) approvers(ctx context.Context) (map[string][]string, error) {
	rounds, err := x.set.Approvals.ListRounds(ctx)
	if err != nil {
		return nil, err
	}
	ids := map[string][]string{}
	for _, r := range rounds {
		for _, level := range r.Levels {
			for _, approver := range level.Approvers {
				ids[r.SubjectID] = append(ids[r.SubjectID], approver.ID)
			}
		}
	}
	return ids, nil
}

// roleCoding reads the attributes with a role and the labels of their options, once per call.
type roleCoding struct {
	byRole map[string]settings.Attribute
	labels map[string]string
}

func (x prLineItems) roleCoding(ctx context.Context) (roleCoding, error) {
	return codingOf(ctx, x.set.Settings)
}

func codingOf(ctx context.Context, repo stores.SettingsStore) (roleCoding, error) {
	attributes, err := repo.ListAttributes(ctx)
	if err != nil {
		return roleCoding{}, err
	}
	options, err := repo.ListOptions(ctx)
	if err != nil {
		return roleCoding{}, err
	}
	c := roleCoding{byRole: map[string]settings.Attribute{}, labels: map[string]string{}}
	for _, a := range attributes {
		if a.Role != "" {
			c.byRole[a.Role] = a
		}
	}
	for _, o := range options {
		c.labels[o.ID] = o.Label
	}
	return c, nil
}

// texts returns the value of the attribute with role among values: a text as typed, the labels of chosen options.
func (c roleCoding) texts(role string, values settings.AttributeValues) []string {
	a, found := c.byRole[role]
	if !found {
		return nil
	}
	raw, found := values[a.ID]
	if !found {
		return nil
	}
	var list []string
	for _, v := range decodedValues(raw) {
		if label, isOption := c.labels[v]; isOption && a.IsChoice() {
			v = label
		}
		if v = strings.TrimSpace(v); v != "" && !slices.Contains(list, v) {
			list = append(list, v)
		}
	}
	return list
}

func (c roleCoding) text(role string, values settings.AttributeValues) string {
	if list := c.texts(role, values); len(list) > 0 {
		return list[0]
	}
	return ""
}
