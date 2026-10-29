package apibench_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mapp/internal/platform/apibench"
	"mapp/internal/platform/sheet"
)

const (
	workbookCheckPath = "/api/v1/plans/P-1001/imports/check"
	// fixtureEnv names the folder the 5,000-row workbook is written to for the end-to-end tests.
	fixtureEnv = "MAPP_WORKBOOK_FIXTURE_DIR"
)

// BenchmarkWorkbookCheck checks generated workbooks of 250, 1,000 and 5,000 valid rows on the demo plan (038 SC-002).
func BenchmarkWorkbookCheck(b *testing.B) {
	server := apibench.New(b, apibench.Demo)
	planner := server.SignIn(b, priya)
	for _, rows := range []int{250, 1000, 5000} {
		body := checkBody(b, rows)
		b.Run(fmt.Sprintf("%d rows", rows), func(b *testing.B) {
			planner.Measure(b, http.MethodPost, workbookCheckPath, body)
		})
	}
}

// TestWorkbookCheckStatements_038_FR008_H2 proves the statements of a check grow by at most 4 per row on top of
// a fixed part, and that 1,000 rows take at most 5 seconds (SC-002).
func TestWorkbookCheckStatements_038_FR008_H2(t *testing.T) {
	if testing.Short() {
		t.Skip("needs PostgreSQL")
	}
	server := apibench.New(t, apibench.Demo)
	planner := server.SignIn(t, priya)
	small := checked(t, planner, 10)
	large := checked(t, planner, 100)
	if grown := large.Queries - small.Queries; grown > 4*90 {
		t.Errorf("100 rows sent %d statements, 10 rows %d: %d more, want at most %d", large.Queries, small.Queries, grown, 4*90)
	}
	start := time.Now()
	full := checked(t, planner, 1000)
	took := time.Since(start)
	t.Logf("1,000 rows: %d statements in %v", full.Queries, took)
	if full.Queries > 4150 || took > 5*time.Second {
		t.Errorf("1,000 rows sent %d statements in %v, want at most 4,150 within 5s", full.Queries, took)
	}
	writeFixture(t)
}

func checked(t *testing.T, c *apibench.Client, rows int) apibench.Response {
	t.Helper()
	answer := c.Do(t, http.MethodPost, workbookCheckPath, checkBody(t, rows))
	var result struct {
		Rows         int `json:"rows"`
		WithProblems int `json:"withProblems"`
	}
	answer.Decode(t, &result)
	if result.Rows != rows || result.WithProblems != 0 {
		t.Fatalf("check of %d rows read %d with %d problems: %s", rows, result.Rows, result.WithProblems, answer.Body)
	}
	return answer
}

func checkBody(tb testing.TB, rows int) map[string]string {
	tb.Helper()
	return map[string]string{"fileName": "bench.xlsx", "content": base64.StdEncoding.EncodeToString(generated(tb, rows))}
}

// generated is a valid workbook of rows data rows for the demo plan: one new campaign, an initiative per 100
// rows, then activities and their line items in equal parts.
func generated(tb testing.TB, rows int) []byte {
	tb.Helper()
	initiatives := max(1, rows/100)
	activities := (rows - 1 - initiatives) / 2
	lineItems := rows - 1 - initiatives - activities
	const campaign = "Bench campaign"
	wb := sheet.Workbook{Sheets: []sheet.Sheet{
		{Name: "Campaigns", Rows: texts([]string{"Campaign name", "Owner"}, [][]string{{campaign, priya}})},
		{Name: "Initiatives", Rows: texts([]string{"Campaign", "Initiative name"}, numbered(initiatives, func(i int) []string {
			return []string{campaign, initiativeName(i)}
		}))},
		{Name: "Activities", Rows: texts([]string{"Campaign", "Initiative", "Activity name", "Activity type", "Planned cost"},
			numbered(activities, func(i int) []string {
				return []string{campaign, initiativeName(i % initiatives), activityName(i), "Event", "1,000.00"}
			}))},
		{Name: "Line items", Rows: texts([]string{"Campaign", "Initiative", "Activity", "Line item name", "Currency", "Planned cost"},
			numbered(lineItems, func(i int) []string {
				a := i % activities
				return []string{campaign, initiativeName(a % initiatives), activityName(a), fmt.Sprintf("Item %d", i), "EUR", "250"}
			}))},
	}}
	var buf bytes.Buffer
	if err := sheet.WriteXLSX(&buf, wb); err != nil {
		tb.Fatal(err)
	}
	return buf.Bytes()
}

func initiativeName(i int) string { return fmt.Sprintf("Bench initiative %d", i+1) }

func activityName(i int) string { return fmt.Sprintf("Bench activity %d", i+1) }

func numbered(n int, row func(i int) []string) [][]string {
	out := make([][]string, n)
	for i := range n {
		out[i] = row(i)
	}
	return out
}

func texts(heading []string, rows [][]string) [][]sheet.Cell {
	out := [][]sheet.Cell{}
	for _, r := range append([][]string{heading}, rows...) {
		cells := []sheet.Cell{}
		for _, text := range r {
			cells = append(cells, sheet.Text(text))
		}
		out = append(out, cells)
	}
	return out
}

// writeFixture writes the 5,000-row workbook of the end-to-end tests when the folder is named.
func writeFixture(t *testing.T) {
	t.Helper()
	dir := os.Getenv(fixtureEnv)
	if dir == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "workbook-5000.xlsx"), generated(t, 5000), 0o600); err != nil {
		t.Fatal(err)
	}
}
