-- +goose Up
CREATE TABLE bulk_actions (
    id             text        PRIMARY KEY CHECK (id ~ '^BLK-[0-9]+$'),
    action         text        NOT NULL CHECK (action IN ('changeOwner', 'changeField', 'submit', 'copy', 'move', 'delete')),
    note           text        NOT NULL CHECK (note <> ''),
    actor_id       text        NOT NULL CHECK (actor_id <> ''),
    actor_name     text        NOT NULL,
    ran_at         timestamptz NOT NULL,
    undone_at      timestamptz,
    undone_by_id   text,
    undone_by_name text,
    CHECK ((undone_at IS NULL) = (undone_by_id IS NULL)),
    CHECK ((undone_at IS NULL) = (undone_by_name IS NULL)),
    CHECK (undone_at IS NULL OR action = 'delete')
);

CREATE INDEX bulk_actions_actor_idx ON bulk_actions (actor_id);

CREATE TABLE bulk_action_bin_entries (
    bulk_action_id text    NOT NULL REFERENCES bulk_actions (id) ON DELETE CASCADE,
    position       integer NOT NULL CHECK (position >= 1),
    bin_entry_id   text    NOT NULL CHECK (bin_entry_id ~ '^BIN-[0-9]+$'),
    PRIMARY KEY (bulk_action_id, position)
);

CREATE TABLE plan_imports (
    id               text        PRIMARY KEY CHECK (id ~ '^IMP-[0-9]+$'),
    plan_id          text        NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    file_name        text        NOT NULL CHECK (length(file_name) BETWEEN 1 AND 255),
    imported_at      timestamptz NOT NULL,
    imported_by_id   text        NOT NULL CHECK (imported_by_id <> ''),
    imported_by_name text        NOT NULL,
    rows_read        integer     NOT NULL CHECK (rows_read >= 0),
    rows_left_out    integer     NOT NULL CHECK (rows_left_out BETWEEN 0 AND rows_read),
    undone_at        timestamptz,
    undone_by_id     text,
    undone_by_name   text,
    CHECK ((undone_at IS NULL) = (undone_by_id IS NULL)),
    CHECK ((undone_at IS NULL) = (undone_by_name IS NULL)),
    CHECK (undone_at IS NULL OR undone_at >= imported_at)
);

CREATE INDEX plan_imports_plan_idx ON plan_imports (plan_id);

CREATE TABLE plan_import_records (
    import_id   text    NOT NULL REFERENCES plan_imports (id) ON DELETE CASCADE,
    position    integer NOT NULL CHECK (position >= 1),
    level       text    NOT NULL CHECK (level IN ('campaign', 'initiative', 'activity', 'lineItem')),
    record_id   text    NOT NULL CHECK (record_id ~ '^(C|I|A|L)-[0-9]+$'),
    record_name text    NOT NULL,
    parent_id   text    NOT NULL,
    action      text    NOT NULL CHECK (action IN ('created', 'updated')),
    PRIMARY KEY (import_id, position)
);

CREATE TABLE plan_import_problems (
    import_id      text    NOT NULL REFERENCES plan_imports (id) ON DELETE CASCADE,
    position       integer NOT NULL CHECK (position >= 1),
    sheet          text    NOT NULL,
    row_number     integer NOT NULL CHECK (row_number >= 0),
    column_heading text    NOT NULL,
    value          text    NOT NULL,
    reason         text    NOT NULL CHECK (reason <> ''),
    PRIMARY KEY (import_id, position)
);
