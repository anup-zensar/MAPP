package access

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// The history kinds of this package (data model §1.1, §1.3).
const (
	RoleKind  = "role"
	RulesKind = "accessRules"
)

// The mid-sentence names of roles and the access rules, as the log, the recycle bin and the messages show
// them.
const (
	roleSingular  = "role"
	rulesSingular = "access rules"
)

// errRulesNeverDeleted is what the access-rules kind's Remove answers: the rules are never deleted.
var errRulesNeverDeleted = errors.New("access: the access rules are never deleted")

// Kinds returns the history kinds of roles and the access rules, both top level, for internal/app to
// register (FR-037). Assignments have no kind: they are links, logged on the person (D-233).
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{
		{
			Name:           RoleKind,
			Singular:       roleSingular,
			Plural:         "roles",
			Snapshot:       s.roleSnapshot,
			Remove:         s.repo.DeleteRole,
			Put:            s.putRole,
			Describe:       s.describeRole,
			PrepareRestore: s.prepareRoleRestore,
		},
		{
			Name:     RulesKind,
			Singular: rulesSingular,
			Plural:   rulesSingular,
			Snapshot: s.rulesSnapshot,
			Remove:   func(context.Context, string) error { return errRulesNeverDeleted },
			Put:      s.putRules,
			Describe: s.describeRules,
		},
	}
}

// roleSnapshot reads the role id as history keeps it.
func (s *Service) roleSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	r, found, err := s.repo.GetRole(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: r.ID, Name: r.Name, Data: data}, true, nil
}

// putRole stores the role of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putRole(ctx context.Context, snapshot history.Snapshot) error {
	var r Role
	if err := json.Unmarshal(snapshot.Data, &r); err != nil {
		return err
	}
	return s.repo.PutRole(ctx, r)
}

// rulesSnapshot reads the access rules as history keeps them; there are none to read while the defaults
// were never saved.
func (s *Service) rulesSnapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	rules, found, err := s.repo.GetRules(ctx)
	if err != nil || !found || id != rulesID {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(rules)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: rulesID, Name: rulesRecordName, Data: data}, true, nil
}

// putRules stores the access rules of a snapshot again.
func (s *Service) putRules(ctx context.Context, snapshot history.Snapshot) error {
	var rules Rules
	if err := json.Unmarshal(snapshot.Data, &rules); err != nil {
		return err
	}
	return s.repo.PutRules(ctx, rules)
}

// describeRole gives the activity log's label and text of a role's field (data model §3): the permissions by
// their labels, the default scope in words.
func (s *Service) describeRole(_ context.Context, field string, value any) (string, string, error) {
	switch field {
	case "permissions":
		return labelPermissions, permissionLabels(value), nil
	case "defaultScope":
		return labelDefaultScope, scopeTypeText(value), nil
	}
	return "", "", nil
}

// describeRules gives the activity log's label and text of an access rule (data model §3).
func (s *Service) describeRules(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "noSelfApproval":
		if value == true {
			return labelNoSelfApproval, onText, nil
		}
		return labelNoSelfApproval, offText, nil
	case "newPersonRoleId":
		id, _ := value.(string)
		if id == "" {
			return labelNewPersonRole, noRoleText, nil
		}
		r, found, err := s.repo.GetRole(ctx, id)
		if err != nil || !found {
			return labelNewPersonRole, id, err
		}
		return labelNewPersonRole, r.Name, nil
	}
	return "", "", nil
}

// permissionLabels writes a stored list of permission keys as their labels, in Table A order.
func permissionLabels(value any) string {
	list, _ := value.([]any)
	var keys []authz.Permission
	for _, item := range list {
		if key, ok := item.(string); ok {
			keys = append(keys, authz.Permission(key))
		}
	}
	return labelsOf(authz.InOrder(keys))
}

// labelsOf joins the labels of permissions with ", "; "" for none, which the log shows as "Not filled in".
func labelsOf(keys []authz.Permission) string {
	labels := make([]string, 0, len(keys))
	for _, key := range keys {
		if info, found := authz.Lookup(string(key)); found {
			labels = append(labels, info.Label)
		}
	}
	return strings.Join(labels, ", ")
}

// scopeTypeText writes a default scope as screens show it.
func scopeTypeText(value any) string {
	switch value {
	case string(authz.ScopeAll):
		return allMarketsText
	case string(authz.ScopeMyTeams):
		return myTeamsText
	}
	return ""
}

// prepareRoleRestore applies the name rule of roles just before a role comes back (001's rule): when another
// role has meanwhile taken the name, an Undo is refused with name_taken and the role stays in the bin; a
// Restore renames it "<name> (restored)", "<name> (restored 2)", … and says so. Its assignments are not
// restored: a role is deleted only when nobody holds it (FR-018).
func (s *Service) prepareRoleRestore(ctx context.Context, rc history.RestoreContext, snapshot *history.Snapshot) ([]string, error) {
	var r Role
	if err := json.Unmarshal(snapshot.Data, &r); err != nil {
		return nil, err
	}
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	if !nameTaken(roles, r.Name, r.ID) {
		return nil, nil
	}
	if rc.Undo {
		return nil, errs.NameTaken(fmt.Sprintf(msgUndoNameTaken, r.Name))
	}
	name := r.Name + " (restored)"
	for n := 2; nameTaken(roles, name, r.ID); n++ {
		name = fmt.Sprintf("%s (restored %d)", r.Name, n)
	}
	r.Name = name
	r.Touch(rc.Actor, rc.Now)
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	snapshot.Name, snapshot.Data = r.Name, data
	return []string{fmt.Sprintf(msgRestoredName, r.Name)}, nil
}

// nameTaken reports whether a role other than exceptID has name, ignoring letter case and surrounding spaces.
func nameTaken(roles []Role, name, exceptID string) bool {
	for _, r := range roles {
		if r.ID != exceptID && strings.EqualFold(strings.TrimSpace(r.Name), strings.TrimSpace(name)) {
			return true
		}
	}
	return false
}
