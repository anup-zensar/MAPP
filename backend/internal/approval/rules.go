package approval

import (
	"slices"
	"strings"

	"mapp/internal/platform/money"
)

// The rule engine of feature 017 (data-model §2, §3 steps 1–3): pure functions over a workflow's levels and
// an activity's rule facts. Rules are evaluated once, at submission, and their outcome is kept in the round
// (017-P1).

// Matches reports whether the condition matches the facts (017 FR-005, D-277): every part it holds must
// match. An activity without planned cost matches neither cost part; a business unit part matches when one
// of the plan's business units is listed; a type part when the activity's type is listed.
func (c Condition) Matches(f RuleFacts) bool {
	if c.PlannedCostAtLeast != nil && (f.PlannedCost == nil || f.PlannedCost.Amount.Cmp(*c.PlannedCostAtLeast) < 0) {
		return false
	}
	if c.PlannedCostBelow != nil && (f.PlannedCost == nil || f.PlannedCost.Amount.Cmp(*c.PlannedCostBelow) >= 0) {
		return false
	}
	if len(c.BusinessUnitIDs) > 0 && !slices.ContainsFunc(f.BusinessUnits, func(o FactOption) bool {
		return slices.Contains(c.BusinessUnitIDs, o.ID)
	}) {
		return false
	}
	if len(c.ActivityTypeIDs) > 0 && (f.ActivityType == nil || !slices.Contains(c.ActivityTypeIDs, f.ActivityType.ID)) {
		return false
	}
	return true
}

// empty reports whether the condition holds no part at all.
func (c Condition) empty() bool {
	return c.PlannedCostAtLeast == nil && c.PlannedCostBelow == nil && len(c.BusinessUnitIDs) == 0 && len(c.ActivityTypeIDs) == 0
}

// choiceNames are the labels of the values conditions name, by id: business units and activity types.
type choiceNames struct {
	businessUnits map[string]Choice
	activityTypes map[string]Choice
}

// namesOf indexes the choices by id.
func namesOf(businessUnits, activityTypes []Choice) choiceNames {
	names := choiceNames{businessUnits: map[string]Choice{}, activityTypes: map[string]Choice{}}
	for _, c := range businessUnits {
		names.businessUnits[c.ID] = c
	}
	for _, c := range activityTypes {
		names.activityTypes[c.ID] = c
	}
	return names
}

// label returns the label of id among choices, or the id itself when it is unknown.
func label(choices map[string]Choice, id string) string {
	if c, found := choices[id]; found {
		return c.Label
	}
	return id
}

// dollars writes a USD amount as the rule texts do: "$10,000.00".
func dollars(a money.Amount) string {
	return "$" + a.Format()
}

// conditionText is a condition in words (data-model §2): its parts in order, joined with " and " —
// "planned cost is at least $100,000.00 and business unit is Mobility or Lubricants".
func conditionText(c Condition, names choiceNames) string {
	var parts []string
	if c.PlannedCostAtLeast != nil {
		parts = append(parts, "planned cost is at least "+dollars(*c.PlannedCostAtLeast))
	}
	if c.PlannedCostBelow != nil {
		parts = append(parts, "planned cost is below "+dollars(*c.PlannedCostBelow))
	}
	if len(c.BusinessUnitIDs) > 0 {
		labels := make([]string, 0, len(c.BusinessUnitIDs))
		for _, id := range c.BusinessUnitIDs {
			labels = append(labels, label(names.businessUnits, id))
		}
		parts = append(parts, "business unit is "+orList(labels))
	}
	if len(c.ActivityTypeIDs) > 0 {
		labels := make([]string, 0, len(c.ActivityTypeIDs))
		for _, id := range c.ActivityTypeIDs {
			labels = append(labels, label(names.activityTypes, id))
		}
		parts = append(parts, "activity type is "+orList(labels))
	}
	return strings.Join(parts, " and ")
}

// orList joins texts as "A", "A or B", "A, B or C".
func orList(texts []string) string {
	switch len(texts) {
	case 0:
		return ""
	case 1:
		return texts[0]
	}
	return strings.Join(texts[:len(texts)-1], ", ") + " or " + texts[len(texts)-1]
}

// levelSummary is one level's part of the levels text (data-model §2): "<label> · <source>[ · all must
// approve][ · automatic when <condition>][ · only when <condition>]". source is the approvers' names, "Line
// manager" or "Role: <name>".
func levelSummary(l Level, n int, source string, names choiceNames) string {
	text := levelLabel(l.Name, n) + " · " + source
	if l.Mode == ModeAll && l.ApproverType != SourceManager {
		text += " · all must approve"
	}
	if l.AutoApproveWhen != nil {
		text += " · automatic when " + conditionText(*l.AutoApproveWhen, names)
	}
	if l.AppliesWhen != nil {
		text += " · only when " + conditionText(*l.AppliesWhen, names)
	}
	return text
}

// levelPlan is what the rules decided for one level at submission (data-model §3 steps 1–3).
type levelPlan struct {
	level      Level
	number     int    // 1-based
	resolution string // ResolutionDecide, ResolutionSkipped or ResolutionAutoApproved
	ruleText   string // the reason of skipped and autoApproved; "" for decide
}

// evaluate applies the rules of each level to the facts, in order (017 FR-003, FR-004, FR-010): a level
// whose "Only needed when" does not match is not needed (skipped); otherwise one whose "Approve
// automatically when" matches is approved automatically; otherwise its approvers decide.
func evaluate(levels []Level, f RuleFacts, names choiceNames) []levelPlan {
	plans := make([]levelPlan, 0, len(levels))
	for i, l := range levels {
		plan := levelPlan{level: l, number: i + 1, resolution: ResolutionDecide}
		switch {
		case l.AppliesWhen != nil && !l.AppliesWhen.Matches(f):
			plan.resolution, plan.ruleText = ResolutionSkipped, "only when "+conditionText(*l.AppliesWhen, names)
		case l.AutoApproveWhen != nil && l.AutoApproveWhen.Matches(f):
			plan.resolution, plan.ruleText = ResolutionAutoApproved, conditionText(*l.AutoApproveWhen, names)
		}
		plans = append(plans, plan)
	}
	return plans
}

// ruleFacts are the values of facts the rules look at (data-model §3), with the activity type's name and
// the business units' labels as they are now.
func ruleFacts(facts ActivityFacts) RuleFacts {
	f := RuleFacts{PlannedCost: facts.PlannedCost, BusinessUnits: []FactOption{}}
	if facts.ActivityTypeID != "" {
		f.ActivityType = &FactRef{ID: facts.ActivityTypeID, Name: facts.TypeName}
	}
	for _, unit := range facts.BusinessUnits {
		f.BusinessUnits = append(f.BusinessUnits, FactOption{ID: unit.ID, Label: unit.Label})
	}
	return f
}
