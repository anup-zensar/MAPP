package access

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/authz"
)

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
