-- name: GetSampleFolder :one
SELECT id, name, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM sample_folders
WHERE id = $1;

-- name: ListSampleFolders :many
SELECT id, name, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM sample_folders
ORDER BY length(id), id COLLATE "C";

-- name: PutSampleFolder :exec
INSERT INTO sample_folders (
    id, name, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (id) DO UPDATE SET
    name = excluded.name,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteSampleFolder :exec
DELETE FROM sample_folders
WHERE id = $1;

-- name: GetSampleNote :one
SELECT id, folder_id, title, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id,
    updated_by_name
FROM sample_notes
WHERE id = $1;

-- name: ListSampleNotes :many
SELECT id, folder_id, title, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id,
    updated_by_name
FROM sample_notes
ORDER BY length(id), id COLLATE "C";

-- name: ListSampleNotesInFolder :many
SELECT id, folder_id, title, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id,
    updated_by_name
FROM sample_notes
WHERE folder_id = $1
ORDER BY length(id), id COLLATE "C";

-- name: PutSampleNote :exec
INSERT INTO sample_notes (
    id, folder_id, title, version, created_at, created_by_id, created_by_name, updated_at, updated_by_id,
    updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
ON CONFLICT (id) DO UPDATE SET
    folder_id = excluded.folder_id,
    title = excluded.title,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteSampleNote :exec
DELETE FROM sample_notes
WHERE id = $1;
