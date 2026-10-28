package approval

import (
	"slices"
	"strings"

	"mapp/internal/platform/money"
)

// Matches reports whether every part of the condition matches the facts.
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

func (c Condition) empty() bool {
	return c.PlannedCostAtLeast == nil && c.PlannedCostBelow == nil && len(c.BusinessUnitIDs) == 0 && len(c.ActivityTypeIDs) == 0
}

func (c Condition) namesActivityFacts() bool {
	return len(c.BusinessUnitIDs) > 0 || len(c.ActivityTypeIDs) > 0
}

// choiceNames are the labels conditions name, by id, and the word for the amount they compare.
type choiceNames struct {
	businessUnits map[string]Choice
	activityTypes map[string]Choice
	amount        string
}

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

func (names choiceNames) forKind(kind SubjectKind) choiceNames {
	names.amount = ""
	if kind.isPurchaseRequest() {
		names.amount = "amount"
	}
	return names
}

func (names choiceNames) amountWord() string {
	if names.amount == "" {
		return "planned cost"
	}
	return names.amount
}

func label(choices map[string]Choice, id string) string {
	if c, found := choices[id]; found {
		return c.Label
	}
	return id
}

func dollars(a money.Amount) string {
	return "$" + a.Format()
}

func conditionText(c Condition, names choiceNames) string {
	var parts []string
	if c.PlannedCostAtLeast != nil {
		parts = append(parts, names.amountWord()+" is at least "+dollars(*c.PlannedCostAtLeast))
	}
	if c.PlannedCostBelow != nil {
		parts = append(parts, names.amountWord()+" is below "+dollars(*c.PlannedCostBelow))
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

func orList(texts []string) string {
	switch len(texts) {
	case 0:
		return ""
	case 1:
		return texts[0]
	}
	return strings.Join(texts[:len(texts)-1], ", ") + " or " + texts[len(texts)-1]
}

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

type levelPlan struct {
	level      Level
	number     int
	resolution string
	ruleText   string
}

// evaluate applies each level's rules to the facts once, at submission.
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

func ruleFacts(sub subject) RuleFacts {
	f := RuleFacts{PlannedCost: sub.Amount, BusinessUnits: []FactOption{}}
	if sub.ActivityTypeID != "" {
		f.ActivityType = &FactRef{ID: sub.ActivityTypeID, Name: sub.TypeName}
	}
	for _, unit := range sub.BusinessUnits {
		f.BusinessUnits = append(f.BusinessUnits, FactOption{ID: unit.ID, Label: unit.Label})
	}
	return f
}
