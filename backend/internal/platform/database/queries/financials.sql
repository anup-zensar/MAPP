-- name: ListTransactions :many
SELECT id, type, document_number, line_number, po_number, po_line_number, posting_date, accrual_month,
    amount_minor, currency, vendor_code, vendor_name, description, swbs_code, tracking_key, sent_line_item_id,
    line_item_id, matched_by, not_linked_reason, manual_link_by_id, manual_link_by_name, manual_link_at, import_id,
    version, created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM financial_transactions
WHERE sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id)
ORDER BY length(id), id COLLATE "C";

-- name: ListTransactionsInScope :many
SELECT t.id, t.type, t.document_number, t.line_number, t.po_number, t.po_line_number, t.posting_date,
    t.accrual_month, t.amount_minor, t.currency, t.vendor_code, t.vendor_name, t.description, t.swbs_code,
    t.tracking_key, t.sent_line_item_id, t.line_item_id, t.matched_by, t.not_linked_reason, t.manual_link_by_id,
    t.manual_link_by_name, t.manual_link_at, t.import_id, t.version, t.created_at, t.created_by_id,
    t.created_by_name, t.updated_at, t.updated_by_id, t.updated_by_name
FROM financial_transactions t
WHERE (t.line_item_id IS NULL AND sqlc.arg(can_import)::boolean)
    OR (t.line_item_id IS NOT NULL AND sqlc.arg(finance_everything)::boolean)
    OR EXISTS (
        SELECT 1
        FROM line_items li
        JOIN activities a ON a.id = li.activity_id
        JOIN initiatives i ON i.id = a.initiative_id
        JOIN campaigns c ON c.id = i.campaign_id
        WHERE li.id = t.line_item_id
            AND plan_in_scope(c.plan_id, sqlc.arg(sees_everything)::boolean, sqlc.arg(viewer_id)::text,
                sqlc.arg(views_plans)::boolean, sqlc.arg(viewer_team_ids)::text[])
            AND plan_in_permission_scope(c.plan_id, false, sqlc.arg(viewer_id)::text,
                sqlc.arg(holds_finance)::boolean, sqlc.arg(finance_team_ids)::text[]))
ORDER BY length(t.id), t.id COLLATE "C";

-- name: PutTransaction :exec
INSERT INTO financial_transactions (
    id, type, document_number, line_number, po_number, po_line_number, posting_date, accrual_month, amount_minor,
    currency, vendor_code, vendor_name, description, swbs_code, tracking_key, sent_line_item_id, line_item_id,
    matched_by, not_linked_reason, manual_link_by_id, manual_link_by_name, manual_link_at, import_id, version,
    created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24,
    $25, $26, $27, $28, $29, $30
)
ON CONFLICT (id) DO UPDATE SET
    type = excluded.type,
    document_number = excluded.document_number,
    line_number = excluded.line_number,
    po_number = excluded.po_number,
    po_line_number = excluded.po_line_number,
    posting_date = excluded.posting_date,
    accrual_month = excluded.accrual_month,
    amount_minor = excluded.amount_minor,
    currency = excluded.currency,
    vendor_code = excluded.vendor_code,
    vendor_name = excluded.vendor_name,
    description = excluded.description,
    swbs_code = excluded.swbs_code,
    tracking_key = excluded.tracking_key,
    sent_line_item_id = excluded.sent_line_item_id,
    line_item_id = excluded.line_item_id,
    matched_by = excluded.matched_by,
    not_linked_reason = excluded.not_linked_reason,
    manual_link_by_id = excluded.manual_link_by_id,
    manual_link_by_name = excluded.manual_link_by_name,
    manual_link_at = excluded.manual_link_at,
    import_id = excluded.import_id,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteTransaction :exec
DELETE FROM financial_transactions
WHERE id = $1;

-- name: ListFinancialImports :many
SELECT id, source, feed_name, file_name, rows, new, updated, unchanged, refused, not_linked, relinked, version,
    created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
FROM financial_imports
WHERE sqlc.narg(id)::text IS NULL OR id = sqlc.narg(id)
ORDER BY length(id), id COLLATE "C";

-- name: PutFinancialImport :exec
INSERT INTO financial_imports (
    id, source, feed_name, file_name, rows, new, updated, unchanged, refused, not_linked, relinked, version,
    created_at, created_by_id, created_by_name, updated_at, updated_by_id, updated_by_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
)
ON CONFLICT (id) DO UPDATE SET
    source = excluded.source,
    feed_name = excluded.feed_name,
    file_name = excluded.file_name,
    rows = excluded.rows,
    new = excluded.new,
    updated = excluded.updated,
    unchanged = excluded.unchanged,
    refused = excluded.refused,
    not_linked = excluded.not_linked,
    relinked = excluded.relinked,
    version = excluded.version,
    created_at = excluded.created_at,
    created_by_id = excluded.created_by_id,
    created_by_name = excluded.created_by_name,
    updated_at = excluded.updated_at,
    updated_by_id = excluded.updated_by_id,
    updated_by_name = excluded.updated_by_name;

-- name: DeleteFinancialImport :exec
DELETE FROM financial_imports
WHERE id = $1;
