package app

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/bulk"
	"mapp/internal/campaign"
	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/settings"
)

func (p bulkPlanning) SetOwner(ctx context.Context, a actor.Actor, r bulk.Record, ownerID string) (bool, error) {
	switch r.Level {
	case bulk.LevelCampaign:
		return p.editCampaign(ctx, a, r.ID, func(u *campaign.CampaignUpdate) bool { return setText(&u.OwnerID, ownerID) })
	case bulk.LevelInitiative:
		return p.editInitiative(ctx, a, r.ID, func(u *campaign.InitiativeUpdate) bool { return setText(&u.OwnerID, ownerID) })
	case bulk.LevelActivity:
	}
	return p.editActivity(ctx, a, r.ID, func(u *activity.Update) bool { return setText(&u.OwnerID, ownerID) })
}

func (p bulkPlanning) SetField(ctx context.Context, a actor.Actor, r bulk.Record, c bulk.SetField) (bool, error) {
	switch r.Level {
	case bulk.LevelCampaign:
		return p.editCampaign(ctx, a, r.ID, func(u *campaign.CampaignUpdate) bool {
			return setContainerField(&u.StartDate, &u.EndDate, u.Attributes, c)
		})
	case bulk.LevelInitiative:
		return p.editInitiative(ctx, a, r.ID, func(u *campaign.InitiativeUpdate) bool {
			return setContainerField(&u.StartDate, &u.EndDate, u.Attributes, c)
		})
	case bulk.LevelActivity:
	}
	return p.editActivity(ctx, a, r.ID, func(u *activity.Update) bool {
		u.RestartApproval = c.RestartApproval
		return setActivityField(u, c)
	})
}

func setContainerField(start, end **date.Date, attributes settings.AttributeValues, c bulk.SetField) bool {
	var day *date.Date
	if c.Value != nil {
		day = c.Value.Date
	}
	switch c.Field.Key {
	case keyStartDate:
		return setDate(start, day)
	case keyEndDate:
		return setDate(end, day)
	}
	return setAttribute(attributes, c)
}

func setActivityField(u *activity.Update, c bulk.SetField) bool {
	var day *string
	if c.Value != nil && c.Value.Date != nil {
		text := c.Value.Date.String()
		day = &text
	}
	switch c.Field.Key {
	case keyStartDate:
		return setOptional(&u.StartDate, day)
	case keyEndDate:
		return setOptional(&u.EndDate, day)
	case keyActivityType:
		changed := u.ActivityTypeID != c.Value.OptionIDs[0]
		u.ActivityTypeID = c.Value.OptionIDs[0]
		return changed
	case keyApprovalWorkflow:
		var id *string
		if c.Value != nil {
			id = &c.Value.OptionIDs[0]
		}
		return setOptional(&u.ApprovalWorkflowID, id)
	}
	return setAttribute(u.Attributes, c)
}

func setText(field **string, value string) bool {
	return setOptional(field, &value)
}

func setOptional(field **string, value *string) bool {
	if (*field == nil && value == nil) || (*field != nil && value != nil && **field == *value) {
		return false
	}
	*field = value
	return true
}

func setDate(field **date.Date, value *date.Date) bool {
	if (*field == nil && value == nil) || (*field != nil && value != nil && (*field).Compare(*value) == 0) {
		return false
	}
	*field = value
	return true
}

// setAttribute sets the Shell attribute of c in values, which the edit's form owns; an empty value clears it.
func setAttribute(values settings.AttributeValues, c bulk.SetField) bool {
	id := strings.TrimPrefix(c.Field.Key, attrKeyPrefix)
	raw := attributeRaw(c.Field.Kind, c.Value)
	if sameRaw(values[id], raw) {
		return false
	}
	if raw == nil {
		delete(values, id)
	} else {
		values[id] = raw
	}
	return true
}

func attributeRaw(kind string, v *bulk.FieldValue) json.RawMessage {
	var value any
	switch {
	case v == nil:
		return nil
	case kind == bulkKindYesNo && v.YesNo != nil:
		value = *v.YesNo
	case kind == bulkKindSingle:
		value = v.OptionIDs[0]
	default:
		value = v.OptionIDs
	}
	raw, _ := json.Marshal(value) // a bool, a string or a list of strings always encodes
	return raw
}

// sameRaw compares two stored values, every empty form (missing, null, "", []) being the same.
func sameRaw(a, b json.RawMessage) bool {
	if emptyValue(a) || emptyValue(b) {
		return emptyValue(a) && emptyValue(b)
	}
	var x, y bytes.Buffer
	return json.Compact(&x, a) == nil && json.Compact(&y, b) == nil && x.String() == y.String()
}

func (p bulkPlanning) Submit(ctx context.Context, a actor.Actor, r bulk.Record, comment string) error {
	if err := p.permit(ctx, r.ID, authz.ActivitiesSubmit); err != nil {
		return err
	}
	stored, found, err := p.activitiesRepo.Get(ctx, r.ID)
	if err != nil || !found {
		return readProblem(err, found)
	}
	_, err = p.approvals.Submit(ctx, a, r.ID, approval.SubmitInput{Version: &stored.Version, Comment: comment})
	return err
}

func (p bulkPlanning) Delete(ctx context.Context, a actor.Actor, r bulk.Record) (history.BinEntry, error) {
	permission := authz.CampaignsDelete
	if r.Level == bulk.LevelActivity {
		permission = authz.ActivitiesDelete
	}
	if err := p.permit(ctx, r.ID, permission); err != nil {
		return history.BinEntry{}, err
	}
	switch r.Level {
	case bulk.LevelCampaign:
		return p.campaigns.DeleteCampaign(ctx, a, r.ID)
	case bulk.LevelInitiative:
		return p.campaigns.DeleteInitiative(ctx, a, r.ID)
	case bulk.LevelActivity:
	}
	return p.activities.Delete(ctx, a, r.ID)
}
