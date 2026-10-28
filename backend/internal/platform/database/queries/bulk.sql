-- name: AddBulkAction :exec
INSERT INTO bulk_actions (id, action, note, actor_id, actor_name, ran_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: AddBulkActionBinEntries :copyfrom
INSERT INTO bulk_action_bin_entries (bulk_action_id, position, bin_entry_id)
VALUES ($1, $2, $3);

-- name: GetBulkActionOfActor :one
SELECT sqlc.embed(bulk_actions),
    coalesce((SELECT array_agg(e.bin_entry_id ORDER BY e.position) FROM bulk_action_bin_entries e
        WHERE e.bulk_action_id = bulk_actions.id), '{}')::text[] AS bin_entry_ids
FROM bulk_actions
WHERE id = sqlc.arg(id) AND actor_id = sqlc.arg(actor_id);

-- name: MarkBulkActionUndone :exec
UPDATE bulk_actions
SET undone_at = sqlc.arg(undone_at), undone_by_id = sqlc.arg(undone_by_id), undone_by_name = sqlc.arg(undone_by_name)
WHERE id = sqlc.arg(id);
