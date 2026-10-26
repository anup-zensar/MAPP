-- name: AppendLogEntry :exec
INSERT INTO activity_log (
    id, seq, at, actor_id, actor_name, action, verb, record_type, record_type_label, record_id, record_name,
    path, changes, comment, details, items
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
);

-- name: ListLogEntries :many
SELECT id, seq, at, actor_id, actor_name, action, verb, record_type, record_type_label, record_id, record_name,
    path, changes, comment, details, items
FROM activity_log
ORDER BY seq;

-- name: PutBinEntry :exec
INSERT INTO recycle_bin (
    id, record_type, record_type_label, record_id, record_name, path, parent, deleted_at, deleted_by_id,
    deleted_by_name, contents, snapshots, resolution_how, resolution_by_id, resolution_by_name, resolution_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
ON CONFLICT (id) DO UPDATE SET
    record_type = excluded.record_type,
    record_type_label = excluded.record_type_label,
    record_id = excluded.record_id,
    record_name = excluded.record_name,
    path = excluded.path,
    parent = excluded.parent,
    deleted_at = excluded.deleted_at,
    deleted_by_id = excluded.deleted_by_id,
    deleted_by_name = excluded.deleted_by_name,
    contents = excluded.contents,
    snapshots = excluded.snapshots,
    resolution_how = excluded.resolution_how,
    resolution_by_id = excluded.resolution_by_id,
    resolution_by_name = excluded.resolution_by_name,
    resolution_at = excluded.resolution_at;

-- name: GetBinEntry :one
SELECT id, record_type, record_type_label, record_id, record_name, path, parent, deleted_at, deleted_by_id,
    deleted_by_name, contents, snapshots, resolution_how, resolution_by_id, resolution_by_name, resolution_at
FROM recycle_bin
WHERE id = $1;

-- name: ListBinEntries :many
SELECT id, record_type, record_type_label, record_id, record_name, path, parent, deleted_at, deleted_by_id,
    deleted_by_name, contents, snapshots, resolution_how, resolution_by_id, resolution_by_name, resolution_at
FROM recycle_bin
ORDER BY length(id), id COLLATE "C";
