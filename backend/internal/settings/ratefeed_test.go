package settings

import (
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"testing"

	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

func upload(lines ...string) RateImportUpload {
	content := strings.Join(lines, "\n") + "\n"
	return RateImportUpload{FileName: "gsap_rates_2026-10-20.csv", Content: base64.StdEncoding.EncodeToString([]byte(content))}
}

func feedFixture(t *testing.T) *fixture {
	t.Helper()
	f := newTestService(t)
	f.addCurrency(t, "USD", "1", true)
	f.addCurrency(t, "EUR", "0.90", true)
	f.addCurrency(t, "GBP", "0.80", true)
	f.addCurrency(t, "INR", "83.50", false)
	f.addRate(t, "EUR", "2026-10-19", "0.92", RateGSAPImport, "")
	f.addRate(t, "GBP", "2026-10-19", "0.79", RateManual, "")
	f.clock.now = october20
	return f
}

func TestGSAPRateFile_Header_FR006(t *testing.T) {
	t.Parallel()
	rows, err := GSAPRateFile{}.Read("x.csv", []byte("\xEF\xBB\xBF RATE , Valid_From,to_currency,FROM_CURRENCY,rate_type\n0.9214,2026-10-20,EUR,USD,M\n\n"))
	want := []FeedRow{{Row: 2, RateType: "M", From: "USD", To: "EUR", ValidFrom: "2026-10-20", Rate: "0.9214"}}
	if err != nil || !reflect.DeepEqual(rows, want) {
		t.Errorf("Read = %+v, %v; want %+v", rows, err, want)
	}

	_, missing := GSAPRateFile{}.Read("x.csv", []byte("rate_type,from_currency,to_currency,rate\nM,USD,EUR,0.92\n"))
	assertError(t, missing, problems("content", "The file needs the column 'valid_from'."))
	_, empty := GSAPRateFile{}.Read("x.csv", []byte("rate_type,from_currency,to_currency,valid_from,rate\n"))
	assertError(t, empty, problems("content", "The file has no rows below the header."))
	_, binary := GSAPRateFile{}.Read("x.csv", []byte{0xff, 0xfe, 0x00})
	assertError(t, binary, problems("content", "This file can't be read. Save it as a .csv file in UTF-8 and try again."))
}

func TestGSAPRateFile_DateFormats(t *testing.T) {
	t.Parallel()
	for text, ok := range map[string]bool{"2026-10-19": true, "20261019": true, " 2026-10-19 ": true, "19.10.2026": false, "20261332": false, "": false} {
		day, err := parseRateDate(text)
		if ok && (err != nil || day != date.MustParse("2026-10-19")) {
			t.Errorf("parseRateDate(%q) = %v, %v", text, day, err)
		}
		if !ok && err == nil {
			t.Errorf("parseRateDate(%q) = %v, want an error", text, day)
		}
	}
}

func TestGSAPRateFile_TooLarge(t *testing.T) {
	t.Parallel()
	big := make([]byte, maxRateFileBytes+1)
	_, err := GSAPRateFile{}.Read("big.csv", big)
	assertError(t, err, problems("content", "The file is larger than 2 MB. Split it and import each part."))
}

func checkOne(t *testing.T, f *fixture, row string) RateImportRow {
	t.Helper()
	check, err := f.service.CheckRateImport(t.Context(), alex, upload("rate_type,from_currency,to_currency,valid_from,rate", row))
	if err != nil || len(check.Rows) != 1 {
		t.Fatalf("CheckRateImport(%q) = %+v, %v", row, check, err)
	}
	return check.Rows[0]
}

func assertRow(t *testing.T, got RateImportRow, outcome RateOutcome, message string) {
	t.Helper()
	gotMessage := ""
	if got.Message != nil {
		gotMessage = *got.Message
	}
	if got.Outcome != outcome || gotMessage != message {
		t.Errorf("row %d = %s %q, want %s %q", got.Row, got.Outcome, gotMessage, outcome, message)
	}
}

func TestGSAPRateFile_OtherRateTypeNotUsed_FR006(t *testing.T) {
	t.Parallel()
	f := feedFixture(t)
	assertRow(t, checkOne(t, f, "B,USD,EUR,2026-10-20,0.92"), OutcomeNotUsed, "Rate type B is not used — only M rates are imported.")
	assertRow(t, checkOne(t, f, "m,usd,eur,2026-10-20,0.93"), OutcomeAdd, "")
}

func TestGSAPRateFile_NotFromUSDNotUsed_FR006(t *testing.T) {
	t.Parallel()
	assertRow(t, checkOne(t, feedFixture(t), "M,EUR,GBP,2026-10-20,0.86"), OutcomeNotUsed, "Only rates quoted from USD are used.")
}

func TestGSAPRateFile_USDNotUsed_FR006(t *testing.T) {
	t.Parallel()
	assertRow(t, checkOne(t, feedFixture(t), "M,USD,USD,2026-10-20,1"), OutcomeNotUsed, "USD is always 1.")
}

func TestGSAPRateFile_CurrencyNotSetUpNotUsed_FR006(t *testing.T) {
	t.Parallel()
	row := checkOne(t, feedFixture(t), "M,USD,CHF,2026-10-20,0.88")
	assertRow(t, row, OutcomeNotUsed, "CHF is not set up in Settings › Currencies.")
	if row.Currency == nil || *row.Currency != "CHF" || row.Rate == nil || row.Rate.String() != "0.88" {
		t.Errorf("row = %+v", row)
	}
}

func TestGSAPRateFile_DateAndRateProblems_FR006(t *testing.T) {
	t.Parallel()
	f := feedFixture(t)
	assertRow(t, checkOne(t, f, "M,USD,EUR,19/10/2026,0.92"), OutcomeProblem, "Row 2: 'valid_from' must be a date such as 2026-10-19.")
	for _, rate := range []string{"0.1234567", "0", "abc"} {
		assertRow(t, checkOne(t, f, "M,USD,EUR,2026-10-20,"+rate), OutcomeProblem, "Row 2: the rate must be above 0 with at most 6 decimals.")
	}
}

func TestGSAPRateFile_TwiceInFile_FR006(t *testing.T) {
	t.Parallel()
	check, err := feedFixture(t).service.CheckRateImport(t.Context(), alex, upload("rate_type,from_currency,to_currency,valid_from,rate",
		"M,USD,EUR,2026-10-20,0.92", "M,USD,GBP,2026-10-20,0.79", "M,USD,EUR,20261020,0.93"))
	if err != nil || len(check.Rows) != 3 {
		t.Fatalf("CheckRateImport = %+v, %v", check, err)
	}
	assertRow(t, check.Rows[0], OutcomeAdd, "")
	assertRow(t, check.Rows[2], OutcomeProblem, "Row 4: EUR on 20 Oct 2026 appears twice (also row 2).")
}

func TestGSAPRateFile_PresentReplaceAdd_FR006(t *testing.T) {
	t.Parallel()
	f := feedFixture(t)
	assertRow(t, checkOne(t, f, "M,USD,EUR,2026-10-19,0.920000"), OutcomePresent, "")
	replace := checkOne(t, f, "M,USD,GBP,2026-10-19,0.7905")
	assertRow(t, replace, OutcomeReplace, "0.79 → 0.7905")
	if replace.PreviousRate == nil || replace.PreviousRate.String() != "0.79" || replace.Rate.String() != "0.7905" {
		t.Errorf("replace row = %+v", replace)
	}
	add := checkOne(t, f, "M,USD,INR,2026-10-20,83.62")
	assertRow(t, add, OutcomeAdd, "")
	if add.Currency == nil || *add.Currency != "INR" || add.Date == nil || *add.Date != date.MustParse("2026-10-20") {
		t.Errorf("add row = %+v", add)
	}
}

func TestGSAPRateFile_StrayQuotesAreARow_038_FR034_M13(t *testing.T) {
	t.Parallel()
	check, err := feedFixture(t).service.CheckRateImport(t.Context(), alex, upload(rateHeader,
		`=HYPERLINK("x"),USD,EUR,2026-10-20,0.92`, `M,USD,EUR,2026-10-21,0.9"2`, "M,USD,GBP,2026-10-20,0.79"))
	if err != nil || len(check.Rows) != 3 {
		t.Fatalf("CheckRateImport = %+v, %v", check, err)
	}
	if check.Rows[0].Outcome == OutcomeAdd {
		t.Errorf("the HYPERLINK row was added: %+v", check.Rows[0])
	}
	assertRow(t, check.Rows[1], OutcomeProblem, "Row 3: the rate must be above 0 with at most 6 decimals.")
	assertRow(t, check.Rows[2], OutcomeAdd, "")
}

// An empty file says so, and every missing column is named at once — 038 FR-034 (L-21).
func TestGSAPRateFile_EmptyAndMissingColumns_038_FR034_L21(t *testing.T) {
	t.Parallel()
	for _, content := range []string{"", "\xEF\xBB\xBF", " \n\n"} {
		_, err := GSAPRateFile{}.Read("x.csv", []byte(content))
		assertError(t, err, problems("content", "This file is empty."))
	}
	_, err := GSAPRateFile{}.Read("x.csv", []byte("rate_type,to_currency\nM,EUR\n"))
	assertError(t, err, problems("content", "The file needs the columns 'from_currency', 'valid_from' and 'rate'."))
}

func TestGSAPRateFile_ThreeMissingColumnsOneMessage_038_FR034_L21(t *testing.T) {
	t.Parallel()
	_, err := GSAPRateFile{}.Read("x.csv", []byte("rate_type,from_currency\nM,USD\n"))
	assertError(t, err, problems("content", "The file needs the columns 'to_currency', 'valid_from' and 'rate'."))
	_, err = GSAPRateFile{}.Read("x.csv", []byte("rate_type,from_currency,to_currency,valid_from\nM,USD,EUR,2026-03-13\n"))
	assertError(t, err, problems("content", "The file needs the column 'rate'."))
}

func TestGSAPRateFile_TooLargeRateAndEarlyDate_038_FR034_L21(t *testing.T) {
	t.Parallel()
	f := feedFixture(t)
	assertRow(t, checkOne(t, f, "M,USD,EUR,2026-10-20,1000001"), OutcomeProblem, "Row 2: use a rate of 1,000,000 or less.")
	assertRow(t, checkOne(t, f, "M,USD,EUR,1999-12-31,0.92"), OutcomeProblem, "Row 2: 'valid_from' must be on or after 1 Jan 2000.")
	assertRow(t, checkOne(t, f, "M,USD,EUR,2000-01-01,0.92"), OutcomeAdd, "")
}

func FuzzGSAPRateFile(f *testing.F) {
	for _, seed := range []string{"", "\xEF\xBB\xBF", rateHeader + "\nM,USD,EUR,2026-10-20,0.92\n",
		rateHeader + "\n=HYPERLINK(\"x\"),USD,EUR,2026-10-20,0.92\n", rateHeader + "\n\"M,USD,EUR,2026-10-20,0.9\"2\"\n",
		rateHeader + "\nM,\"USD,EUR\n", "rate\n\"\n"} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, content []byte) {
		rows, err := GSAPRateFile{}.Read("fuzz.csv", content)
		if _, isProblem := errors.AsType[*errs.Error](err); err != nil && !isProblem {
			t.Fatalf("Read = %v, want rows or a problem", err)
		}
		if err == nil && len(rows) == 0 {
			t.Fatalf("Read gave no rows and no problem")
		}
	})
}
