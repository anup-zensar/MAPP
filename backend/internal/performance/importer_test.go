package performance_test

import (
	"testing"

	"mapp/internal/performance"
)

func TestClassify_ARepeatedReferenceInOneFileIsRejectedAndTheFirstRowKeepsItsOutcome_FR011(t *testing.T) {
	t.Parallel()

	results := classifiedFile([]performance.Row{
		measureRow("r1", "inq", "5", "A-1"),
		measureRow("r1", "inq", "9", "A-1"),
	})

	if results[0].Outcome != performance.OutcomeMatched {
		t.Errorf("the first row carrying r1 is %s, want matched: the repeat is the second row's fault",
			results[0].Outcome)
	}
	if results[1].Outcome != performance.OutcomeRejected {
		t.Fatalf("the second row carrying r1 is %s, want rejected", results[1].Outcome)
	}
	if !codesOf(results[1].Faults)["repeatedReference"] {
		t.Errorf("the second row reported %v, want repeatedReference", codesOf(results[1].Faults))
	}
}

func TestClassify_TheSameReferenceInALaterLoadIsNotARepeat_046C4(t *testing.T) {
	t.Parallel()

	first := classifiedFile([]performance.Row{measureRow("r1", "inq", "5", "A-1")})
	second := classifiedFile([]performance.Row{measureRow("r1", "inq", "9", "A-1")})

	if first[0].Outcome != performance.OutcomeMatched || second[0].Outcome != performance.OutcomeMatched {
		t.Errorf("loads gave %s then %s, want both matched: replacement across loads is not a fault",
			first[0].Outcome, second[0].Outcome)
	}
}

func TestClassify_AnInvalidRowIsNeverMatched_FR011(t *testing.T) {
	t.Parallel()

	row := measureRow("r1", "inq", "-3", "A-1")

	result := classifiedFile([]performance.Row{row})[0]

	if result.Outcome != performance.OutcomeRejected {
		t.Fatalf("a row with a negative count is %s, want rejected", result.Outcome)
	}
	if result.ActivityID != "" {
		t.Errorf("a rejected row was bound to %s, want no activity: validation comes before matching",
			result.ActivityID)
	}
}

func TestClassify_AHeldRowIsMatchedToAnActivityThatIsNotApproved_FR020(t *testing.T) {
	t.Parallel()

	result := classifiedFile([]performance.Row{measureRow("r1", "inq", "5", "A-2")})[0]

	if result.Outcome != performance.OutcomeHeld {
		t.Fatalf("a row on an unapproved activity is %s, want held", result.Outcome)
	}
	if result.ActivityID != "A-2" {
		t.Errorf("a held row names activity %q, want A-2: it is matched, only not counted", result.ActivityID)
	}
}

func TestClassify_ARowWithNoKeyIsUnattributedNotUnmatched_046C19(t *testing.T) {
	t.Parallel()

	result := classifiedFile([]performance.Row{measureRow("r1", "inq", "5", "")})[0]

	if result.Outcome != performance.OutcomeUnattributed {
		t.Errorf("a row carrying no key is %s, want unattributed", result.Outcome)
	}
}

func TestClassify_MeasuresAreParsedOnceAndCarried_FR014(t *testing.T) {
	t.Parallel()

	row := performance.Row{
		Reference: performance.Cell{Column: "reference", Value: "r1"},
		Date:      performance.Cell{Column: "date", Value: "2026-02-01"},
		Currency:  performance.Cell{Column: "currency", Value: "eur"},
		Keys:      performance.RowKeys{ActivityID: "A-1"},
		Readings: []performance.Reading{
			{Measure: performance.Measure{Stage: "inq"}, Cell: performance.Cell{Column: "inq", Value: "5"}},
			{Measure: performance.Measure{Stage: "revenue"}, Cell: performance.Cell{Column: "rev", Value: "1234.50"}},
			{Measure: performance.Measure{Stage: "mql"}, Cell: performance.Cell{Column: "mql", Value: ""}},
		},
	}

	result := classifiedFile([]performance.Row{row})[0]

	if len(result.Measured) != 2 {
		t.Fatalf("Classify() carried %d measures, want 2: the empty mql column is an absent fact",
			len(result.Measured))
	}
	if result.Measured[0].Count == nil || *result.Measured[0].Count != 5 {
		t.Errorf("the inq count came through as %v, want 5", result.Measured[0].Count)
	}
	revenue := result.Measured[1].Amount
	if revenue == nil || revenue.Amount.Format() != "1,234.50" || revenue.Currency != "EUR" {
		t.Errorf("the revenue came through as %v, want 1,234.50 EUR exactly", revenue)
	}
}

// FuzzClassify checks what every caller relies on, whatever a file holds: the summary always reconciles,
// every row gets exactly one of the five outcomes, and nothing panics.
func FuzzClassify(f *testing.F) {
	f.Add("r1", "2026-02-01", "5", "A-1", "USD")
	f.Add("", "", "", "", "")
	f.Add("r1", "not-a-date", "-1.5", "A-missing", "ZZZ")
	f.Add("r2", "2999-01-01", "99999999999999999999", "A-shared", "")

	f.Fuzz(func(t *testing.T, ref, date, count, key, ccy string) {
		rows := []performance.Row{{
			Reference: performance.Cell{Column: "reference", Value: ref},
			Date:      performance.Cell{Column: "date", Value: date},
			Currency:  performance.Cell{Column: "currency", Value: ccy},
			Keys:      performance.RowKeys{ActivityID: key},
			Readings: []performance.Reading{
				{Measure: performance.Measure{Stage: "inq"}, Cell: performance.Cell{Column: "inq", Value: count}},
				{Measure: performance.Measure{Stage: "revenue"}, Cell: performance.Cell{Column: "rev", Value: count}},
			},
		}}

		summary := performance.Summarise(performance.Classify(rows, performance.Env{
			Today: today, Currencies: knownCcy,
			Resolved: performance.Resolved{performance.MatchedActivityID: map[string][]string{
				"A-1": {"A-1"}, "A-shared": {"A-2", "A-3"},
			}},
			Approved: func(string) bool { return true },
		}))

		if !summary.Counts.Reconciles() || summary.Counts.Read != 1 {
			t.Fatalf("Classify(%q, %q, %q, %q, %q) summarised as %+v, which does not account for the row",
				ref, date, count, key, ccy, summary.Counts)
		}
	})
}
