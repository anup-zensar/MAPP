-- name: GetCurrency :one
SELECT code, name, symbol, rate_micros, rate_changed_at, rate_changed_by_id, rate_changed_by_name, active, version,
    created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM currencies
WHERE code = $1;

-- name: ListCurrencies :many
SELECT code, name, symbol, rate_micros, rate_changed_at, rate_changed_by_id, rate_changed_by_name, active, version,
    created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM currencies
ORDER BY length(code), code COLLATE "C";

-- name: PutCurrency :exec
INSERT INTO currencies (
    code, name, symbol, rate_micros, rate_changed_at, rate_changed_by_id, rate_changed_by_name, active, version,
    created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
ON CONFLICT (code) DO UPDATE SET
    name = excluded.name,
    symbol = excluded.symbol,
    rate_micros = excluded.rate_micros,
    rate_changed_at = excluded.rate_changed_at,
    rate_changed_by_id = excluded.rate_changed_by_id,
    rate_changed_by_name = excluded.rate_changed_by_name,
    active = excluded.active,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteCurrency :exec
DELETE FROM currencies
WHERE code = $1;

-- name: GetRate :one
SELECT id, currency, date, rate_micros, source, import_id, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name
FROM exchange_rates
WHERE id = $1;

-- name: ListRates :many
SELECT id, currency, date, rate_micros, source, import_id, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name
FROM exchange_rates
ORDER BY length(id), id COLLATE "C";

-- name: RatesOf :many
SELECT id, currency, date, rate_micros, source, import_id, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name
FROM exchange_rates
WHERE currency = $1
ORDER BY date, length(id), id COLLATE "C";

-- name: PutRate :exec
INSERT INTO exchange_rates (
    id, currency, date, rate_micros, source, import_id, version, created_at, created_by_id, created_by_name,
    updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
ON CONFLICT (id) DO UPDATE SET
    currency = excluded.currency,
    date = excluded.date,
    rate_micros = excluded.rate_micros,
    source = excluded.source,
    import_id = excluded.import_id,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteRate :exec
DELETE FROM exchange_rates
WHERE id = $1;

-- name: GetRateImport :one
SELECT id, file_name, feed, first_date, last_date, added, replaced, unchanged, not_used, problems, version,
    created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM rate_imports
WHERE id = $1;

-- name: ListRateImports :many
SELECT id, file_name, feed, first_date, last_date, added, replaced, unchanged, not_used, problems, version,
    created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM rate_imports
ORDER BY length(id), id COLLATE "C";

-- name: InsertRateImport :exec
INSERT INTO rate_imports (
    id, file_name, feed, first_date, last_date, added, replaced, unchanged, not_used, problems, version,
    created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
);
