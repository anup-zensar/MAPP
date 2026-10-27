package people

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// ListPeople returns everyone, active or not, with their teams, sorted by name ignoring letter case, then
// by e-mail (FR-001). It is also the data of every person picker, which filters it in the browser (FR-017,
// FR-018, D-069): inactive people are included, so that a field holding one can label them "(inactive)".
// Every signed-in person may see it (FR-027), so a is not used yet; it is where a permission check will go
// when roles arrive. Each person comes with their line manager and how many people report to them (017
// FR-016).
func (s *Service) ListPeople(ctx context.Context, a actor.Actor) ([]PersonListItem, error) {
	var views []PersonListItem
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		d, err := s.readDirectory(ctx)
		if err != nil {
			return err
		}
		views = make([]PersonListItem, 0, len(d.people))
		for _, p := range d.sortedPeople() {
			views = append(views, d.listItem(p))
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
		form := checkPersonForm(&problems, d, "", personForm{input.Name, input.Email, input.JobTitle, input.TeamIDs, input.ManagerID})
		if err := problems.Err(); err != nil {
			return err
		}

		id, err := s.ids.Next(ctx, ids.PrefixPerson)
		if err != nil {
			return err
		}
		p := Person{ID: id, Name: form.Name, Email: form.Email, JobTitle: form.JobTitle, Active: true,
			ManagerID: form.ManagerID, Meta: record.NewMeta(a, s.now())}
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
		if s.access.NewPerson != nil { // the role for new people, in the same save, logged on the person (014 FR-023)
			if err := s.access.NewPerson(ctx, a, p.ID); err != nil {
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
		changed.Name, changed.Email, changed.JobTitle, changed.ManagerID = form.Name, form.Email, form.JobTitle, form.ManagerID
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
