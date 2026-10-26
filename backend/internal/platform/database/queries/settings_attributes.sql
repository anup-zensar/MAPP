-- name: GetAttribute :one
SELECT sqlc.embed(shell_attributes),
    coalesce((SELECT array_agg(c.option_id ORDER BY c.position) FROM shell_attribute_conditions c
        WHERE c.attribute_id = shell_attributes.id), '{}')::text[] AS condition
FROM shell_attributes
WHERE id = $1;

-- name: ListAttributes :many
SELECT sqlc.embed(shell_attributes),
    coalesce((SELECT array_agg(c.option_id ORDER BY c.position) FROM shell_attribute_conditions c
        WHERE c.attribute_id = shell_attributes.id), '{}')::text[] AS condition
FROM shell_attributes
ORDER BY length(id), id COLLATE "C";

-- name: PutAttribute :exec
INSERT INTO shell_attributes (
    id, label, help_text, level, type, marking, section, position, parent_attribute_id, role, active, version,
    created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
)
ON CONFLICT (id) DO UPDATE SET
    label = excluded.label,
    help_text = excluded.help_text,
    level = excluded.level,
    type = excluded.type,
    marking = excluded.marking,
    section = excluded.section,
    position = excluded.position,
    parent_attribute_id = excluded.parent_attribute_id,
    role = excluded.role,
    active = excluded.active,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteAttributeConditions :exec
DELETE FROM shell_attribute_conditions
WHERE attribute_id = $1;

-- name: InsertAttributeConditions :exec
INSERT INTO shell_attribute_conditions (attribute_id, position, option_id)
SELECT sqlc.arg(attribute_id), ordinality::integer, option_id
FROM unnest(sqlc.arg(option_ids)::text[]) WITH ORDINALITY AS t(option_id, ordinality);

-- name: DeleteAttribute :exec
DELETE FROM shell_attributes
WHERE id = $1;

-- name: GetOption :one
SELECT sqlc.embed(shell_attribute_options),
    coalesce((SELECT array_agg(p.parent_option_id ORDER BY p.position) FROM shell_option_parents p
        WHERE p.option_id = shell_attribute_options.id), '{}')::text[] AS parent_option_ids
FROM shell_attribute_options
WHERE id = $1;

-- name: ListOptions :many
SELECT sqlc.embed(shell_attribute_options),
    coalesce((SELECT array_agg(p.parent_option_id ORDER BY p.position) FROM shell_option_parents p
        WHERE p.option_id = shell_attribute_options.id), '{}')::text[] AS parent_option_ids
FROM shell_attribute_options
WHERE sqlc.narg(attribute_id)::text IS NULL OR attribute_id = sqlc.narg(attribute_id)
ORDER BY length(id), id COLLATE "C";

-- name: PutOption :exec
INSERT INTO shell_attribute_options (
    id, attribute_id, label, position, active, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (id) DO UPDATE SET
    attribute_id = excluded.attribute_id,
    label = excluded.label,
    position = excluded.position,
    active = excluded.active,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteOptionParents :exec
DELETE FROM shell_option_parents
WHERE option_id = $1;

-- name: InsertOptionParents :exec
INSERT INTO shell_option_parents (option_id, position, parent_option_id)
SELECT sqlc.arg(option_id), ordinality::integer, parent_option_id
FROM unnest(sqlc.arg(parent_option_ids)::text[]) WITH ORDINALITY AS t(parent_option_id, ordinality);

-- name: DeleteOption :exec
DELETE FROM shell_attribute_options
WHERE id = $1;
