-- name: GetPerson :one
SELECT id, name, email, job_title, active, manager_id, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name
FROM people
WHERE id = $1;

-- name: FindPersonByEmail :one
SELECT id, name, email, job_title, active, manager_id, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name
FROM people
WHERE email = $1;

-- name: ListPeople :many
SELECT id, name, email, job_title, active, manager_id, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name
FROM people
ORDER BY length(id), id COLLATE "C";

-- name: PutPerson :exec
INSERT INTO people (
    id, name, email, job_title, active, manager_id, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    email = excluded.email,
    job_title = excluded.job_title,
    active = excluded.active,
    manager_id = excluded.manager_id,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: GetPreferences :one
SELECT person_id, theme, sidebar_collapsed, display_currency, date_format, home_scope, history_include_inside,
    pinned_plan_ids
FROM preferences
WHERE person_id = $1;

-- name: ListPreferences :many
SELECT person_id, theme, sidebar_collapsed, display_currency, date_format, home_scope, history_include_inside,
    pinned_plan_ids
FROM preferences
ORDER BY length(person_id), person_id COLLATE "C";

-- name: PutPreferences :exec
INSERT INTO preferences (
    person_id, theme, sidebar_collapsed, display_currency, date_format, home_scope, history_include_inside,
    pinned_plan_ids
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (person_id) DO UPDATE SET
    theme = excluded.theme,
    sidebar_collapsed = excluded.sidebar_collapsed,
    display_currency = excluded.display_currency,
    date_format = excluded.date_format,
    home_scope = excluded.home_scope,
    history_include_inside = excluded.history_include_inside,
    pinned_plan_ids = excluded.pinned_plan_ids;

-- name: GetTeam :one
SELECT id, name, description, colour, active, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
FROM teams
WHERE id = $1;

-- name: ListTeams :many
SELECT id, name, description, colour, active, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
FROM teams
ORDER BY length(id), id COLLATE "C";

-- name: PutTeam :exec
INSERT INTO teams (
    id, name, description, colour, active, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    colour = excluded.colour,
    active = excluded.active,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteTeam :exec
DELETE FROM teams
WHERE id = $1;

-- name: GetMembership :one
SELECT id, team_id, person_id, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id,
    updated_by_name
FROM team_memberships
WHERE id = $1;

-- name: ListMemberships :many
SELECT id, team_id, person_id, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id,
    updated_by_name
FROM team_memberships
WHERE (sqlc.narg(team_id)::text IS NULL OR team_id = sqlc.narg(team_id))
    AND (sqlc.narg(person_id)::text IS NULL OR person_id = sqlc.narg(person_id))
ORDER BY length(id), id COLLATE "C";

-- name: PutMembership :exec
INSERT INTO team_memberships (
    id, team_id, person_id, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id,
    updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
ON CONFLICT (id) DO UPDATE SET
    team_id = excluded.team_id,
    person_id = excluded.person_id,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteMembership :exec
DELETE FROM team_memberships
WHERE id = $1;
