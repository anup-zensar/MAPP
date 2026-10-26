-- name: LockWrites :exec
SELECT pg_advisory_xact_lock(hashtext('mapp.write'));
