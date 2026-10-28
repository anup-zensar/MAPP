package seed

import (
	"context"

	"mapp/internal/people"
	"mapp/internal/platform/ids"
)

// demoTeams are the teams of 003 data model §7, T-1001 … T-1003, with their members in membership order
// (TM-1001 … TM-1009; Lea Santos joins in 032).
var demoTeams = []struct {
	name, colour, description string
	members                   []string
}{
	{"Lubricants Europe", "blue", "Marketing for lubricants across Europe.", []string{"Priya Raman", "Daniel Okoro", "Sara Novak"}},
	{"Mobility China", "violet", "Mobility and retail marketing in China.", []string{"Mei Lin", "Carlos Mendes", "Sara Novak"}},
	{globalMarketingOps, "teal", "Marketing operations and Settings for every business.", []string{alexMorgan, carlosMendes, leaSantos}},
}

func (d *demo) addTeams(ctx context.Context) error {
	repo := d.stores.People
	d.teams = map[string]string{}
	for _, t := range demoTeams {
		teamID, err := d.next(ctx, ids.PrefixTeam)
		if err != nil {
			return err
		}
		team := people.Team{ID: teamID, Name: t.name, Description: t.description, Colour: t.colour, Active: true, Meta: d.meta()}
		if err := repo.PutTeam(ctx, team); err != nil {
			return err
		}
		d.teams[t.name] = teamID
		for _, member := range t.members {
			personID, err := find(d.people, "person", member)
			if err != nil {
				return err
			}
			membershipID, err := d.next(ctx, ids.PrefixTeamMembership)
			if err != nil {
				return err
			}
			membership := people.Membership{ID: membershipID, TeamID: teamID, PersonID: personID, Meta: d.meta()}
			if err := repo.PutMembership(ctx, membership); err != nil {
				return err
			}
		}
	}
	return nil
}
