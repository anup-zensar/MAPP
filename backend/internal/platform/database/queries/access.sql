-- name: GetRole :one
SELECT id, name, description, built_in, permissions, default_scope, version, created_at, created_by_id,
    created_by_name, updated_at, updated_by_id, updated_by_name
FROM roles
WHERE id = $1;

-- name: ListRoles :many
SELECT id, name, description, built_in, permissions, default_scope, version, created_at, created_by_id,
    created_by_name, updated_at, updated_by_id, updated_by_name
FROM roles
ORDER BY length(id), id COLLATE "C";

-- name: PutRole :exec
INSERT INTO roles (
    id, name, description, built_in, permissions, default_scope, version, created_at, created_by_id,
    created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    built_in = excluded.built_in,
    permissions = excluded.permissions,
    default_scope = excluded.default_scope,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteRole :exec
DELETE FROM roles
WHERE id = $1;

-- name: ListAssignments :many
SELECT sqlc.embed(role_assignments),
    coalesce((SELECT array_agg(t.team_id ORDER BY t.position) FROM role_assignment_teams t
        WHERE t.assignment_id = role_assignments.id), '{}')::text[] AS team_ids
FROM role_assignments
WHERE sqlc.narg(person_id)::text IS NULL OR person_id = sqlc.narg(person_id)
ORDER BY length(id), id COLLATE "C";

-- name: PutAssignment :exec
INSERT INTO role_assignments (
    id, person_id, role_id, scope_type, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
ON CONFLICT (id) DO UPDATE SET
    person_id = excluded.person_id,
    role_id = excluded.role_id,
    scope_type = excluded.scope_type,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteAssignmentTeams :exec
DELETE FROM role_assignment_teams
WHERE assignment_id = $1;

-- name: InsertAssignmentTeams :exec
INSERT INTO role_assignment_teams (assignment_id, position, team_id)
SELECT sqlc.arg(assignment_id), ordinality::integer, team_id
FROM unnest(sqlc.arg(team_ids)::text[]) WITH ORDINALITY AS t(team_id, ordinality);

-- name: DeleteAssignment :exec
DELETE FROM role_assignments
WHERE id = $1;

-- name: GetAccessRules :one
SELECT id, no_self_approval, new_person_role_id, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
FROM access_rules
WHERE id = $1;

-- name: PutAccessRules :exec
INSERT INTO access_rules (
    id, no_self_approval, new_person_role_id, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
ON CONFLICT (id) DO UPDATE SET
    no_self_approval = excluded.no_self_approval,
    new_person_role_id = excluded.new_person_role_id,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;
