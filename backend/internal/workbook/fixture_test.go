package workbook

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/money"
	"mapp/internal/platform/sheet"
)

var priya = actor.Actor{ID: "U-1", Name: "Priya Raman"}

// fakePlanning records what an import asks of the planning areas and refuses the records named in refusals.
type fakePlanning struct {
	plan     Plan
	refusals map[string]error
	created  []string
	updated  []map[string]bool
	next     int
}

func (f *fakePlanning) Plan(context.Context, actor.Actor, string) (Plan, error) { return f.plan, nil }

func (f *fakePlanning) PlanFacts(context.Context, string) (authz.PlanFacts, bool, error) {
	return authz.PlanFacts{}, true, nil
}

func (f *fakePlanning) Create(_ context.Context, _ actor.Actor, r NewRecord) (string, error) {
	if err := f.refusals[r.Values.Name]; err != nil {
		return "", err
	}
	f.next++
	prefix := map[Level]string{LevelCampaign: "C", LevelInitiative: "I", LevelActivity: "A", LevelLineItem: "L"}[r.Level]
	id := fmt.Sprintf("%s-%d", prefix, 900+f.next)
	f.created = append(f.created, id+" "+r.Values.Name+" under "+r.ParentID)
	return id, nil
}

func (f *fakePlanning) Update(_ context.Context, _ actor.Actor, c Change) error {
	if err := f.refusals[c.Values.Name]; err != nil {
		return err
	}
	f.updated = append(f.updated, c.Present)
	return nil
}

func (f *fakePlanning) Delete(context.Context, actor.Actor, Level, string) error { return nil }

func (f *fakePlanning) Exists(context.Context, Level, string) (bool, error) { return true, nil }

// testPlan has the campaign Helix launch (C-1) with the initiative Digital (I-1), its approved activity Launch film
// (A-1) and created activity Podcast (A-2).
func testPlan() Plan {
	cost := mustAmount("1000.00")
	return Plan{ID: "P-1", Name: "FY2026 Lubricants Europe", Currency: "USD", Choices: testChoices, Records: []Stored{
		{Level: LevelCampaign, ID: "C-1", ParentID: "P-1", Values: Values{Name: "Helix launch", OwnerID: "U-1"}},
		{Level: LevelInitiative, ID: "I-1", ParentID: "C-1", Values: Values{Name: "Digital"}},
		{Level: LevelActivity, ID: "A-1", ParentID: "I-1", Status: "approved",
			Values: Values{Name: "Launch film", ActivityTypeID: "AT-1", PlannedCost: &cost}},
		{Level: LevelActivity, ID: "A-2", ParentID: "I-1", Status: "created", Values: Values{Name: "Podcast", ActivityTypeID: "AT-1"}},
	}}
}

func uploadOf(tb testing.TB, sheets map[string][][]string) Upload {
	tb.Helper()
	var wb sheet.Workbook
	for _, level := range levels {
		rows, found := sheets[sheetName(level)]
		if !found {
			continue
		}
		cells := make([][]sheet.Cell, len(rows))
		for i, row := range rows {
			for _, text := range row {
				cells[i] = append(cells[i], sheet.Text(text))
			}
		}
		wb.Sheets = append(wb.Sheets, sheet.Sheet{Name: sheetName(level), Rows: cells})
	}
	var buf bytes.Buffer
	if err := sheet.WriteXLSX(&buf, wb); err != nil {
		tb.Fatal(err)
	}
	return Upload{FileName: "plan.xlsx", Content: base64.StdEncoding.EncodeToString(buf.Bytes())}
}

// applied checks an upload against the fake planning and returns the check.
func applied(t *testing.T, f *fakePlanning, up Upload) Check {
	t.Helper()
	sheets, ignored, err := readUpload(up, f.plan.Choices)
	if err != nil {
		t.Fatalf("readUpload: %v", err)
	}
	ap := newApplier(priya, f, f.plan)
	ap.result.IgnoredColumns = ignored
	if err := ap.run(t.Context(), sheets); err != nil {
		t.Fatalf("run: %v", err)
	}
	return ap.result
}

func problemsOf(c Check) []string {
	got := []string{}
	for _, p := range c.Problems {
		got = append(got, fmt.Sprintf("%s %d %s: %s", p.Sheet, p.Row, p.Column, p.Reason))
	}
	return got
}

func mustAmount(s string) money.Amount {
	a, err := money.ParseAmount(s)
	if err != nil {
		panic(err)
	}
	return a
}
