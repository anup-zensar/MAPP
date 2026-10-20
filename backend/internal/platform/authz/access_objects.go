package authz

import "context"

// The access objects of the API's answers (feature 014, D-229): for each action on a record, whether the
// signed-in person may take it now and, when not, the exact refusal the API would give (FR-011). The web
// app shows them and never works a permission out itself. Each builder answers nil for an internal call (a
// context without Grants), so answers built outside a request leave the optional property out; every API
// request carries Grants, so the API always sends it.

// PlanContentAccess is what the person may do with the records inside one plan (PlanContentAccess): tree
// rows, Add buttons and row menus.
type PlanContentAccess struct {
	AddCampaign      Availability `json:"addCampaign"`
	EditCampaigns    Availability `json:"editCampaigns"` // add initiatives, edit, move and copy campaigns and initiatives
	DeleteCampaigns  Availability `json:"deleteCampaigns"`
	AddActivity      Availability `json:"addActivity"`
	EditActivities   Availability `json:"editActivities"` // edit, move and copy activities; add, edit and duplicate line items
	DeleteActivities Availability `json:"deleteActivities"`
	SubmitActivities Availability `json:"submitActivities"`
}

// PlanAccess is what the person may do with one plan (PlanAccess).
type PlanAccess struct {
	Edit       Availability      `json:"edit"`
	EditBudget Availability      `json:"editBudget"` // also not available for an allocated plan (global budgets)
	Delete     Availability      `json:"delete"`
	Publish    Availability      `json:"publish"` // Publish and Unpublish
	Copy       Availability      `json:"copy"`
	Content    PlanContentAccess `json:"content"`
}

// ContainerAccess is what the person may do with a campaign or an initiative (ContainerAccess); AddChild is
// Add initiative or Add activity.
type ContainerAccess struct {
	Edit     Availability `json:"edit"`
	Delete   Availability `json:"delete"`
	Move     Availability `json:"move"`
	Copy     Availability `json:"copy"`
	AddChild Availability `json:"addChild"`
}

// ActivityAccess is what the person may do with an activity (ActivityAccess).
type ActivityAccess struct {
	Edit            Availability `json:"edit"`
	Delete          Availability `json:"delete"`
	Move            Availability `json:"move"`
	Copy            Availability `json:"copy"`
	AddLineItem     Availability `json:"addLineItem"`
	EditLineItems   Availability `json:"editLineItems"`
	DeleteLineItems Availability `json:"deleteLineItems"`
	Submit          Availability `json:"submit"`     // the permission only; 008's actions add the activity's own state
	SeeFinance      Availability `json:"seeFinance"` // the "From finance" sections (FR-015)
}

// LineItemAccess is what the person may do with the line items of an activity (LineItemAccess).
type LineItemAccess struct {
	Add        Availability `json:"add"`
	Edit       Availability `json:"edit"`
	Delete     Availability `json:"delete"`
	Duplicate  Availability `json:"duplicate"`
	SeeFinance Availability `json:"seeFinance"`
}

// BinEntryAccess is what the person may do with a recycle-bin entry (BinEntryAccess).
type BinEntryAccess struct {
	Restore           Availability `json:"restore"` // also Undo and the restore preview
	DeletePermanently Availability `json:"deletePermanently"`
}

// ContentAccessOf returns what the person acting may do inside the plan; nil for an internal call.
func ContentAccessOf(ctx context.Context, plan PlanFacts) *PlanContentAccess {
	if _, restricted := From(ctx); !restricted {
		return nil
	}
	content := contentAccess(ctx, plan)
	return &content
}

// contentAccess is ContentAccessOf for a request.
func contentAccess(ctx context.Context, plan PlanFacts) PlanContentAccess {
	return PlanContentAccess{
		AddCampaign:      Available(ctx, CampaignsEdit, plan),
		EditCampaigns:    Available(ctx, CampaignsEdit, plan),
		DeleteCampaigns:  Available(ctx, CampaignsDelete, plan),
		AddActivity:      Available(ctx, ActivitiesEdit, plan),
		EditActivities:   Available(ctx, ActivitiesEdit, plan),
		DeleteActivities: Available(ctx, ActivitiesDelete, plan),
		SubmitActivities: Available(ctx, ActivitiesSubmit, plan),
	}
}

// PlanAccessOf returns what the person acting may do with the plan; nil for an internal call. Copying needs
// "Create and copy plans" for the plan's teams, as the copy keeps them.
func PlanAccessOf(ctx context.Context, plan PlanFacts) *PlanAccess {
	g, restricted := From(ctx)
	if !restricted {
		return nil
	}
	copyFacts := PlanFacts{ID: plan.ID, TeamIDs: plan.TeamIDs, OwnerID: plan.OwnerID, CreatedByID: g.PersonID}
	edit := Available(ctx, PlansEdit, plan)
	return &PlanAccess{
		Edit:       edit,
		EditBudget: edit,
		Delete:     Available(ctx, PlansDelete, plan),
		Publish:    Available(ctx, PlansPublish, plan),
		Copy:       Available(ctx, PlansCreate, copyFacts),
		Content:    contentAccess(ctx, plan),
	}
}

// CampaignAccessOf returns what the person acting may do with a campaign of the plan; nil for an internal
// call. Add initiative is a change of campaigns and initiatives.
func CampaignAccessOf(ctx context.Context, plan PlanFacts) *ContainerAccess {
	return containerAccess(ctx, plan, CampaignsEdit)
}

// InitiativeAccessOf returns what the person acting may do with an initiative of the plan; nil for an
// internal call. Add activity is a change of activities.
func InitiativeAccessOf(ctx context.Context, plan PlanFacts) *ContainerAccess {
	return containerAccess(ctx, plan, ActivitiesEdit)
}

// containerAccess is the ContainerAccess of a campaign or initiative whose children need addChild.
func containerAccess(ctx context.Context, plan PlanFacts, addChild Permission) *ContainerAccess {
	if _, restricted := From(ctx); !restricted {
		return nil
	}
	edit := Available(ctx, CampaignsEdit, plan)
	return &ContainerAccess{
		Edit: edit, Delete: Available(ctx, CampaignsDelete, plan), Move: edit, Copy: edit,
		AddChild: Available(ctx, addChild, plan),
	}
}

// ActivityAccessOf returns what the person acting may do with an activity of the plan; nil for an internal
// call.
func ActivityAccessOf(ctx context.Context, plan PlanFacts) *ActivityAccess {
	if _, restricted := From(ctx); !restricted {
		return nil
	}
	edit, remove := Available(ctx, ActivitiesEdit, plan), Available(ctx, ActivitiesDelete, plan)
	return &ActivityAccess{
		Edit: edit, Delete: remove, Move: edit, Copy: edit, AddLineItem: edit, EditLineItems: edit, DeleteLineItems: remove,
		Submit: Available(ctx, ActivitiesSubmit, plan), SeeFinance: Available(ctx, FinanceView, plan),
	}
}

// LineItemAccessOf returns what the person acting may do with the line items of an activity of the plan;
// nil for an internal call.
func LineItemAccessOf(ctx context.Context, plan PlanFacts) *LineItemAccess {
	if _, restricted := From(ctx); !restricted {
		return nil
	}
	edit := Available(ctx, ActivitiesEdit, plan)
	return &LineItemAccess{
		Add: edit, Edit: edit, Delete: Available(ctx, ActivitiesDelete, plan), Duplicate: edit,
		SeeFinance: Available(ctx, FinanceView, plan),
	}
}
