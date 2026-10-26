-- +goose Up
CREATE TABLE sample_folders (
    id              text        PRIMARY KEY CHECK (id ~ '^SF-[0-9]+$'),
    name            text        NOT NULL CHECK (name <> ''),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

-- COLLATE "C" lowers ASCII letters only, so the index never refuses a name that the service's
-- case-insensitive rule (strings.EqualFold) accepts.
CREATE UNIQUE INDEX sample_folders_name_key ON sample_folders (lower(name COLLATE "C"));

CREATE TABLE sample_notes (
    id              text        PRIMARY KEY CHECK (id ~ '^SN-[0-9]+$'),
    folder_id       text        NOT NULL REFERENCES sample_folders (id) DEFERRABLE INITIALLY DEFERRED,
    title           text        NOT NULL CHECK (title <> ''),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE INDEX sample_notes_folder_id_idx ON sample_notes (folder_id);
