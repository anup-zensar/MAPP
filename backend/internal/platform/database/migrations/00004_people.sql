-- +goose Up
CREATE TABLE people (
    id              text        PRIMARY KEY CHECK (id ~ '^U-[0-9]+$'),
    name            text        NOT NULL CHECK (name <> ''),
    email           text        NOT NULL CHECK (email <> ''),
    job_title       text        NOT NULL,
    active          boolean     NOT NULL,
    manager_id      text        REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED CHECK (manager_id <> id),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE UNIQUE INDEX people_email_key ON people (lower(email COLLATE "C"));

CREATE TABLE preferences (
    person_id              text    PRIMARY KEY REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    theme                  text    NOT NULL CHECK (theme IN ('system', 'light', 'dark')),
    sidebar_collapsed      boolean NOT NULL,
    display_currency       text    NOT NULL,
    date_format            text    NOT NULL
                                   CHECK (date_format IN ('', 'DD MMM YYYY', 'DD/MM/YYYY', 'MM/DD/YYYY', 'YYYY-MM-DD')),
    home_scope             text    NOT NULL CHECK (home_scope IN ('', 'myTeams', 'everything')),
    history_include_inside boolean NOT NULL,
    pinned_plan_ids        text[]  NOT NULL
);

CREATE TABLE teams (
    id              text        PRIMARY KEY CHECK (id ~ '^T-[0-9]+$'),
    name            text        NOT NULL CHECK (name <> ''),
    description     text        NOT NULL,
    colour          text        NOT NULL
                                CHECK (colour IN ('blue', 'violet', 'teal', 'magenta', 'orange', 'green', 'slate', 'gold')),
    active          boolean     NOT NULL,
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE UNIQUE INDEX teams_name_key ON teams (lower(name COLLATE "C"));

CREATE TABLE team_memberships (
    id              text        PRIMARY KEY CHECK (id ~ '^TM-[0-9]+$'),
    team_id         text        NOT NULL REFERENCES teams (id) DEFERRABLE INITIALLY DEFERRED,
    person_id       text        NOT NULL REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL,
    UNIQUE (team_id, person_id)
);

CREATE INDEX team_memberships_person_id_idx ON team_memberships (person_id);
