package campaign

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

// A campaign copy has every stored field of the original, the chosen name and its destination, with new
// bookkeeping; its initiatives are copied with their names and fields, and their activities and line items
// through feature 006, within the year (FR-027, R13, US6 scenario 5).
func TestCopyCampaign_CopiesEverythingBelow_FR027(t *testing.T) { //nolint:gocyclo // 13: one copy, then the campaign, an initiative and the activities checked in turn
	t.Parallel()
	f := newFixture(t)
	d := f.newDemo(t)
	f.advance(time.Hour)

	copied, err := f.service.CopyCampaign(t.Context(), daniel, d.helix.ID, CampaignCopy{PlanID: "P-1001", Name: " Helix launch (copy) "})

	if err != nil {
		t.Fatalf("CopyCampaign: %v", err)
	}
	original, _ := f.storedCampaign(t, d.helix.ID)
	want := original
	want.ID, want.Name, want.Meta = copied.ID, "Helix launch (copy)", record.NewMeta(daniel, start.Add(time.Hour))
	if got, _ := f.storedCampaign(t, copied.ID); copied.ID == d.helix.ID || !reflect.DeepEqual(got, want) {
		t.Errorf("stored copy =\n%+v\nwant\n%+v", got, want)
	}
	if len(copied.Initiatives) != 2 || copied.Counts.Activities != 3 || copied.Counts.LineItems != 2 {
		t.Fatalf("copy holds %+v, want 2 initiatives, 3 activities and 2 line items", copied.Counts)
	}
	var digitalCopy InitiativeView
	for _, i := range copied.Initiatives {
		if i.Name == "Digital & social" {
			digitalCopy = i
		}
	}
	originalDigital, _ := f.storedInitiative(t, d.digital.ID)
	wantDigital := originalDigital
	wantDigital.ID, wantDigital.CampaignID, wantDigital.Meta = digitalCopy.ID, copied.ID, record.NewMeta(daniel, start.Add(time.Hour))
	if got, _ := f.storedInitiative(t, digitalCopy.ID); digitalCopy.ID == d.digital.ID || !reflect.DeepEqual(got, wantDigital) {
		t.Errorf("initiative copy =\n%+v\nwant\n%+v", got, wantDigital)
	}
	if len(f.activities.copies) != 2 || !strings.HasSuffix(f.activities.copies[0], " +0") {
		t.Errorf("activity copies = %q, want one per initiative, within the year", f.activities.copies)
	}
}

// A copy is made completely or not at all: when copying the activities fails, no campaign, initiative or
// log entry is stored and no ID is used up (FR-027).
func TestCopyCampaign_AllOrNothing_FR027(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.newDemo(t)
	f.activities.copyErr = errors.New("disk full")
	entries := len(f.log(t, "", ""))

	_, err := f.service.CopyCampaign(t.Context(), priya, d.helix.ID, CampaignCopy{PlanID: "P-1001", Name: "Helix launch (copy)"})

	if !errors.Is(err, f.activities.copyErr) {
		t.Fatalf("CopyCampaign = %v, want the activities' error", err)
	}
	if tree := f.list(t, "P-1001"); tree.Totals.Campaigns != 2 || tree.Totals.Initiatives != 3 {
		t.Errorf("plan holds %d campaigns and %d initiatives, want the 2 and 3 of before", tree.Totals.Campaigns, tree.Totals.Initiatives)
	}
	if got := len(f.log(t, "", "")); got != entries {
		t.Errorf("log entries = %d, want %d", got, entries)
	}
	if next := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Next"}); next.ID != "C-1003" {
		t.Errorf("next campaign = %s, want C-1003: the failed copy used up no ID", next.ID)
	}
}

// The copy's name is checked like a new campaign's name in the destination plan, and the destination must
// be a plan of the same year; all problems come at once. The same name is fine in another plan (FR-026).
func TestCopyCampaign_NameCheckedInDestination_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.newDemo(t)
	f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "HELIX launch"})
	copyTo := func(planID, name string) (CampaignView, error) {
		return f.service.CopyCampaign(t.Context(), priya, d.helix.ID, CampaignCopy{PlanID: planID, Name: name})
	}

	_, taken := copyTo("P-1002", "helix launch")
	_, both := copyTo("P-1003", " ")
	_, gone := copyTo("P-9999", "Helix launch")
	_, own := copyTo("P-1001", "Rimula FLEET")
	elsewhere, err := copyTo("P-1001", "Helix launch 2")
	_, missing := f.service.CopyCampaign(t.Context(), priya, "C-9999", CampaignCopy{PlanID: "P-1001", Name: "Copy"})

	assertError(t, taken, problems("name", "'FY2026 Mobility China' already has a campaign called 'HELIX launch'."))
	assertError(t, both, problems("name", "Enter a name.", "planId", "Campaigns can be copied only into plans of the same year (2026)."))
	assertError(t, gone, problems("planId", "That plan no longer exists — choose another one."))
	assertError(t, own, problems("name", "'FY2026 Lubricants Europe' already has a campaign called 'Rimula fleet'."))
	assertError(t, missing, errs.NotFound(campaignGoneMessage))
	if err != nil || elsewhere.Name != "Helix launch 2" || elsewhere.PlanID != "P-1001" {
		t.Errorf("CopyCampaign = %q in %s, %v; want Helix launch 2 in P-1001", elsewhere.Name, elsewhere.PlanID, err)
	}
}

// An initiative is copied only into a campaign of its plan whose dates contain its dates; the name is
// checked in that campaign (FR-026, R3).
func TestCopyInitiative_DatesMustFit_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.newDemo(t)
	late := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Late launch", StartDate: day("2026-06-01"), EndDate: day("2026-12-31")})
	other := f.createCampaign(t, priya, "P-1002", CampaignInput{Name: "Retail growth"})
	copyTo := func(campaignID, name string) (InitiativeView, error) {
		return f.service.CopyInitiative(t.Context(), priya, d.digital.ID, InitiativeCopy{CampaignID: campaignID, Name: name})
	}

	_, dates := copyTo(late.ID, "Digital & social")
	_, otherPlan := copyTo(other.ID, "Digital & social")
	_, gone := copyTo("C-9999", "Digital & social")
	_, taken := copyTo(d.helix.ID, "digital & SOCIAL")
	copied, err := copyTo(d.rimula.ID, "Digital & social")
	_, missing := f.service.CopyInitiative(t.Context(), priya, "I-9999", InitiativeCopy{CampaignID: d.helix.ID, Name: "Copy"})

	assertError(t, dates, problems("campaignId", "Dates don't fit (01 Jun – 31 Dec 2026) — change the initiative's dates first."))
	assertError(t, otherPlan, problems("campaignId", "Initiatives can be copied only into campaigns of the same plan."))
	assertError(t, gone, problems("campaignId", "That campaign no longer exists — choose another one."))
	assertError(t, taken, problems("name", "'Helix launch' already has an initiative called 'Digital & social'."))
	assertError(t, missing, errs.NotFound(initiativeGoneMessage))
	if err != nil || copied.CampaignID != d.rimula.ID || copied.ID == d.digital.ID || copied.Counts.Activities != 3 ||
		copied.StartDate.String() != "2026-03-01" {
		t.Errorf("CopyInitiative = %+v, %v; want a new initiative in Rimula fleet with the 3 activities", copied, err)
	}
}

// A copy writes one "copied" entry, on the new record only, naming the original and what came with it;
// the records copied below it get none (D-097, FR-030).
func TestCopy_OneCopiedEntryWithDetails_DecisionH(t *testing.T) { //nolint:gocyclo // 10: two copies, then the entries of each checked in turn
	t.Parallel()
	f := newFixture(t)
	d := f.newDemo(t)
	initiativeEntries := len(f.log(t, initiativeKind, ""))

	campaignCopy, err := f.service.CopyCampaign(t.Context(), daniel, d.helix.ID, CampaignCopy{PlanID: "P-1001", Name: "Helix launch (copy)"})
	if err != nil {
		t.Fatalf("CopyCampaign: %v", err)
	}
	printCopy, err := f.service.CopyInitiative(t.Context(), daniel, d.print.ID, InitiativeCopy{CampaignID: d.helix.ID, Name: "Print & outdoor (copy)"})
	if err != nil {
		t.Fatalf("CopyInitiative: %v", err)
	}

	entries := f.log(t, campaignKind, campaignCopy.ID)
	if len(entries) != 1 || entries[0].Action != history.ActionCopied || entries[0].Actor != daniel ||
		entries[0].Details != "Copied from 'Helix launch' ("+d.helix.ID+"), with 2 initiatives and 3 activities" {
		t.Errorf("entries of the copy = %+v, want one copied entry with the counts", entries)
	}
	printEntries := f.log(t, initiativeKind, printCopy.ID)
	if len(printEntries) != 1 || printEntries[0].Details != "Copied from 'Print & outdoor' ("+d.print.ID+")" {
		t.Errorf("entries of the initiative copy = %+v, want one, without contents", printEntries)
	}
	if got := len(f.log(t, initiativeKind, "")); got != initiativeEntries+1 {
		t.Errorf("initiative entries = %d, want %d: the initiatives copied with the campaign write none", got, initiativeEntries+1)
	}
}

// A copy suggests "<name> (copy)", then "(copy 2)", "(copy 3)", … — the first name taken does not report
// (R13).
func TestSuggestCopyName_CopyThenNumbers_R13(t *testing.T) {
	t.Parallel()
	taken := []string{"helix launch (COPY)", "Helix launch (copy 2)"}
	isTaken := func(name string) bool {
		return slices.ContainsFunc(taken, func(t string) bool { return strings.EqualFold(t, name) })
	}

	if got := suggestCopyName("Helix launch", func(string) bool { return false }); got != "Helix launch (copy)" {
		t.Errorf("first copy = %q", got)
	}
	if got := suggestCopyName("Helix launch", isTaken); got != "Helix launch (copy 3)" {
		t.Errorf("third copy = %q, want Helix launch (copy 3)", got)
	}
}

// A copy name stays within 120 characters: the original part is shortened, without trailing spaces, so the
// suffix fits (edge case).
func TestSuggestCopyName_ShortensLongNames_EdgeCase(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("ü", 112) + " " + strings.Repeat("x", 7) // 120 characters
	if got, want := suggestCopyName(long, func(string) bool { return false }), strings.Repeat("ü", 112)+" (copy)"; got != want {
		t.Errorf("copy name = %q, want %q", got, want)
	}
	second := suggestCopyName(long, func(n string) bool { return strings.HasSuffix(n, " (copy)") })
	if utf8.RuneCountInString(second) != 120 || !strings.HasSuffix(second, " (copy 2)") {
		t.Errorf("second copy name = %q (%d characters), want 120 ending in (copy 2)", second, utf8.RuneCountInString(second))
	}
}
