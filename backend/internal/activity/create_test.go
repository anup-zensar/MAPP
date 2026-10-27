package activity_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"mapp/internal/activity"
	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// User story 1 — create an activity.

// Only the name and the type are needed to save; the needed-to-submit fields may stay empty (FR-003). An
// activity is created as Created with the next readable ID, and one "created" entry names the actor and the
// initiative (FR-004).
func TestCreate_OnlyNameAndTypeRequired_FR003_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view := f.create(t, digitalAndSocial, activity.Input{Name: " Launch video ", ActivityTypeID: eventType})

	if view.ID != "A-1013" || view.Name != "Launch video" || view.Status != activity.StatusCreated ||
		view.DisplayStatus != activity.DisplayCreated || view.Owner != nil || view.StartDate != nil ||
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

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activity.Input{Name: "  "})
	assertError(t, err, problems("name", "Enter a name.", "activityTypeId", "Choose an activity type."))
	_, err = f.service.Create(t.Context(), priya, "I-9999", activity.Input{Name: "Launch video", ActivityTypeID: eventType})
	assertError(t, err, errs.NotFound("This initiative no longer exists — someone may have deleted it."))
}

// Names are unique in the initiative, ignoring letter case and surrounding spaces; another initiative may
// use the same name; the external name is optional (FR-006).
func TestCreate_NameUniqueInInitiative_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activity.Input{Name: " launch FILM ", ActivityTypeID: eventType})

	assertError(t, err, problems("name", "'Digital & social' already has an activity called 'Launch film'."))
	other := f.create(t, printAndOutdoor, activity.Input{Name: "Launch film", ActivityTypeID: eventType})
	if other.ExternalName != "" {
		t.Errorf("external name = %q, want empty: the activity name is used", other.ExternalName)
	}
	_, err = f.service.Create(t.Context(), priya, digitalAndSocial,
		activity.Input{Name: strings.Repeat("é", 121), ExternalName: strings.Repeat("a", 121), ActivityTypeID: eventType})
	assertError(t, err, problems("name", "Use 120 characters or fewer.", "externalName", "Use 120 characters or fewer."))
}

// The type must exist, be active and be allowed by the plan's model (FR-007, R7).
func TestCreate_TypeActiveAndAllowed_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.deactivateType(t, "AT-1005")
	refused := problems("activityTypeId", "Choose an activity type that the model 'Lubricants 1.0' allows.")

	for _, typeID := range []string{mobilityType, "AT-1005", "AT-9999"} {
		_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activity.Input{Name: "Launch video", ActivityTypeID: typeID})
		assertError(t, err, refused)
	}
	if view := f.create(t, fleetCards, activity.Input{Name: "Fleet promo", ActivityTypeID: mobilityType}); view.ActivityType.Name != "MOBILITY – Long-term contract" {
		t.Errorf("Mobility 1.0 allows the Mobility type: %+v", view.ActivityType)
	}
}

// Every problem is reported at once: a taken name, an end before the start and dates outside the
// initiative's (FR-006, FR-009); nothing is saved.
func TestCreate_AllProblemsAtOnce_FR009(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activity.Input{
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
	_, err = f.service.Create(t.Context(), priya, fleetCards, activity.Input{
		Name: "Long promo", ActivityTypeID: eventType, StartDate: text("2026-13-01"), EndDate: text("2037-01-01"),
	})
	assertError(t, err, problems("startDate", "Enter a valid date.",
		"endDate", "The end date can't be after 31 Dec 2036 — 10 years after the plan's year."))
}

// Ongoing removes the end date and checks only the start against the initiative's dates (FR-010).
func TestCreate_OngoingClearsEndDate_FR010(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	view := f.create(t, digitalAndSocial, activity.Input{
		Name: "Always on", ActivityTypeID: eventType, StartDate: text("2026-02-01"), EndDate: text("2027-12-31"), Ongoing: true,
	})

	if !view.Ongoing || view.EndDate != nil || view.StartDate.String() != "2026-02-01" {
		t.Errorf("ongoing activity = start %v, end %v, ongoing %v; want the end date removed", view.StartDate, view.EndDate, view.Ongoing)
	}
}

// The planned cost is in the plan currency (016 FR-011), 0.00 or more with at most two decimals; 0.00 is a value
// (FR-011, D-190).
func TestCreate_PlannedCostRules_FR011(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tests := []struct {
		name, amount, currency, message string
	}{
		{"negative", "-0.01", "USD", "Planned cost can't be negative."},
		{"three decimals", "12.345", "USD", "Enter an amount, such as 12,500.00."},
		{"not a number", "12,500", "USD", "Enter an amount, such as 12,500.00."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activity.Input{
				Name: "Costly", ActivityTypeID: eventType, PlannedCost: &money.Input{Amount: tt.amount, Currency: tt.currency},
			})
			assertError(t, err, problems("plannedCost", tt.message))
		})
	}
	view := f.create(t, digitalAndSocial, activity.Input{Name: "Free", ActivityTypeID: eventType, PlannedCost: usdInput("0")})
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
	t.Parallel()
	f := newFixture(t)
	f.deactivatePerson(t, "U-1006")
	f.deactivateWorkflow(t, chinaFlow)

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activity.Input{
		Name: "Launch video", ActivityTypeID: eventType, OwnerID: text("U-1006"), ApprovalWorkflowID: text(chinaFlow),
	})

	assertError(t, err, problems("ownerId", "Choose an active person.", "approvalWorkflowId", "Choose an active approval workflow."))
	view := f.create(t, digitalAndSocial, activity.Input{
		Name: "Launch video", ActivityTypeID: eventType, OwnerID: text("U-1004"), ApprovalWorkflowID: text(lubricantsFlow),
	})
	if view.Owner.Name != "Mei Lin" || view.ApprovalWorkflow.Name != "Lubricants EU approval" {
		t.Errorf("owner %+v, workflow %+v", view.Owner, view.ApprovalWorkflow)
	}
}

// Shell attribute values are checked on save with Settings' rules: unknown attributes and options, an
// option of another attribute, a wrong share; a split may be saved with any total.
func TestCreate_AttributeValuesChecked_FR016_FR017(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activity.Input{
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

	view := f.create(t, digitalAndSocial, activity.Input{
		Name: "Launch video", ActivityTypeID: eventType, Attributes: settings.AttributeValues{
			countrySplit: split("SO-1034", "50", "SO-1032", "33.33", "SO-1033", "33.34"),
		},
	})
	if got := string(view.Attributes[countrySplit]); got != `[{"optionId":"SO-1034","share":"50.00"},{"optionId":"SO-1032","share":"33.33"},{"optionId":"SO-1033","share":"33.34"}]` {
		t.Errorf("stored split = %s", got)
	}
}

// INQ and typed MQL are whole numbers of 0 or more, typed revenue is USD and not negative; MQL and revenue
// are stored only when the person sets them (FR-018).
func TestCreate_GoalsMustBeWholeAndNotNegative_FR018(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	minus := int64(-5)

	_, err := f.service.Create(t.Context(), priya, digitalAndSocial, activity.Input{
		Name: "Launch video", ActivityTypeID: eventType,
		Goals: activity.GoalsInput{INQ: &minus, SetLaterStagesMyself: true, MQL: &minus, Revenue: usdInput("-1.00")},
	})

	assertError(t, err, problems("goals.inq", "Enter a whole number of 0 or more.",
		"goals.mql", "Enter a whole number of 0 or more.", "goals.revenue", "Revenue can't be negative."))
	inq, mql := int64(250_000), int64(7)
	view := f.create(t, digitalAndSocial, activity.Input{
		Name: "Launch video", ActivityTypeID: eventType, Goals: activity.GoalsInput{INQ: &inq, MQL: &mql},
	})
	stored, _ := f.stored(t, view.ID)
	if stored.MQL != nil || *view.Goals.MQL != 5_000 || view.Goals.Revenue.Format() != "2,500,000.00 USD" || !view.Goals.Calculated {
		t.Errorf("goals = %+v (stored MQL %v), want calculated 5,000 and 2,500,000.00 USD, nothing typed stored", view.Goals, stored.MQL)
	}
}

// The planned costs that roll up to initiatives, campaigns and plans follow every change: they are read
// from the activities each time (FR-005, D-105).
func TestSummaries_FollowEveryChange_FR005(t *testing.T) {
	t.Parallel()
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

	created := f.create(t, digitalAndSocial, activity.Input{Name: "Launch video", ActivityTypeID: eventType, PlannedCost: usdInput("25000")})
	if got := total(); got != "3 activities, 225000.00" {
		t.Errorf("after create = %s", got)
	}
	f.mustEdit(t, created.ID, func(u *activity.Update) { u.PlannedCost = usdInput("30000.00") })
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
