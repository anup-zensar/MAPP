package app

import (
	"context"
	"slices"

	"mapp/internal/access"
	"mapp/internal/approvals"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

// accessDirectory implements access.Directory (feature 014) over the people service (003): everyone with
// their teams, every team, and the person's version, which a change of their roles raises (D-233).
type accessDirectory struct {
	people *people.Service
}

// The compiler checks here that accessDirectory implements access.Directory.
var _ access.Directory = accessDirectory{}

// People returns everyone, active or not, with their teams and version.
func (d accessDirectory) People(ctx context.Context) ([]access.Person, error) {
	everyone, err := d.people.EveryoneWithTeams(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]access.Person, 0, len(everyone))
	for _, p := range everyone {
		ref := p.Person.Ref()
		list = append(list, access.Person{
			Ref:   authz.PersonRef{ID: ref.ID, Name: ref.Name, Initials: ref.Initials, JobTitle: ref.JobTitle, Active: ref.Active},
			Email: p.Person.Email, TeamIDs: p.TeamIDs, Version: p.Person.Version,
		})
	}
	return list, nil
}

// Teams returns every team, active or not.
func (d accessDirectory) Teams(ctx context.Context) ([]authz.TeamRef, error) {
	teams, err := d.people.AllTeams(ctx)
	if err != nil {
		return nil, err
	}
	refs := make([]authz.TeamRef, 0, len(teams))
	for _, t := range teams {
		refs = append(refs, authz.TeamRef{ID: t.ID, Name: t.Name, Colour: t.Colour, Active: t.Active})
	}
	return refs, nil
}

// CheckPersonVersion refuses a change of the person's roles made from an out-of-date copy.
func (d accessDirectory) CheckPersonVersion(ctx context.Context, personID string, version int) error {
	return d.people.CheckPersonVersion(ctx, personID, version)
}

// TouchPerson gives the person a new version for a change of their roles.
func (d accessDirectory) TouchPerson(ctx context.Context, a actor.Actor, personID string) (int, error) {
	return d.people.TouchPerson(ctx, a, personID)
}

// accessApprovals implements access.ApprovalFacts over the approval workflows and rounds of feature 008,
// for the roles impact check (FR-022).
type accessApprovals struct {
	repo approvals.Repository
}

// ApproverIn counts the active approval workflows that name the person as an approver of any level, and
// the activities whose open round waits for them at its current level.
func (f accessApprovals) ApproverIn(ctx context.Context, personID string) (workflows, waiting int, err error) {
	all, err := f.repo.ListWorkflows(ctx)
	if err != nil {
		return 0, 0, err
	}
	for _, w := range all {
		if w.Active && slices.ContainsFunc(w.Levels, func(l approvals.Level) bool { return slices.Contains(l.ApproverIDs, personID) }) {
			workflows++
		}
	}
	rounds, err := f.repo.ListRounds(ctx)
	if err != nil {
		return 0, 0, err
	}
	for _, r := range rounds {
		if r.Outcome != approvals.OutcomeOpen || r.CurrentLevel < 1 || r.CurrentLevel > len(r.Levels) {
			continue
		}
		level := r.Levels[r.CurrentLevel-1]
		if slices.ContainsFunc(level.Approvers, func(a actor.Actor) bool { return a.ID == personID }) {
			waiting++
		}
	}
	return workflows, waiting, nil
}

// peopleAccessHooks returns what people asks of access control (014): CurrentUser.access, the role for new
// people, the last-administrator guard on deactivation, and team deletes leaving Chosen teams scopes.
func peopleAccessHooks(s *access.Service) people.AccessHooks {
	return people.AccessHooks{
		Summary:       s.Summary,
		NewPerson:     s.AssignNewPerson,
		CanDeactivate: s.CheckDeactivate,
		TeamDeleted:   s.TeamDeleted,
	}
}
