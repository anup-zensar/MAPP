package access

import "context"

// Store stores roles, role assignments and the access rules. The JSON store implements it now
// (store_json.go) and PostgreSQL will later, without any change to the service. Every method runs inside the
// caller's transaction, which the context carries.
type Store interface {
	// GetRole returns the role with id, and false when there is none.
	GetRole(ctx context.Context, id string) (Role, bool, error)
	// ListRoles returns every role, ordered by id.
	ListRoles(ctx context.Context) ([]Role, error)
	// PutRole saves r, replacing the role with the same id.
	PutRole(ctx context.Context, r Role) error
	// DeleteRole removes the role with id.
	DeleteRole(ctx context.Context, id string) error

	// ListAssignments returns every role assignment, ordered by id.
	ListAssignments(ctx context.Context) ([]Assignment, error)
	// PutAssignment saves a, replacing the assignment with the same id.
	PutAssignment(ctx context.Context, a Assignment) error
	// DeleteAssignment removes the assignment with id.
	DeleteAssignment(ctx context.Context, id string) error

	// GetRules returns the access rules, and false while none are stored (the defaults apply).
	GetRules(ctx context.Context) (Rules, bool, error)
	// PutRules saves the access rules.
	PutRules(ctx context.Context, r Rules) error
}
