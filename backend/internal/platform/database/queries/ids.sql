-- name: AdvanceCounter :one
INSERT INTO counters (prefix, last)
VALUES (sqlc.arg(prefix), sqlc.arg(first_number)::integer + sqlc.arg(count)::integer - 1)
ON CONFLICT (prefix) DO UPDATE SET last = counters.last + sqlc.arg(count)::integer
RETURNING last;

-- name: AnyCounter :one
SELECT EXISTS (SELECT 1 FROM counters);
