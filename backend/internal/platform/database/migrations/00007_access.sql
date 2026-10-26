-- +goose Up
CREATE TABLE roles (
    id              text        PRIMARY KEY CHECK (id ~ '^RL-[0-9]+$'),
    name            text        NOT NULL CHECK (name <> ''),
    description     text        NOT NULL,
    built_in        text        NOT NULL
                                CHECK (built_in IN ('', 'administrator', 'globalPlanner', 'planner', 'approver', 'viewer')),
    permissions     text[]      NOT NULL CHECK (cardinality(permissions) >= 1 AND permissions <@ ARRAY[
                                    'plans.view', 'plans.create', 'plans.edit', 'plans.publish', 'plans.delete',
                                    'campaigns.edit', 'campaigns.delete', 'activities.edit', 'activities.delete',
                                    'activities.submit', 'data.export', 'finance.view', 'finance.link', 'finance.import',
                                    'approvals.decide', 'approvals.decideOnBehalf', 'globalBudgets.view',
                                    'globalBudgets.manage', 'settings.manage', 'exchangeRates.manage', 'workflows.manage',
                                    'people.manage', 'access.manage', 'activityLog.viewAll', 'recycleBin.purge']),
    default_scope   text        NOT NULL CHECK (default_scope IN ('all', 'myTeams')),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL
);

CREATE UNIQUE INDEX roles_name_key ON roles (lower(name COLLATE "C"));
CREATE UNIQUE INDEX roles_built_in_key ON roles (built_in) WHERE built_in <> '';

CREATE TABLE role_assignments (
    id              text        PRIMARY KEY CHECK (id ~ '^RA-[0-9]+$'),
    person_id       text        NOT NULL REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    role_id         text        NOT NULL REFERENCES roles (id) DEFERRABLE INITIALLY DEFERRED,
    scope_type      text        NOT NULL CHECK (scope_type IN ('all', 'myTeams', 'teams')),
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL,
    UNIQUE (person_id, role_id)
);

CREATE TABLE role_assignment_teams (
    assignment_id text    NOT NULL REFERENCES role_assignments (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    position      integer NOT NULL CHECK (position >= 1),
    team_id       text    NOT NULL REFERENCES teams (id) DEFERRABLE INITIALLY DEFERRED,
    PRIMARY KEY (assignment_id, team_id)
);

CREATE TABLE access_rules (
    id                 text        PRIMARY KEY CHECK (id = 'access-rules'),
    no_self_approval   boolean     NOT NULL,
    new_person_role_id text        REFERENCES roles (id) DEFERRABLE INITIALLY DEFERRED,
    version            integer     NOT NULL CHECK (version >= 1),
    created_at         timestamptz NOT NULL,
    created_by_id      text        NOT NULL,
    created_by_name    text        NOT NULL,
    updated_at         timestamptz NOT NULL,
    updated_by_id      text        NOT NULL,
    updated_by_name    text        NOT NULL
);
