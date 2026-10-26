-- name: CountCampaignsOwnedBy :one
SELECT count(*)::integer FROM campaigns
WHERE owner_id = sqlc.arg(person_id)::text;

-- name: CountInitiativesOwnedBy :one
SELECT count(*)::integer FROM initiatives
WHERE owner_id = sqlc.arg(person_id)::text;

-- name: CountActivitiesOwnedBy :one
SELECT count(*)::integer FROM activities
WHERE owner_id = sqlc.arg(person_id)::text;

-- name: CountLineItemsOwnedBy :one
SELECT count(*)::integer FROM line_items
WHERE owner_id = sqlc.arg(person_id)::text;
