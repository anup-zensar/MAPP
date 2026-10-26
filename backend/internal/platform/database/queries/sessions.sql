-- name: GetSession :one
SELECT id, person_id, created_at, last_used_at
FROM sessions
WHERE id = $1;

-- name: ListSessions :many
SELECT id, person_id, created_at, last_used_at
FROM sessions
ORDER BY length(id), id COLLATE "C";

-- name: PutSession :exec
INSERT INTO sessions (id, person_id, created_at, last_used_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    person_id = excluded.person_id,
    created_at = excluded.created_at,
    last_used_at = excluded.last_used_at;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1;
