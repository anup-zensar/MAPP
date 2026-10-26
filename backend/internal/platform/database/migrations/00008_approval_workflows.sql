-- +goose Up
CREATE TABLE approval_workflows (
    id              text        PRIMARY KEY CHECK (id ~ '^WF-[0-9]+$'),
    name            text        NOT NULL CHECK (name <> ''),
    description     text        NOT NULL,
    active          boolean     NOT NULL,
    reminder_days   integer     CHECK (reminder_days BETWEEN 1 AND 60),
    escalation_days integer     CHECK (escalation_days BETWEEN 1 AND 90),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL,
    CHECK (escalation_days > reminder_days)
);

CREATE UNIQUE INDEX approval_workflows_name_key ON approval_workflows (lower(name COLLATE "C"));

CREATE TABLE approval_workflow_levels (
    workflow_id        text    NOT NULL REFERENCES approval_workflows (id) ON DELETE CASCADE
                               DEFERRABLE INITIALLY DEFERRED,
    number             integer NOT NULL CHECK (number >= 1),
    name               text    NOT NULL,
    approver_type      text    NOT NULL CHECK (approver_type IN ('people', 'manager', 'role')),
    role_id            text    NOT NULL,
    mode               text    NOT NULL CHECK (mode IN ('any', 'all')),
    applies_when       json    NOT NULL CHECK (json_typeof(applies_when) IN ('object', 'null')),
    auto_approve_when  json    NOT NULL CHECK (json_typeof(auto_approve_when) IN ('object', 'null')),
    backup_approver_id text    REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    PRIMARY KEY (workflow_id, number)
);

CREATE TABLE approval_workflow_level_approvers (
    workflow_id  text    NOT NULL,
    level_number integer NOT NULL,
    position     integer NOT NULL CHECK (position >= 1),
    person_id    text    NOT NULL REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    PRIMARY KEY (workflow_id, level_number, person_id),
    FOREIGN KEY (workflow_id, level_number) REFERENCES approval_workflow_levels (workflow_id, number)
        ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED
);
