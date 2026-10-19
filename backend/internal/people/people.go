package people

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
)

// The longest name, job title and e-mail of a person, in characters (003 data model §1, FR-002).
const (
	maxPersonNameLength = 120
	maxJobTitleLength   = 120
	maxEmailLength      = 254
)

// ListPeople returns everyone, active or not, with their teams, sorted by name ignoring letter case, then
// by e-mail (FR-001). It is also the data of every person picker, which filters it in the browser (FR-017,
// FR-018, D-069): inactive people are included, so that a field holding one can label them "(inactive)".
// Every signed-in person may see it (FR-027), so a is not used yet; it is where a permission check will go
// when roles arrive.
func (s *Service) ListPeople(ctx context.Context, a actor.Actor) ([]PersonView, error) {
	var views []PersonView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		d, err := s.readDirectory(ctx)
		if err != nil {
			return err
		}
		views = make([]PersonView, 0, len(d.people))
		for _, p := range d.sortedPeople() {
			views = append(views, d.personView(p))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return views, nil
}

// GetPerson returns the person id with their teams and what they own or approve (FR-004). A person who
// does not exist answers not_found; people are never deleted (FR-008), so this means a wrong id.
func (s *Service) GetPerson(ctx context.Context, a actor.Actor, id string) (PersonDetails, error) {
	var details PersonDetails
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		p, err := s.storedPerson(ctx, id)
		if err != nil {
			return err
		}
		details, err = s.details(ctx, p)
		return err
	})
	if err != nil {
		return PersonDetails{}, err
	}
	return details, nil
}

// CreatePerson adds an active person with their teams (FR-001, FR-002). Every problem of the form is
// reported at once. The e-mail is their sign-in name, so they can sign in straight away with the demo
// password: feature 001's sign-in finds them in the same table (FR-003). The log gets the person's
// "created" entry and a "changed" entry for each of their teams (FR-012, FR-028).
func (s *Service) CreatePerson(ctx context.Context, a actor.Actor, input PersonInput) (PersonDetails, error) {
	var details PersonDetails
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		d, err := s.readDirectory(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		form := checkPersonForm(&problems, d, "", personForm{input.Name, input.Email, input.JobTitle, input.TeamIDs})
		if err := problems.Err(); err != nil {
			return err
		}

		id, err := s.ids.Next(ctx, ids.PrefixPerson)
		if err != nil {
			return err
		}
		p := Person{ID: id, Name: form.Name, Email: form.Email, JobTitle: form.JobTitle, Active: true,
			Meta: record.NewMeta(a, s.now())}
		if err := s.repo.Put(ctx, p); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: personKind, RecordID: p.ID, RecordName: p.Name, Action: history.ActionCreated,
		})
		if err != nil {
			return err
		}
		add, _ := membershipDelta(nil, form.TeamIDs, func(teamID string) pair { return pair{teamID, p.ID} })
		if err := s.applyMemberships(ctx, a, p.ID, add, nil); err != nil {
			return err
		}
		details, err = s.details(ctx, p)
		return err
	})
	if err != nil {
		return PersonDetails{}, err
	}
	return details, nil
}

// UpdatePerson changes the person id with the fields update sends — name, e-mail, job title, every team
// after the change — and keeps the others (FR-002, FR-012, D-210). Every problem of the form the change
// leads to, a missing version included, is reported at once; an out-of-date version is refused with
// version_conflict (001 FR-029). Teams the person is already in may be inactive (FR-014). Removing a team
// whose plan they own is allowed: the impact check warns first (FR-013). A form that changes nothing is
// not a change: no new version, no log entry. Otherwise the person gets a new version and one "changed"
// entry with the old and new values, and every team they joined or left gets one too (D-057). The next
// sign-in uses a new e-mail (US1 scenario 4).
func (s *Service) UpdatePerson(ctx context.Context, a actor.Actor, id string, update PersonUpdate) (PersonDetails, error) {
	var details PersonDetails
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		p, err := s.storedPerson(ctx, id)
		if err != nil {
			return err
		}
		d, err := s.readDirectory(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		form := checkPersonForm(&problems, d, p.ID, update.applyTo(storedPersonForm(p, d)))
		checkVersionSent(&problems, update.Version)
		if err := problems.Err(); err != nil {
			return err
		}
		if err := p.CheckVersion(*update.Version, personSingular); err != nil {
			return err
		}

		before := d.teamIDsOf(p.ID)
		changed := p
		changed.Name, changed.Email, changed.JobTitle = form.Name, form.Email, form.JobTitle
		changes, err := history.Diff(p.logView(before), changed.logView(form.TeamIDs))
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			changed.Touch(a, s.now())
			if err := s.repo.Put(ctx, changed); err != nil {
				return err
			}
			_, err = s.history.Log(ctx, a, history.LogInput{
				Kind: personKind, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionChanged, Changes: changes,
			})
			if err != nil {
				return err
			}
			add, remove := membershipDelta(before, form.TeamIDs, func(teamID string) pair { return pair{teamID, p.ID} })
			if err := s.applyMemberships(ctx, a, p.ID, add, remove); err != nil {
				return err
			}
		}
		details, err = s.details(ctx, changed)
		return err
	})
	if err != nil {
		return PersonDetails{}, err
	}
	return details, nil
}

// PersonImpact says what saving update would affect, before it is saved, and saves nothing (FR-013,
// D-058): one ownerNeedsAttention warning when the teams the person leaves are teams of plans they own and
// they would then belong to none of those plans' teams; teams that are not sent are not left. The form is
// checked exactly like UpdatePerson's, except for the version.
func (s *Service) PersonImpact(ctx context.Context, a actor.Actor, id string, update PersonUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		p, err := s.storedPerson(ctx, id)
		if err != nil {
			return err
		}
		d, err := s.readDirectory(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		form := checkPersonForm(&problems, d, p.ID, update.applyTo(storedPersonForm(p, d)))
		if err := problems.Err(); err != nil {
			return err
		}

		var removedTeams []string
		for _, teamID := range d.teamIDsOf(p.ID) {
			if !slices.Contains(form.TeamIDs, teamID) {
				removedTeams = append(removedTeams, teamID)
			}
		}
		if len(removedTeams) == 0 {
			return nil
		}
		usesOne, usesMany := usesTheRemovedTeams, useTheRemovedTeams
		if len(removedTeams) == 1 {
			name := d.teams[removedTeams[0]].Name
			usesOne, usesMany = fmt.Sprintf(usesTeamNamed, name), fmt.Sprintf(useTeamNamed, name)
		}
		removed, after := map[string][]string{p.ID: removedTeams}, map[string][]string{p.ID: form.TeamIDs}
		impact.Warnings, err = s.ownerImpact(ctx, d, removed, after, usesOne, usesMany)
		return err
	})
	if err != nil {
		return ChangeImpact{}, err
	}
	return impact, nil
}

// SetPersonStatus deactivates (active false) or reactivates (active true) the person id: people are never
// deleted (FR-005 – FR-008). A deactivated person can no longer sign in — feature 001's sign-in check ends
// their open sessions at their next request — and is no longer offered in pickers; every record that names
// them keeps them, and so do their teams. Nobody may deactivate themselves (FR-006). A person already in
// that status is answered unchanged: no new version, no log entry.
func (s *Service) SetPersonStatus(ctx context.Context, a actor.Actor, id string, update StatusUpdate) (PersonDetails, error) {
	if err := checkStatusUpdate(update); err != nil {
		return PersonDetails{}, err
	}
	var details PersonDetails
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		p, err := s.storedPerson(ctx, id)
		if err != nil {
			return err
		}
		if p.ID == a.ID && !*update.Active {
			return errs.NotAllowed(deactivateYourselfMessage)
		}
		if err := p.CheckVersion(*update.Version, personSingular); err != nil {
			return err
		}
		if p.Active != *update.Active {
			if p, err = s.changePersonStatus(ctx, a, p, *update.Active); err != nil {
				return err
			}
		}
		details, err = s.details(ctx, p)
		return err
	})
	if err != nil {
		return PersonDetails{}, err
	}
	return details, nil
}

// changePersonStatus saves p with the status active, a new version and a "deactivated" or "reactivated"
// log entry, inside the caller's Write, and returns the person as they are stored now. Memberships are
// kept (FR-005, FR-007).
func (s *Service) changePersonStatus(ctx context.Context, a actor.Actor, p Person, active bool) (Person, error) {
	changed := p
	changed.Active = active
	changes, err := history.Diff(p.logView(nil), changed.logView(nil))
	if err != nil {
		return Person{}, err
	}
	changed.Touch(a, s.now())
	if err := s.repo.Put(ctx, changed); err != nil {
		return Person{}, err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{
		Kind: personKind, RecordID: changed.ID, RecordName: changed.Name,
		Action: history.ActionStatusChanged, Verb: statusVerb(active), Changes: changes,
	})
	return changed, err
}

// storedPerson returns the stored person id, or not_found with the message the web app shows.
func (s *Service) storedPerson(ctx context.Context, id string) (Person, error) {
	p, found, err := s.repo.Get(ctx, id)
	if err != nil {
		return Person{}, err
	}
	if !found {
		return Person{}, errs.NotFound(personGoneMessage)
	}
	return p, nil
}

// details returns p as the details, create, update and status operations answer them, inside the caller's
// transaction: with their teams as stored now and what they own or approve.
func (s *Service) details(ctx context.Context, p Person) (PersonDetails, error) {
	d, err := s.readDirectory(ctx)
	if err != nil {
		return PersonDetails{}, err
	}
	plans, err := s.allPlans(ctx)
	if err != nil {
		return PersonDetails{}, err
	}
	usage, err := s.usageOf(ctx, p.ID, plans)
	if err != nil {
		return PersonDetails{}, err
	}
	return PersonDetails{PersonView: d.personView(p), Usage: usage}, nil
}

// personForm is what the person form sends: the fields of PersonInput and PersonUpdate without the version.
type personForm struct {
	Name, Email, JobTitle string
	TeamIDs               []string
}

// storedPersonForm is the person p as their form starts: the stored name, e-mail, job title and teams.
// PersonUpdate.applyTo puts the sent fields in place.
func storedPersonForm(p Person, d directory) personForm {
	return personForm{Name: p.Name, Email: p.Email, JobTitle: p.JobTitle, TeamIDs: d.teamIDsOf(p.ID)}
}

// checkPersonForm returns form with its texts trimmed, the e-mail in lower case and the team ids without
// repeats, and adds every problem to problems (FR-002, FR-011, 003 data model §1): the name is required with
// at most 120 characters; the e-mail is required, valid, at most 254 characters and no other person has it,
// active or not, in any letter case — the message names who has it; the job title has at most 120
// characters; every team exists, and a team the person is not in yet must be active (D-070). personID is
// the person being edited, "" on create: they keep their own e-mail and their inactive teams.
func checkPersonForm(problems *errs.Problems, d directory, personID string, form personForm) personForm {
	form.Name = checkPersonName(problems, form.Name)
	form.Email = strings.ToLower(strings.TrimSpace(form.Email))
	form.TeamIDs = uniqueIDs(form.TeamIDs)

	holder, taken := personWithEmail(d, form.Email, personID)
	switch {
	case form.Email == "":
		problems.Add("email", enterEmailMessage)
	case !validEmail(form.Email):
		problems.Add("email", invalidEmailMessage)
	case utf8.RuneCountInString(form.Email) > maxEmailLength:
		problems.Add("email", use254Message)
	case taken && holder.Active:
		problems.Add("email", fmt.Sprintf(emailTakenMessage, form.Email, holder.Name))
	case taken:
		problems.Add("email", fmt.Sprintf(emailTakenInactiveMessage, form.Email, holder.Name))
	}

	form.JobTitle = checkJobTitle(problems, form.JobTitle)

	teams := d.teamIDsOf(personID)
	if slices.ContainsFunc(form.TeamIDs, func(id string) bool { _, known := d.teams[id]; return !known }) {
		problems.Add("teamIds", unknownTeamMessage)
	}
	for _, id := range form.TeamIDs {
		if t, known := d.teams[id]; known && !t.Active && !slices.Contains(teams, id) {
			problems.Add("teamIds", fmt.Sprintf(inactiveTeamMessage, t.Name))
		}
	}
	return form
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
