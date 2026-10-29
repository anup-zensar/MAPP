package workbook_test

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/apitest"
	"mapp/internal/platform/sheet"
	"mapp/internal/workbook"
)

const (
	priyasEmail = "priya.raman@shell.com"
	sarasEmail  = "sara.novak@shell.com"
	planPath    = "/api/v1/plans/P-1001"
)

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Fields  []struct {
			Message string `json:"message"`
		} `json:"fields"`
	} `json:"error"`
}

func decoded[T any](t *testing.T, response *apitest.Response, status int) T {
	t.Helper()
	if response.Status != status {
		t.Fatalf("status %d %s, want %d", response.Status, response.Body, status)
	}
	var v T
	response.Decode(t, &v)
	return v
}

// upload is a workbook of the given sheets, each a list of rows of texts, ready to send.
func upload(t *testing.T, name string, sheets map[string][][]string) map[string]any {
	t.Helper()
	var wb sheet.Workbook
	for _, sheetName := range []string{"Campaigns", "Initiatives", "Activities", "Line items"} {
		rows, found := sheets[sheetName]
		if !found {
			continue
		}
		cells := make([][]sheet.Cell, len(rows))
		for i, row := range rows {
			for _, text := range row {
				cells[i] = append(cells[i], sheet.Text(text))
			}
		}
		wb.Sheets = append(wb.Sheets, sheet.Sheet{Name: sheetName, Rows: cells})
	}
	var buf bytes.Buffer
	if err := sheet.WriteXLSX(&buf, wb); err != nil {
		t.Fatal(err)
	}
	return map[string]any{"fileName": name, "content": base64.StdEncoding.EncodeToString(buf.Bytes())}
}

func reasons(check workbook.Check) []string {
	got := []string{}
	for _, p := range check.Problems {
		got = append(got, strings.Join([]string{p.Sheet, strconv.Itoa(p.Row), p.Column, p.Value, p.Reason}, " | "))
	}
	return got
}

func TestTemplate_SheetsColumnsMarkings_Contract_012FR025_FR026(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, planPath+"/import-template", nil)

	if response.Status != http.StatusOK || !strings.Contains(response.Header.Get("Content-Disposition"), "import template") {
		t.Fatalf("status %d, disposition %q", response.Status, response.Header.Get("Content-Disposition"))
	}
	tables, err := sheet.ReadXLSX(response.Body, 1000)
	if err != nil {
		t.Fatalf("the template does not read back: %v", err)
	}
	names := []string{}
	for _, table := range tables {
		names = append(names, table.Name)
	}
	if diff := cmp.Diff([]string{"Campaigns", "Initiatives", "Activities", "Line items", "Allowed values", "Instructions"}, names); diff != "" {
		t.Errorf("sheets (-want +got):\n%s", diff)
	}
	activities := tables[2]
	start := [][]string{activities.Rows[0].Cells[:6], activities.Rows[1].Cells[:4]}
	want := [][]string{{"ID", "Campaign", "Initiative", "Activity name", "External name", "Activity type"},
		{"Empty for a new record", "Required", "Required", "Required"}}
	if diff := cmp.Diff(want, start); diff != "" {
		t.Errorf("Activities headings and markings (-want +got):\n%s", diff)
	}
	if len(activities.Rows) != 2 || len(activities.Rows[0].Cells) <= 13 {
		t.Errorf("the template has %d rows and %d columns; want no data and the Shell attribute columns", len(activities.Rows), len(activities.Rows[0].Cells))
	}
}

func TestExportThenCheck_NothingToChange_Contract_012FR027_SC004(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)

	exported := priya.Do(t, http.MethodGet, planPath+"/workbook", nil)
	if exported.Status != http.StatusOK {
		t.Fatalf("export status %d %s", exported.Status, exported.Body)
	}
	body := map[string]any{"fileName": "lubricants.xlsx", "content": base64.StdEncoding.EncodeToString(exported.Body)}
	check := decoded[workbook.Check](t, priya.Do(t, http.MethodPost, planPath+"/imports/check", body), http.StatusOK)

	if !check.NothingToChange || len(check.Problems) != 0 {
		t.Errorf("an unchanged export: nothing to change %v, problems %v", check.NothingToChange, reasons(check))
	}
	if check.Levels[2].Unchanged != 7 || check.Rows != check.Ready {
		t.Errorf("levels %+v, rows %d, ready %d", check.Levels, check.Rows, check.Ready)
	}
}

func TestCheck_EveryProblemAtOnce_LeftOutBelow_Contract_012FR031_FR033(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)
	file := upload(t, "plan.xlsx", map[string][][]string{
		"Campaigns":   {{"ID", "Campaign name", "Start date"}, {"", "Spring push", "31 Feb 2026"}},
		"Initiatives": {{"Campaign", "Initiative name"}, {"Spring push", "Forecourt"}},
		"Activities": {{"Campaign", "Initiative", "Activity name", "Activity type", "Owner"},
			{"Helix launch", "Digital & social", "Podcast", "Teleportation", "nobody@shell.com"},
			{"Helix launch", "Digital & social", "Webinar", "Event", ""},
			{"Helix launch", "Digital & social", "Webinar", "Event", ""}},
	})

	check := decoded[workbook.Check](t, priya.Do(t, http.MethodPost, planPath+"/imports/check", file), http.StatusOK)

	want := []string{
		"Campaigns | 2 | Start date | 31 Feb 2026 | Not a real date — write it as 2026-03-12",
		"Initiatives | 2 |  | Spring push | Its campaign on Campaigns row 2 has problems",
		"Activities | 2 | Activity type | Teleportation | 'Teleportation' is not an activity type the model 'Lubricants 1.0' allows",
		"Activities | 2 | Owner | nobody@shell.com | No active person has this e-mail",
		"Activities | 3 |  | Webinar | Appears twice in the file for 'Digital & social' (rows 3 and 4)",
		"Activities | 4 |  | Webinar | Appears twice in the file for 'Digital & social' (rows 3 and 4)",
	}
	if diff := cmp.Diff(want, reasons(check)); diff != "" {
		t.Errorf("problems (-want +got):\n%s", diff)
	}
	if check.Rows != 5 || check.Ready != 0 || check.WithProblems != 5 {
		t.Errorf("rows %d, ready %d, with problems %d", check.Rows, check.Ready, check.WithProblems)
	}
}

func TestCheck_ViewerRefused_Contract_012FR024(t *testing.T) {
	t.Parallel()
	sara := apitest.New(t).SignIn(t, sarasEmail)
	file := upload(t, "plan.xlsx", map[string][][]string{"Campaigns": {{"Campaign name"}, {"Spring push"}}})

	refused := sara.Do(t, http.MethodPost, planPath+"/imports/check", file)

	if body := decoded[apiError](t, refused, http.StatusForbidden); body.Error.Code != "forbidden" {
		t.Errorf("a viewer's check = %+v", body)
	}
}

func TestCheck_FileProblems_Contract_012FR028(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)
	tests := []struct {
		name string
		body map[string]any
		want string
	}{
		{name: "not base64", body: map[string]any{"fileName": "a.xlsx", "content": "%%%"},
			want: "SP&A One can't read this file — save it as .xlsx or .csv and try again."},
		{name: "an old .xls", body: map[string]any{"fileName": "a.xls", "content": base64.StdEncoding.EncodeToString([]byte("abc"))},
			want: "Choose an .xlsx or .csv file."},
		{name: "a missing required column", body: upload(t, "a.xlsx", map[string][][]string{"Activities": {{"Activity name"}, {"Podcast"}}}),
			want: "The Activities sheet has no 'Campaign' column — download a fresh template"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := decoded[apiError](t, priya.Do(t, http.MethodPost, planPath+"/imports/check", tt.body), http.StatusBadRequest)
			if len(body.Error.Fields) == 0 || body.Error.Fields[0].Message != tt.want {
				t.Errorf("error = %+v, want %q", body.Error, tt.want)
			}
		})
	}
}
