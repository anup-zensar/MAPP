package access

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/memo"
	"mapp/internal/platform/record"
)

// keptArea is what the transaction memo keeps of access: roles, assignments and rules.
const keptArea = "access"

// PostgresStore is the Repository on PostgreSQL. A role's permissions are a text[] column (values of a
// fixed list); an assignment's chosen teams are a child table, so each team must exist.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) GetRole(ctx context.Context, id string) (Role, bool, error) {
	roles, err := s.roles(ctx)
	row, found := roles.Find(id)
	if err != nil || !found {
		return Role{}, false, err
	}
	return roleFromRow(row), true, nil
}

func (s *PostgresStore) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := s.roles(ctx)
	if err != nil {
		return nil, err
	}
	roles := make([]Role, 0, len(rows.Rows))
	for _, row := range rows.Rows {
		roles = append(roles, roleFromRow(row))
	}
	return roles, nil
}

func (s *PostgresStore) PutRole(ctx context.Context, r Role) error {
	permissions := make([]string, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		permissions = append(permissions, string(p))
	}
	err := s.change(ctx).PutRole(ctx, sqlcdb.PutRoleParams{
		ID: r.ID, Name: r.Name, Description: r.Description, BuiltIn: r.BuiltIn, Permissions: permissions,
		DefaultScope: string(r.DefaultScope), Version: r.Version, CreatedAt: r.CreatedAt,
		CreatedByID: r.CreatedBy.ID, CreatedByName: r.CreatedBy.Name, UpdatedAt: r.UpdatedAt,
		UpdatedByID: r.UpdatedBy.ID, UpdatedByName: r.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save role %s: %w", r.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteRole(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteRole(ctx, id); err != nil {
		return fmt.Errorf("delete role %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) ListAssignments(ctx context.Context) ([]Assignment, error) {
	return s.assignments(ctx, nil)
}

func (s *PostgresStore) AssignmentsOfPerson(ctx context.Context, personID string) ([]Assignment, error) {
	return s.assignments(ctx, &personID)
}

func (s *PostgresStore) PutAssignment(ctx context.Context, a Assignment) error {
	q := s.change(ctx)
	err := q.PutAssignment(ctx, sqlcdb.PutAssignmentParams{
		ID: a.ID, PersonID: a.PersonID, RoleID: a.RoleID, ScopeType: string(a.ScopeType), Version: a.Version,
		CreatedAt: a.CreatedAt, CreatedByID: a.CreatedBy.ID, CreatedByName: a.CreatedBy.Name,
		UpdatedAt: a.UpdatedAt, UpdatedByID: a.UpdatedBy.ID, UpdatedByName: a.UpdatedBy.Name,
	})
	if err == nil {
		err = q.DeleteAssignmentTeams(ctx, a.ID)
	}
	if err == nil {
		err = q.InsertAssignmentTeams(ctx, sqlcdb.InsertAssignmentTeamsParams{AssignmentID: a.ID, TeamIds: a.TeamIDs})
	}
	if err != nil {
		return fmt.Errorf("save role assignment %s: %w", a.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteAssignment(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteAssignment(ctx, id); err != nil {
		return fmt.Errorf("delete role assignment %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) GetRules(ctx context.Context) (Rules, bool, error) {
	kept, err := memo.Get(ctx, keptArea+":rules", func(ctx context.Context) (*sqlcdb.AccessRule, error) {
		row, err := s.db.Queries(ctx).GetAccessRules(ctx, rulesID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return &row, err
	})
	if err != nil {
		return Rules{}, false, fmt.Errorf("select access rules: %w", err)
	}
	if kept == nil {
		return Rules{}, false, nil
	}
	row := *kept
	return Rules{
		ID: row.ID, NoSelfApproval: row.NoSelfApproval, NewPersonRoleID: database.EmptyIfNull(row.NewPersonRoleID),
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}, true, nil
}

func (s *PostgresStore) PutRules(ctx context.Context, r Rules) error {
	err := s.change(ctx).PutAccessRules(ctx, sqlcdb.PutAccessRulesParams{
		ID: rulesID, NoSelfApproval: r.NoSelfApproval, NewPersonRoleID: database.NullIfEmpty(r.NewPersonRoleID),
		Version: r.Version, CreatedAt: r.CreatedAt, CreatedByID: r.CreatedBy.ID, CreatedByName: r.CreatedBy.Name,
		UpdatedAt: r.UpdatedAt, UpdatedByID: r.UpdatedBy.ID, UpdatedByName: r.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save access rules: %w", err)
	}
	return nil
}

func (s *PostgresStore) assignments(ctx context.Context, personID *string) ([]Assignment, error) {
	rows, err := memo.Get(ctx, keptArea+":assignments", func(ctx context.Context) ([]sqlcdb.ListAssignmentsRow, error) {
		return s.db.Queries(ctx).ListAssignments(ctx, nil)
	})
	if err != nil {
		return nil, fmt.Errorf("select role assignments: %w", err)
	}
	assignments := make([]Assignment, 0, len(rows))
	for _, row := range rows {
		a := row.RoleAssignment
		if personID != nil && a.PersonID != *personID {
			continue
		}
		assignments = append(assignments, Assignment{
			ID: a.ID, PersonID: a.PersonID, RoleID: a.RoleID, ScopeType: authz.ScopeType(a.ScopeType),
			TeamIDs: slices.Clone(row.TeamIds),
			Meta: record.Meta{
				Version:   a.Version,
				CreatedAt: a.CreatedAt,
				CreatedBy: actor.Actor{ID: a.CreatedByID, Name: a.CreatedByName},
				UpdatedAt: a.UpdatedAt,
				UpdatedBy: actor.Actor{ID: a.UpdatedByID, Name: a.UpdatedByName},
			},
		})
	}
	return assignments, nil
}

// roles reads every role once per transaction (memo): every request checks the actor's roles, often more
// than once, and only this store changes them.
func (s *PostgresStore) roles(ctx context.Context) (memo.Table[sqlcdb.Role], error) {
	table, err := memo.GetTable(ctx, keptArea+":roles", func(ctx context.Context) ([]sqlcdb.Role, error) {
		return s.db.Queries(ctx).ListRoles(ctx)
	}, func(r sqlcdb.Role) string { return r.ID })
	if err != nil {
		return memo.Table[sqlcdb.Role]{}, fmt.Errorf("select roles: %w", err)
	}
	return table, nil
}

func (s *PostgresStore) change(ctx context.Context) *sqlcdb.Queries {
	memo.Forget(ctx, keptArea)
	return s.db.Queries(ctx)
}

func roleFromRow(row sqlcdb.Role) Role {
	permissions := make([]authz.Permission, 0, len(row.Permissions))
	for _, p := range row.Permissions {
		permissions = append(permissions, authz.Permission(p))
	}
	return Role{
		ID: row.ID, Name: row.Name, Description: row.Description, BuiltIn: row.BuiltIn, Permissions: permissions,
		DefaultScope: authz.ScopeType(row.DefaultScope),
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}
