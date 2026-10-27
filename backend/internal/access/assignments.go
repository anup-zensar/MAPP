package access

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// personKind is the history kind of people (feature 003): a change of a person's roles is logged on the
// person (D-233).
const personKind = "person"

// rolesField is the field of a person's log entry that holds their roles (data model §3).
const rolesField = "roles"

// ListPeopleAccess returns everyone with their roles, scopes and teams for the People tab (FR-017): active
// people first, then by name.
func (s *Service) ListPeopleAccess(ctx context.Context, a actor.Actor) ([]PersonRoles, error) {
	w, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	everyone := make([]Person, 0, len(w.people))
	for _, p := range w.people {
		everyone = append(everyone, p)
	}
	slices.SortFunc(everyone, func(x, y Person) int {
		if x.Ref.Active != y.Ref.Active {
			if x.Ref.Active {
				return -1
			}
			return 1
		}
		return cmp.Or(compareNames(x.Ref.Name, y.Ref.Name), strings.Compare(x.Ref.ID, y.Ref.ID))
	})
	list := make([]PersonRoles, 0, len(everyone))
	for _, p := range everyone {
		list = append(list, w.personRoles(p))
	}
	return list, nil
}

// PersonRoles returns the person personID with their roles and what the roles allow together (FR-017).
func (s *Service) PersonRoles(ctx context.Context, a actor.Actor, personID string) (PersonRoles, error) {
	w, err := s.read(ctx)
	if err != nil {
		return PersonRoles{}, err
	}
	p, known := w.people[personID]
	if !known {
		return PersonRoles{}, errs.NotFound(msgPersonGone)
	}
	return w.personRoles(p), nil
}

// SetPersonRoles replaces the roles of the person personID with the complete list update sends (FR-017,
// D-233), with the person's version: every problem is reported at once, an out-of-date version is refused
// with version_conflict, and no change may leave MAPP without an active administrator (409
// not_allowed, FR-021). A change gives the person a new version and one "changed" entry "Roles: old → new"
// (FR-037); assignment rows are created, changed or deleted outright to match. It applies to the person's
// next request, without signing in again (FR-020). A list that changes nothing writes nothing.
func (s *Service) SetPersonRoles(ctx context.Context, a actor.Actor, personID string, update PersonRolesUpdate) (PersonRoles, error) {
	var result PersonRoles
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		w, err := s.read(ctx)
		if err != nil {
			return err
		}
		p, known := w.people[personID]
		if !known {
			return errs.NotFound(msgPersonGone)
		}
		var problems errs.Problems
		wanted := w.checkAssignments(&problems, personID, update.Roles)
		if update.Version == nil || *update.Version < 1 {
			problems.Add("version", msgReload)
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if err := s.directory.CheckPersonVersion(ctx, personID, *update.Version); err != nil {
			return err
		}

		before := w.assignmentsOf(personID)
		oldText, newText := w.rolesText(personID, before), w.rolesText(personID, wanted)
		if sameAssignments(before, wanted) {
			result = w.personRoles(p)
			return nil
		}
		if err := w.checkAdministratorLeft(personID, wanted, p.Ref.Active); err != nil {
			return err
		}
		if err := s.replaceAssignments(ctx, a, before, wanted); err != nil {
			return err
		}
		if err := s.logRoles(ctx, a, p, oldText, newText); err != nil {
			return err
		}
		after, err := s.read(ctx)
		if err != nil {
			return err
		}
		result = after.personRoles(after.people[personID])
		return nil
	})
	if err != nil {
		return PersonRoles{}, err
	}
	return result, nil
}

// CheckRolesImpact says what saving update would cause, without saving (FR-022): the person is an approver in
// active approval workflows and would no longer hold "Approve or decline when named as an approver"
// (message 18), or the administrator a removes their own Administrator role (message 19). The list is
// checked like SetPersonRoles's, except for the version.
func (s *Service) CheckRolesImpact(ctx context.Context, a actor.Actor, personID string, update PersonRolesUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		w, err := s.read(ctx)
		if err != nil {
			return err
		}
		p, known := w.people[personID]
		if !known {
			return errs.NotFound(msgPersonGone)
		}
		var problems errs.Problems
		wanted := w.checkAssignments(&problems, personID, update.Roles)
		if err := problems.Err(); err != nil {
			return err
		}
		now, later := w.grantsOf(personID, w.assignmentsOf(personID)), w.grantsOf(personID, wanted)
		if now.Has(authz.ApprovalsDecide) && !later.Has(authz.ApprovalsDecide) && s.approvals != nil {
			workflows, waiting, err := s.approvals.ApproverIn(ctx, personID)
			if err != nil {
				return err
			}
			if workflows > 0 {
				impact.Warnings = append(impact.Warnings, ImpactWarning{
					Kind:    ImpactApproverCannotApprove,
					Message: fmt.Sprintf(msgApproverCannotApprove, p.Ref.Name, workflowCount(workflows), waitingCount(waiting)),
					Count:   waiting,
				})
			}
		}
		if personID == a.ID && now.IsAdministrator() && !later.IsAdministrator() {
			impact.Warnings = append(impact.Warnings, ImpactWarning{
				Kind: ImpactRemovesOwnAdministration, Message: msgRemovesOwnAdministration, Count: 1,
			})
		}
		return nil
	})
	if err != nil {
		return ChangeImpact{}, err
	}
	return impact, nil
}

// workflowCount says in how many workflows: "1 approval workflow", "2 approval workflows".
func workflowCount(n int) string {
	if n == 1 {
		return "1 approval workflow"
	}
	return fmt.Sprintf("%d approval workflows", n)
}

// waitingCount says how many activities wait: "has 1 activity waiting", "has 2 activities waiting", "has no
// activities waiting".
func waitingCount(n int) string {
	switch n {
	case 0:
		return "has no activities waiting"
	case 1:
		return "has 1 activity waiting"
	}
	return fmt.Sprintf("has %d activities waiting", n)
}

// replaceAssignments makes the stored assignments of a person match wanted, inside the caller's Write:
// new roles get a new assignment, changed scopes a new version, and roles no longer held are deleted
// outright (they are links; the log keeps the old value, D-233).
func (s *Service) replaceAssignments(ctx context.Context, a actor.Actor, before, wanted []Assignment) error {
	for _, want := range wanted {
		i := slices.IndexFunc(before, func(b Assignment) bool { return b.RoleID == want.RoleID })
		switch {
		case i < 0:
			id, err := s.ids.Next(ctx, ids.PrefixRoleAssignment)
			if err != nil {
				return err
			}
			want.ID, want.Meta = id, record.NewMeta(a, s.now())
		case before[i].ScopeType == want.ScopeType && slices.Equal(before[i].TeamIDs, want.TeamIDs):
			continue
		default:
			want.Touch(a, s.now())
		}
		if err := s.repo.PutAssignment(ctx, want); err != nil {
			return err
		}
	}
	for _, b := range before {
		if !slices.ContainsFunc(wanted, func(want Assignment) bool { return want.RoleID == b.RoleID }) {
			if err := s.repo.DeleteAssignment(ctx, b.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// logRoles gives the person a new version and logs the change of their roles on them, inside the caller's
// Write (data model §3): "Roles: <old> → <new>".
func (s *Service) logRoles(ctx context.Context, a actor.Actor, p Person, oldText, newText string) error {
	if _, err := s.directory.TouchPerson(ctx, a, p.Ref.ID); err != nil {
		return err
	}
	return s.logRolesEntry(ctx, a, p, oldText, newText)
}

// logRolesEntry logs the change of a person's roles on them, inside the caller's Write, without a new
// version of the person: for the role a new person gets in the save that creates them (FR-023).
func (s *Service) logRolesEntry(ctx context.Context, a actor.Actor, p Person, oldText, newText string) error {
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: personKind, RecordID: p.Ref.ID, RecordName: p.Ref.Name, Action: history.ActionChanged,
		Changes: []history.Change{{Field: rolesField, Old: encode(oldText), New: encode(newText)}},
	})
	return err
}
