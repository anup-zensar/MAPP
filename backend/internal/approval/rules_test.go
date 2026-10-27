package approval

import (
	"context"
	"fmt"
	"testing"

	"mapp/internal/platform/money"
)

// amount returns a pointer to the USD amount text, for conditions.
func amount(t *testing.T, text string) *money.Amount {
	t.Helper()
	a := mustAmount(t, text)
	return &a
}

// factsOf are rule facts of a planned cost (USD text, "" for none), an activity type id ("" for none) and
// business unit ids.
func factsOf(t *testing.T, cost, typeID string, units ...string) RuleFacts {
	t.Helper()
	f := RuleFacts{BusinessUnits: []FactOption{}}
	if cost != "" {
		usd := money.USD(mustAmount(t, cost))
		f.PlannedCost = &usd
	}
	if typeID != "" {
		f.ActivityType = &FactRef{ID: typeID}
	}
	for _, u := range units {
		f.BusinessUnits = append(f.BusinessUnits, FactOption{ID: u})
	}
	return f
}

// fixtureNames are the labels of the fixture's rule choices.
var fixtureNames = namesOf(
	[]Choice{{ID: lubricantsUnit, Label: "Lubricants", Active: true}, {ID: mobilityUnit, Label: "Mobility", Active: true}},
	[]Choice{{ID: eventType, Label: "Event", Active: true}, {ID: sponsorshipType, Label: "Sponsorship", Active: true}},
)

// A condition matches when every part it holds matches (017 FR-005, D-277): planned cost at least (≥) and
// below (<) in USD — an activity without planned cost matches neither —, one of the plan's business units
// listed, the activity's type listed.
func TestConditionMatches_Table_FR005(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		condition Condition
		facts     RuleFacts
		want      bool
	}{
		{"at least, equal", Condition{PlannedCostAtLeast: amount(t, "100000.00")}, factsOf(t, "100000.00", ""), true},
		{"at least, below it", Condition{PlannedCostAtLeast: amount(t, "100000.00")}, factsOf(t, "99999.99", ""), false},
		{"below, under it", Condition{PlannedCostBelow: amount(t, "10000.00")}, factsOf(t, "8000.00", ""), true},
		{"below, equal", Condition{PlannedCostBelow: amount(t, "10000.00")}, factsOf(t, "10000.00", ""), false},
		{"no planned cost, below", Condition{PlannedCostBelow: amount(t, "10000.00")}, factsOf(t, "", ""), false},
		{"no planned cost, at least", Condition{PlannedCostAtLeast: amount(t, "0.00")}, factsOf(t, "", ""), false},
		{"between", Condition{PlannedCostAtLeast: amount(t, "1000.00"), PlannedCostBelow: amount(t, "2000.00")}, factsOf(t, "1500.00", ""), true},
		{"business unit listed", Condition{BusinessUnitIDs: []string{mobilityUnit}}, factsOf(t, "", "", lubricantsUnit, mobilityUnit), true},
		{"business unit not listed", Condition{BusinessUnitIDs: []string{mobilityUnit}}, factsOf(t, "", "", lubricantsUnit), false},
		{"no business unit", Condition{BusinessUnitIDs: []string{mobilityUnit}}, factsOf(t, "", ""), false},
		{"type listed", Condition{ActivityTypeIDs: []string{eventType, sponsorshipType}}, factsOf(t, "", sponsorshipType), true},
		{"type not listed", Condition{ActivityTypeIDs: []string{eventType}}, factsOf(t, "", sponsorshipType), false},
		{"no type", Condition{ActivityTypeIDs: []string{eventType}}, factsOf(t, "", ""), false},
		{"all parts match", Condition{PlannedCostAtLeast: amount(t, "100.00"), BusinessUnitIDs: []string{lubricantsUnit},
			ActivityTypeIDs: []string{eventType}}, factsOf(t, "200.00", eventType, lubricantsUnit), true},
		{"one part fails", Condition{PlannedCostAtLeast: amount(t, "100.00"), BusinessUnitIDs: []string{mobilityUnit},
			ActivityTypeIDs: []string{eventType}}, factsOf(t, "200.00", eventType, lubricantsUnit), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.condition.Matches(tt.facts); got != tt.want {
				t.Errorf("Matches = %v, want %v", got, tt.want)
			}
		})
	}
}

// A condition in words: its parts in order joined with " and ", amounts as "$10,000.00", lists of values
// with "or" (017 data-model §2, FR-009).
func TestConditionText_FR009(t *testing.T) {
	t.Parallel()
	tests := []struct {
		condition Condition
		want      string
	}{
		{Condition{PlannedCostBelow: amount(t, "10000.00")}, "planned cost is below $10,000.00"},
		{Condition{PlannedCostAtLeast: amount(t, "100000.00")}, "planned cost is at least $100,000.00"},
		{Condition{BusinessUnitIDs: []string{mobilityUnit}}, "business unit is Mobility"},
		{Condition{BusinessUnitIDs: []string{mobilityUnit, lubricantsUnit}}, "business unit is Mobility or Lubricants"},
		{Condition{ActivityTypeIDs: []string{eventType, sponsorshipType, "AT-9999"}}, "activity type is Event, Sponsorship or AT-9999"},
		{Condition{PlannedCostAtLeast: amount(t, "1000.00"), PlannedCostBelow: amount(t, "2000.00"), BusinessUnitIDs: []string{lubricantsUnit},
			ActivityTypeIDs: []string{eventType}},
			"planned cost is at least $1,000.00 and planned cost is below $2,000.00 and business unit is Lubricants and activity type is Event"},
	}
	for _, tt := range tests {
		if got := conditionText(tt.condition, fixtureNames); got != tt.want {
			t.Errorf("conditionText = %q, want %q", got, tt.want)
		}
	}
}

// The levels text summarises each level's rule (017 FR-009, data-model §2): label, source, "all must
// approve", "automatic when …", "only when …", joined with " → ".
func TestLevelsText_FR009(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	levels := []Level{
		{Name: "Line manager", ApproverType: SourceManager, Mode: ModeAny, AutoApproveWhen: &Condition{PlannedCostBelow: amount(t, "10000.00")}},
		{ApproverType: SourcePeople, ApproverIDs: []string{alex.ID}, Mode: ModeAny, AppliesWhen: &Condition{PlannedCostAtLeast: amount(t, "100000.00")}},
		{ApproverType: SourcePeople, ApproverIDs: []string{daniel.ID, alex.ID}, Mode: ModeAll, AppliesWhen: &Condition{BusinessUnitIDs: []string{mobilityUnit}}},
		{ApproverType: SourceRole, RoleID: approverRole.ID, Mode: ModeAny},
	}
	var rc ruleContext
	f.read(t, func(ctx context.Context) (err error) {
		rc, err = f.service.ruleContext(ctx)
		return err
	})

	got := levelsText(levelViews(levels, rc))

	want := "Line manager · Line manager · automatic when planned cost is below $10,000.00 → " +
		"Level 2 · Alex Morgan · only when planned cost is at least $100,000.00 → " +
		"Level 3 · Daniel Okoro, Alex Morgan · all must approve · only when business unit is Mobility → " +
		"Level 4 · Role: Approver"
	if got != want {
		t.Errorf("levels text =\n%s\nwant\n%s", got, want)
	}
}

// The rules decide each level in order, once (017 FR-003, FR-004, FR-010, SC-001): a level whose "Only
// needed when" does not match is not needed; otherwise one whose "Approve automatically when" matches is
// approved automatically; otherwise its approvers decide. "Only needed when" is checked first when a level
// has both.
func TestEvaluate_Table_FR003_FR004_SC001(t *testing.T) {
	t.Parallel()
	below10k := &Condition{PlannedCostBelow: amount(t, "10000.00")}
	from100k := &Condition{PlannedCostAtLeast: amount(t, "100000.00")}
	mobility := &Condition{BusinessUnitIDs: []string{mobilityUnit}}
	levels := []Level{
		{ApproverType: SourceManager, Mode: ModeAny, AutoApproveWhen: below10k},
		{ApproverType: SourcePeople, Mode: ModeAny, AppliesWhen: from100k},
		{ApproverType: SourcePeople, Mode: ModeAll, AppliesWhen: mobility},
		{ApproverType: SourcePeople, Mode: ModeAny, AppliesWhen: mobility, AutoApproveWhen: below10k},
	}
	const (
		decide = ResolutionDecide
		skip   = ResolutionSkipped
		auto   = ResolutionAutoApproved
	)
	tests := []struct {
		cost, unit string
		want       [4]string
	}{
		{"8000.00", lubricantsUnit, [4]string{auto, skip, skip, skip}},
		{"30000.00", lubricantsUnit, [4]string{decide, skip, skip, skip}},
		{"120000.00", lubricantsUnit, [4]string{decide, decide, skip, skip}},
		{"50000.00", mobilityUnit, [4]string{decide, skip, decide, decide}},
		{"5000.00", mobilityUnit, [4]string{auto, skip, decide, auto}},
		{"", mobilityUnit, [4]string{decide, skip, decide, decide}},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.cost, tt.unit), func(t *testing.T) {
			t.Parallel()
			plans := evaluate(levels, factsOf(t, tt.cost, eventType, tt.unit), fixtureNames)
			var got [4]string
			for i, p := range plans {
				got[i] = p.resolution
				if p.number != i+1 {
					t.Errorf("level %d numbered %d", i+1, p.number)
				}
			}
			if got != tt.want {
				t.Errorf("outcomes = %v, want %v", got, tt.want)
			}
		})
	}
	plans := evaluate(levels, factsOf(t, "8000.00", eventType, lubricantsUnit), fixtureNames)
	if plans[0].ruleText != "planned cost is below $10,000.00" || plans[1].ruleText != "only when planned cost is at least $100,000.00" ||
		plans[2].ruleText != "only when business unit is Mobility" {
		t.Errorf("reasons = %q, %q, %q", plans[0].ruleText, plans[1].ruleText, plans[2].ruleText)
	}
}
