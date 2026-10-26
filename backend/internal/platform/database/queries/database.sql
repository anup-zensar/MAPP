-- name: LockWrites :exec
SELECT pg_advisory_xact_lock(hashtext('mapp.write'));

-- name: DataTables :many
SELECT tablename::text FROM pg_catalog.pg_tables
WHERE schemaname = 'public' AND tablename <> 'goose_db_version'
ORDER BY tablename;

-- name: DatabaseExists :one
SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_database WHERE datname = sqlc.arg(name)::text);
