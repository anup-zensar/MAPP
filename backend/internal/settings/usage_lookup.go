package settings

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"mapp/internal/history"
)

// part returns one usage part: text, followed by " (<n> in the recycle bin)" when some of the records
// counted are in the bin.
func part(kind string, t tally, text string) UsagePart {
	if t.inBin > 0 {
		text += fmt.Sprintf(" (%d in the recycle bin)", t.inBin)
	}
	return UsagePart{Kind: kind, Text: text, Count: t.count, InRecycleBin: t.inBin}
}

// newUsage returns the usage of parts: in use when there is any part, the parts joined with " · " as its
// summary, or "Not used yet". records lists plans for models and finance budgets; it is never nil.
func newUsage(parts []UsagePart, records []history.RecordRef) Usage {
	if records == nil {
		records = []history.RecordRef{}
	}
	if len(parts) == 0 {
		return Usage{InUse: false, Summary: notUsedYet, Parts: []UsagePart{}, Records: records}
	}
	texts := make([]string, 0, len(parts))
	for _, p := range parts {
		texts = append(texts, p.Text)
	}
	return Usage{InUse: true, Summary: strings.Join(texts, " · "), Parts: parts, Records: records}
}

// usedBy returns the part "Used by <n> <records>" of tallies[key], or false when nothing is counted.
func usedBy(tallies map[string]*tally, key, kind string, nouns [2]string) (UsagePart, bool) {
	t := tallies[key]
	if t == nil {
		return UsagePart{}, false
	}
	return part(kind, *t, "Used by "+plural(t.count, nouns[0], nouns[1])), true
}

// model returns the usage of the model id: the plans using it, and the live ones among them as records.
func (u *usageIndex) model(id string) Usage {
	var parts []UsagePart
	if p, used := usedBy(u.plansByModel, id, string(LevelPlan), [2]string{"plan", "plans"}); used {
		parts = append(parts, p)
	}
	return newUsage(parts, u.livePlans(func(ref Reference) bool { return ref.ModelID == id }))
}

// activityType returns the usage of the activity type id: the activities of that type and the models that
// allow it.
func (u *usageIndex) activityType(id string) Usage {
	var parts []UsagePart
	if p, used := usedBy(u.activitiesByType, id, string(LevelActivity), [2]string{"activity", "activities"}); used {
		parts = append(parts, p)
	}
	if t := u.modelsByType[id]; t != nil {
		parts = append(parts, part(usedByModel, *t, "Allowed in "+plural(t.count, "model", "models")))
	}
	return newUsage(parts, nil)
}

// attribute returns the usage of the Shell attribute a: the records of its level with a value, the
// attributes depending on it and, for the Business unit attribute, the attributes with a condition.
func (u *usageIndex) attribute(a Attribute) Usage {
	var parts []UsagePart
	nouns := levelNouns[a.Level]
	if p, used := usedBy(u.valuesByAttribute, a.ID, string(a.Level), nouns); used {
		parts = append(parts, p)
	}
	if dependents := u.dependents[a.ID]; len(dependents) > 0 {
		var t tally
		for _, d := range dependents {
			t.add(d.inBin)
		}
		text := fmt.Sprintf("%d attributes depend on it", t.count)
		if t.count == 1 {
			text = dependents[0].record.Label + " depends on it"
		}
		parts = append(parts, part(usedByDependent, t, text))
	}
	if a.Role == RoleBusinessUnit && u.conditions.count > 0 {
		parts = append(parts, part(usedByCondition, u.conditions, "Condition of "+plural(u.conditions.count, "attribute", "attributes")))
	}
	return newUsage(parts, nil)
}

// option returns the usage of the option o of the attribute a: the records whose value has it, the options
// of dependent attributes linked to it, and the attributes whose condition has it.
func (u *usageIndex) option(o Option, a Attribute) Usage {
	var parts []UsagePart
	nouns := levelNouns[a.Level]
	if p, used := usedBy(u.valuesByOption, o.ID, string(a.Level), nouns); used {
		parts = append(parts, p)
	}
	linked := u.linkedOptions[o.ID]
	for _, attributeID := range slices.SortedFunc(maps.Keys(linked), compareIDs) {
		t := linked[attributeID]
		label := u.attributes[attributeID].Label
		parts = append(parts, part(usedByLinkedOption, *t,
			fmt.Sprintf("Linked to %s", plural(t.count, label+" option", label+" options"))))
	}
	if t := u.conditionsByOption[o.ID]; t != nil {
		parts = append(parts, part(usedByCondition, *t, "Condition of "+plural(t.count, "attribute", "attributes")))
	}
	return newUsage(parts, nil)
}

// currency returns the usage of the currency code: the line items in it, the people showing amounts in it
// and the plans planned in it (016 FR-010), with those plans as records.
func (u *usageIndex) currency(code string) Usage {
	var parts []UsagePart
	if p, used := usedBy(u.lineItemsByCurr, code, string(LevelLineItem), [2]string{"line item", "line items"}); used {
		parts = append(parts, p)
	}
	if t := u.peopleByCurr[code]; t != nil {
		parts = append(parts, part(usedByPerson, *t, "Display currency of "+plural(t.count, "person", "people")))
	}
	var plans []history.RecordRef
	if t := u.plansByCurr[code]; t != nil {
		parts = append(parts, part(string(LevelPlan), *t, "Plan currency of "+plural(t.count, "plan", "plans")))
		plans = u.livePlans(func(ref Reference) bool { return ref.Currency == code })
	}
	return newUsage(parts, plans)
}

// item returns the usage of the finance budget item id: the line items linked to it.
func (u *usageIndex) item(id string) Usage {
	var parts []UsagePart
	if t := u.lineItemsByItem[id]; t != nil {
		parts = append(parts, part(string(LevelLineItem), *t, "Linked to "+plural(t.count, "line item", "line items")))
	}
	return newUsage(parts, nil)
}

// budget returns the usage of the finance budget id: the plans using it as their default finance budget,
// and the live ones among them as records. It never blocks anything: budgets are not deleted (D-047).
func (u *usageIndex) budget(id string) Usage {
	var parts []UsagePart
	if t := u.plansByBudget[id]; t != nil {
		parts = append(parts, part(string(LevelPlan), *t, "Default finance budget of "+plural(t.count, "plan", "plans")))
	}
	return newUsage(parts, u.livePlans(func(ref Reference) bool { return ref.FinanceBudgetID == id }))
}

// livePlans lists the plans outside the recycle bin that match, sorted by name, at most 50.
func (u *usageIndex) livePlans(match func(Reference) bool) []history.RecordRef {
	plans := []history.RecordRef{}
	for _, ref := range u.refs {
		if ref.Level == LevelPlan && !ref.InBin && match(ref) {
			plans = append(plans, ref.Record)
		}
	}
	slices.SortStableFunc(plans, func(a, b history.RecordRef) int { return compareNames(a.Name, b.Name) })
	return plans[:min(len(plans), maxUsageRecords)]
}
