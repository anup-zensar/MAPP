package people_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
)

// Teams and memberships are saved in their own tables and read back exactly; a team's and a person's
// memberships are found from both sides; preferences of feature 001 read with empty 003 fields (003 data
// model §2–§4).
func TestJSONRepository_TeamsMembershipsPreferences(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	repo := people.NewJSONRepository()
	meta := record.NewMeta(actor.System, time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC))
	team := people.Team{ID: "T-1001", Name: "Lubricants Europe", Description: "Marketing for lubricants across Europe.", Colour: "blue", Active: true, Meta: meta}
	other := people.Team{ID: "T-1002", Name: "Mobility China", Colour: "violet", Active: false, Meta: meta}
	memberships := []people.Membership{
		{ID: "TM-1001", TeamID: "T-1001", PersonID: "U-1002", Meta: meta},
		{ID: "TM-1002", TeamID: "T-1001", PersonID: "U-1006", Meta: meta},
		{ID: "TM-1003", TeamID: "T-1002", PersonID: "U-1006", Meta: meta},
	}
	prefs := people.Preferences{PersonID: "U-1002", Theme: "dark", DisplayCurrency: "EUR", DateFormat: "DD/MM/YYYY",
		PinnedPlanIDs: []string{"P-1002", "P-1001"}} // 018: pinned plans round-trip in the order pinned

	err = db.Write(t.Context(), func(ctx context.Context) error {
		for _, err := range []error{
			repo.PutTeam(ctx, team), repo.PutTeam(ctx, other), repo.PutPreferences(ctx, prefs),
			repo.PutMembership(ctx, memberships[0]), repo.PutMembership(ctx, memberships[1]), repo.PutMembership(ctx, memberships[2]),
			store.Put(ctx, "preferences", "U-1003", map[string]any{"personId": "U-1003", "theme": "light", "sidebarCollapsed": true}),
		} {
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = db.Read(t.Context(), func(ctx context.Context) error {
		if got, found, err := repo.GetTeam(ctx, "T-1001"); err != nil || !found || got != team {
			t.Errorf("GetTeam = %+v, %v, %v", got, found, err)
		}
		if teams, err := repo.ListTeams(ctx); err != nil || !reflect.DeepEqual(teams, []people.Team{team, other}) {
			t.Errorf("ListTeams = %+v, %v", teams, err)
		}
		if got, found, err := repo.GetMembership(ctx, "TM-1002"); err != nil || !found || got != memberships[1] {
			t.Errorf("GetMembership = %+v, %v, %v", got, found, err)
		}
		if ofTeam, err := repo.MembershipsOfTeam(ctx, "T-1001"); err != nil || !reflect.DeepEqual(ofTeam, memberships[:2]) {
			t.Errorf("MembershipsOfTeam = %+v, %v", ofTeam, err)
		}
		if ofPerson, err := repo.MembershipsOfPerson(ctx, "U-1006"); err != nil || !reflect.DeepEqual(ofPerson, memberships[1:]) {
			t.Errorf("MembershipsOfPerson = %+v, %v", ofPerson, err)
		}
		if none, err := repo.MembershipsOfPerson(ctx, "U-1001"); err != nil || none == nil || len(none) != 0 {
			t.Errorf("MembershipsOfPerson(no teams) = %#v, %v; want an empty list", none, err)
		}
		all, err := repo.ListPreferences(ctx)
		want := []people.Preferences{prefs, {PersonID: "U-1003", Theme: "light", SidebarCollapsed: true}}
		if err != nil || !reflect.DeepEqual(all, want) {
			t.Errorf("ListPreferences = %+v, %v; want %+v", all, err, want)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	err = db.Write(t.Context(), func(ctx context.Context) error {
		if err := repo.DeleteMembership(ctx, "TM-1003"); err != nil {
			return err
		}
		return repo.DeleteTeam(ctx, "T-1002")
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = db.Read(t.Context(), func(ctx context.Context) error {
		if _, found, _ := repo.GetTeam(ctx, "T-1002"); found {
			t.Error("the deleted team is still stored")
		}
		if left, _ := repo.ListMemberships(ctx); len(left) != 2 {
			t.Errorf("%d memberships after the delete, want 2", len(left))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// Other records show a person and a team by their references (PersonRef, TeamRef).
func TestRefs_PersonAndTeam(t *testing.T) {
	priya := people.Person{ID: "U-1002", Name: "Priya Raman", JobTitle: "Marketer", Active: true}
	if got := priya.Ref(); got != (people.PersonRef{ID: "U-1002", Name: "Priya Raman", Initials: "PR", JobTitle: "Marketer", Active: true}) {
		t.Errorf("Person.Ref = %+v", got)
	}
	team := people.Team{ID: "T-1001", Name: "Lubricants Europe", Colour: "blue"}
	if got := team.Ref(); got != (people.TeamRef{ID: "T-1001", Name: "Lubricants Europe", Colour: "blue", Active: false}) {
		t.Errorf("Team.Ref = %+v", got)
	}
	if len(people.Palette) != 8 || people.Palette[0].Key != "blue" || people.DateFormats[0] != people.DefaultDateFormat {
		t.Error("palette or date formats are wrong")
	}
	currencies, err := people.OnlyUSD(t.Context())
	if err != nil || len(currencies) != 1 || currencies[0] != (people.Currency{Code: "USD", Name: "US dollar", Rate: "1", Active: true}) {
		t.Errorf("OnlyUSD = %+v, %v", currencies, err)
	}
}
