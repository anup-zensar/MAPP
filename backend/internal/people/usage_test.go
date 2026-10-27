package people

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"testing"
)

// Other features name owners, approvers and members with PersonRef, read when they answer, so a rename
// shows everywhere; an id that is no person is left out instead of failing the whole answer.
func TestPersonRefs_UnknownIdsLeftOut(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	mei := f.addPerson(t, "Mei Lin", "", false)

	got, err := f.service.PersonRefs(t.Context(), []string{priya.ID, "U-9999", mei.ID})
	if err != nil {
		t.Fatalf("PersonRefs: %v", err)
	}

	want := map[string]PersonRef{
		priya.ID: {ID: priya.ID, Name: "Priya Raman", Initials: "PR", JobTitle: "Marketer", Active: true},
		mei.ID:   {ID: mei.ID, Name: "Mei Lin", Initials: "ML", JobTitle: "", Active: false},
	}
	if !maps.Equal(got, want) {
		t.Errorf("PersonRefs = %+v, want %+v", got, want)
	}
}

// Plans and other features name teams with TeamRef, with the colour of their pill and whether they are
// inactive (D-065, FR-014).
func TestTeamRefs(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true)
	china := f.addTeam(t, "Mobility China", "violet", false)

	got, err := f.service.TeamRefs(t.Context(), []string{china.ID, "T-9999", europe.ID})
	if err != nil {
		t.Fatalf("TeamRefs: %v", err)
	}

	want := map[string]TeamRef{
		europe.ID: {ID: europe.ID, Name: "Lubricants Europe", Colour: "blue", Active: true},
		china.ID:  {ID: china.ID, Name: "Mobility China", Colour: "violet", Active: false},
	}
	if !maps.Equal(got, want) {
		t.Errorf("TeamRefs = %+v, want %+v", got, want)
	}
}

// Feature 004 checks a plan's owner against the members of its teams: every member counts, active or not;
// a team without members has an empty list, an unknown team none.
func TestMemberIDs(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	mei := f.addPerson(t, "Mei Lin", "", false)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, daniel, priya, mei)
	empty := f.addTeam(t, "Empty", "teal", true)

	got, err := f.service.MemberIDs(t.Context(), []string{europe.ID, empty.ID, "T-9999"})
	if err != nil {
		t.Fatalf("MemberIDs: %v", err)
	}

	want := map[string][]string{europe.ID: {priya.ID, daniel.ID, mei.ID}, empty.ID: {}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MemberIDs = %v, want %v", got, want)
	}
}

// R1 (feature 004): an owner must belong to one of the plan's teams. An inactive team still counts,
// because plans keep it and its members (FR-014).
func TestIsMemberOfAny_InactiveTeamCounts_FR014(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", false, priya)
	china := f.addTeam(t, "Mobility China", "violet", true)

	tests := []struct {
		name    string
		teamIDs []string
		want    bool
	}{
		{"her inactive team", []string{europe.ID}, true},
		{"one of two teams", []string{china.ID, europe.ID}, true},
		{"another team", []string{china.ID}, false},
		{"no team", nil, false},
	}
	for _, tt := range tests {
		got, err := f.service.IsMemberOfAny(t.Context(), priya.ID, tt.teamIDs)
		if err != nil || got != tt.want {
			t.Errorf("%s: IsMemberOfAny = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}

// "My teams" of other features is every team of the signed-in person, active or not (FR-026).
func TestTeamIDsOf_FR026(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	sara := f.addPerson(t, "Sara Novak", "Local finance advisor", true)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	europe := f.addTeam(t, "Lubricants Europe", "blue", true, sara)
	f.addTeam(t, "Global Marketing Ops", "teal", true, alex)
	china := f.addTeam(t, "Mobility China", "violet", false, sara)

	got, err := f.service.TeamIDsOf(t.Context(), sara.ID)
	if err != nil || !slices.Equal(got, []string{europe.ID, china.ID}) {
		t.Errorf("TeamIDsOf(Sara) = %v, %v; want [%s %s]", got, err, europe.ID, china.ID)
	}
	none, err := f.service.TeamIDsOf(t.Context(), "U-9999")
	if err != nil || none == nil || len(none) != 0 {
		t.Errorf("TeamIDsOf(nobody) = %#v, %v; want an empty list", none, err)
	}
}

// The APIs of 004–012 convert USD amounts with the person's display currency and its current rate
// (D-061). A display currency set inactive later stays in use (D-063); a code no longer known falls back to
// US dollars; everyone who never chose one sees USD (FR-023).
func TestDisplayRateOf_EURAndFallback_FR023(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		chosen     string // "" = no preferences saved
		currencies []Currency
		want       DisplayRate
	}{
		{"EUR", "EUR", []Currency{usd, {Code: "EUR", Name: "Euro", Rate: "0.92", Active: true}}, DisplayRate{"EUR", "0.92"}},
		{"inactive EUR kept", "EUR", []Currency{usd, {Code: "EUR", Name: "Euro", Rate: "0.92", Active: false}}, DisplayRate{"EUR", "0.92"}},
		{"unknown code", "XYZ", []Currency{usd}, DisplayRate{"USD", "1"}},
		{"never chosen", "", []Currency{usd}, DisplayRate{"USD", "1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			priya := f.addPerson(t, "Priya Raman", "Marketer", true)
			f.service.UseCurrencies(func(context.Context) ([]Currency, error) { return tt.currencies, nil })
			if tt.chosen != "" {
				f.savePreferences(t, Preferences{PersonID: priya.ID, Theme: "system", DisplayCurrency: tt.chosen})
			}

			got, err := f.service.DisplayRateOf(t.Context(), priya.ID)

			if err != nil || got != tt.want {
				t.Errorf("DisplayRateOf = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

// Feature 002 says which currencies people show amounts in; a person with no saved choice, or a record
// saved by feature 001 without one, shows USD.
func TestPeopleByDisplayCurrency_EmptyIsUSD(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	alex := f.addPerson(t, "Alex Morgan", "", true)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	daniel := f.addPerson(t, "Daniel Okoro", "", true)
	f.savePreferences(t, Preferences{PersonID: priya.ID, Theme: "dark", DisplayCurrency: "EUR"})
	f.savePreferences(t, Preferences{PersonID: daniel.ID, Theme: "dark"}) // feature 001's record

	got, err := f.service.PeopleByDisplayCurrency(t.Context())
	if err != nil {
		t.Fatalf("PeopleByDisplayCurrency: %v", err)
	}

	want := map[string][]string{"USD": {alex.ID, daniel.ID}, "EUR": {priya.ID}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PeopleByDisplayCurrency = %v, want %v", got, want)
	}
}

// Until feature 002 is wired, US dollars are the only currency (spec Assumptions), and the service uses
// them without being told.
func TestOnlyUSD(t *testing.T) {
	t.Parallel()
	got, err := OnlyUSD(t.Context())
	if err != nil || !slices.Equal(got, []Currency{usd}) {
		t.Errorf("OnlyUSD = %+v, %v; want %+v", got, err, []Currency{usd})
	}

	f := newFixture(t)
	priya := f.addPerson(t, "Priya Raman", "Marketer", true)
	f.savePreferences(t, Preferences{PersonID: priya.ID, Theme: "system", DisplayCurrency: "EUR"})
	if rate, err := f.service.DisplayRateOf(t.Context(), priya.ID); err != nil || rate != (DisplayRate{"USD", "1"}) {
		t.Errorf("DisplayRateOf without currencies = %+v, %v; want USD 1", rate, err)
	}
}

// usd is the US dollar as OnlyUSD gives it.
var usd = Currency{Code: "USD", Name: "US dollar", Rate: "1", Active: true}

// savePreferences stores prefs through the repository.
func (f *fixture) savePreferences(t *testing.T, prefs Preferences) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		return NewJSONRepository().PutPreferences(ctx, prefs)
	})
	if err != nil {
		t.Fatalf("PutPreferences: %v", err)
	}
}
