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

// jsonRepository is the Repository of the JSON store.
type jsonRepository struct{}

// NewJSONRepository returns the Repository of the JSON store.
func NewJSONRepository() Repository {
	return jsonRepository{}
}

// GetRole reads one role from the roles table.
func (jsonRepository) GetRole(ctx context.Context, id string) (Role, bool, error) {
	return store.Get[Role](ctx, rolesTable, id)
}

// ListRoles reads the whole roles table.
func (jsonRepository) ListRoles(ctx context.Context) ([]Role, error) {
	return store.List[Role](ctx, rolesTable)
}

// PutRole saves a role under its id.
func (jsonRepository) PutRole(ctx context.Context, r Role) error {
	return store.Put(ctx, rolesTable, r.ID, r)
}

// DeleteRole removes one role.
func (jsonRepository) DeleteRole(ctx context.Context, id string) error {
	return store.Delete(ctx, rolesTable, id)
}

// ListAssignments reads the whole role_assignments table.
func (jsonRepository) ListAssignments(ctx context.Context) ([]Assignment, error) {
	return store.List[Assignment](ctx, assignmentsTable)
}

// AssignmentsOfPerson reads every assignment to find those of one person, which is quick at demo scale.
// PostgreSQL would use an index instead.
func (r jsonRepository) AssignmentsOfPerson(ctx context.Context, personID string) ([]Assignment, error) {
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
func (jsonRepository) PutAssignment(ctx context.Context, a Assignment) error {
	return store.Put(ctx, assignmentsTable, a.ID, a)
}

// DeleteAssignment removes one assignment.
func (jsonRepository) DeleteAssignment(ctx context.Context, id string) error {
	return store.Delete(ctx, assignmentsTable, id)
}

// GetRules reads the one access-rules record.
func (jsonRepository) GetRules(ctx context.Context) (Rules, bool, error) {
	return store.Get[Rules](ctx, rulesTable, rulesID)
}

// PutRules saves the one access-rules record under its fixed id.
func (jsonRepository) PutRules(ctx context.Context, r Rules) error {
	r.ID = rulesID
	return store.Put(ctx, rulesTable, rulesID, r)
}
