-- name: ListPurchaseRequests :many
SELECT * FROM purchase_requests
WHERE sqlc.narg(ids)::text[] IS NULL OR id = ANY (sqlc.narg(ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: ListPurchaseRequestsInScope :many
SELECT * FROM purchase_requests
WHERE (sqlc.narg(ids)::text[] IS NULL OR id = ANY (sqlc.narg(ids)::text[]))
    AND (sqlc.narg(line_item_ids)::text[] IS NULL OR EXISTS (
        SELECT 1 FROM purchase_request_lines l
        WHERE l.purchase_request_id = purchase_requests.id AND l.line_item_id = ANY (sqlc.narg(line_item_ids)::text[])))
    AND purchase_request_in_scope(id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
        sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[], sqlc.arg(lists)::boolean,
        sqlc.arg(list_team_ids)::text[])
ORDER BY length(id), id COLLATE "C";

-- name: ListPurchaseRequestLines :many
SELECT * FROM purchase_request_lines
WHERE (sqlc.narg(purchase_request_ids)::text[] IS NULL OR purchase_request_id = ANY (sqlc.narg(purchase_request_ids)::text[]))
    AND (sqlc.narg(ids)::text[] IS NULL OR id = ANY (sqlc.narg(ids)::text[]))
ORDER BY length(purchase_request_id), purchase_request_id COLLATE "C", position;

-- name: ListPurchaseRequestClaims :many
SELECT l.id, l.purchase_request_id, l.line_item_id, l.currency, l.amount_minor, p.status, p.name
FROM purchase_request_lines l
JOIN purchase_requests p ON p.id = l.purchase_request_id
WHERE l.line_item_id = ANY (sqlc.arg(line_item_ids)::text[])
ORDER BY length(l.purchase_request_id), l.purchase_request_id COLLATE "C", l.position;

-- name: ListPurchaseRequestChangesInReview :many
SELECT * FROM purchase_requests
WHERE change_pending AND EXISTS (
    SELECT 1 FROM json_array_elements(approved_version -> 'lines') AS v
    WHERE v ->> 'lineItemId' = ANY (sqlc.arg(line_item_ids)::text[]))
ORDER BY length(id), id COLLATE "C";

-- name: PutPurchaseRequest :exec
INSERT INTO purchase_requests (
    id, plan_id, name, description, owner_id, moa_approver_id, approval_workflow_id, modification_workflow_id, status,
    change_pending, attributes, vendor, approved_version, gsap_pr_number, gsap_pr_date, gsap_source, closed_at,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
)
ON CONFLICT (id) DO UPDATE SET
    plan_id = excluded.plan_id,
    name = excluded.name,
    description = excluded.description,
    owner_id = excluded.owner_id,
    moa_approver_id = excluded.moa_approver_id,
    approval_workflow_id = excluded.approval_workflow_id,
    modification_workflow_id = excluded.modification_workflow_id,
    status = excluded.status,
    change_pending = excluded.change_pending,
    attributes = excluded.attributes,
    vendor = excluded.vendor,
    approved_version = excluded.approved_version,
    gsap_pr_number = excluded.gsap_pr_number,
    gsap_pr_date = excluded.gsap_pr_date,
    gsap_source = excluded.gsap_source,
    closed_at = excluded.closed_at,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: PutPurchaseRequestLine :exec
INSERT INTO purchase_request_lines (
    id, purchase_request_id, position, line_item_id, currency, amount_minor, delivery_date, overall_limit_minor,
    description, attributes, sent_on, po_number, po_line, po_line_status, po_date, po_value_minor, po_currency,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
)
ON CONFLICT (id) DO UPDATE SET
    purchase_request_id = excluded.purchase_request_id,
    position = excluded.position,
    line_item_id = excluded.line_item_id,
    currency = excluded.currency,
    amount_minor = excluded.amount_minor,
    delivery_date = excluded.delivery_date,
    overall_limit_minor = excluded.overall_limit_minor,
    description = excluded.description,
    attributes = excluded.attributes,
    sent_on = excluded.sent_on,
    po_number = excluded.po_number,
    po_line = excluded.po_line,
    po_line_status = excluded.po_line_status,
    po_date = excluded.po_date,
    po_value_minor = excluded.po_value_minor,
    po_currency = excluded.po_currency,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeletePurchaseRequest :exec
DELETE FROM purchase_requests WHERE id = $1;

-- name: DeletePurchaseRequestLine :exec
DELETE FROM purchase_request_lines WHERE id = $1;

-- name: AddPurchaseRequestRouting :exec
INSERT INTO purchase_request_routings (
    id, purchase_request_id, type, routed_at, routed_by_id, routed_by_name, recipients, form
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListPurchaseRequestRoutings :many
SELECT * FROM purchase_request_routings
WHERE (sqlc.narg(purchase_request_id)::text IS NULL OR purchase_request_id = sqlc.narg(purchase_request_id))
    AND (sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id))
ORDER BY routed_at DESC, length(id) DESC, id COLLATE "C" DESC;

-- name: GetPurchaseRequestConfiguration :many
SELECT * FROM purchase_request_configuration WHERE id = 'purchase-requests';

-- name: PutPurchaseRequestConfiguration :exec
INSERT INTO purchase_request_configuration (
    id, purchase_support, lines_on_approved, version, created_at, created_by_id, created_by_name, updated_at,
    updated_by_id, updated_by_name
) VALUES ('purchase-requests', $1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
    purchase_support = excluded.purchase_support,
    lines_on_approved = excluded.lines_on_approved,
    version = excluded.version,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: PagePurchaseRequests :many
WITH matches AS (
    SELECT p.id, p.name, p.status, p.vendor, p.created_at,
        coalesce((SELECT sum(round(l.amount_minor::numeric * 1000000 / coalesce((sqlc.arg(rate_micros)::bigint[])[
                array_position(sqlc.arg(rate_currencies)::text[], l.currency)], 1000000)))
            FROM purchase_request_lines l WHERE l.purchase_request_id = p.id), 0)::bigint AS amount_usd_minor
    FROM purchase_requests p
    WHERE plan_in_permission_scope(p.plan_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
            sqlc.arg(holds)::boolean, sqlc.arg(team_ids)::text[])
        AND (sqlc.arg(search)::text = ''
            OR strpos(lower(p.id), sqlc.arg(search)::text) > 0
            OR strpos(lower(p.name), sqlc.arg(search)::text) > 0
            OR strpos(lower(p.vendor), sqlc.arg(search)::text) > 0
            OR strpos(lower(p.gsap_pr_number), sqlc.arg(search)::text) > 0
            OR EXISTS (SELECT 1 FROM purchase_request_lines l WHERE l.purchase_request_id = p.id
                AND (strpos(lower('HX' || substr(l.line_item_id, 3)), sqlc.arg(search)::text) > 0
                    OR strpos(lower(l.po_number), sqlc.arg(search)::text) > 0)))
        AND (cardinality(sqlc.arg(statuses)::text[]) = 0 OR p.status = ANY (sqlc.arg(statuses)::text[]))
        AND ((cardinality(sqlc.arg(type_ids)::text[]) = 0 AND NOT sqlc.arg(type_not_completed)::boolean)
            OR EXISTS (SELECT 1 FROM json_array_elements_text(CASE json_typeof(p.attributes -> sqlc.arg(type_attribute)::text)
                    WHEN 'array' THEN p.attributes -> sqlc.arg(type_attribute)::text ELSE '[]' END) AS v(option_id)
                WHERE v.option_id = ANY (sqlc.arg(type_ids)::text[]))
            OR (sqlc.arg(type_not_completed)::boolean AND coalesce(json_array_length(CASE
                json_typeof(p.attributes -> sqlc.arg(type_attribute)::text) WHEN 'array'
                THEN p.attributes -> sqlc.arg(type_attribute)::text ELSE '[]' END), 0) = 0))
        AND ((cardinality(sqlc.arg(vendors)::text[]) = 0 AND NOT sqlc.arg(vendor_not_completed)::boolean)
            OR lower(p.vendor) = ANY (sqlc.arg(vendors)::text[])
            OR (sqlc.arg(vendor_not_completed)::boolean AND p.vendor = ''))
        AND (cardinality(sqlc.arg(owner_ids)::text[]) = 0 OR p.owner_id = ANY (sqlc.arg(owner_ids)::text[]))
        AND (sqlc.narg(created_from)::timestamptz IS NULL OR p.created_at >= sqlc.narg(created_from)::timestamptz)
        AND (sqlc.narg(created_before)::timestamptz IS NULL OR p.created_at < sqlc.narg(created_before)::timestamptz)
        AND (NOT sqlc.arg(waiting_for_gsap)::boolean OR (p.status = 'approved'
            AND EXISTS (SELECT 1 FROM purchase_request_routings g WHERE g.purchase_request_id = p.id)
            AND EXISTS (SELECT 1 FROM purchase_request_lines l WHERE l.purchase_request_id = p.id
                AND (l.po_line = '' OR l.po_value_minor <> l.amount_minor OR l.po_currency <> l.currency))))
)
SELECT m.id, m.vendor, m.amount_usd_minor
FROM matches m
ORDER BY (m.vendor = ''), lower(m.vendor COLLATE "C"),
    CASE WHEN sqlc.arg(sort)::text = 'name' THEN lower(m.name COLLATE "C") END,
    CASE WHEN sqlc.arg(sort)::text = '-name' THEN lower(m.name COLLATE "C") END DESC,
    CASE WHEN sqlc.arg(sort)::text = 'status'
        THEN array_position(ARRAY['created', 'submitted', 'inReview', 'declined', 'approved', 'closed'], m.status) END,
    CASE WHEN sqlc.arg(sort)::text = '-status'
        THEN array_position(ARRAY['created', 'submitted', 'inReview', 'declined', 'approved', 'closed'], m.status) END DESC,
    CASE WHEN sqlc.arg(sort)::text = 'amount' THEN m.amount_usd_minor END,
    CASE WHEN sqlc.arg(sort)::text = '-amount' THEN m.amount_usd_minor END DESC,
    CASE WHEN sqlc.arg(sort)::text = 'created' THEN m.created_at END,
    CASE WHEN sqlc.arg(sort)::text = '-created' THEN m.created_at END DESC,
    CASE WHEN sqlc.arg(sort)::text = '-id' THEN length(m.id) END DESC,
    CASE WHEN sqlc.arg(sort)::text = '-id' THEN m.id END DESC,
    length(m.id), m.id COLLATE "C";
