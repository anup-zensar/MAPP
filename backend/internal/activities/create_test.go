package activities_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"northstar/internal/activities"
	"northstar/internal/history"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// User story 1 — create an activity.

// The new-activity form offers only the active activity types the plan's model allows, each with its
// default planned cost, sorted by name (FR-007).
func TestNewForm_OnlyActiveAllowedTypes_FR007(t *testing.T) {
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
	f := newFixture(t)
	form, err := f.service.NewForm(t.Context(), priya, digitalAndSocial)
	if err != nil {
		t.Fatalf("NewForm: %v", err)
	}
	d := form.Defaults
	if d == nil || d.Owner == nil || d.Owner.Name != "Priya Raman" || d.StartDate.String() != "2026-01-01" ||
		d.EndDate.String() != "2026-06-30" || d.ApprovalWorkflowID != nil {
		t.Fatalf("defaults = %+v, want Priya Raman, 1 Jan – 30 Jun 2026 and no workflow (three are active)", d)
	}
	if form.Context.Initiative.Owner == nil || form.Context.LatestEndDate.String() != "2036-12-31" ||
		form.Context.Model.Name != "Lubricants 1.0" || len(form.Context.BusinessUnits) != 1 ||
		form.Context.BusinessUnits[0].Label != "Lubricants" || form.Context.Campaign.Name != "Helix launch" {
		t.Errorf("context = %+v", form.Context)
	}

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

// A model without activity types gives an empty type list: the form explains it (edge case).
func TestNewForm_ModelWithoutTypes_EdgeCase(t *testing.T) {
	f := newFixture(t)
	repo := settings.NewJSONRepository()
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

// Only the name and the type are needed to save; the needed-to-submit fields may stay empty (FR-003). An
// activity is created as Created with the next readable ID, and one "created" entry names the actor and the
// initiative (FR-004).
func TestCreate_OnlyNameAndTypeRequired_FR003_FR004(t *testing.T) {
	f := newFixture(t)

	view := f.create(t, digitalAndSocial, activities.Input{Name: " Launch video ", ActivityTypeID: eventType})

	if view.ID != "A-1013" || view.Name != "Launch video" || view.Status != activities.StatusCreated ||
		view.DisplayStatus != activities.DisplayCreated || view.Owner != nil || view.StartDate != nil ||
		view.PlannedCost != nil || view.ApprovalWorkflow != nil || view.Version != 1 || view.CreatedBy != priya {
		t.Errorf("view = %+v", view)
	}
	if view.Initiative.Name != "Digital & social" || view.Campaign.Name != "Helix launch" || view.Plan.ID != "P-1001" {
		t.Errorf("path = %+v › %+v › %+v", view.Plan, view.Campaign, view.Initiative)
	}
	if view.Completeness.MissingFields == 0 || view.Readiness.Ready {
		t.Errorf("a new activity without owner, dates or cost must list what it still needs: %+v", view.Readiness)
	}
	entries := f.entries(t)
	want := []history.RecordRef{{Type: "initiative", ID: digitalAndSocial, Name: "Digital & social"}}
	if len(entries) != 1 || entries[0].Action != history.ActionCreated || entries[0].Actor != priya ||
		entries[0].RecordID != "A-1013" || entries[0].RecordName != "Launch video" || !reflect.DeepEqual(entries[0].Path, want) {
		t.Errorf("entries = %+v, want one created entry by Priya under the initiative", entries)
	}

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activities.Input{Name: "  "})
	assertError(t, err, problems("name", "Enter a name.", "activityTypeId", "Choose an activity type."))
}

// Names are unique in the initiative, ignoring letter case and surrounding spaces; another initiative may
// use the same name; the external name is optional (FR-006).
func TestCreate_NameUniqueInInitiative_FR006(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activities.Input{Name: " launch FILM ", ActivityTypeID: eventType})

	assertError(t, err, problems("name", "'Digital & social' already has an activity called 'Launch film'."))
	other := f.create(t, printAndOutdoor, activities.Input{Name: "Launch film", ActivityTypeID: eventType})
	if other.ExternalName != "" {
		t.Errorf("external name = %q, want empty: the activity name is used", other.ExternalName)
	}
	_, err = f.service.Create(t.Context(), priya, digitalAndSocial,
		activities.Input{Name: strings.Repeat("é", 121), ExternalName: strings.Repeat("a", 121), ActivityTypeID: eventType})
	assertError(t, err, problems("name", "Use 120 characters or fewer.", "externalName", "Use 120 characters or fewer."))
}

// The type must exist, be active and be allowed by the plan's model (FR-007, R7).
func TestCreate_TypeActiveAndAllowed_FR007(t *testing.T) {
	f := newFixture(t)
	f.deactivateType(t, "AT-1005")
	refused := problems("activityTypeId", "Choose an activity type that the model 'Lubricants 1.0' allows.")

	for _, typeID := range []string{mobilityType, "AT-1005", "AT-9999"} {
		_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activities.Input{Name: "Launch video", ActivityTypeID: typeID})
		assertError(t, err, refused)
	}
	if view := f.create(t, fleetCards, activities.Input{Name: "Fleet promo", ActivityTypeID: mobilityType}); view.ActivityType.Name != "MOBILITY – Long-term contract" {
		t.Errorf("Mobility 1.0 allows the Mobility type: %+v", view.ActivityType)
	}
}

// Every problem is reported at once: a taken name, an end before the start and dates outside the
// initiative's (FR-006, FR-009); nothing is saved.
func TestCreate_AllProblemsAtOnce_FR009(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activities.Input{
		Name: "Launch film", ActivityTypeID: eventType, StartDate: text("2026-07-01"), EndDate: text("2026-06-01"),
		PlannedCost: usdInput("-1.00"),
	})

	outside := "Dates must be inside the initiative's dates (01 Jan 2026 – 30 Jun 2026)."
	assertError(t, err, problems(
		"name", "'Digital & social' already has an activity called 'Launch film'.",
		"endDate", "End date must be on or after the start date (01 Jul 2026).",
		"startDate", outside,
		"plannedCost", "Planned cost can't be negative."))
	if _, found := f.stored(t, "A-1013"); found || len(f.entries(t)) != 0 {
		t.Error("a refused create saved something")
	}
	_, err = f.service.Create(t.Context(), priya, fleetCards, activities.Input{
		Name: "Long promo", ActivityTypeID: eventType, StartDate: text("2026-13-01"), EndDate: text("2037-01-01"),
	})
	assertError(t, err, problems("startDate", "Enter a valid date.",
		"endDate", "The end date can't be after 31 Dec 2036 — 10 years after the plan's year."))
}

// Ongoing removes the end date and checks only the start against the initiative's dates (FR-010).
func TestCreate_OngoingClearsEndDate_FR010(t *testing.T) {
	f := newFixture(t)

	view := f.create(t, digitalAndSocial, activities.Input{
		Name: "Always on", ActivityTypeID: eventType, StartDate: text("2026-02-01"), EndDate: text("2027-12-31"), Ongoing: true,
	})

	if !view.Ongoing || view.EndDate != nil || view.StartDate.String() != "2026-02-01" {
		t.Errorf("ongoing activity = start %v, end %v, ongoing %v; want the end date removed", view.StartDate, view.EndDate, view.Ongoing)
	}
}

// The planned cost is in the plan currency (016 FR-011), 0.00 or more with at most two decimals; 0.00 is a value
// (FR-011, D-190).
func TestCreate_PlannedCostRules_FR011(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name, amount, currency, message string
	}{
		{"negative", "-0.01", "USD", "Planned cost can't be negative."},
		{"three decimals", "12.345", "USD", "Enter an amount, such as 12,500.00."},
		{"not a number", "12,500", "USD", "Enter an amount, such as 12,500.00."},
		{"another currency", "100.00", "EUR", "Enter the amount in USD, the plan's currency."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activities.Input{
				Name: "Costly", ActivityTypeID: eventType, PlannedCost: &money.Input{Amount: tt.amount, Currency: tt.currency},
			})
			assertError(t, err, problems("plannedCost", tt.message))
		})
	}
	view := f.create(t, digitalAndSocial, activities.Input{Name: "Free", ActivityTypeID: eventType, PlannedCost: usdInput("0")})
	if view.PlannedCost == nil || view.PlannedCost.Format() != "0.00 USD" {
		t.Errorf("planned cost = %v, want 0.00 USD kept", view.PlannedCost)
	}
	for _, p := range view.Readiness.Activity {
		if p.Field == "plannedCost" {
			t.Errorf("0.00 is a planned cost, but the check lists %q", p.Message)
		}
	}
}

// The owner must be an active person and the approval workflow an active one when chosen (FR-008, FR-012).
func TestCreate_OwnerAndWorkflowMustBeActive_FR008_FR012(t *testing.T) {
	f := newFixture(t)
	f.deactivatePerson(t, "U-1006")
	f.deactivateWorkflow(t, chinaFlow)

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activities.Input{
		Name: "Launch video", ActivityTypeID: eventType, OwnerID: text("U-1006"), ApprovalWorkflowID: text(chinaFlow),
	})

	assertError(t, err, problems("ownerId", "Choose an active person.", "approvalWorkflowId", "Choose an active approval workflow."))
	view := f.create(t, digitalAndSocial, activities.Input{
		Name: "Launch video", ActivityTypeID: eventType, OwnerID: text("U-1004"), ApprovalWorkflowID: text(lubricantsFlow),
	})
	if view.Owner.Name != "Mei Lin" || view.ApprovalWorkflow.Name != "Lubricants EU approval" {
		t.Errorf("owner %+v, workflow %+v", view.Owner, view.ApprovalWorkflow)
	}
}

// Shell attribute values are checked on save with Settings' rules: unknown attributes and options, an
// option of another attribute, a wrong share; a split may be saved with any total, but the check lists it
// (FR-016, FR-017).
func TestCreate_AttributeValuesChecked_FR016_FR017(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activities.Input{
		Name: "Launch video", ActivityTypeID: eventType, Attributes: settings.AttributeValues{
			"SA-1012":    json.RawMessage(`["SO-1027","SO-1027"]`),
			countrySplit: split("SO-1034", "12.345"),
			companyCode:  json.RawMessage(`"SO-1030"`),
			"SA-9999":    json.RawMessage(`"x"`),
			"SA-1003":    json.RawMessage(`["SO-1003"]`),
		},
	})

	assertError(t, err, problems(
		"attributes.SA-1003", "Cluster belongs to the Campaign form.",
		"attributes.SA-9999", "This Shell attribute no longer exists — reload the page.",
		"attributes.SA-1012", "Each option can appear only once.",
		"attributes.SA-1013", "Enter a share above 0 % with at most two decimals.",
		"attributes.SA-1014", "Choose one of the options of Company code."))

	view := f.create(t, digitalAndSocial, activities.Input{
		Name: "Launch video", ActivityTypeID: eventType, Attributes: settings.AttributeValues{
			countrySplit: split("SO-1034", "50", "SO-1032", "33.33", "SO-1033", "33.34"),
		},
	})
	if got := string(view.Attributes[countrySplit]); got != `[{"optionId":"SO-1034","share":"50.00"},{"optionId":"SO-1032","share":"33.33"},{"optionId":"SO-1033","share":"33.34"}]` {
		t.Errorf("stored split = %s", got)
	}
	if !slices.ContainsFunc(view.Readiness.Activity, func(p activities.SubmitProblem) bool {
		return p.Code == activities.ProblemSplitIncomplete &&
			p.Message == "Country / OU shares add up to 116.67 % — they must add up to 100.00 %"
	}) {
		t.Errorf("readiness = %+v, want the 116.67 %% split listed", view.Readiness.Activity)
	}
}

// INQ and typed MQL are whole numbers of 0 or more, typed revenue is USD and not negative; MQL and revenue
// are stored only when the person sets them (FR-018).
func TestCreate_GoalsMustBeWholeAndNotNegative_FR018(t *testing.T) {
	f := newFixture(t)
	minus := int64(-5)

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activities.Input{
		Name: "Launch video", ActivityTypeID: eventType,
		Goals: activities.GoalsInput{INQ: &minus, SetLaterStagesMyself: true, MQL: &minus, Revenue: usdInput("-1.00")},
	})

	assertError(t, err, problems("goals.inq", "Enter a whole number of 0 or more.",
		"goals.mql", "Enter a whole number of 0 or more.", "goals.revenue", "Revenue can't be negative."))
	inq, mql := int64(250_000), int64(7)
	view := f.create(t, digitalAndSocial, activities.Input{
		Name: "Launch video", ActivityTypeID: eventType, Goals: activities.GoalsInput{INQ: &inq, MQL: &mql},
	})
	stored, _ := f.stored(t, view.ID)
	if stored.MQL != nil || *view.Goals.MQL != 5_000 || view.Goals.Revenue.Format() != "2,500,000.00 USD" || !view.Goals.Calculated {
		t.Errorf("goals = %+v (stored MQL %v), want calculated 5,000 and 2,500,000.00 USD, nothing typed stored", view.Goals, stored.MQL)
	}
}

// An initiative that is gone answers not found.
func TestCreate_InitiativeGone_404(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Create(t.Context(), priya, "I-9999", activities.Input{Name: "Launch video", ActivityTypeID: eventType})

	assertError(t, err, errs.NotFound("This initiative no longer exists — someone may have deleted it."))
}

// The planned costs that roll up to initiatives, campaigns and plans follow every change: they are read
// from the activities each time (FR-005, D-105).
func TestSummaries_FollowEveryChange_FR005(t *testing.T) {
	f := newFixture(t)
	total := func() string {
		t.Helper()
		summaries, err := f.service.Summaries(t.Context(), []string{digitalAndSocial})
		if err != nil {
			t.Fatalf("Summaries: %v", err)
		}
		var sum money.Amount
		for _, s := range summaries {
			if s.PlannedCost != nil {
				sum = sum.Add(s.PlannedCost.Amount)
			}
		}
		return fmt.Sprintf("%d activities, %s", len(summaries), sum)
	}
	if got := total(); got != "2 activities, 200000.00" {
		t.Fatalf("before = %s, want Launch film 120,000.00 + Social always-on 80,000.00", got)
	}

	created := f.create(t, digitalAndSocial, activities.Input{Name: "Launch video", ActivityTypeID: eventType, PlannedCost: usdInput("25000")})
	if got := total(); got != "3 activities, 225000.00" {
		t.Errorf("after create = %s", got)
	}
	f.mustEdit(t, created.ID, func(u *activities.Update) { u.PlannedCost = usdInput("30000.00") })
	if got := total(); got != "3 activities, 230000.00" {
		t.Errorf("after edit = %s", got)
	}
	entry, err := f.service.Delete(t.Context(), priya, created.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := total(); got != "2 activities, 200000.00" {
		t.Errorf("after delete = %s", got)
	}
	if _, err := f.history.Undo(t.Context(), priya, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if got := total(); got != "3 activities, 230000.00" {
		t.Errorf("after Undo = %s", got)
	}
}
