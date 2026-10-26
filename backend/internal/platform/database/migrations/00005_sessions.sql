-- +goose Up
CREATE TABLE sessions (
    id           text        PRIMARY KEY CHECK (id ~ '^[0-9a-f]{64}$'),
    person_id    text        NOT NULL REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    created_at   timestamptz NOT NULL,
    last_used_at timestamptz NOT NULL
);
