package campaign

import (
	"testing"
)

// New initiative dates that would leave activities outside are refused, naming each; an ongoing activity
// has only its start checked, an undated one is ignored (FR-007).
func TestUpdateInitiative_DatesLeavingActivitiesOutside_Refused_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social", StartDate: day("2026-01-01"), EndDate: day("2026-12-31")})
	f.activities.add(
		ActivitySummary{ID: "A-1001", InitiativeID: i.ID, Name: "Launch film", StartDate: day("2026-03-01"), EndDate: day("2026-06-30")},
		ActivitySummary{ID: "A-1002", InitiativeID: i.ID, Name: "Always-on search", StartDate: day("2026-02-15"), Ongoing: true},
		ActivitySummary{ID: "A-1003", InitiativeID: i.ID, Name: "Undated idea"},
	)

	from := initiativeUpdateOf(i, 1)
	from.StartDate = day("2026-03-01")
	_, err := f.service.UpdateInitiative(t.Context(), priya, i.ID, from)
	assertError(t, err, problems("startDate",
		"Always-on search (ongoing from 15 Feb 2026) would be outside these dates. Change its dates first, or choose dates that include it."))

	narrow := initiativeUpdateOf(i, 1)
	narrow.StartDate, narrow.EndDate = day("2026-04-01"), day("2026-05-31")
	_, err = f.service.UpdateInitiative(t.Context(), priya, i.ID, narrow)
	assertError(t, err, problems("startDate",
		"These activities would be outside these dates: Launch film (01 Mar – 30 Jun 2026), Always-on search (ongoing from 15 Feb 2026). "+
			"Change their dates first, or choose dates that include them."))

	fits := initiativeUpdateOf(i, 1)
	fits.StartDate, fits.EndDate = day("2026-02-01"), day("2026-06-30")
	if _, err := f.service.UpdateInitiative(t.Context(), priya, i.ID, fits); err != nil {
		t.Errorf("dates that hold every activity: %v, want them saved", err)
	}
}

// New initiative dates outside the campaign's are refused (FR-006, R3).
func TestUpdateInitiative_OutsideCampaign_Refused_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch", StartDate: day("2026-03-01"), EndDate: day("2026-12-31")})
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social", StartDate: day("2026-03-01"), EndDate: day("2026-11-30")})
	update := initiativeUpdateOf(i, 1)
	update.StartDate = day("2026-02-01")

	_, err := f.service.UpdateInitiative(t.Context(), priya, i.ID, update)

	assertError(t, err, problems("startDate", "Dates must be inside the campaign's dates (01 Mar 2026 – 31 Dec 2026)."))
}

// A saved initiative change gets a new version and one "changed" entry; its name stays unique in the
// campaign (FR-005, FR-030).
func TestUpdateInitiative_SavedAndLogged_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	c := f.createCampaign(t, priya, "P-1001", CampaignInput{Name: "Helix launch"})
	f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Print & outdoor"})
	i := f.createInitiative(t, priya, c.ID, InitiativeInput{Name: "Digital & social"})

	taken := initiativeUpdateOf(i, 1)
	taken.Name = "print & OUTDOOR"
	_, err := f.service.UpdateInitiative(t.Context(), priya, i.ID, taken)
	assertError(t, err, problems("name", "'Helix launch' already has an initiative called 'Print & outdoor'."))

	update := initiativeUpdateOf(i, 1)
	update.Budget = input("125000.50")
	view, err := f.service.UpdateInitiative(t.Context(), daniel, i.ID, update)
	if err != nil || view.Version != 2 || amountText(view.Budget) != "125000.50" {
		t.Fatalf("UpdateInitiative = %+v, %v", view, err)
	}
	entries := f.log(t, initiativeKind, i.ID)
	if len(entries) != 2 || len(entries[0].Changes) != 1 || entries[0].Changes[0].NewText != "125,000.50 USD" {
		t.Errorf("entries = %+v", entries)
	}
}
