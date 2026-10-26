package settings

import (
	"encoding/base64"
	"fmt"
	"slices"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/date"
)

// rateHeader is the header of the GSAP daily rate extract.
const rateHeader = "rate_type,from_currency,to_currency,valid_from,rate"

// ninetyTwoRows is the file of US1-2: 87 rates to add (EUR, GBP and the inactive INR for the 29 days of
// 20 Sep – 18 Oct 2026), 2 that replace a different rate for 19 Oct, 1 already present, 1 not used (CHF) and
// 1 problem on row 41.
func ninetyTwoRows() RateImportUpload {
	var rows []string
	first := date.MustParse("2026-09-20")
	for day := range 29 {
		for _, code := range []string{"EUR", "GBP", "INR"} {
			rows = append(rows, fmt.Sprintf("M,USD,%s,%s,%d.%03d", code, first.AddDays(day), 1+day, 100+day))
		}
	}
	rows = slices.Insert(rows, 39, "M,USD,EUR,2026-10-20,abc") // data row 40 is row 41 of the file
	rows = append(rows, "M,USD,EUR,2026-10-19,0.9214", "M,USD,GBP,20261019,0.7905", "M,USD,INR,2026-10-19,83.50", "M,USD,CHF,2026-10-19,0.88")
	return upload(append([]string{rateHeader}, rows...)...)
}

// importFixture is feedFixture with INR's stored rate of 19 Oct.
func importFixture(t *testing.T) *fixture {
	t.Helper()
	f := feedFixture(t)
	f.addRate(t, "INR", "2026-10-19", "83.5", RateManual, "")
	return f
}

// The review answers every row with its outcome and row number before anything is saved (US1-2).
func TestCheckRateImport_Review_FR006(t *testing.T) {
	t.Parallel()
	f := importFixture(t)

	check, err := f.service.CheckRateImport(t.Context(), alex, ninetyTwoRows())

	if err != nil || len(check.Rows) != 92 || check.FileName != "gsap_rates_2026-10-20.csv" || check.Feed != "gsapDailyRatesFile" {
		t.Fatalf("CheckRateImport = %d rows, %+v", len(check.Rows), err)
	}
	if want := (RateImportCounts{Add: 87, Replace: 2, Present: 1, NotUsed: 1, Problem: 1}); check.Counts != want {
		t.Errorf("counts = %+v, want %+v", check.Counts, want)
	}
	assertRow(t, check.Rows[39], OutcomeProblem, "Row 41: the rate must be above 0 with at most 6 decimals.")
	assertRow(t, check.Rows[88], OutcomeReplace, "0.92 → 0.9214")
	assertRow(t, check.Rows[91], OutcomeNotUsed, "CHF is not set up in Settings › Currencies.")
	for i, row := range check.Rows {
		if row.Row != i+2 {
			t.Errorf("row %d has number %d, want %d", i, row.Row, i+2)
		}
	}
	if rates := f.storedRates(t, "EUR"); len(rates) != 1 {
		t.Errorf("the review saved rates: %+v", rates)
	}
}

// The import saves exactly the rows the review counts as to add and replace (FR-006).
func TestApplyRateImport_SameCountsAsCheck_FR006(t *testing.T) {
	t.Parallel()
	f := importFixture(t)
	check, _ := f.service.CheckRateImport(t.Context(), alex, ninetyTwoRows())

	imported, err := f.service.ApplyRateImport(t.Context(), alex, ninetyTwoRows())

	if err != nil || imported.Added != check.Counts.Add || imported.Replaced != check.Counts.Replace ||
		imported.Unchanged != check.Counts.Present || imported.NotUsed != check.Counts.NotUsed || imported.Problems != check.Counts.Problem {
		t.Fatalf("imported = %+v, %v; review %+v", imported, err, check.Counts)
	}
	eur := f.storedRates(t, "EUR")
	if len(eur) != 30 || eur[29].Date != date.MustParse("2026-10-19") || eur[29].Rate.String() != "0.9214" ||
		eur[29].Source != RateGSAPImport || eur[29].ImportID != imported.ID || eur[29].Version != 2 {
		t.Errorf("EUR rates = %d, last %+v", len(eur), eur[len(eur)-1])
	}
	_, again := f.service.ApplyRateImport(t.Context(), alex, ninetyTwoRows())
	assertError(t, again, problems("content", "Nothing to import: every rate is already present or not used."))
}

// Every import is recorded — file, feed, who, when, dates, counts — and writes one log entry (FR-007, US1-3).
func TestApplyRateImport_RecordAndLog_FR007(t *testing.T) {
	t.Parallel()
	f := importFixture(t)
	before := len(f.logEntries(t))

	imported, err := f.service.ApplyRateImport(t.Context(), daniel, ninetyTwoRows())

	if err != nil || imported.ID != "RI-1001" || imported.FileName != "gsap_rates_2026-10-20.csv" || imported.Feed != "gsapDailyRatesFile" ||
		*imported.FirstDate != date.MustParse("2026-09-20") || *imported.LastDate != date.MustParse("2026-10-19") ||
		imported.ImportedBy != daniel || !imported.ImportedAt.Equal(october20) {
		t.Fatalf("imported = %+v, %v", imported, err)
	}
	entries := f.logEntries(t)
	if len(entries) != before+1 {
		t.Fatalf("%d new log entries, want 1", len(entries)-before)
	}
	entry := entries[len(entries)-1]
	if entry.RecordType != rateImportKind || entry.RecordID != "RI-1001" || entry.RecordName != "gsap_rates_2026-10-20.csv" ||
		entry.Action != history.ActionCreated || entry.Verb != "imported" || entry.Actor != daniel ||
		entry.Details != "89 exchange rates (87 added, 2 replaced) for 20 Sep – 19 Oct 2026" {
		t.Errorf("log entry = %+v", entry)
	}
	page, err := f.service.ListRateImports(t.Context(), alex, 10)
	if err != nil || page.Total != 1 || page.Items[0].ID != "RI-1001" {
		t.Errorf("recent imports = %+v, %v", page, err)
	}
}

// Inactive currencies keep converting, so their rates are imported (data-model §2.4).
func TestApplyRateImport_InactiveCurrencyImported(t *testing.T) {
	t.Parallel()
	f := feedFixture(t)

	imported, err := f.service.ApplyRateImport(t.Context(), alex, upload(rateHeader, "M,USD,INR,2026-10-20,83.62"))

	inr := f.storedRates(t, "INR")
	if err != nil || imported.Added != 1 || len(inr) != 1 || inr[0].Rate.String() != "83.62" {
		t.Errorf("imported = %+v, %v; INR rates %+v", imported, err, inr)
	}
	if rates, _ := f.service.Rates(t.Context()); rates["INR"].String() != "83.62" {
		t.Errorf("INR's current rate = %v", rates["INR"])
	}
}

// ApplyRateRows is the core every rate feed uses: a later automatic feed hands it its rows and the rules,
// the record and the log are the same (FR-008).
func TestApplyRateRows_SharedCore_FR008(t *testing.T) {
	t.Parallel()
	f := feedFixture(t)
	rows := []FeedRow{
		{Row: 1, RateType: "M", From: "USD", To: "EUR", ValidFrom: "2026-10-20", Rate: "0.921"},
		{Row: 2, RateType: "M", From: "USD", To: "GBP", ValidFrom: "2026-10-19", Rate: "0.79"},
	}

	imported, err := f.service.ApplyRateRows(t.Context(), alex, "gsapDailyRatesFeed", "GSAP feed 20 Oct 2026", rows)

	if err != nil || imported.Feed != "gsapDailyRatesFeed" || imported.FileName != "GSAP feed 20 Oct 2026" || imported.Added != 1 ||
		imported.Unchanged != 1 {
		t.Errorf("ApplyRateRows = %+v, %v", imported, err)
	}
	if rates, _ := f.service.Rates(t.Context()); rates["EUR"].String() != "0.921" {
		t.Errorf("EUR's current rate = %v, want the fed 0.921", rates["EUR"])
	}
}

// The sample file has one M row per active currency but USD, dated today, at the current rate + 0.5 %, and
// imports as it is (FR-006).
func TestImportSample_TodayRates(t *testing.T) {
	t.Parallel()
	f := feedFixture(t)

	name, content, err := f.service.ImportSample(t.Context())

	want := rateHeader + "\nM,USD,EUR,2026-10-20,0.9246\nM,USD,GBP,2026-10-20,0.79395\n"
	if err != nil || name != "gsap_rates_2026-10-20.csv" || string(content) != want {
		t.Fatalf("ImportSample = %q, %q, %v\nwant %q", name, content, err, want)
	}
	imported, err := f.service.ApplyRateImport(t.Context(), alex,
		RateImportUpload{FileName: name, Content: base64.StdEncoding.EncodeToString(content)})
	if err != nil || imported.Added != 2 {
		t.Errorf("importing the sample = %+v, %v", imported, err)
	}
}

// The upload needs a file name of at most 200 characters and base64 content (FR-006).
func TestRateImportUpload_Checked(t *testing.T) {
	t.Parallel()
	f := feedFixture(t)

	_, err := f.service.CheckRateImport(t.Context(), alex, RateImportUpload{FileName: " ", Content: "not base64!"})

	assertError(t, err, problems("fileName", "Choose a file to import.", "content", "This file can't be read. Save it as a .csv file in UTF-8 and try again."))
}
