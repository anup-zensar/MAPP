package seed_test

import (
	"slices"
	"testing"
)

// The teams of 003 data model §7, T-1001 … T-1003, with their colours, descriptions and the nine
// memberships TM-1001 … TM-1009 in order (FR-029, 032 FR-051).
func TestLoad_TeamsAndMemberships_FR029(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	var teams []string
	for _, team := range d.teams {
		teams = append(teams, team.ID+" "+team.Name+" "+team.Colour)
		if !team.Active || team.Description == "" {
			t.Errorf("team %s: active %v, description %q", team.ID, team.Active, team.Description)
		}
	}
	wantTeams := []string{"T-1001 Lubricants Europe blue", "T-1002 Mobility China violet", "T-1003 Global Marketing Ops teal"}
	if !slices.Equal(teams, wantTeams) {
		t.Errorf("teams = %q, want %q", teams, wantTeams)
	}

	var memberships []string
	for _, m := range d.memberships {
		memberships = append(memberships, m.ID+" "+m.TeamID+" "+m.PersonID)
		if m.Version != 1 {
			t.Errorf("membership %s has version %d, want 1", m.ID, m.Version)
		}
	}
	wantMemberships := []string{
		"TM-1001 T-1001 U-1002", "TM-1002 T-1001 U-1003", "TM-1003 T-1001 U-1006",
		"TM-1004 T-1002 U-1004", "TM-1005 T-1002 U-1005", "TM-1006 T-1002 U-1006",
		"TM-1007 T-1003 U-1001", "TM-1008 T-1003 U-1005", "TM-1009 T-1003 U-1007",
	}
	if !slices.Equal(memberships, wantMemberships) {
		t.Errorf("memberships = %q, want %q", memberships, wantMemberships)
	}
}
