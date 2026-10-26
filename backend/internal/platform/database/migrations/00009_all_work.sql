-- +goose Up
CREATE TABLE all_work_preferences (
    person_id      text        PRIMARY KEY REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    columns        text[],
    sort_column    text,
    sort_direction text        CHECK (sort_direction IN ('asc', 'desc')),
    updated_at     timestamptz NOT NULL,
    CHECK (num_nulls(sort_column, sort_direction) IN (0, 2))
);

CREATE TABLE saved_views (
    id         text        PRIMARY KEY CHECK (id ~ '^SV-[0-9]+$'),
    person_id  text        NOT NULL REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    name       text        NOT NULL CHECK (name <> ''),
    is_default boolean     NOT NULL,
    state      json        NOT NULL CHECK (json_typeof(state) = 'object'),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX saved_views_name_key ON saved_views (person_id, lower(name COLLATE "C"));
CREATE UNIQUE INDEX saved_views_default_key ON saved_views (person_id) WHERE is_default;
