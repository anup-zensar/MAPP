package workbook_test

import (
	"encoding/base64"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/apitest"
	"mapp/internal/platform/sheet"
	"mapp/internal/workbook"
)

func skeleton(t *testing.T) map[string]any {
	t.Helper()
	return upload(t, "skeleton.xlsx", map[string][][]string{
		"Campaigns":   {{"ID", "Campaign name", "Owner"}, {"", "Spring push", "priya.raman@shell.com"}},
		"Initiatives": {{"Campaign", "Initiative name"}, {"Spring push", "Forecourt"}},
		"Activities": {{"Campaign", "Initiative", "Activity name", "Activity type", "Planned cost"},
			{"Spring push", "Forecourt", "Pump toppers", "Event", "12,500.00"}, {"Spring push", "Forecourt", "Bad", "Teleportation", ""}},
		"Line items": {{"Campaign", "Initiative", "Activity", "Line item name", "Currency", "Planned cost"},
			{"Spring push", "Forecourt", "Pump toppers", "Printing", "EUR", "2000"}},
	})
}

func TestImport_SkeletonListedAndUndone_Contract_012FR034_FR035_FR036(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)

	detail := decoded[workbook.Detail](t, priya.Do(t, http.MethodPost, planPath+"/imports", skeleton(t)), http.StatusCreated)

	created := workbook.LevelTotals{Campaigns: 1, Initiatives: 1, Activities: 1, LineItems: 1}
	if detail.Import.Created != created || detail.Import.RowsLeftOut != 1 || len(detail.Problems) != 1 || detail.Undo == nil {
		t.Fatalf("detail = %+v, problems %+v", detail.Import, detail.Problems)
	}
	assertLoggedAndListed(t, priya, detail)

	undone := decoded[workbook.Detail](t, priya.Do(t, http.MethodPost, "/api/v1/imports/"+detail.Import.ID+"/undo", nil), http.StatusOK)

	if undone.Import.Status != "undone" || undone.Undo != nil || slices.ContainsFunc(undone.Records, func(r workbook.RecordRef) bool { return r.Exists }) {
		t.Errorf("after undo: %+v %+v", undone.Import, undone.Records)
	}
	again := priya.Do(t, http.MethodPost, "/api/v1/imports/"+detail.Import.ID+"/undo", nil)
	if body := decoded[apiError](t, again, http.StatusConflict); body.Error.Code != "already_resolved" {
		t.Errorf("a second undo = %+v", body)
	}
}

// assertLoggedAndListed checks the created activity's log entry carries the import's note and the plan lists it.
func assertLoggedAndListed(t *testing.T, priya *apitest.Client, detail workbook.Detail) {
	t.Helper()
	activityID := detail.Records[2].ID
	log := decoded[struct {
		Items []struct{ Details string } `json:"items"`
	}](t, priya.Do(t, http.MethodGet, "/api/v1/activity-log?recordId="+activityID, nil, apitest.SkipValidation()), http.StatusOK)
	if len(log.Items) == 0 || !strings.Contains(log.Items[0].Details, "Imported from 'skeleton.xlsx' ("+detail.Import.ID+")") {
		t.Errorf("log of %s = %+v", activityID, log.Items)
	}
	page := decoded[workbook.Page](t, priya.Do(t, http.MethodGet, planPath+"/imports", nil), http.StatusOK)
	if page.Total != 1 || page.Items[0].ID != detail.Import.ID || page.Items[0].Status != "imported" {
		t.Errorf("imports = %+v", page)
	}
}

func TestImport_OtherPeopleSeeNoImportOfAPlanTheyCannotSee_Contract_012FR035(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	detail := decoded[workbook.Detail](t, server.SignIn(t, priyasEmail).Do(t, http.MethodPost, planPath+"/imports", skeleton(t)), http.StatusCreated)
	mei := server.SignIn(t, "mei.lin@shell.com")

	if got := mei.Do(t, http.MethodGet, "/api/v1/imports/"+detail.Import.ID, nil); got.Status != http.StatusNotFound {
		t.Errorf("Mei reads Priya's import: %d", got.Status)
	}
}

// exportedWith exports the plan and changes the Planned cost of the activities named in costs.
func exportedWith(t *testing.T, priya *apitest.Client, costs map[string]string) map[string]any {
	t.Helper()
	tables, err := sheet.ReadXLSX(priya.Do(t, http.MethodGet, planPath+"/workbook", nil).Body, 1000)
	if err != nil {
		t.Fatal(err)
	}
	sheets := map[string][][]string{}
	for _, table := range tables {
		var rows [][]string
		for _, row := range table.Rows {
			cells := slices.Clone(row.Cells)
			if table.Name == "Activities" && costs[row.Cell(0)] != "" {
				cells[slices.Index(table.Rows[0].Cells, "Planned cost")] = costs[row.Cell(0)]
			}
			rows = append(rows, cells)
		}
		sheets[table.Name] = rows
	}
	return upload(t, "updated.xlsx", sheets)
}

func TestImport_UpdatesByIDApprovedOnScreen_Contract_012FR029_FR032(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)

	detail := decoded[workbook.Detail](t, priya.Do(t, http.MethodPost, planPath+"/imports",
		exportedWith(t, priya, map[string]string{"A-1001": "4321.00", "A-1002": "1.00"})), http.StatusCreated)

	if detail.Import.Updated.Activities != 1 || len(detail.Records) != 1 || detail.Records[0].ID != "A-1001" {
		t.Errorf("updated = %+v, records %+v", detail.Import.Updated, detail.Records)
	}
	if len(detail.Problems) != 1 || !strings.HasPrefix(detail.Problems[0].Reason, "Approved — changing its planned cost") {
		t.Errorf("problems = %+v", detail.Problems)
	}
	activity := decoded[struct {
		PlannedCost struct{ Amount string } `json:"plannedCost"`
	}](t, priya.Do(t, http.MethodGet, "/api/v1/activities/A-1001", nil, apitest.SkipValidation()), http.StatusOK)
	if activity.PlannedCost.Amount != "4321.00" {
		t.Errorf("planned cost of A-1001 = %s", activity.PlannedCost.Amount)
	}
}

func TestCheck_CSVIsTheActivitiesSheet_Contract_012FR028(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)
	csv := "\xEF\xBB\xBFCampaign;Initiative;Activity name;Activity type\nHelix launch;Digital & social;Podcast;Event\n"

	check := decoded[workbook.Check](t, priya.Do(t, http.MethodPost, planPath+"/imports/check",
		map[string]any{"fileName": "activities.csv", "content": base64.StdEncoding.EncodeToString([]byte(csv))}), http.StatusOK)

	if diff := cmp.Diff(workbook.LevelCounts{Level: workbook.LevelActivity, Create: 1}, check.Levels[2]); diff != "" || len(check.Problems) != 0 {
		t.Errorf("activities (-want +got):\n%s problems %v", diff, reasons(check))
	}
}
