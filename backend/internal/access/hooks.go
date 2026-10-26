package access

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// The methods below are what other packages ask of access control, through adapters in internal/app (plan
// "internal/access", hooks.go). Each runs inside the caller's transaction.

// AssignNewPerson gives the person personID, just created in Settings › People, the role for new people with
// scope My teams, in the save that creates them (so the person keeps version 1), and logs it on the person
// right after their "created" entry (FR-023). Nothing happens when the rule names no role.
func (s *Service) AssignNewPerson(ctx context.Context, a actor.Actor, personID string) error {
	return s.tx.Write(ctx, func(ctx context.Context) error {
		w, err := s.read(ctx)
		if err != nil {
			return err
		}
		r, known := w.roles[w.rules.NewPersonRoleID]
		p, personKnown := w.people[personID]
		if !known || !personKnown || r.administrator() {
			return nil
		}
		id, err := s.ids.Next(ctx, ids.PrefixRoleAssignment)
		if err != nil {
			return err
		}
		assignment := Assignment{ID: id, PersonID: personID, RoleID: r.ID, ScopeType: authz.ScopeMyTeams,
			TeamIDs: []string{}, Meta: record.NewMeta(a, s.now())}
		if err := s.repo.PutAssignment(ctx, assignment); err != nil {
			return err
		}
		return s.logRolesEntry(ctx, a, p, noRoleText, w.rolesText(personID, []Assignment{assignment}))
	})
}

// CheckDeactivate refuses to deactivate the person personID when they are the only active administrator
// (FR-021, message 10).
func (s *Service) CheckDeactivate(ctx context.Context, personID string) error {
	w, err := s.read(ctx)
	if err != nil {
		return err
	}
	if admins := w.administratorIDs(); len(admins) == 1 && admins[0] == personID {
		return errs.NotAllowed(fmtOnlyAdministrator(w.people[personID].Ref.Name))
	}
	return nil
}

// TeamDeleted takes the deleted team teamID out of every Chosen teams scope, in the same Write as the delete
// (data model §1.2). An assignment left with no team stays and grants nothing; the People tab flags it.
// Restoring the team does not put it back.
func (s *Service) TeamDeleted(ctx context.Context, a actor.Actor, teamID string) error {
	return s.tx.Write(ctx, func(ctx context.Context) error {
		assignments, err := s.repo.ListAssignments(ctx)
		if err != nil {
			return err
		}
		for _, assignment := range assignments {
			if !slices.Contains(assignment.TeamIDs, teamID) {
				continue
			}
			assignment.TeamIDs = slices.DeleteFunc(slices.Clone(assignment.TeamIDs), func(id string) bool { return id == teamID })
			assignment.Touch(a, s.now())
			if err := s.repo.PutAssignment(ctx, assignment); err != nil {
				return err
			}
		}
		return nil
	})
}

// CanApprove reports, for each of personIDs, whether they hold "Approve or decline when named as an
// approver" (FR-028): approval workflows refuse new approvers without it and flag approvers who lost it.
func (s *Service) CanApprove(ctx context.Context, personIDs []string) (map[string]bool, error) {
	w, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	can := map[string]bool{}
	for _, id := range personIDs {
		can[id] = w.grantsOf(id, w.assignmentsOf(id)).Has(authz.ApprovalsDecide)
	}
	return can, nil
}

// HoldersOf returns the active people who hold p — for All markets when all is true — sorted by name: the
// owners a global budget may have (FR-029) and whom features 015–017 notify.
func (s *Service) HoldersOf(ctx context.Context, p authz.Permission, all bool) ([]string, error) {
	w, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	var holders []string
	for _, ref := range w.personRefs(w.activeIDs()) {
		g := w.grantsOf(ref.ID, w.assignmentsOf(ref.ID))
		everywhere, _ := g.Teams(p)
		if g.Has(p) && (!all || everywhere) {
			holders = append(holders, ref.ID)
		}
	}
	return holders, nil
}

// RoleHolders returns the active people who hold the role roleID with a scope that covers the plan, sorted by
// name: feature 017's role approvers.
func (s *Service) RoleHolders(ctx context.Context, roleID string, plan authz.PlanFacts) ([]string, error) {
	w, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	holders := []string{}
	if _, known := w.roles[roleID]; !known {
		return holders, nil
	}
	for _, ref := range w.personRefs(w.activeIDs()) {
		mine := w.assignmentsOf(ref.ID)
		i := slices.IndexFunc(mine, func(a Assignment) bool { return a.RoleID == roleID })
		if i >= 0 && coversPlan(w, ref.ID, mine[i], plan) {
			holders = append(holders, ref.ID)
		}
	}
	return holders, nil
}

// coversPlan reports whether the scope of the assignment covers the plan (FR-005), whatever the role's
// permissions.
func coversPlan(w world, personID string, a Assignment, plan authz.PlanFacts) bool {
	switch a.ScopeType {
	case authz.ScopeAll:
		return true
	case authz.ScopeMyTeams:
		return slices.ContainsFunc(plan.TeamIDs, func(id string) bool { return slices.Contains(w.people[personID].TeamIDs, id) })
	}
	return slices.ContainsFunc(plan.TeamIDs, func(id string) bool { return slices.Contains(a.TeamIDs, id) })
}

// ApprovingRoles returns the roles that include "Approve or decline when named as an approver", sorted by
// name: what feature 017 offers as role approvers.
func (s *Service) ApprovingRoles(ctx context.Context) ([]RoleView, error) {
	w, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	views := []RoleView{}
	for _, r := range sortedRoles(w.roles) {
		if slices.Contains(r.permissions(), authz.ApprovalsDecide) {
			views = append(views, w.roleView(r))
		}
	}
	return views, nil
}

// activeIDs returns the ids of the active people.
func (w world) activeIDs() []string {
	var list []string
	for id, p := range w.people {
		if p.Ref.Active {
			list = append(list, id)
		}
	}
	return list
}

// fmtOnlyAdministrator is message 10 for the person called name.
func fmtOnlyAdministrator(name string) string {
	return fmt.Sprintf(msgOnlyAdministrator, name)
}
