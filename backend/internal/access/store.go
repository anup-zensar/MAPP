package access

import "context"

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists roles and assignments by id; GetRules reports false while the defaults apply.
type Store interface {
	GetRole(ctx context.Context, id string) (Role, bool, error)
	ListRoles(ctx context.Context) ([]Role, error)
	PutRole(ctx context.Context, r Role) error
	DeleteRole(ctx context.Context, id string) error

	ListAssignments(ctx context.Context) ([]Assignment, error)
	PutAssignment(ctx context.Context, a Assignment) error
	DeleteAssignment(ctx context.Context, id string) error

	GetRules(ctx context.Context) (Rules, bool, error)
	PutRules(ctx context.Context, r Rules) error
}
