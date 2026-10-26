-- +goose Up
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE approval_rounds (
    id              text        PRIMARY KEY CHECK (id ~ '^AR-[0-9]+$'),
    activity_id     text        NOT NULL,
    number          integer     NOT NULL CHECK (number >= 1),
    workflow_id     text        NOT NULL,
    workflow_name   text        NOT NULL,
    levels          json        NOT NULL CHECK (json_typeof(levels) IN ('array', 'null')),
    current_level   integer     NOT NULL,
    outcome         text        NOT NULL
                                CHECK (outcome IN ('open', 'approved', 'declined', 'recalled', 'restarted', 'cancelled')),
    submitted_at    timestamptz NOT NULL,
    submitted_by_id text        NOT NULL,
    submitted_by_name text      NOT NULL,
    comment         text        NOT NULL,
    ended_at        timestamptz,
    facts           json        NOT NULL CHECK (json_typeof(facts) IN ('object', 'null')),
    reminder_days   integer,
    escalation_days integer,
    auto_approved   boolean     NOT NULL,
    version         integer     NOT NULL CHECK (version >= 1),
    created_at      timestamptz NOT NULL,
    created_by_id   text        NOT NULL,
    created_by_name text        NOT NULL,
    updated_at      timestamptz NOT NULL,
    updated_by_id   text        NOT NULL,
    updated_by_name text        NOT NULL,
    UNIQUE (activity_id, number) DEFERRABLE INITIALLY DEFERRED,
    EXCLUDE USING btree (activity_id WITH =) WHERE (outcome = 'open') DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE approval_actions (
    id                  text        PRIMARY KEY CHECK (id ~ '^AH-[0-9]+$'),
    round_id            text        NOT NULL REFERENCES approval_rounds (id) DEFERRABLE INITIALLY DEFERRED,
    round_number        integer     NOT NULL,
    activity_id         text        NOT NULL,
    action              text        NOT NULL CHECK (action IN ('submitted', 'resubmitted', 'approved', 'declined',
                                        'recalled', 'restarted', 'cancelled', 'autoApproved', 'skipped', 'reminded',
                                        'escalated')),
    level               integer     NOT NULL CHECK (level >= 0),
    level_name          text        NOT NULL,
    approvers           json        NOT NULL CHECK (json_typeof(approvers) IN ('array', 'null')),
    actor_id            text        NOT NULL,
    actor_name          text        NOT NULL,
    on_behalf_of        json        NOT NULL CHECK (json_typeof(on_behalf_of) IN ('array', 'null')),
    at                  timestamptz NOT NULL,
    comment             text        NOT NULL,
    changes             json        NOT NULL CHECK (json_typeof(changes) IN ('array', 'null')),
    capacity            text        NOT NULL CHECK (capacity IN ('', 'approver', 'delegate', 'backup', 'onBehalf')),
    as_delegate_of_id   text,
    as_delegate_of_name text,
    rule                text        NOT NULL,
    escalated_to_id     text,
    escalated_to_name   text,
    waited_days         integer     NOT NULL CHECK (waited_days >= 0),
    CHECK (num_nulls(as_delegate_of_id, as_delegate_of_name) IN (0, 2)),
    CHECK (num_nulls(escalated_to_id, escalated_to_name) IN (0, 2))
);

CREATE INDEX approval_actions_activity_id_idx ON approval_actions (activity_id);

CREATE TABLE notifications (
    id            text        PRIMARY KEY CHECK (id ~ '^NT-[0-9]+$'),
    recipient_id  text        NOT NULL REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    kind          text        NOT NULL CHECK (kind IN ('approvalNeeded', 'levelApproved', 'approved', 'declined',
                                  'recalled', 'restarted', 'autoApproved', 'reminder', 'escalated')),
    title         text        NOT NULL,
    text          text        NOT NULL,
    activity_id   text        NOT NULL,
    activity_name text        NOT NULL,
    actor_id      text        NOT NULL,
    actor_name    text        NOT NULL,
    at            timestamptz NOT NULL,
    read          boolean     NOT NULL,
    read_at       timestamptz,
    CHECK (read = (read_at IS NOT NULL))
);

CREATE INDEX notifications_recipient_idx ON notifications (recipient_id, at DESC);

CREATE TABLE approval_delegations (
    id                text        PRIMARY KEY CHECK (id ~ '^DLG-[0-9]+$'),
    person_id         text        NOT NULL REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    delegate_id       text        NOT NULL REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    start_date        date        NOT NULL,
    end_date          date        NOT NULL,
    note              text        NOT NULL,
    cancelled_at      timestamptz,
    cancelled_by_id   text,
    cancelled_by_name text,
    version           integer     NOT NULL CHECK (version >= 1),
    created_at        timestamptz NOT NULL,
    created_by_id     text        NOT NULL,
    created_by_name   text        NOT NULL,
    updated_at        timestamptz NOT NULL,
    updated_by_id     text        NOT NULL,
    updated_by_name   text        NOT NULL,
    CHECK (end_date >= start_date AND end_date <= start_date + 90),
    CHECK (num_nulls(cancelled_by_id, cancelled_by_name) IN (0, 2)),
    EXCLUDE USING gist (person_id WITH =, daterange(start_date, end_date, '[]') WITH &&)
        WHERE (cancelled_at IS NULL) DEFERRABLE INITIALLY DEFERRED
);

-- The approvers named in any level of any round of an activity see it too (authz.CanViewActivity).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION activity_in_scope(
    target_activity text, sees_everything boolean, viewer_id text, views_plans boolean, viewer_team_ids text[]
) RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT sees_everything
        OR coalesce(a.owner_id = viewer_id, false)
        OR plan_in_scope(c.plan_id, sees_everything, viewer_id, views_plans, viewer_team_ids)
        OR EXISTS (
            SELECT 1
            FROM approval_rounds r,
                json_array_elements(CASE json_typeof(r.levels) WHEN 'array' THEN r.levels ELSE '[]' END) AS l(level),
                json_array_elements(CASE json_typeof(l.level -> 'approvers') WHEN 'array' THEN l.level -> 'approvers'
                    ELSE '[]' END) AS ap(approver)
            WHERE r.activity_id = a.id AND ap.approver ->> 'id' = viewer_id)
    FROM activities a
    JOIN initiatives i ON i.id = a.initiative_id
    JOIN campaigns c ON c.id = i.campaign_id
    WHERE a.id = target_activity
$$;
-- +goose StatementEnd
