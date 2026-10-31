package performance_test

import (
	"strconv"
	"testing"
	"testing/quick"
	"time"

	"mapp/internal/performance"
)

func classifiedFile(rows []performance.Row) []performance.RowResult {
	return performance.Classify(rows, performance.Env{
		Today:      today,
		Currencies: knownCcy,
		Resolved: performance.Resolved{performance.MatchedActivityID: map[string][]string{
			"A-1": {"A-1"}, "A-2": {"A-2"}, "A-shared": {"A-2", "A-3"},
		}},
		Approved: func(id string) bool { return id != "A-2" },
	})
}

// TestSummarise_FiveOutcomesSumToRowsRead_FR022 is the property SC-001 rests on: whatever the file, the
// five counts account for every row read, overall and inside each measure group.
func TestSummarise_FiveOutcomesSumToRowsRead_FR022(t *testing.T) {
	t.Parallel()

	reconciles := func(refs []string, dates []uint8, counts []int16, keys []uint8) bool {
		rows := generatedRows(refs, dates, counts, keys)
		summary := performance.Summarise(classifiedFile(rows))
		if !summary.Counts.Reconciles() || summary.Counts.Read != len(rows) {
			return false
		}
		for _, group := range summary.Breakdown {
			if !group.Counts.Reconciles() {
				return false
			}
		}
		return true
	}

	if err := quick.Check(reconciles, &quick.Config{MaxCount: 500}); err != nil {
		t.Errorf("the five outcomes did not account for every row read: %v", err)
	}
}

// generatedRows turns four generated slices into a file whose rows reach every outcome: unreadable dates
// and negative counts reject, repeated references reject, unknown and ambiguous keys leave rows unmatched,
// a missing key leaves a row unattributed, and A-2 is the activity that is not approved.
func generatedRows(refs []string, dates []uint8, counts []int16, keys []uint8) []performance.Row {
	possibleKeys := []string{"", "A-1", "A-2", "A-shared", "A-missing"}
	rows := make([]performance.Row, 0, len(refs))
	for i, ref := range refs {
		date := "2026-02-01"
		if at(dates, i)%5 == 0 {
			date = "not-a-date"
		}
		count := int64(at(counts, i))
		rows = append(rows, performance.Row{
			Reference: performance.Cell{Column: "reference", Value: ref},
			Date:      performance.Cell{Column: "date", Value: date},
			Keys:      performance.RowKeys{ActivityID: possibleKeys[int(at(keys, i))%len(possibleKeys)]},
			Readings: []performance.Reading{{
				Measure: performance.Measure{Stage: stageFor(i)},
				Cell:    performance.Cell{Column: "n", Value: strconv.FormatInt(count, 10)},
			}},
		})
	}
	return rows
}

func stageFor(i int) string {
	if i%2 == 0 {
		return "inq"
	}
	return "mql"
}

func at[T any](values []T, i int) T {
	var zero T
	if len(values) == 0 {
		return zero
	}
	return values[i%len(values)]
}

func TestSummarise_BreaksTheCountsDownByMeasure_046C18(t *testing.T) {
	t.Parallel()

	rows := []performance.Row{
		measureRow("r1", "inq", "5", "A-1"),
		measureRow("r2", "mql", "3", "A-missing"),
		measureRow("r3", "mql", "4", "A-missing"),
	}

	summary := performance.Summarise(classifiedFile(rows))

	if len(summary.Breakdown) != 2 {
		t.Fatalf("Summarise() gave %d groups, want one for inq and one for mql", len(summary.Breakdown))
	}
	byStage := map[string]performance.Counts{}
	for _, group := range summary.Breakdown {
		byStage[group.Stage] = group.Counts
	}
	if byStage["mql"].Unmatched != 2 {
		t.Errorf("mql reported %d unmatched, want 2: the whole point is seeing which stage went wrong",
			byStage["mql"].Unmatched)
	}
	if byStage["inq"].Matched != 1 {
		t.Errorf("inq reported %d matched, want 1", byStage["inq"].Matched)
	}
}

func TestSummarise_ARowWithTwoMeasuresCountsInBoth_046C18(t *testing.T) {
	t.Parallel()

	wide := performance.Row{
		Reference: performance.Cell{Column: "reference", Value: "r1"},
		Date:      performance.Cell{Column: "date", Value: "2026-02-01"},
		Keys:      performance.RowKeys{ActivityID: "A-1"},
		Readings: []performance.Reading{
			{Measure: performance.Measure{Stage: "inq"}, Cell: performance.Cell{Column: "inq", Value: "5"}},
			{Measure: performance.Measure{Stage: "mql"}, Cell: performance.Cell{Column: "mql", Value: "2"}},
		},
	}

	summary := performance.Summarise(classifiedFile([]performance.Row{wide}))

	if summary.Counts.Read != 1 {
		t.Errorf("rows read = %d, want 1: one line of the file is one row however many measures it carries",
			summary.Counts.Read)
	}
	if len(summary.Breakdown) != 2 {
		t.Errorf("Summarise() gave %d groups, want 2: the row carries both measures", len(summary.Breakdown))
	}
}

func TestSummarise_DataThroughIsTheLatestFactDate_FR005(t *testing.T) {
	t.Parallel()

	rows := []performance.Row{
		dated("r1", "2026-01-31", "A-1"),
		dated("r2", "2026-02-28", "A-1"),
		dated("r3", "2026-03-01", "A-missing"),
	}

	summary := performance.Summarise(classifiedFile(rows))

	if summary.DataThrough == nil {
		t.Fatal("Summarise() reported no data-through date, want the latest date that became a fact")
	}
	want := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	if !summary.DataThrough.Equal(want) {
		t.Errorf("data through %s, want %s: an unmatched row delivered no data", summary.DataThrough, want)
	}
}

func TestSummarise_AnEmptyFileReconcilesAndCarriesNoGroups_FR022(t *testing.T) {
	t.Parallel()

	summary := performance.Summarise(classifiedFile(nil))

	if !summary.Counts.Reconciles() || summary.Counts.Read != 0 {
		t.Errorf("an empty file summarised as %+v, want every count zero", summary.Counts)
	}
	if summary.Breakdown == nil {
		t.Error("Breakdown is nil, want an empty list: a nil slice stores as JSON null, not []")
	}
}

func measureRow(ref, stage, count, activityID string) performance.Row {
	return performance.Row{
		Reference: performance.Cell{Column: "reference", Value: ref},
		Date:      performance.Cell{Column: "date", Value: "2026-02-01"},
		Keys:      performance.RowKeys{ActivityID: activityID},
		Readings: []performance.Reading{{
			Measure: performance.Measure{Stage: stage},
			Cell:    performance.Cell{Column: stage, Value: count},
		}},
	}
}

func dated(ref, date, activityID string) performance.Row {
	row := measureRow(ref, "inq", "1", activityID)
	row.Date = performance.Cell{Column: "date", Value: date}
	return row
}

func TestSummarise_AnOutcomeItDoesNotKnowIsNotCountedAsRead_FR022(t *testing.T) {
	t.Parallel()

	summary := performance.Summarise([]performance.RowResult{
		{Outcome: performance.OutcomeMatched},
		{Outcome: performance.Outcome("something new")},
	})

	if !summary.Counts.Reconciles() {
		t.Errorf("Summarise() gave %+v, which does not reconcile: an outcome the summary does not know "+
			"must not raise rows read", summary.Counts)
	}
	if summary.Counts.Read != 1 {
		t.Errorf("rows read = %d, want 1", summary.Counts.Read)
	}
}
