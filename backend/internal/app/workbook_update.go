package app

import (
	"context"
	"strings"

	"mapp/internal/activity"
	"mapp/internal/campaign"
	"mapp/internal/lineitem"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
	"mapp/internal/workbook"
)

const (
	workbookAttrPrefix = "attributes."
	formName           = "name"
	formDescription    = "description"
	formOwner          = "ownerId"
	formCurrency       = "currency"
	formPlannedCost    = "plannedCost"
)

// containerForm points at the fields a campaign's and an initiative's forms share.
type containerForm struct {
	name, description *string
	owner             **string
	start, end        **date.Date
	budget            **money.Input
	attributes        settings.AttributeValues
}

// Update sets the fields named in present on the record's form and saves it with the single edit (FR-029).
func (p workbookPlanning) Update(ctx context.Context, a actor.Actor, c workbook.Change) error {
	id, v, present := c.ID, c.Values, c.Present
	var err error
	switch prefix, _, _ := strings.Cut(id, "-"); prefix {
	case "C":
		_, err = p.editCampaign(ctx, a, id, func(u *campaign.CampaignUpdate) bool {
			return setContainer(containerForm{&u.Name, &u.Description, &u.OwnerID, &u.StartDate, &u.EndDate, &u.Budget, u.Attributes}, v, present)
		})
	case "I":
		_, err = p.editInitiative(ctx, a, id, func(u *campaign.InitiativeUpdate) bool {
			return setContainer(containerForm{&u.Name, &u.Description, &u.OwnerID, &u.StartDate, &u.EndDate, &u.Budget, u.Attributes}, v, present)
		})
	case "A":
		_, err = p.editActivity(ctx, a, id, func(u *activity.Update) bool { return setActivity(u, v, present) })
	default:
		_, err = p.editLineItem(ctx, a, id, func(u *lineitem.Update) bool { return setLineItem(u, v, present) })
	}
	return err
}

func setContainer(f containerForm, v workbook.Values, present map[string]bool) bool {
	for field := range present {
		switch field {
		case formName:
			*f.name = v.Name
		case formDescription:
			*f.description = v.Description
		case formOwner:
			*f.owner = textOrNil(v.OwnerID)
		case keyStartDate:
			*f.start = v.StartDate
		case keyEndDate:
			*f.end = v.EndDate
		case "budget":
			*f.budget = amountInput(v.Budget, v.Currency)
		}
	}
	setAttributes(f.attributes, v, present)
	return true
}

func setActivity(u *activity.Update, v workbook.Values, present map[string]bool) bool {
	for field := range present {
		switch field {
		case formName:
			u.Name = v.Name
		case formDescription:
			u.Description = v.Description
		case "externalName":
			u.ExternalName = v.ExternalName
		case formOwner:
			u.OwnerID = textOrNil(v.OwnerID)
		case keyStartDate:
			u.StartDate = dateText(v.StartDate)
		case keyEndDate:
			u.EndDate = dateText(v.EndDate)
		}
	}
	setActivityMore(u, v, present)
	setAttributes(u.Attributes, v, present)
	return true
}

func setActivityMore(u *activity.Update, v workbook.Values, present map[string]bool) {
	if present["ongoing"] {
		u.Ongoing = v.Ongoing
	}
	if present[formPlannedCost] {
		u.PlannedCost = amountInput(v.PlannedCost, v.Currency)
	}
	if present["activityTypeId"] {
		u.ActivityTypeID = v.ActivityTypeID
	}
	if present["approvalWorkflowId"] {
		u.ApprovalWorkflowID = textOrNil(v.WorkflowID)
	}
}

func setLineItem(u *lineitem.Update, v workbook.Values, present map[string]bool) bool {
	for field := range present {
		switch field {
		case formName:
			u.Name = v.Name
		case formDescription:
			u.Description = v.Description
		case formOwner:
			u.OwnerID = textOrNil(v.OwnerID)
		case formCurrency:
			u.Currency = v.Currency
		case formPlannedCost:
			u.PlannedCost = amountText(v.PlannedCost)
		}
	}
	setAttributes(u.Attributes, v, present)
	return true
}

func setAttributes(values settings.AttributeValues, v workbook.Values, present map[string]bool) {
	for field := range present {
		id, isAttribute := strings.CutPrefix(field, workbookAttrPrefix)
		if !isAttribute {
			continue
		}
		if raw := v.Attributes[id]; raw != nil {
			values[id] = raw
		} else {
			delete(values, id)
		}
	}
}
