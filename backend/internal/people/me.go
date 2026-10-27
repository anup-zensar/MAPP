package people

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// CurrentUser returns the signed-in person a with their preferences: what the sidebar and the account menu
// show (FR-012), with the person's version, which My profile sends back (003 FR-021), and their teams,
// active or not, by name: "My teams" (003 FR-026). The display currency and date format of a person who
// never chose one are the defaults (003 FR-023, FR-024). A person who no longer exists must sign in again.
func (s *Service) CurrentUser(ctx context.Context, a actor.Actor) (CurrentUser, error) {
	var user CurrentUser
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		person, err := s.signedInPerson(ctx, a)
		if err != nil {
			return err
		}
		user, err = s.currentUser(ctx, person)
		return err
	})
	if err != nil {
		return CurrentUser{}, err
	}
	return user, nil
}

// UpdateProfile changes the name and job title of the signed-in person a from My profile (003 FR-021,
// FR-022), with the rules of the People form (FR-002): the name is required with at most 120 characters,
// the job title has at most 120. Every problem, a missing version included, is reported at once; an
// out-of-date version is refused with version_conflict. The e-mail is the sign-in name and is changed only in
// Settings › People (FR-021). A change gives the person a new version and one "changed" entry in the log
// (FR-028); it shows wherever they are named, because other records keep only their id. A form that changes
// nothing writes nothing. A person who no longer exists must sign in again.
func (s *Service) UpdateProfile(ctx context.Context, a actor.Actor, update ProfileUpdate) (CurrentUser, error) {
	var user CurrentUser
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		p, err := s.signedInPerson(ctx, a)
		if err != nil {
			return err
		}
		var problems errs.Problems
		name := checkPersonName(&problems, update.Name)
		jobTitle := checkJobTitle(&problems, update.JobTitle)
		checkVersionSent(&problems, update.Version)
		if err = problems.Err(); err != nil {
			return err
		}
		if err = p.CheckVersion(*update.Version, personSingular); err != nil {
			return err
		}

		changed := p
		changed.Name, changed.JobTitle = name, jobTitle
		changes, err := history.Diff(p.logView(nil), changed.logView(nil)) // the teams do not change here
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			changed.Touch(a, s.now())
			if err = s.repo.Put(ctx, changed); err != nil {
				return err
			}
			_, err = s.history.Log(ctx, a, history.LogInput{
				Kind: personKind, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionChanged, Changes: changes,
			})
			if err != nil {
				return err
			}
		}
		user, err = s.currentUser(ctx, changed)
		return err
	})
	if err != nil {
		return CurrentUser{}, err
	}
	return user, nil
}

// signedInPerson returns the stored record of the signed-in person a, inside the caller's transaction. A
// person who no longer exists must sign in again.
func (s *Service) signedInPerson(ctx context.Context, a actor.Actor) (Person, error) {
	p, found, err := s.repo.Get(ctx, a.ID)
	if err != nil {
		return Person{}, err
	}
	if !found {
		return Person{}, errs.SignInRequired()
	}
	return p, nil
}

// currentUser returns p as the signed-in person, with their preferences and teams, inside the caller's
// transaction.
func (s *Service) currentUser(ctx context.Context, p Person) (CurrentUser, error) {
	prefs, err := s.preferences(ctx, p.ID)
	if err != nil {
		return CurrentUser{}, err
	}
	d, err := s.readDirectory(ctx)
	if err != nil {
		return CurrentUser{}, err
	}
	user := p.currentUser(prefs, d.teamRefs(d.teamIDsOf(p.ID)))
	if user.Access, err = s.accessOf(ctx, p.ID); err != nil {
		return CurrentUser{}, err
	}
	return user, nil
}
