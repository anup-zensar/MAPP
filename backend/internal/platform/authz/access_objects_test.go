package authz_test

import (
	"context"
	"testing"

	"mapp/internal/platform/authz"
)

// An answer built outside a request (no Grants in the context) leaves its access object out (FR-011).
func TestAccessObjects_InternalCallLeavesThemOut_FR011(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := plan(lubricants)

	if authz.PlanAccessOf(ctx, p) != nil || authz.ContentAccessOf(ctx, p) != nil || authz.CampaignAccessOf(ctx, p) != nil ||
		authz.InitiativeAccessOf(ctx, p) != nil || authz.ActivityAccessOf(ctx, p) != nil || authz.LineItemAccessOf(ctx, p) != nil {
		t.Error("an internal call got an access object, want none")
	}
}

// A Planner of the plan's team may change its contents and submit but not delete the plan; the reason of
// each unavailable action is the refusal the API would answer (FR-011, FR-009).
func TestPlanAccessOf_Planner_FR011(t *testing.T) {
	t.Parallel()
	priya := authz.With(context.Background(), grants("U-1002", []string{lubricants}, planner))

	got := authz.PlanAccessOf(priya, plan(lubricants))

	for name, a := range map[string]authz.Availability{
		"edit": got.Edit, "editBudget": got.EditBudget, "publish": got.Publish, "copy": got.Copy,
		"addCampaign": got.Content.AddCampaign, "editCampaigns": got.Content.EditCampaigns,
		"deleteCampaigns": got.Content.DeleteCampaigns, "addActivity": got.Content.AddActivity,
		"editActivities": got.Content.EditActivities, "deleteActivities": got.Content.DeleteActivities,
		"submitActivities": got.Content.SubmitActivities,
	} {
		if !a.Available || a.Reason != "" {
			t.Errorf("%s = %+v, want available", name, a)
		}
	}
	want := "Deleting plans isn't part of your role (Planner). Ask an administrator if you need it."
	if got.Delete.Available || got.Delete.Reason != want {
		t.Errorf("delete = %+v, want unavailable with %q", got.Delete, want)
	}
}

// Outside the Planner's teams nothing can be changed, and the reason names the teams the person may change
// (message 3, FR-011).
func TestPlanAccessOf_OtherMarket_FR011(t *testing.T) {
	t.Parallel()
	priya := authz.With(context.Background(), grants("U-1002", []string{lubricants}, planner))

	got := authz.PlanAccessOf(priya, plan(mobility))

	want := "Editing plans is allowed for you only in plans of Lubricants Europe. Ask an administrator if you need it here."
	if got.Edit.Available || got.Edit.Reason != want {
		t.Errorf("edit = %+v\nwant unavailable with %q", got.Edit, want)
	}
	if got.Content.EditActivities.Available || got.Copy.Available {
		t.Errorf("content = %+v, copy = %+v, want unavailable", got.Content, got.Copy)
	}
}

// A Viewer sees records but every change action is unavailable; activities and line items still show the
// "From finance" sections, as every default role may see financial data (FR-011, FR-015).
func TestActivityAndLineItemAccessOf_Viewer_FR015(t *testing.T) {
	t.Parallel()
	sara := authz.With(context.Background(), grants("U-1006", []string{lubricants, mobility}, viewer))
	p := plan(lubricants)

	activity := authz.ActivityAccessOf(sara, p)
	lineItems := authz.LineItemAccessOf(sara, p)

	for name, a := range map[string]authz.Availability{
		"activity edit": activity.Edit, "activity delete": activity.Delete, "activity move": activity.Move,
		"activity copy": activity.Copy, "addLineItem": activity.AddLineItem, "editLineItems": activity.EditLineItems,
		"deleteLineItems": activity.DeleteLineItems, "submit": activity.Submit,
		"line item add": lineItems.Add, "line item edit": lineItems.Edit, "line item delete": lineItems.Delete,
		"line item duplicate": lineItems.Duplicate,
	} {
		if a.Available || a.Reason == "" {
			t.Errorf("%s = %+v, want unavailable with a reason", name, a)
		}
	}
	if !activity.SeeFinance.Available || !lineItems.SeeFinance.Available {
		t.Errorf("seeFinance = %+v / %+v, want available", activity.SeeFinance, lineItems.SeeFinance)
	}
}

// Without "See financial data" in the plan the "From finance" sections are not shown (FR-015).
func TestActivityAccessOf_NoFinance_FR015(t *testing.T) {
	t.Parallel()
	custom := role("RL-1010", "Plan editor", "", authz.ScopeMyTeams)
	g := authz.NewGrants("U-1006", []authz.GrantedRole{custom}, []string{lubricants},
		map[string][]authz.Permission{"RL-1010": {authz.PlansView, authz.ActivitiesEdit}}, nil, authz.Rules{})
	ctx := authz.With(context.Background(), g)

	activity := authz.ActivityAccessOf(ctx, plan(lubricants))
	lineItems := authz.LineItemAccessOf(ctx, plan(lubricants))

	if activity.SeeFinance.Available || lineItems.SeeFinance.Available {
		t.Errorf("seeFinance = %+v / %+v, want unavailable", activity.SeeFinance, lineItems.SeeFinance)
	}
	if !activity.Edit.Available || !lineItems.Edit.Available || activity.Delete.Available {
		t.Errorf("edit = %+v, line items %+v, delete %+v", activity.Edit, lineItems.Edit, activity.Delete)
	}
}

// A campaign's Add initiative is a change of campaigns; an initiative's Add activity is a change of
// activities (FR-011, FR-014).
func TestContainerAccessOf_AddChild_FR011(t *testing.T) {
	t.Parallel()
	custom := role("RL-1011", "Activity editor", "", authz.ScopeMyTeams)
	g := authz.NewGrants("U-1006", []authz.GrantedRole{custom}, []string{lubricants},
		map[string][]authz.Permission{"RL-1011": {authz.PlansView, authz.ActivitiesEdit}}, nil, authz.Rules{})
	ctx := authz.With(context.Background(), g)

	campaign := authz.CampaignAccessOf(ctx, plan(lubricants))
	initiative := authz.InitiativeAccessOf(ctx, plan(lubricants))

	if campaign.AddChild.Available || campaign.Edit.Available || campaign.Move.Available {
		t.Errorf("campaign = %+v, want no changes", campaign)
	}
	if !initiative.AddChild.Available || initiative.Edit.Available || initiative.Delete.Available {
		t.Errorf("initiative = %+v, want Add activity only", initiative)
	}
}
