package people

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
)

// The longest team name and description, in characters (003 data model §2).
const (
	maxTeamNameLength    = 80
	maxDescriptionLength = 300
)

// teamForm is what the team form sends: the fields of TeamInput and TeamUpdate without the version.
type teamForm struct {
	Name, Description, Colour string
	MemberIDs                 []string
}

// storedTeamForm is the team t as its form starts: the stored name, description, colour and members.
// TeamUpdate.applyTo puts the sent fields in place.
func storedTeamForm(t Team, d directory) teamForm {
	return teamForm{Name: t.Name, Description: t.Description, Colour: t.Colour, MemberIDs: d.memberIDsOf(t.ID)}
}

// checkTeamForm returns form with its texts trimmed and its member ids without repeats, and adds every
// problem to problems (FR-011, 003 data model §2): the name is required, has at most 80 characters and no
// other team has it, ignoring letter case; the description has at most 300 characters; the colour is one of
// the palette — or empty on create, which picks one; every member is a person, and a person who is not a
// member yet must be active (D-070). teamID is the team being edited, "" on create: it may keep its name
// and its inactive members.
func checkTeamForm(problems *errs.Problems, d directory, teamID string, form teamForm) teamForm {
	form.Name = checkTeamName(problems, d, teamID, form.Name)
	form.Description = strings.TrimSpace(form.Description)
	form.MemberIDs = uniqueIDs(form.MemberIDs)
	checkLength(problems, "description", form.Description, maxDescriptionLength, use300Message)
	if form.Colour != "" || teamID != "" {
		if !slices.ContainsFunc(Palette, func(c Colour) bool { return c.Key == form.Colour }) {
			problems.Add("colour", chooseColourMessage)
		}
	}
	checkTeamMembers(problems, d, teamID, form.MemberIDs)
	return form
}

// checkTeamName returns name trimmed, and adds its problem to problems.
func checkTeamName(problems *errs.Problems, d directory, teamID, name string) string {
	name = strings.TrimSpace(name)
	other, taken := teamNamed(d.sortedTeams(), name, teamID)
	switch {
	case name == "":
		problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(name) > maxTeamNameLength:
		problems.Add("name", use80Message)
	case taken:
		problems.Add("name", fmt.Sprintf(teamNameTakenMessage, other.Name))
	}
	return name
}

// checkTeamMembers adds the problems of the team's members, memberIDs without repeats, to problems.
func checkTeamMembers(problems *errs.Problems, d directory, teamID string, memberIDs []string) {
	members := d.memberIDsOf(teamID)
	if slices.ContainsFunc(memberIDs, func(id string) bool { _, known := d.people[id]; return !known }) {
		problems.Add("memberIds", unknownPersonMessage)
	}
	for _, id := range memberIDs {
		if p, known := d.people[id]; known && !p.Active && !slices.Contains(members, id) {
			problems.Add("memberIds", fmt.Sprintf(inactivePersonMessage, p.Name))
		}
	}
}

// leastUsedColour returns the palette colour the fewest of teams have, the first in palette order when
// several have as few (D-025, D-065).
func leastUsedColour(teams []Team) string {
	best, fewest := Palette[0].Key, len(teams)+1
	for _, c := range Palette {
		used := 0
		for _, t := range teams {
			if t.Colour == c.Key {
				used++
			}
		}
		if used < fewest {
			best, fewest = c.Key, used
		}
	}
	return best
}
