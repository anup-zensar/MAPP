package workbook

import (
	"bytes"
	"encoding/base64"
	"slices"
	"testing"

	"mapp/internal/platform/sheet"
)

// The export writes budget and planned cost as numbers, and the unchanged export checks as nothing to
// change — 038 FR-059 (L-10).
func TestExport_MoneyAsNumbers_ReimportNothingToChange_038_FR059_L10(t *testing.T) {
	t.Parallel()
	plan := testPlan()
	budget := mustAmount("25000.50")
	plan.Records[0].Values.Budget = &budget
	columns := columnsOf(LevelActivity, plan.Choices)
	byID := map[string]Stored{}
	for _, r := range plan.Records {
		byID[r.ID] = r
	}

	planned := recordRow(plan, columns, plan.Records[2], byID)[slices.IndexFunc(columns, func(c column) bool { return c.field == fieldPlanned })]
	campaignColumns := columnsOf(LevelCampaign, plan.Choices)
	budgetCell := recordRow(plan, campaignColumns, plan.Records[0], byID)[slices.IndexFunc(campaignColumns, func(c column) bool { return c.field == fieldBudget })]

	if planned != sheet.Money(mustAmount("1000.00")) || budgetCell != sheet.Money(budget) {
		t.Errorf("planned cost %+v, budget %+v; want money numbers", planned, budgetCell)
	}
	var buf bytes.Buffer
	if err := sheet.WriteXLSX(&buf, buildWorkbook(plan, true)); err != nil {
		t.Fatal(err)
	}
	f := &fakePlanning{plan: plan}
	check := applied(t, f, Upload{FileName: "export.xlsx", Content: base64.StdEncoding.EncodeToString(buf.Bytes())})
	if !check.NothingToChange || len(f.updated) != 0 {
		t.Errorf("re-imported export: nothing to change %v, updated %v, problems %v", check.NothingToChange, f.updated, problemsOf(check))
	}
}
