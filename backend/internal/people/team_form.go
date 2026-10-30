package people

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

const (
	maxTeamNameLength    = 80
	maxDescriptionLength = 300
)

type teamForm struct {
	Name, Description, Colour string
	MemberIDs                 []string
}

func storedTeamForm(t Team, d directory) teamForm {
	return teamForm{Name: t.Name, Description: t.Description, Colour: t.Colour, MemberIDs: d.memberIDsOf(t.ID)}
}

// checkTeamForm trims form, drops repeated member ids and adds every problem; a person not yet a member must
// be active (D-070). teamID is the team edited, "" on create: it may keep its name and inactive members.
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

func checkTeamName(problems *errs.Problems, d directory, teamID, name string) string {
	name = strings.TrimSpace(name)
	other, taken := teamNamed(d.sortedTeams(), name, teamID)
	switch {
	case name == "":
		problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(name) > maxTeamNameLength:
		problems.Add("name", use80Message)
	case record.CheckSingleLine(name) != "":
		problems.Add("name", record.CheckSingleLine(name))
	case taken:
		problems.Add("name", fmt.Sprintf(teamNameTakenMessage, other.Name))
	}
	return name
}

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

// leastUsedColour breaks a tie by palette order (D-025, D-065).
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
