-- name: GetNotification :one
SELECT id, recipient_id, kind, title, text, subject_id, subject_name, actor_id, actor_name, at, read, read_at,
    subject_kind
FROM notifications
WHERE id = $1;

-- name: NotificationsOf :many
SELECT id, recipient_id, kind, title, text, subject_id, subject_name, actor_id, actor_name, at, read, read_at,
    subject_kind
FROM notifications
WHERE recipient_id = $1
ORDER BY at DESC, length(id) DESC, id COLLATE "C" DESC;

-- name: PutNotification :exec
INSERT INTO notifications (
    id, recipient_id, kind, title, text, subject_id, subject_name, actor_id, actor_name, at, read, read_at, subject_kind
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
ON CONFLICT (id) DO UPDATE SET
    recipient_id = excluded.recipient_id,
    kind = excluded.kind,
    title = excluded.title,
    text = excluded.text,
    subject_kind = excluded.subject_kind,
    subject_id = excluded.subject_id,
    subject_name = excluded.subject_name,
    actor_id = excluded.actor_id,
    actor_name = excluded.actor_name,
    at = excluded.at,
    read = excluded.read,
    read_at = excluded.read_at;

-- name: GetDelegation :one
SELECT id, person_id, delegate_id, start_date, end_date, note, cancelled_at, cancelled_by_id, cancelled_by_name,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM approval_delegations
WHERE id = $1;

-- name: ListDelegations :many
SELECT id, person_id, delegate_id, start_date, end_date, note, cancelled_at, cancelled_by_id, cancelled_by_name,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM approval_delegations
ORDER BY length(id), id COLLATE "C";

-- name: PutDelegation :exec
INSERT INTO approval_delegations (
    id, person_id, delegate_id, start_date, end_date, note, cancelled_at, cancelled_by_id, cancelled_by_name,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
ON CONFLICT (id) DO UPDATE SET
    person_id = excluded.person_id,
    delegate_id = excluded.delegate_id,
    start_date = excluded.start_date,
    end_date = excluded.end_date,
    note = excluded.note,
    cancelled_at = excluded.cancelled_at,
    cancelled_by_id = excluded.cancelled_by_id,
    cancelled_by_name = excluded.cancelled_by_name,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: MarkNotificationsRead :exec
UPDATE notifications
SET read = true, read_at = sqlc.arg(read_at)
WHERE id = ANY (sqlc.arg(ids)::text[]);
