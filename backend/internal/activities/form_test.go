package activities_test

import (
	"context"
	"slices"
	"testing"

	"mapp/internal/activities"
	"mapp/internal/platform/errs"
	"mapp/internal/settings"
)

// The new-activity form offers only the active activity types the plan's model allows, each with its
// default planned cost, sorted by name (FR-007).
func TestNewForm_OnlyActiveAllowedTypes_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.deactivateType(t, "AT-1005") // Sponsorship

	form, err := f.service.NewForm(t.Context(), priya, digitalAndSocial)

	if err != nil {
		t.Fatalf("NewForm: %v", err)
	}
	var offered []string
	for _, option := range form.ActivityTypes {
		cost := "none"
		if option.DefaultPlannedCost != nil {
			cost = option.DefaultPlannedCost.Format()
		}
		offered = append(offered, option.Name+" · "+cost)
		if !option.Active || !option.Allowed {
			t.Errorf("type %s offered as active %v, allowed %v", option.Name, option.Active, option.Allowed)
		}
	}
	want := []string{"Agency fees – Creative · none", "Content production · none", "Digital media · 10,000.00 USD", "Event · 25,000.00 USD"}
	if !slices.Equal(offered, want) {
		t.Errorf("types = %q, want %q (no Mobility type, no inactive Sponsorship)", offered, want)
	}
}

// A new activity starts with the initiative's owner (when active) and dates, and with the approval workflow
// when exactly one is active (FR-008, FR-009, FR-012).
func TestNewForm_Defaults_FR008_FR009_FR012(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	form, err := f.service.NewForm(t.Context(), priya, digitalAndSocial)
	if err != nil {
		t.Fatalf("NewForm: %v", err)
	}
	assertInitiativeDefaults(t, form.Defaults)
	assertDigitalAndSocialContext(t, form.Context)

	f.deactivateWorkflow(t, chinaFlow)
	f.deactivateWorkflow(t, "WF-1003") // 017's Standard marketing approval (017 FR-037)
	f.deactivatePerson(t, priya.ID)
	form, err = f.service.NewForm(t.Context(), priya, digitalAndSocial)
	if err != nil {
		t.Fatalf("NewForm: %v", err)
	}
	if d := form.Defaults; d.Owner != nil || d.ApprovalWorkflowID == nil || *d.ApprovalWorkflowID != lubricantsFlow {
		t.Errorf("defaults = %+v, want no owner (inactive) and the only active workflow %s", d, lubricantsFlow)
	}
	if len(form.ApprovalWorkflows) != 1 || form.ApprovalWorkflows[0].Levels[0].Name != "Level 1" ||
		form.ApprovalWorkflows[0].Levels[0].Approvers[0].Name != "Daniel Okoro" {
		t.Errorf("workflows = %+v, want only the active one with its level and approver", form.ApprovalWorkflows)
	}
}

// assertInitiativeDefaults checks the defaults of a new activity of Digital & social while its owner and three
// workflows are active: the owner, the initiative's dates and no workflow.
func assertInitiativeDefaults(t *testing.T, d *activities.Defaults) {
	t.Helper()
	if d == nil || d.Owner == nil || d.Owner.Name != "Priya Raman" || d.StartDate.String() != "2026-01-01" ||
		d.EndDate.String() != "2026-06-30" || d.ApprovalWorkflowID != nil {
		t.Fatalf("defaults = %+v, want Priya Raman, 1 Jan – 30 Jun 2026 and no workflow (three are active)", d)
	}
}

// assertDigitalAndSocialContext checks the plan context of the form of Digital & social: its owner, the
// latest end date, the model, the business unit and the campaign.
func assertDigitalAndSocialContext(t *testing.T, c activities.FormContext) {
	t.Helper()
	if c.Initiative.Owner == nil || c.LatestEndDate.String() != "2036-12-31" ||
		c.Model.Name != "Lubricants 1.0" || len(c.BusinessUnits) != 1 ||
		c.BusinessUnits[0].Label != "Lubricants" || c.Campaign.Name != "Helix launch" {
		t.Errorf("context = %+v", c)
	}
}

// A model without activity types gives an empty type list: the form explains it (edge case).
func TestNewForm_ModelWithoutTypes_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	repo := settings.NewJSONStore()
	f.change(t, func(ctx context.Context) error {
		model, _, err := repo.GetModel(ctx, "MOD-1001")
		if err != nil {
			return err
		}
		model.ActivityTypeIDs = []string{}
		return repo.PutModel(ctx, model)
	})

	form, err := f.service.NewForm(t.Context(), priya, digitalAndSocial)

	if err != nil || form.ActivityTypes == nil || len(form.ActivityTypes) != 0 {
		t.Errorf("types = %#v, %v; want an empty list", form.ActivityTypes, err)
	}
}

// The form lists the active activity-level Shell attributes in display order, and the names already used
// in the initiative (FR-014, FR-006); a gone initiative is not found.
func TestNewForm_AttributesAndNames_FR014(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	form, err := f.service.NewForm(t.Context(), priya, digitalAndSocial)

	if err != nil {
		t.Fatalf("NewForm: %v", err)
	}
	if len(form.Attributes) != 20 || form.Attributes[0].Label != "Channel / audience" || form.Attributes[1].ID != countrySplit {
		t.Errorf("attributes: %d, first %q; want the 20 activity attributes in display order", len(form.Attributes), form.Attributes[0].Label)
	}
	for i := 1; i < len(form.Attributes); i++ {
		if form.Attributes[i].Position <= form.Attributes[i-1].Position || form.Attributes[i].Level != settings.LevelActivity {
			t.Errorf("attribute %s out of order or of another level", form.Attributes[i].Label)
		}
	}
	if !slices.Equal(form.ExistingNames, []string{"Launch film", "Social always-on"}) {
		t.Errorf("existing names = %q", form.ExistingNames)
	}
	_, err = f.service.NewForm(t.Context(), priya, "I-9999")
	assertError(t, err, errs.NotFound("This initiative no longer exists — someone may have deleted it."))
}
