-- name: IncrementCounter :one
INSERT INTO counters (prefix, last)
VALUES (sqlc.arg(prefix), sqlc.arg(first_number))
ON CONFLICT (prefix) DO UPDATE SET last = counters.last + 1
RETURNING last;
