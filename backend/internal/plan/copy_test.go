package plan

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// The suggestion is the first name no plan has, ignoring letter case.
func TestCopyName_Suffixes_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	source := f.create(t, priya, input("FY2026 Lubricants Europe"))

	first := f.copyOptions(t, source.ID).Name
	f.create(t, priya, input("fy2026 LUBRICANTS europe (COPY)"))
	second := f.copyOptions(t, source.ID).Name
	f.create(t, priya, input("FY2026 Lubricants Europe (copy 2)"))
	third := f.copyOptions(t, source.ID).Name

	got := []string{first, second, third}
	want := []string{"FY2026 Lubricants Europe (copy)", "FY2026 Lubricants Europe (copy 2)", "FY2026 Lubricants Europe (copy 3)"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("suggested names = %q, want %q", got, want)
	}
}

// The original part is shortened, without trailing spaces, so the suffix fits; characters, not bytes, count.
func TestCopyName_ShortenedTo120_EdgeCase(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("é", 112) + " " + strings.Repeat("b", 7) // 120 characters, a space at 113
	free := func(string) bool { return false }
	copyTaken := func(name string) bool { return strings.HasSuffix(name, " (copy)") }

	if got, want := copyName(long, free), strings.Repeat("é", 112)+" (copy)"; got != want {
		t.Errorf("copyName = %q, want %q", got, want)
	}
	if got, want := copyName(long, copyTaken), strings.Repeat("é", 111)+" (copy 2)"; got != want {
		t.Errorf("copyName, (copy) taken = %q, want %q", got, want)
	}
	if got := copyName("Short", free); got != "Short (copy)" {
		t.Errorf("copyName(Short) = %q, want Short (copy)", got)
	}
}

// The next year is offered only up to the current year + 5; the copy refuses a year the dialog does not offer.
func TestCopyOptions_NextYearAvailability_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addLevel(t, "campaign", map[string]Facts{
		"P-1001": {Campaigns: 2, Initiatives: 4, Activities: 7, ApprovedActivities: 4, LineItems: 15},
	})
	europe := f.create(t, priya, input("FY2026 Lubricants Europe"))
	farAhead := input("FY2031 Lubricants Europe")
	farAhead.Year = ptr(2031)
	last := f.create(t, priya, farAhead)

	got := f.copyOptions(t, europe.ID)
	want := CopyOptions{
		Name: "FY2026 Lubricants Europe (copy)", SameYear: 2026, NextYear: NextYearOption{Year: 2027, Available: true},
		Contents: Contents{Campaigns: 2, Initiatives: 4, Activities: 7, ApprovedActivities: 4, LineItems: 15},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CopyOptions(2026 plan) = %+v\nwant %+v", got, want)
	}
	if got := f.copyOptions(t, last.ID).NextYear; got != (NextYearOption{Year: 2032, Reason: "Plans can only be created up to 2031."}) {
		t.Errorf("next year of a 2031 plan = %+v, want 2032, unavailable, with the reason", got)
	}

	_, beyond := f.service.Copy(t.Context(), daniel, last.ID, CopyRequest{Name: "Copy", Year: ptr(2032)})
	_, other := f.service.Copy(t.Context(), daniel, last.ID, CopyRequest{Name: "Copy", Year: ptr(2030)})
	_, optionsGone := f.service.CopyOptions(t.Context(), daniel, "P-9999")
	_, copyGone := f.service.Copy(t.Context(), daniel, "P-9999", CopyRequest{Name: "Copy", Year: ptr(2026)})

	assertError(t, beyond, problems("year", "Plans can only be created up to 2031."))
	assertError(t, other, problems("year", "Copy into 2031 or 2032."))
	assertError(t, optionsGone, errs.NotFound(planGoneMessage))
	assertError(t, copyGone, errs.NotFound(planGoneMessage))
}

// The name is checked as on create; every problem, the year's included, is reported at once; nothing is saved.
func TestCopy_Validation_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	source := f.create(t, priya, input("FY2026 Lubricants Europe"))
	entries := len(f.entries(t))

	_, empty := f.service.Copy(t.Context(), daniel, source.ID, CopyRequest{Name: "  "})
	_, taken := f.service.Copy(t.Context(), daniel, source.ID, CopyRequest{Name: " fy2026 lubricants EUROPE ", Year: ptr(2026)})
	_, long := f.service.Copy(t.Context(), daniel, source.ID, CopyRequest{Name: strings.Repeat("x", 121), Year: ptr(2027)})

	assertError(t, empty, problems("name", "Enter a plan name.", "year", "Copy into 2026 or 2027."))
	assertError(t, taken, problems("name", "There is already a plan called 'FY2026 Lubricants Europe'. Choose another name."))
	assertError(t, long, problems("name", "Use 120 characters or fewer."))
	if got := len(f.entries(t)); got != entries {
		t.Errorf("log entries = %d, want %d: nothing is saved", got, entries)
	}
}

// Without its contents, a copy of a Published plan is a Draft with a new ID and the copier as its creator.
func TestCopy_PlanOnly_FR027(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	level := f.addLevel(t, "campaign", nil)
	in := input("FY2026 Lubricants Europe")
	in.AOP, in.BCPIR, in.Description = true, true, "Lubricants marketing in Europe."
	in.TeamIDs = []string{teamLubricants, teamGlobal}
	in.Budget = &money.Input{Amount: "1200000.00", Currency: "USD"}
	in.DefaultFinanceBudgetID, in.DefaultSpread, in.GoalAmount = ptr(budget2026), string(SpreadFirstMonth), ptr("250000")
	source := f.create(t, priya, in)
	if _, err := f.service.Publish(t.Context(), priya, source.ID); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	f.clock.Advance(time.Hour)

	result, err := f.service.Copy(t.Context(), daniel, source.ID, CopyRequest{Name: " FY2026 Lubricants Europe (copy) ", Year: ptr(2026)})

	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if result.Plan.ID != "P-1002" || result.Plan.Status != StatusDraft || result.Plan.Name != "FY2026 Lubricants Europe (copy)" ||
		result.Notes == nil || len(result.Notes) != 0 {
		t.Errorf("result = %s %s %q, notes %v; want P-1002, Draft, the trimmed name, no notes", result.Plan.ID,
			result.Plan.Status, result.Plan.Name, result.Notes)
	}
	want := f.stored(t, source.ID)
	want.ID, want.Name, want.Status = "P-1002", "FY2026 Lubricants Europe (copy)", StatusDraft
	want.Meta = record.NewMeta(daniel, f.clock.Now())
	if got := f.stored(t, "P-1002"); !reflect.DeepEqual(got, want) {
		t.Errorf("stored copy =\n%+v\nwant\n%+v", got, want)
	}
	if len(level.copyJobs) != 0 {
		t.Errorf("the level below copied %d times, want none without the contents", len(level.copyJobs))
	}
	newest := f.entries(t)[0]
	if newest.Action != history.ActionCopied || newest.Actor != daniel || newest.RecordID != "P-1002" ||
		newest.Details != "from 'FY2026 Lubricants Europe' (P-1001)" {
		t.Errorf("log entry = %s by %s on %s, details %q; want copied by Daniel on P-1002, from the original",
			newest.Action, newest.Actor.Name, newest.RecordID, newest.Details)
	}
}

// Each level, top-down, is given the year shift and the old → new IDs copied above it, the plan's first.
func TestCopy_WithContents_FR027(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.registerStubKind(t, "campaign", "campaigns", kindName)
	f.registerStubKind(t, "initiative", "initiatives", "campaign")
	campaignLevel := f.addLevel(t, "campaign", nil)
	campaignLevel.copy = func(_ context.Context, job *CopyJob) error {
		job.NewIDs["C-1001"], job.NewIDs["C-1002"] = "C-1003", "C-1004"
		job.Counts["campaign"] = 2
		return nil
	}
	initiativeLevel := f.addLevel(t, "initiative", nil)
	initiativeLevel.copy = func(_ context.Context, job *CopyJob) error {
		job.Counts["initiative"] = 1
		return nil
	}
	source := f.create(t, priya, input("FY2026 Lubricants Europe"))

	_, err := f.service.Copy(t.Context(), daniel, source.ID, CopyRequest{Name: "FY2027 Lubricants Europe", Year: ptr(2027), IncludeContents: true})
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	_, err = f.service.Copy(t.Context(), daniel, source.ID, CopyRequest{Name: "Same year", Year: ptr(2026), IncludeContents: true})
	if err != nil {
		t.Fatalf("Copy (same year): %v", err)
	}

	first := CopyJob{
		Actor: daniel, Now: start, SourcePlanID: "P-1001", TargetPlanID: "P-1002", Years: 1,
		NewIDs: map[string]string{"P-1001": "P-1002"}, Counts: map[string]int{},
	}
	if len(campaignLevel.copyJobs) != 2 || !reflect.DeepEqual(campaignLevel.copyJobs[0], first) {
		t.Errorf("campaign level's jobs = %+v\nwant first %+v", campaignLevel.copyJobs, first)
	} else if campaignLevel.copyJobs[1].Years != 0 || campaignLevel.copyJobs[1].TargetPlanID != "P-1003" {
		t.Errorf("same-year job = %+v, want 0 years into P-1003", campaignLevel.copyJobs[1])
	}
	below := initiativeLevel.copyJobs[0]
	if wantIDs := map[string]string{"P-1001": "P-1002", "C-1001": "C-1003", "C-1002": "C-1004"}; !reflect.DeepEqual(below.NewIDs, wantIDs) ||
		!reflect.DeepEqual(below.Counts, map[string]int{"campaign": 2}) {
		t.Errorf("initiative level got IDs %v and counts %v, want %v and 2 campaigns", below.NewIDs, below.Counts, wantIDs)
	}
	if got := f.entries(t)[1].Details; got != "from 'FY2026 Lubricants Europe' (P-1001), with 2 campaigns and 1 initiative" {
		t.Errorf("details = %q, want the source and the counts", got)
	}
}

// When a level fails, nothing is saved: no plan, no copied record, no log entry, no ID used up.
func TestCopy_AllOrNothing_FR027(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	copies := fake.TableOf(f.db, "test_copies", func(c map[string]string) string { return c["id"] })
	first := f.addLevel(t, "campaign", nil)
	first.copy = func(ctx context.Context, _ *CopyJob) error {
		return copies.Put(ctx, map[string]string{"id": "C-1003"})
	}
	failure := errors.New("disk full")
	f.addLevel(t, "initiative", nil).copy = func(context.Context, *CopyJob) error { return failure }
	source := f.create(t, priya, input("FY2026 Lubricants Europe"))
	entries := len(f.entries(t))

	_, err := f.service.Copy(t.Context(), daniel, source.ID, CopyRequest{Name: "Copy", Year: ptr(2026), IncludeContents: true})

	if !errors.Is(err, failure) {
		t.Fatalf("Copy = %v, want the level's error", err)
	}
	if copied, err := copies.List(); err != nil || len(copied) != 0 {
		t.Errorf("records the level copied = %v, %v; want none", copied, err)
	}
	if got := len(f.entries(t)); got != entries {
		t.Errorf("log entries = %d, want %d", got, entries)
	}
	if next := f.create(t, priya, input("Next plan")); next.ID != "P-1002" {
		t.Errorf("next plan = %s, want P-1002: the failed copy used up no ID", next.ID)
	}
}

// A next-year copy clears the default finance budget, which belongs to the plan's year, and notes it.
func TestCopy_NextYear_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	withBudget := input("FY2026 Lubricants Europe")
	withBudget.DefaultFinanceBudgetID = ptr(budget2026)
	europe := f.create(t, priya, withBudget)
	plain := f.create(t, priya, input("FY2026 Lubricants Test"))

	result, err := f.service.Copy(t.Context(), daniel, europe.ID, CopyRequest{Name: "FY2027 Lubricants Europe", Year: ptr(2027)})
	plainResult, plainErr := f.service.Copy(t.Context(), daniel, plain.ID, CopyRequest{Name: "FY2027 Lubricants Test", Year: ptr(2027)})

	if err != nil || plainErr != nil {
		t.Fatalf("Copy = %v, %v", err, plainErr)
	}
	copied := f.stored(t, result.Plan.ID)
	if copied.Year != 2027 || copied.DefaultFinanceBudgetID != "" || result.Plan.DefaultFinanceBudget != nil {
		t.Errorf("next-year copy: year %d, finance budget %q; want 2027 and none", copied.Year, copied.DefaultFinanceBudgetID)
	}
	if want := []string{"Default finance budget cleared — it belonged to 2026."}; !reflect.DeepEqual(result.Notes, want) {
		t.Errorf("notes = %q, want %q", result.Notes, want)
	}
	if plainResult.Plan.Year != 2027 || len(plainResult.Notes) != 0 {
		t.Errorf("copy without a finance budget: year %d, notes %q; want 2027 and none", plainResult.Plan.Year, plainResult.Notes)
	}
}

func (f *fixture) copyOptions(t *testing.T, id string) CopyOptions {
	t.Helper()
	options, err := f.service.CopyOptions(t.Context(), daniel, id)
	if err != nil {
		t.Fatalf("CopyOptions(%s): %v", id, err)
	}
	return options
}

// registerStubKind registers a history kind without records, so the log can name its counts.
func (f *fixture) registerStubKind(t *testing.T, name, plural, parent string) {
	t.Helper()
	err := f.history.Register(history.Kind{
		Name: name, Singular: name, Plural: plural, Parent: parent,
		Snapshot: func(context.Context, string) (history.Snapshot, bool, error) { return history.Snapshot{}, false, nil },
		ChildIDs: func(context.Context, string) ([]string, error) { return nil, nil },
		Remove:   func(context.Context, string) error { return nil },
		Put:      func(context.Context, history.Snapshot) error { return nil },
	})
	if err != nil {
		t.Fatalf("Register(%s): %v", name, err)
	}
}
