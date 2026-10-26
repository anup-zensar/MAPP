-- name: AddRates :copyfrom
INSERT INTO exchange_rates (
    id, currency, date, rate_micros, source, import_id, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);

-- name: DeleteRates :exec
DELETE FROM exchange_rates
WHERE id = ANY (sqlc.arg(ids)::text[]);

-- name: AddOptions :copyfrom
INSERT INTO shell_attribute_options (
    id, attribute_id, label, position, active, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
);

-- name: DeleteOptions :exec
DELETE FROM shell_attribute_options
WHERE id = ANY (sqlc.arg(ids)::text[]);

-- name: AddOptionParents :copyfrom
INSERT INTO shell_option_parents (option_id, position, parent_option_id)
VALUES ($1, $2, $3);
