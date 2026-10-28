package workbook

import (
	"strings"
)

// The fields of the columns; an attribute column's field is attrPrefix + its id.
const (
	fieldID          = "id"
	fieldCampaign    = "campaign"
	fieldInitiative  = "initiative"
	fieldActivity    = "activity"
	fieldName        = "name"
	fieldOwner       = "ownerId"
	fieldStart       = "startDate"
	fieldEnd         = "endDate"
	fieldBudget      = "budget"
	fieldDescription = "description"
	fieldExternal    = "externalName"
	fieldType        = "activityTypeId"
	fieldOngoing     = "ongoing"
	fieldPlanned     = "plannedCost"
	fieldWorkflow    = "approvalWorkflowId"
	fieldCurrency    = "currency"
	attrPrefix       = "attributes."
)

const (
	headOwner       = "Owner"
	headStart       = "Start date"
	headEnd         = "End date"
	headPlanned     = "Planned cost"
	headDescription = "Description"
	sheetCampaigns  = "Campaigns"
	sheetInitiative = "Initiatives"
)

const (
	markRequired = "Required"
	markPublish  = "Needed to publish"
	markSubmit   = "Needed to submit"
	markOptional = "Optional"
	markNewEmpty = "Empty for a new record"
)

// levels are the level sheets in the order an import reads them: parents before children.
var levels = []Level{LevelCampaign, LevelInitiative, LevelActivity, LevelLineItem} //nolint:gochecknoglobals // the fixed order of the sheets

type column struct {
	heading string
	field   string
	marking string
	attr    *Attribute
}

func (c column) required() bool { return c.marking == markRequired }

func sheetName(l Level) string {
	switch l {
	case LevelCampaign:
		return sheetCampaigns
	case LevelInitiative:
		return sheetInitiative
	case LevelActivity:
		return "Activities"
	case LevelLineItem:
	}
	return "Line items"
}

func withArticle(l Level) string {
	if l == LevelActivity || l == LevelInitiative {
		return "an " + levelWord(l)
	}
	return "a " + levelWord(l)
}

func levelWord(l Level) string {
	if l == LevelLineItem {
		return "line item"
	}
	return string(l)
}

func baseColumns(l Level) []column {
	id := column{heading: "ID", field: fieldID, marking: markNewEmpty}
	parents := []column{
		{heading: "Campaign", field: fieldCampaign, marking: markRequired},
		{heading: "Initiative", field: fieldInitiative, marking: markRequired},
		{heading: "Activity", field: fieldActivity, marking: markRequired},
	}
	switch l {
	case LevelCampaign:
		return append([]column{id, {heading: "Campaign name", field: fieldName, marking: markRequired}}, containerColumns(markPublish)...)
	case LevelInitiative:
		return append([]column{id, parents[0], {heading: "Initiative name", field: fieldName, marking: markRequired}},
			containerColumns(markPublish)...)
	case LevelActivity:
		return []column{id, parents[0], parents[1], {heading: "Activity name", field: fieldName, marking: markRequired},
			{heading: "External name", field: fieldExternal, marking: markOptional},
			{heading: "Activity type", field: fieldType, marking: markRequired},
			{heading: headOwner, field: fieldOwner, marking: markSubmit}, {heading: headStart, field: fieldStart, marking: markSubmit},
			{heading: headEnd, field: fieldEnd, marking: markSubmit}, {heading: "Ongoing", field: fieldOngoing, marking: markOptional},
			{heading: headPlanned, field: fieldPlanned, marking: markSubmit},
			{heading: "Approval workflow", field: fieldWorkflow, marking: markSubmit},
			{heading: headDescription, field: fieldDescription, marking: markOptional}}
	case LevelLineItem:
	}
	return []column{id, parents[0], parents[1], parents[2], {heading: "Line item name", field: fieldName, marking: markRequired},
		{heading: headOwner, field: fieldOwner, marking: markSubmit}, {heading: "Currency", field: fieldCurrency, marking: markRequired},
		{heading: headPlanned, field: fieldPlanned, marking: markRequired},
		{heading: headDescription, field: fieldDescription, marking: markOptional}}
}

func containerColumns(needed string) []column {
	return []column{
		{heading: headOwner, field: fieldOwner, marking: needed}, {heading: headStart, field: fieldStart, marking: needed},
		{heading: headEnd, field: fieldEnd, marking: needed}, {heading: "Budget", field: fieldBudget, marking: markOptional},
		{heading: headDescription, field: fieldDescription, marking: markOptional},
	}
}

// columnsOf is the level's sheet: its base columns, then one per attribute of the level the plan uses.
func columnsOf(l Level, ch Choices) []column {
	columns := baseColumns(l)
	for i := range ch.Attributes {
		a := &ch.Attributes[i]
		if a.Level != l {
			continue
		}
		heading := a.Label
		if columnByHeading(columns, heading) >= 0 {
			heading += " (attribute)"
		}
		columns = append(columns, column{heading: heading, field: attrPrefix + a.ID, marking: attributeMarking(a.Marking), attr: a})
	}
	return columns
}

func attributeMarking(marking string) string {
	switch marking {
	case "neededToPublish":
		return markPublish
	case "neededToSubmit":
		return markSubmit
	}
	return markOptional
}

func columnByHeading(columns []column, heading string) int {
	wanted := normalHeading(heading)
	for i, c := range columns {
		if normalHeading(c.heading) == wanted {
			return i
		}
	}
	return -1
}

// normalHeading compares headings ignoring letter case, spaces and a trailing marking such as " *".
func normalHeading(h string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimRight(h, " *")), " "))
}
