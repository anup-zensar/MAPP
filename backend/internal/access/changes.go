package access

import (
	"cmp"
	"context"
	"slices"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
)

// recentChangesLimit is how many access changes the Roles and access page shows (FR-038).
const recentChangesLimit = 20

// RecentChanges returns the latest 20 activity-log entries about access, newest first, and how many there
// are in all (FR-038): changes of roles, of the access rules, and of people's roles (entries on a person that
// change the field "roles").
func (s *Service) RecentChanges(ctx context.Context, a actor.Actor) ([]history.Entry, int, error) {
	entries, err := s.history.Entries(ctx)
	if err != nil {
		return nil, 0, err
	}
	matching := []history.Entry{}
	for _, e := range entries {
		if isAccessChange(e) {
			matching = append(matching, e)
		}
	}
	slices.SortFunc(matching, func(x, y history.Entry) int { return cmp.Compare(y.Seq, x.Seq) })
	return matching[:min(recentChangesLimit, len(matching))], len(matching), nil
}

// isAccessChange reports whether the log entry is about access: a role, the access rules, or a person's
// roles.
func isAccessChange(e history.Entry) bool {
	switch e.RecordType {
	case RoleKind, RulesKind:
		return true
	case personKind:
		return slices.ContainsFunc(e.Changes, func(c history.Change) bool { return c.Field == rolesField })
	}
	return false
}
