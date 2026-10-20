package access

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/authz"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
)

// The longest role name and description, in characters (data model §1.1).
const (
	maxRoleNameLength    = 60
	maxDescriptionLength = 300
)

// Permissions returns the fixed list of permissions grouped as the role page shows them (Table A).
func (s *Service) Permissions() PermissionCatalogue {
	catalogue := PermissionCatalogue{Groups: []PermissionGroup{}}
	for _, group := range authz.Groups {
		g := PermissionGroup{Name: group, Permissions: []PermissionInfo{}}
		for _, p := range authz.Catalogue() {
			if p.Group == group {
				g.Permissions = append(g.Permissions, PermissionInfo{Key: p.Key, Label: p.Label, Scoped: p.Scoped})
			}
		}
		catalogue.Groups = append(catalogue.Groups, g)
	}
	return catalogue
}

// ListRoles returns every role, the Administrator role first, then by name ignoring letter case (FR-018),
// each with the people who hold it.
func (s *Service) ListRoles(ctx context.Context, a actor.Actor) ([]RoleView, error) {
	w, err := s.read(ctx)
	if err != nil {
		return nil, err
	}
	roles := sortedRoles(w.roles)
	slices.SortStableFunc(roles, func(x, y Role) int {
		switch {
		case x.administrator() && !y.administrator():
			return -1
		case y.administrator() && !x.administrator():
			return 1
		}
		return 0
	})
	views := make([]RoleView, 0, len(roles))
	for _, r := range roles {
		views = append(views, w.roleView(r))
	}
	return views, nil
}

// GetRole returns the role id with its permissions, the people who hold it and its locks (FR-018).
func (s *Service) GetRole(ctx context.Context, a actor.Actor, id string) (RoleView, error) {
	w, err := s.read(ctx)
	if err != nil {
		return RoleView{}, err
	}
	r, found := w.roles[id]
	if !found {
		return RoleView{}, errs.NotFound(msgRoleGone)
	}
	return w.roleView(r), nil
}

// CreateRole creates a role from the whole form (FR-018): a unique name of at most 60 characters, a
// description of at most 300, at least one permission and a default scope. Every problem is reported at
// once. The log gets a "created" entry "with <n> permissions" (FR-037).
func (s *Service) CreateRole(ctx context.Context, a actor.Actor, input RoleInput) (RoleView, error) {
	var view RoleView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		w, err := s.read(ctx)
		if err != nil {
			return err
		}
		var problems errs.Problems
		r := Role{BuiltIn: ""}
		r.Name = checkRoleName(&problems, w, input.Name, "")
		r.Description = checkDescription(&problems, input.Description)
		r.Permissions = checkPermissions(&problems, input.Permissions)
		r.DefaultScope = checkDefaultScope(&problems, input.DefaultScope)
		if err := problems.Err(); err != nil {
			return err
		}
		if r.ID, err = s.ids.Next(ctx, ids.PrefixRole); err != nil {
			return err
		}
		r.Meta = record.NewMeta(a, s.now())
		if err := s.repo.PutRole(ctx, r); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: RoleKind, RecordID: r.ID, RecordName: r.Name, Action: history.ActionCreated,
			Details: permissionCount(len(r.Permissions)),
		}); err != nil {
			return err
		}
		w.roles[r.ID] = r
		view = w.roleView(r)
		return nil
	})
	if err != nil {
		return RoleView{}, err
	}
	return view, nil
}

// roleLog is a role as the activity log compares it (data model §3).
type roleLog struct {
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	DefaultScope authz.ScopeType    `json:"defaultScope"`
	Permissions  []authz.Permission `json:"permissions"`
}

// logView returns r as the activity log compares it.
func (r Role) logView() roleLog {
	return roleLog{Name: r.Name, Description: r.Description, DefaultScope: r.DefaultScope, Permissions: r.Permissions}
}

// UpdateRole changes the role id with the fields update sends and keeps the others (FR-018). The
// Administrator role refuses every change (409 not_allowed, message 11). Every problem, a missing version
// included, is reported at once; an out-of-date version is refused with version_conflict. A change gets one
// "changed" entry with every changed field, permissions as "Added: …; removed: …" (FR-037). A form that
// changes nothing writes nothing. The change applies to the next request of everyone who holds the role
// (FR-020).
func (s *Service) UpdateRole(ctx context.Context, a actor.Actor, id string, update RoleUpdate) (RoleView, error) {
	var view RoleView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		w, err := s.read(ctx)
		if err != nil {
			return err
		}
		r, found := w.roles[id]
		if !found {
			return errs.NotFound(msgRoleGone)
		}
		if r.administrator() {
			return errs.NotAllowed(msgAdministratorLock)
		}
		var problems errs.Problems
		changed := r
		if update.Name != nil {
			changed.Name = checkRoleName(&problems, w, *update.Name, r.ID)
		}
		if update.Description != nil {
			changed.Description = checkDescription(&problems, *update.Description)
		}
		if update.Permissions != nil {
			changed.Permissions = checkPermissions(&problems, *update.Permissions)
		}
		if update.DefaultScope != nil {
			changed.DefaultScope = checkDefaultScope(&problems, *update.DefaultScope)
		}
		if update.Version == nil || *update.Version < 1 {
			problems.Add("version", msgReload)
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if err := r.CheckVersion(*update.Version, roleSingular); err != nil {
			return err
		}
		changes, err := history.Diff(r.logView(), changed.logView())
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			changed.Touch(a, s.now())
			if err := s.repo.PutRole(ctx, changed); err != nil {
				return err
			}
			if _, err := s.history.Log(ctx, a, history.LogInput{
				Kind: RoleKind, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionChanged,
				Changes: changes, Details: permissionsDelta(r.Permissions, changed.Permissions),
			}); err != nil {
				return err
			}
		}
		w.roles[changed.ID] = changed
		view = w.roleView(changed)
		return nil
	})
	if err != nil {
		return RoleView{}, err
	}
	return view, nil
}

// DeleteRole moves the role id to the recycle bin, so it can be undone (FR-018). The Administrator role is
// never deleted (409 not_allowed); a role that someone holds is refused with in_use, naming how many people
// hold it (message 13). When the role was the role for new people, the rule becomes "no role" in the same
// save, logged on the access rules (data model §1.3).
func (s *Service) DeleteRole(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		w, err := s.read(ctx)
		if err != nil {
			return err
		}
		r, found := w.roles[id]
		if !found {
			return errs.NotFound(msgRoleGone)
		}
		if r.administrator() {
			return errs.NotAllowed(msgAdministratorLock)
		}
		if reason := w.inUseReason(r); reason != "" {
			return errs.InUse(reason)
		}
		if w.rules.NewPersonRoleID == r.ID { // logged first, while the log can still name the role
			after := w.rules
			after.NewPersonRoleID = ""
			if err := s.saveRules(ctx, a, w.rules, after); err != nil {
				return err
			}
		}
		entry, err = s.history.MoveToBin(ctx, a, RoleKind, r.ID)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// holders returns the ids of the people who hold the role, active or not.
func (w world) holders(roleID string) []string {
	var personIDs []string
	for _, a := range w.assignments {
		if a.RoleID == roleID && !slices.Contains(personIDs, a.PersonID) {
			personIDs = append(personIDs, a.PersonID)
		}
	}
	return personIDs
}

// inUseReason returns why the role cannot be deleted because people hold it (message 13); "" when nobody
// does.
func (w world) inUseReason(r Role) string {
	switch n := len(w.holders(r.ID)); n {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf(msgRoleInUseOne, r.Name)
	default:
		return fmt.Sprintf(msgRoleInUseMany, r.Name, n)
	}
}

// roleView returns r as the API returns it: every permission for the Administrator role, the people who
// hold it, and its locks (the Administrator role's fields and delete; delete of a role people hold).
func (w world) roleView(r Role) RoleView {
	locks := []Lock{}
	if r.administrator() {
		for _, field := range []string{"name", "description", "permissions", "defaultScope", "delete"} {
			locks = append(locks, Lock{Field: field, Reason: msgAdministratorLock})
		}
	} else if reason := w.inUseReason(r); reason != "" {
		locks = append(locks, Lock{Field: "delete", Reason: reason})
	}
	defaultScope := r.DefaultScope
	if r.administrator() {
		defaultScope = authz.ScopeAll
	}
	return RoleView{
		ID: r.ID, Name: r.Name, Description: r.Description, BuiltIn: r.BuiltIn, Permissions: r.permissions(),
		DefaultScope: defaultScope, People: w.personRefs(w.holders(r.ID)), Locks: locks, Meta: r.Meta,
	}
}

// checkRoleName returns name trimmed and adds its problem: required, at most 60 characters, and no other role
// than exceptID has it, ignoring letter case (message 14).
func checkRoleName(problems *errs.Problems, w world, name, exceptID string) string {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		problems.Add("name", msgEnterRoleName)
	case utf8.RuneCountInString(name) > maxRoleNameLength:
		problems.Add("name", msgUse60)
	case nameTaken(sortedRoles(w.roles), name, exceptID):
		problems.Add("name", fmt.Sprintf(msgRoleNameTaken, name))
	}
	return name
}

// checkDescription returns the description trimmed and adds its problem: at most 300 characters.
func checkDescription(problems *errs.Problems, description string) string {
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		problems.Add("description", msgUse300)
	}
	return description
}

// checkPermissions returns the permissions without repeats in Table A order and adds their problems: at
// least one (message 15), each a key of Table A.
func checkPermissions(problems *errs.Problems, keys []authz.Permission) []authz.Permission {
	for _, key := range keys {
		if _, known := authz.Lookup(string(key)); !known {
			problems.Add("permissions", fmt.Sprintf(msgUnknownPermission, key))
		}
	}
	ordered := authz.InOrder(keys)
	if len(ordered) == 0 && len(keys) == 0 {
		problems.Add("permissions", msgChoosePermission)
	}
	return ordered
}

// checkDefaultScope adds the problem of a default scope that is neither All markets nor My teams.
func checkDefaultScope(problems *errs.Problems, scope authz.ScopeType) authz.ScopeType {
	if scope != authz.ScopeAll && scope != authz.ScopeMyTeams {
		problems.Add("defaultScope", msgChooseDefaultScope)
	}
	return scope
}

// permissionCount is the details of a role's "created" entry: "with 9 permissions".
func permissionCount(n int) string {
	if n == 1 {
		return "with 1 permission"
	}
	return fmt.Sprintf("with %d permissions", n)
}

// permissionsDelta is the details of a role's "changed" entry when its permissions change: "Added: …;
// removed: …", each part only when there is something to say; "" when the permissions did not change.
func permissionsDelta(before, after []authz.Permission) string {
	var added, removed []authz.Permission
	for _, p := range after {
		if !slices.Contains(before, p) {
			added = append(added, p)
		}
	}
	for _, p := range before {
		if !slices.Contains(after, p) {
			removed = append(removed, p)
		}
	}
	var parts []string
	if len(added) > 0 {
		parts = append(parts, "Added: "+labelsOf(authz.InOrder(added)))
	}
	if len(removed) > 0 {
		part := "removed: " + labelsOf(authz.InOrder(removed))
		if len(parts) == 0 {
			part = "Removed: " + labelsOf(authz.InOrder(removed))
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}

// encode returns v as JSON for a log change; v is a string or a bool, which always encode.
func encode(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
