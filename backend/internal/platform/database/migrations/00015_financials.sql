-- +goose Up
CREATE TABLE financial_imports (
    id              text        PRIMARY KEY CHECK (id ~ '^FIM-[0-9]+$'),
    source          text        NOT NULL CHECK (source IN ('file', 'demo')),
    feed_name       text        NOT NULL,
    file_name       text        NOT NULL,
    rows            integer     NOT NULL CHECK (rows >= 0),
    new             integer     NOT NULL CHECK (new >= 0),
    updated         integer     NOT NULL CHECK (updated >= 0),
    unchanged       integer     NOT NULL CHECK (unchanged >= 0),
    refused         integer     NOT NULL CHECK (refused >= 0),
    not_linked      integer     NOT NULL CHECK (not_linked >= 0),
    relinked        integer     NOT NULL CHECK (relinked >= 0),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE TABLE financial_transactions (
    id                  text        PRIMARY KEY CHECK (id ~ '^FT-[0-9]+$'),
    type                text        NOT NULL CHECK (type IN ('purchaseOrder', 'invoice', 'payment', 'accrual')),
    document_number     text        NOT NULL,
    line_number         text        NOT NULL CHECK (line_number ~ '^[0-9]{0,6}$'),
    po_number           text        NOT NULL,
    po_line_number      text        NOT NULL,
    posting_date        date        NOT NULL,
    accrual_month       text        CHECK (accrual_month ~ '^[0-9]{4}-(0[1-9]|1[0-2])$'),
    amount_minor        bigint      NOT NULL,
    currency            text        NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    vendor_code         text        NOT NULL,
    vendor_name         text        NOT NULL,
    description         text        NOT NULL,
    swbs_code           text        NOT NULL,
    tracking_key        text        NOT NULL,
    sent_line_item_id   text        NOT NULL,
    line_item_id        text,
    matched_by          text        NOT NULL
                                    CHECK (matched_by IN ('', 'lineItemId', 'trackingKey', 'poNumber', 'swbsCode', 'manual')),
    not_linked_reason   text        NOT NULL,
    manual_link_by_id   text,
    manual_link_by_name text,
    manual_link_at      timestamptz,
    import_id           text        NOT NULL REFERENCES financial_imports (id) DEFERRABLE INITIALLY DEFERRED,
    version             integer     NOT NULL CHECK (version >= 1),
    created_at          timestamptz NOT NULL,
    created_by_id       text        NOT NULL,
    created_by_name     text        NOT NULL,
    updated_at          timestamptz NOT NULL,
    updated_by_id       text        NOT NULL,
    updated_by_name     text        NOT NULL,
    UNIQUE (type, document_number, line_number) DEFERRABLE INITIALLY DEFERRED,
    CHECK ((type = 'accrual') = (accrual_month IS NOT NULL)),
    CHECK ((matched_by = 'manual') = (manual_link_at IS NOT NULL)),
    CHECK (num_nulls(manual_link_by_id, manual_link_by_name, manual_link_at) IN (0, 3))
);

CREATE INDEX financial_transactions_line_item_id_idx ON financial_transactions (line_item_id);

-- plan_in_permission_scope is authz.InScope in SQL: the permission everywhere, or in one of the plan's teams,
-- or for a plan without teams that the person created.
-- +goose StatementBegin
CREATE FUNCTION plan_in_permission_scope(
    target_plan text, sees_everything boolean, viewer_id text, holds boolean, viewer_team_ids text[]
) RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT sees_everything
        OR (holds AND (
            EXISTS (SELECT 1 FROM plan_teams t WHERE t.plan_id = p.id AND t.team_id = ANY (viewer_team_ids))
            OR (NOT EXISTS (SELECT 1 FROM plan_teams t WHERE t.plan_id = p.id) AND p.created_by_id = viewer_id)))
    FROM plans p
    WHERE p.id = target_plan
$$;
-- +goose StatementEnd
