-- name: GetAllWorkPreferences :one
SELECT person_id, columns, sort_column, sort_direction, updated_at
FROM all_work_preferences
WHERE person_id = $1;

-- name: PutAllWorkPreferences :exec
INSERT INTO all_work_preferences (person_id, columns, sort_column, sort_direction, updated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (person_id) DO UPDATE SET
    columns = excluded.columns,
    sort_column = excluded.sort_column,
    sort_direction = excluded.sort_direction,
    updated_at = excluded.updated_at;

-- name: GetSavedView :one
SELECT id, person_id, name, is_default, state, created_at, updated_at
FROM saved_views
WHERE id = $1;

-- name: ListSavedViews :many
SELECT id, person_id, name, is_default, state, created_at, updated_at
FROM saved_views
WHERE person_id = $1
ORDER BY length(id), id COLLATE "C";

-- name: PutSavedView :exec
INSERT INTO saved_views (id, person_id, name, is_default, state, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE SET
    person_id = excluded.person_id,
    name = excluded.name,
    is_default = excluded.is_default,
    state = excluded.state,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

-- name: DeleteSavedView :exec
DELETE FROM saved_views
WHERE id = $1;
