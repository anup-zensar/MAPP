-- +goose Up
ALTER TABLE line_items
    ADD COLUMN swbs_code         text        NOT NULL DEFAULT '' CHECK (swbs_code = '' OR swbs_code ~ '^[A-Z0-9-]{1,40}$'),
    ADD COLUMN swbs_requested_at timestamptz,
    ADD COLUMN swbs_received_at  timestamptz;

ALTER TABLE line_items
    ADD CONSTRAINT line_items_swbs_received CHECK ((swbs_code = '') = (swbs_received_at IS NULL)),
    ADD CONSTRAINT line_items_swbs_requested CHECK (
        swbs_received_at IS NULL OR (swbs_requested_at IS NOT NULL AND swbs_requested_at <= swbs_received_at)),
    ADD CONSTRAINT line_items_swbs_request_kept CHECK (swbs_code = '' OR request_swbs);

CREATE UNIQUE INDEX line_items_swbs_code_key ON line_items (swbs_code) WHERE swbs_code <> '';

CREATE TABLE swbs_runs (
    id              text        PRIMARY KEY CHECK (id ~ '^SWR-[0-9]+$'),
    source          text        NOT NULL CHECK (source IN ('simulatedGsap', 'demoData')),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE TABLE swbs_run_codes (
    run_id         text    NOT NULL REFERENCES swbs_runs (id) ON DELETE CASCADE,
    position       integer NOT NULL CHECK (position >= 1),
    line_item_id   text    NOT NULL CHECK (line_item_id <> ''),
    line_item_name text    NOT NULL,
    company_code   text    NOT NULL,
    swbs_code      text    NOT NULL UNIQUE CHECK (swbs_code ~ '^[A-Z0-9-]{1,40}$'),
    PRIMARY KEY (run_id, position)
);
