package people

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"mapp/internal/platform/errs"
)

// The longest name, job title and e-mail of a person, in characters (003 data model §1, FR-002).
const (
	maxPersonNameLength = 120
	maxJobTitleLength   = 120
	maxEmailLength      = 254
)

// personForm is what the person form sends: the fields of PersonInput and PersonUpdate without the version.
type personForm struct {
	Name, Email, JobTitle string
	TeamIDs               []string
	ManagerID             string // "" = no line manager (017 FR-016)
}

// storedPersonForm is the person p as their form starts: the stored name, e-mail, job title, teams and line
// manager. PersonUpdate.applyTo puts the sent fields in place.
func storedPersonForm(p Person, d directory) personForm {
	return personForm{Name: p.Name, Email: p.Email, JobTitle: p.JobTitle, TeamIDs: d.teamIDsOf(p.ID), ManagerID: p.ManagerID}
}

// checkPersonForm returns form with its texts trimmed, the e-mail in lower case and the team ids without
// repeats, and adds every problem to problems (FR-002, FR-011, 003 data model §1): the name is required with
// at most 120 characters; the e-mail is required, valid, at most 254 characters and no other person has it,
// active or not, in any letter case — the message names who has it; the job title has at most 120
// characters; every team exists, and a team the person is not in yet must be active (D-070). personID is
// the person being edited, "" on create: they keep their own e-mail and their inactive teams.
func checkPersonForm(problems *errs.Problems, d directory, personID string, form personForm) personForm {
	form.Name = checkPersonName(problems, form.Name)
	form.Email = checkEmail(problems, d, personID, form.Email)
	form.JobTitle = checkJobTitle(problems, form.JobTitle)
	form.TeamIDs = uniqueIDs(form.TeamIDs)
	checkPersonTeams(problems, d, personID, form.TeamIDs)
	form.ManagerID = checkManager(problems, d, personID, form.ManagerID)
	return form
}

// checkEmail returns email trimmed and in lower case, and adds its problem to problems.
func checkEmail(problems *errs.Problems, d directory, personID, email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	holder, taken := personWithEmail(d, email, personID)
	switch {
	case email == "":
		problems.Add("email", enterEmailMessage)
	case !validEmail(email):
		problems.Add("email", invalidEmailMessage)
	case utf8.RuneCountInString(email) > maxEmailLength:
		problems.Add("email", use254Message)
	case taken && holder.Active:
		problems.Add("email", fmt.Sprintf(emailTakenMessage, email, holder.Name))
	case taken:
		problems.Add("email", fmt.Sprintf(emailTakenInactiveMessage, email, holder.Name))
	}
	return email
}

// checkPersonTeams adds the problems of the person's teams, teamIDs without repeats, to problems.
func checkPersonTeams(problems *errs.Problems, d directory, personID string, teamIDs []string) {
	teams := d.teamIDsOf(personID)
	if slices.ContainsFunc(teamIDs, func(id string) bool { _, known := d.teams[id]; return !known }) {
		problems.Add("teamIds", unknownTeamMessage)
	}
	for _, id := range teamIDs {
		if t, known := d.teams[id]; known && !t.Active && !slices.Contains(teams, id) {
			problems.Add("teamIds", fmt.Sprintf(inactiveTeamMessage, t.Name))
		}
	}
}

// checkManager returns managerID trimmed and adds its problem to problems (017 FR-016, data-model §9): the
// line manager is optional; when chosen, a known person, not the person themselves, active when added or
// changed (a manager who became inactive may stay), and not someone who reports to the person, directly or
// through other managers — that would make a loop. personID is the person being edited, "" on create (a
// new person has nobody reporting to them yet).
func checkManager(problems *errs.Problems, d directory, personID, managerID string) string {
	managerID = strings.TrimSpace(managerID)
	if managerID == "" {
		return ""
	}
	manager, known := d.people[managerID]
	switch {
	case !known:
		problems.Add("managerId", chooseManagerMessage)
	case managerID == personID:
		problems.Add("managerId", ownManagerMessage)
	case !manager.Active && d.people[personID].ManagerID != managerID:
		problems.Add("managerId", fmt.Sprintf(inactiveManagerMessage, manager.Name))
	case personID != "" && reportsTo(d, managerID, personID):
		name := d.people[personID].Name
		problems.Add("managerId", fmt.Sprintf(managerLoopMessage, manager.Name, name, name, manager.Name))
	}
	return managerID
}

// reportsTo reports whether the person personID reaches bossID by following line managers upwards. The walk
// stops after as many steps as there are people, so data that already holds a loop cannot hang it.
func reportsTo(d directory, personID, bossID string) bool {
	current := d.people[personID].ManagerID
	for range len(d.people) {
		if current == "" {
			return false
		}
		if current == bossID {
			return true
		}
		current = d.people[current].ManagerID
	}
	return false
}

// checkPersonName returns name trimmed and adds its problem to problems: a person's name is required, with
// at most 120 characters (FR-002). People and My profile share it (FR-022).
func checkPersonName(problems *errs.Problems, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		problems.Add("name", enterNameMessage)
	} else {
		checkLength(problems, "name", name, maxPersonNameLength, use120Message)
	}
	return name
}

// checkJobTitle returns jobTitle trimmed and adds its problem to problems: a job title may be empty and has
// at most 120 characters (FR-002). People and My profile share it (FR-022).
func checkJobTitle(problems *errs.Problems, jobTitle string) string {
	jobTitle = strings.TrimSpace(jobTitle)
	checkLength(problems, "jobTitle", jobTitle, maxJobTitleLength, use120Message)
	return jobTitle
}

// personWithEmail returns the person, other than exceptID, whose e-mail is email ignoring letter case, and
// false when there is none. E-mails are stored in lower case without surrounding spaces.
func personWithEmail(d directory, email, exceptID string) (Person, bool) {
	for _, p := range d.sortedPeople() {
		if p.ID != exceptID && strings.EqualFold(p.Email, email) {
			return p, true
		}
	}
	return Person{}, false
}

// validEmail reports whether email has the form local@domain.tld (003 data model §1): exactly one "@",
// something before it, no spaces, and a "." inside the domain but not at either of its ends.
func validEmail(email string) bool {
	local, domain, found := strings.Cut(email, "@")
	return found && local != "" && !strings.Contains(domain, "@") && !strings.ContainsFunc(email, unicode.IsSpace) &&
		strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}
