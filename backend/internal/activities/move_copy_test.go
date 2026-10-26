package activities_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"mapp/internal/activities"
	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

// User story 5 — Move and copy an activity. The demo plan FY2026 Lubricants Europe holds Helix launch with
// Digital & social (I-1001, 1 Jan – 30 Jun: Launch film A-1001, 1 Mar – 31 May, Created; Social always-on
// A-1002, Approved) and Print & outdoor (I-1002, 1 Jan – 31 Dec), and Rimula fleet with Events (I-1003,
// 1 Mar – 30 Sep) and Retail events (I-1004, 1 Jun – 31 Dec).

const (
	launchFilm      = "A-1001"
	socialAlwaysOn  = "A-1002"
	events          = "I-1003"
	stationPromo    = "I-1005" // FY2026 Mobility China
	launchFilmDates = "'Launch film' runs 01 Mar – 31 May 2026, but 'Retail events' runs 01 Jun – 31 Dec 2026. " +
		"Change the activity's dates first, or pick another initiative."
)

// destinationText writes a destination as "<initiative>: move <problem or ok>, copy <problem or ok>".
func destinationText(d activities.Destination) string {
	problem := func(p *string) string {
		if p == nil {
			return "ok"
		}
		return *p
	}
	return d.Campaign.Name + " › " + d.Initiative.Name + ": move " + problem(d.MoveProblem) + ", copy " + problem(d.CopyProblem)
}

// Every initiative of the plan is listed, grouped by campaign in tree order, each saying why the activity
// can't go there: its own initiative takes no move; dates that do not fit take neither; a name already there
// takes no move, while a copy gets a name of its own (FR-029, FR-030).
func TestDestinations_ReasonsPerTarget_FR029_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.create(t, printAndOutdoor, activities.Input{Name: "LAUNCH film", ActivityTypeID: eventType})

	destinations, err := f.service.Destinations(t.Context(), priya, launchFilm)

	if err != nil {
		t.Fatalf("Destinations: %v", err)
	}
	var got []string
	for _, d := range destinations {
		got = append(got, destinationText(d))
	}
	want := []string{
		"Helix launch › Digital & social: move It's already in this initiative., copy ok",
		"Helix launch › Print & outdoor: move 'Print & outdoor' already has an activity called 'LAUNCH film'. " +
			"Rename the activity first, or pick another initiative., copy ok",
		"Rimula fleet › Events: move ok, copy ok",
		"Rimula fleet › Retail events: move " + launchFilmDates + ", copy " + launchFilmDates,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("destinations =\n%q\nwant\n%q", got, want)
	}
	if first := destinations[0]; !first.Current || first.Initiative.ID != digitalAndSocial || first.StartDate.String() != "2026-01-01" ||
		first.Campaign.Type != "campaign" || first.Initiative.Type != "initiative" {
		t.Errorf("first destination = %+v, want Digital & social, current, with its dates", first)
	}
	_, err = f.service.Destinations(t.Context(), priya, "A-9999")
	assertError(t, err, errs.NotFound("This activity no longer exists — someone may have deleted it."))
}

// A move goes only to another initiative of the same plan, whose dates contain the activity's and which has
// no activity with its name; every refusal says why and what to do, and nothing moves (FR-029, US5
// scenario 1).
func TestMove_Refusals_FR029(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.create(t, printAndOutdoor, activities.Input{Name: "LAUNCH film", ActivityTypeID: eventType})
	move := func(initiativeID string, version *int) error {
		_, err := f.service.Move(t.Context(), priya, launchFilm, activities.MoveInput{InitiativeID: initiativeID, Version: version})
		return err
	}
	one := 1

	chooseAnother := problems("initiativeId", "Choose another initiative of this plan.")
	assertError(t, move(stationPromo, &one), chooseAnother)
	assertError(t, move(digitalAndSocial, &one), chooseAnother)
	assertError(t, move("I-9999", &one), chooseAnother)
	assertError(t, move(retailEvents, &one), problems("initiativeId", launchFilmDates))
	assertError(t, move(printAndOutdoor, nil), problems(
		"initiativeId", "'Print & outdoor' already has an activity called 'LAUNCH film'. Rename the activity first, or pick another initiative.",
		"version", "Reload the page and try again."))
	if stored, _ := f.stored(t, launchFilm); stored.InitiativeID != digitalAndSocial || stored.Version != 1 {
		t.Errorf("stored = in %s, version %d; want it unmoved", stored.InitiativeID, stored.Version)
	}

	f.clock.Advance(time.Hour)
	f.mustEdit(t, launchFilm, func(u *activities.Update) { u.Description = "Edited." })
	_, err := f.service.Move(t.Context(), daniel, launchFilm, activities.MoveInput{InitiativeID: events, Version: &one})
	assertError(t, err, errs.VersionConflict("activity", "Priya Raman", start.Add(time.Hour)))
}

// A moved activity keeps its status, values and line items — moving restarts no approval — and one "moved"
// entry names the old and the new initiative with their campaigns (FR-029).
func TestMove_KeepsStatusAndLogsParent_FR029(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	before, _ := f.stored(t, socialAlwaysOn)
	f.clock.Advance(time.Hour)
	version := before.Version

	moved, err := f.service.Move(t.Context(), daniel, socialAlwaysOn, activities.MoveInput{InitiativeID: printAndOutdoor, Version: &version})

	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	after, _ := f.stored(t, socialAlwaysOn)
	want := before
	want.InitiativeID, want.Meta = printAndOutdoor, after.Meta
	if !reflect.DeepEqual(after, want) || after.Version != before.Version+1 || after.UpdatedBy != daniel {
		t.Errorf("stored after the move =\n%+v\nwant every value of\n%+v\nwith a new version by Daniel", after, want)
	}
	if moved.Status != activities.StatusApproved || moved.Initiative.ID != printAndOutdoor || len(f.hooks.restarted) != 0 {
		t.Errorf("view = %s in %s, restarts %v; want Approved in Print & outdoor, no restart", moved.Status, moved.Initiative.ID, f.hooks.restarted)
	}
	entry := f.entries(t)[0]
	if entry.Action != history.ActionMoved || entry.RecordID != socialAlwaysOn || len(entry.Changes) != 1 {
		t.Fatalf("newest entry = %+v, want one moved entry", entry)
	}
	c := entry.Changes[0]
	var oldParent, newParent errs.RecordRef
	if err := json.Unmarshal(c.Old, &oldParent); err != nil {
		t.Fatalf("old value %s: %v", c.Old, err)
	}
	if err := json.Unmarshal(c.New, &newParent); err != nil {
		t.Fatalf("new value %s: %v", c.New, err)
	}
	if c.Field != "parent" || c.Label != "Initiative" || c.OldText != "Helix launch › Digital & social" ||
		c.NewText != "Helix launch › Print & outdoor" ||
		oldParent != (errs.RecordRef{Type: "initiative", ID: digitalAndSocial, Name: "Digital & social"}) ||
		newParent != (errs.RecordRef{Type: "initiative", ID: printAndOutdoor, Name: "Print & outdoor"}) {
		t.Errorf("change = %+v, want the parent from Digital & social to Print & outdoor", c)
	}
}

// A copy is named "<name> (copy)", " (copy 2)", … — the first name free in its initiative (FR-030).
func TestCopy_Names_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	copyInto := func(initiativeID string) string {
		view, err := f.service.Copy(t.Context(), priya, launchFilm, activities.CopyInput{InitiativeID: initiativeID})
		if err != nil {
			t.Fatalf("Copy into %s: %v", initiativeID, err)
		}
		return view.Name
	}

	got := []string{copyInto(digitalAndSocial), copyInto(digitalAndSocial), copyInto(printAndOutdoor)}

	if want := []string{"Launch film (copy)", "Launch film (copy 2)", "Launch film (copy)"}; !reflect.DeepEqual(got, want) {
		t.Errorf("copy names = %q, want %q", got, want)
	}
}

// A copy of an Approved activity is Created, with a new ID and every value of the original, and copies of
// its line items; one "copied" entry on the copy says where it came from and how many line items came along
// (FR-030, R13, D-110, US5 scenario 2).
func TestCopy_CreatedWithLineItems_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	registerLineItemKind(t, f.history)
	f.lineItems.facts[socialAlwaysOn] = activities.LineItemFacts{Count: 2}
	original, _ := f.stored(t, socialAlwaysOn)
	f.clock.Advance(time.Hour)

	copied, err := f.service.Copy(t.Context(), daniel, socialAlwaysOn, activities.CopyInput{InitiativeID: printAndOutdoor})

	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	stored, _ := f.stored(t, copied.ID)
	want := original
	want.ID, want.InitiativeID, want.Name = copied.ID, printAndOutdoor, "Social always-on (copy)"
	want.Status, want.Meta = activities.StatusCreated, record.NewMeta(daniel, start.Add(time.Hour))
	if copied.ID == socialAlwaysOn || !reflect.DeepEqual(stored, want) {
		t.Errorf("stored copy =\n%+v\nwant\n%+v", stored, want)
	}
	if want := []string{socialAlwaysOn + " → " + copied.ID + " +0"}; !reflect.DeepEqual(f.lineItems.copies, want) {
		t.Errorf("line-item copies = %q, want %q", f.lineItems.copies, want)
	}
	if source, _ := f.stored(t, socialAlwaysOn); source.Status != activities.StatusApproved {
		t.Errorf("original is %s, want it still Approved", source.Status)
	}
	entry := f.entries(t)[0]
	if entry.Action != history.ActionCopied || entry.RecordID != copied.ID || entry.Actor != daniel ||
		entry.Details != "Copied from 'Social always-on' (A-1002), with 2 line items" {
		t.Errorf("newest entry = %s on %s by %s, %q; want the copy's one copied entry", entry.Action, entry.RecordID, entry.Actor.Name, entry.Details)
	}
}

// A copy goes only into an initiative of the same plan whose dates contain the activity's; into its own
// initiative it always fits (FR-030).
func TestCopy_TargetDatesMustFit_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, dates := f.service.Copy(t.Context(), priya, launchFilm, activities.CopyInput{InitiativeID: retailEvents})
	_, otherPlan := f.service.Copy(t.Context(), priya, launchFilm, activities.CopyInput{InitiativeID: stationPromo})
	_, gone := f.service.Copy(t.Context(), priya, "A-9999", activities.CopyInput{InitiativeID: digitalAndSocial})

	assertError(t, dates, problems("initiativeId", launchFilmDates))
	assertError(t, otherPlan, problems("initiativeId", "Choose another initiative of this plan."))
	assertError(t, gone, errs.NotFound("This activity no longer exists — someone may have deleted it."))
	if copied, err := f.service.Copy(t.Context(), priya, launchFilm, activities.CopyInput{InitiativeID: events}); err != nil ||
		copied.Initiative.ID != events {
		t.Errorf("Copy into Events = %+v, %v; want the copy there", copied.Initiative, err)
	}
}

// registerLineItemKind registers a history kind "lineItem" under the activity without records, so the log
// can name line-item counts.
func registerLineItemKind(t *testing.T, hist *history.Service) {
	t.Helper()
	err := hist.Register(history.Kind{
		Name: "lineItem", Singular: "line item", Plural: "line items", Parent: "activity",
		Snapshot: func(context.Context, string) (history.Snapshot, bool, error) { return history.Snapshot{}, false, nil },
		ChildIDs: func(context.Context, string) ([]string, error) { return nil, nil },
		Remove:   func(context.Context, string) error { return nil },
		Put:      func(context.Context, history.Snapshot) error { return nil },
	})
	if err != nil {
		t.Fatalf("Register(lineItem): %v", err)
	}
}
