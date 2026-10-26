-- name: LockWrites :exec
SELECT pg_advisory_xact_lock(hashtext('mapp.write'));

-- name: CountWaitingWrites :one
SELECT count(*)
FROM pg_catalog.pg_locks
WHERE locktype = 'advisory'
    AND NOT granted
    AND database = (SELECT oid FROM pg_catalog.pg_database WHERE datname = current_database());
