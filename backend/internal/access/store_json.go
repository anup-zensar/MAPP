package access

import (
	"context"

	"mapp/internal/platform/store"
)

// The JSON store tables of this package (data model §1.1–§1.3).
const (
	rolesTable       = "roles"
	assignmentsTable = "role_assignments"
	rulesTable       = "access_rules"
)

// JSONStore is the Store of the JSON store.
type JSONStore struct{}

// NewJSONStore returns the Store of the JSON store.
func NewJSONStore() JSONStore {
	return JSONStore{}
}

// GetRole reads one role from the roles table.
func (JSONStore) GetRole(ctx context.Context, id string) (Role, bool, error) {
	return store.Get[Role](ctx, rolesTable, id)
}

// ListRoles reads the whole roles table.
func (JSONStore) ListRoles(ctx context.Context) ([]Role, error) {
	return store.List[Role](ctx, rolesTable)
}

// PutRole saves a role under its id.
func (JSONStore) PutRole(ctx context.Context, r Role) error {
	return store.Put(ctx, rolesTable, r.ID, r)
}

// DeleteRole removes one role.
func (JSONStore) DeleteRole(ctx context.Context, id string) error {
	return store.Delete(ctx, rolesTable, id)
}

// ListAssignments reads the whole role_assignments table.
func (JSONStore) ListAssignments(ctx context.Context) ([]Assignment, error) {
	return store.List[Assignment](ctx, assignmentsTable)
}

// AssignmentsOfPerson reads every assignment to find those of one person, which is quick at demo scale.
// PostgreSQL would use an index instead.
func (r JSONStore) AssignmentsOfPerson(ctx context.Context, personID string) ([]Assignment, error) {
	all, err := r.ListAssignments(ctx)
	if err != nil {
		return nil, err
	}
	mine := []Assignment{}
	for _, a := range all {
		if a.PersonID == personID {
			mine = append(mine, a)
		}
	}
	return mine, nil
}

// PutAssignment saves an assignment under its id.
func (JSONStore) PutAssignment(ctx context.Context, a Assignment) error {
	return store.Put(ctx, assignmentsTable, a.ID, a)
}

// DeleteAssignment removes one assignment.
func (JSONStore) DeleteAssignment(ctx context.Context, id string) error {
	return store.Delete(ctx, assignmentsTable, id)
}

// GetRules reads the one access-rules record.
func (JSONStore) GetRules(ctx context.Context) (Rules, bool, error) {
	return store.Get[Rules](ctx, rulesTable, rulesID)
}

// PutRules saves the one access-rules record under its fixed id.
func (JSONStore) PutRules(ctx context.Context, r Rules) error {
	r.ID = rulesID
	return store.Put(ctx, rulesTable, rulesID, r)
}
