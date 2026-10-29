package workbook

import (
	"mapp/internal/platform/money"
	"mapp/internal/platform/sheet"
)

const allowedValuesSheet = "Allowed values"

// buildWorkbook lays out the plan workbook (FR-025 – FR-027): the level sheets, with the plan's records when
// withRecords, then the allowed values and the instructions.
func buildWorkbook(plan Plan, withRecords bool) sheet.Workbook {
	byID := map[string]Stored{}
	for _, r := range plan.Records {
		byID[r.ID] = r
	}
	var wb sheet.Workbook
	for _, level := range levels {
		columns := columnsOf(level, plan.Choices)
		headings := make([]sheet.Cell, len(columns))
		markings := make([]sheet.Cell, len(columns))
		for i, c := range columns {
			headings[i], markings[i] = sheet.Bold(c.heading), sheet.Text(c.marking)
		}
		rows := [][]sheet.Cell{headings, markings}
		for _, r := range plan.Records {
			if withRecords && r.Level == level {
				rows = append(rows, recordRow(plan, columns, r, byID))
			}
		}
		wb.Sheets = append(wb.Sheets, sheet.Sheet{Name: sheetName(level), Rows: rows})
	}
	wb.Sheets = append(wb.Sheets, allowedValues(plan), instructions(plan))
	return wb
}

func recordRow(plan Plan, columns []column, r Stored, byID map[string]Stored) []sheet.Cell {
	ancestors := map[string]string{}
	for parent, ok := byID[r.ParentID]; ok; parent, ok = byID[parent.ParentID] {
		ancestors[string(parent.Level)] = parent.Values.Name
	}
	cells := make([]sheet.Cell, len(columns))
	for i, c := range columns {
		switch {
		case c.field == fieldID:
			cells[i] = sheet.Text(r.ID)
		case isStructural(c.field):
			cells[i] = sheet.Text(ancestors[c.field])
		case c.field == fieldStart && r.Values.StartDate != nil:
			cells[i] = sheet.Date(*r.Values.StartDate)
		case c.field == fieldEnd && r.Values.EndDate != nil:
			cells[i] = sheet.Date(*r.Values.EndDate)
		case moneyIn(c, r.Values) != nil:
			cells[i] = sheet.Money(*moneyIn(c, r.Values)) // a number people can add up (038 FR-059)
		default:
			cells[i] = sheet.Text(format(c, r.Values, plan.Choices))
		}
	}
	return cells
}

// moneyIn is the budget or planned cost a column shows, or nil.
func moneyIn(c column, v Values) *money.Amount {
	if amount, isAmount := amountField(c.field, &v); isAmount && c.attr == nil {
		return *amount
	}
	return nil
}

// allowedValues lists, one column each, the values a choice column takes (FR-025).
func allowedValues(plan Plan) sheet.Sheet {
	campaigns, initiatives := containerLists(plan)
	lists := [][]string{campaigns, initiatives, {"Activity types"}, {"Approval workflows"}, {"Currencies"}, {"People"}}
	ch := plan.Choices
	lists[2] = append(lists[2], activeLabels(ch.ActivityTypes)...)
	lists[3] = append(lists[3], activeLabels(ch.Workflows)...)
	lists[4] = append(lists[4], activeLabels(ch.Currencies)...)
	for _, p := range ch.People {
		if p.Active {
			lists[5] = append(lists[5], p.Email)
		}
	}
	for _, a := range ch.Attributes {
		if a.Type == typeSingle || a.Type == typeMultiple || a.Type == typeSplit {
			lists = append(lists, append([]string{a.Label + " (" + sheetName(a.Level) + ")"}, activeLabels(a.Options)...))
		}
	}
	return sheet.Sheet{Name: allowedValuesSheet, Rows: columnsToRows(lists)}
}

// containerLists are the plan's campaigns, and its initiatives as "Campaign › Initiative", each headed.
func containerLists(plan Plan) ([]string, []string) {
	campaigns, initiatives := []string{sheetCampaigns}, []string{sheetInitiative}
	names := map[string]string{}
	for _, r := range plan.Records {
		names[r.ID] = r.Values.Name
		switch r.Level {
		case LevelCampaign:
			campaigns = append(campaigns, r.Values.Name)
		case LevelInitiative:
			initiatives = append(initiatives, names[r.ParentID]+pathSeparator+r.Values.Name)
		case LevelActivity, LevelLineItem:
		}
	}
	return campaigns, initiatives
}

func activeLabels(options []Option) []string {
	labels := []string{}
	for _, o := range options {
		if o.Active {
			labels = append(labels, o.Label)
		}
	}
	return labels
}

func columnsToRows(lists [][]string) [][]sheet.Cell {
	longest := 0
	for _, l := range lists {
		longest = max(longest, len(l))
	}
	rows := make([][]sheet.Cell, longest)
	for r := range rows {
		rows[r] = make([]sheet.Cell, len(lists))
		for c, l := range lists {
			switch {
			case r >= len(l):
			case r == 0:
				rows[r][c] = sheet.Bold(l[r])
			default:
				rows[r][c] = sheet.Text(l[r])
			}
		}
	}
	return rows
}

func instructions(plan Plan) sheet.Sheet {
	lines := []string{
		"How to fill in the plan workbook of '" + plan.Name + "'",
		"Row 1 holds the column headings and row 2 says whether a column is Required, Needed to publish, Needed to submit or Optional; write from row 3.",
		"A row without an ID creates a record; a row with the ID of a record of this plan updates it with the row's values.",
		"Rows you delete from the file leave their records alone: delete and move records in SP&A One.",
		"Name a campaign, initiative or activity by its name or its ID; a name may be one the file creates on an earlier sheet.",
		"Dates: 2026-03-12 or 12 Mar 2026. Amounts: 12500.00, in the plan currency (" + plan.Currency + ") except line items, which have a Currency column.",
		"Yes or No. People by e-mail. Several choices separated by ';'. Splits as 'China 60; Hong Kong 40', or without shares to split evenly.",
		"Only active values of the Allowed values sheet are accepted; a value a record already has may stay.",
		"A .csv file is read as the Activities sheet.",
	}
	rows := make([][]sheet.Cell, len(lines))
	for i, line := range lines {
		rows[i] = []sheet.Cell{sheet.Text(line)}
	}
	rows[0][0] = sheet.Bold(lines[0])
	return sheet.Sheet{Name: "Instructions", Rows: rows}
}
