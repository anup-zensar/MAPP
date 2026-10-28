-- +goose Up
-- Feature 032: purchase requests raised from line items, their lines, their routings to the requisitioner
-- (written once) and the Purchase Request Configuration.
CREATE TABLE purchase_requests (
    id                       text        PRIMARY KEY CHECK (id ~ '^PR-[0-9]+$'),
    plan_id                  text        NOT NULL REFERENCES plans (id) DEFERRABLE INITIALLY DEFERRED,
    name                     text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
    description              text        NOT NULL CHECK (char_length(description) <= 2000),
    owner_id                 text        NOT NULL REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    moa_approver_id          text        REFERENCES people (id) DEFERRABLE INITIALLY DEFERRED,
    approval_workflow_id     text        NOT NULL,
    modification_workflow_id text        NOT NULL,
    status                   text        NOT NULL
                                         CHECK (status IN ('created', 'submitted', 'inReview', 'declined', 'approved', 'closed')),
    change_pending           boolean     NOT NULL CHECK (NOT change_pending OR status = 'inReview'),
    attributes               json        NOT NULL CHECK (json_typeof(attributes) = 'object'),
    vendor                   text        NOT NULL,
    approved_version         json        CHECK (approved_version IS NULL OR json_typeof(approved_version) = 'object'),
    gsap_pr_number           text        NOT NULL,
    gsap_pr_date             date,
    gsap_source              text        NOT NULL CHECK (gsap_source IN ('', 'simulatedGsap', 'gsap')),
    closed_at                timestamptz,
    version                  integer     NOT NULL CHECK (version >= 1),
    created_at               timestamptz NOT NULL,
    created_by_id            text        NOT NULL,
    created_by_name          text        NOT NULL,
    updated_at               timestamptz NOT NULL,
    updated_by_id            text        NOT NULL,
    updated_by_name          text        NOT NULL,
    CHECK ((status NOT IN ('approved', 'closed') AND NOT change_pending) OR approved_version IS NOT NULL),
    CHECK ((gsap_pr_number = '') = (gsap_pr_date IS NULL)),
    CHECK ((gsap_source = '') = (gsap_pr_number = '')),
    CHECK ((status = 'closed') = (closed_at IS NOT NULL))
);

CREATE INDEX purchase_requests_plan_id_idx ON purchase_requests (plan_id);
CREATE INDEX purchase_requests_status_idx ON purchase_requests (status);
CREATE INDEX purchase_requests_vendor_idx ON purchase_requests (lower(vendor COLLATE "C"), id);

CREATE TABLE purchase_request_lines (
    id                  text        PRIMARY KEY CHECK (id ~ '^PRL-[0-9]+$'),
    purchase_request_id text        NOT NULL REFERENCES purchase_requests (id) DEFERRABLE INITIALLY DEFERRED,
    position            integer     NOT NULL CHECK (position >= 1),
    line_item_id        text        NOT NULL REFERENCES line_items (id) DEFERRABLE INITIALLY DEFERRED,
    currency            text        NOT NULL REFERENCES currencies (code) DEFERRABLE INITIALLY DEFERRED,
    amount_minor        bigint      NOT NULL CHECK (amount_minor >= 0),
    delivery_date       date        NOT NULL,
    overall_limit_minor bigint      CHECK (overall_limit_minor >= 0),
    description         text        NOT NULL CHECK (char_length(description) <= 2000),
    attributes          json        NOT NULL CHECK (json_typeof(attributes) = 'object'),
    sent_on             date,
    po_number           text        NOT NULL,
    po_line             text        NOT NULL,
    po_line_status      text        NOT NULL,
    po_date             date,
    po_value_minor      bigint,
    po_currency         text        NOT NULL,
    version             integer     NOT NULL CHECK (version >= 1),
    created_at          timestamptz NOT NULL,
    created_by_id       text        NOT NULL,
    created_by_name     text        NOT NULL,
    updated_at          timestamptz NOT NULL,
    updated_by_id       text        NOT NULL,
    updated_by_name     text        NOT NULL,
    CONSTRAINT purchase_request_lines_position_key UNIQUE (purchase_request_id, position) DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT purchase_request_lines_line_item_key UNIQUE (purchase_request_id, line_item_id)
        DEFERRABLE INITIALLY DEFERRED,
    CHECK ((po_line = '') = (po_date IS NULL)),
    CHECK ((po_line = '') = (po_value_minor IS NULL)),
    CHECK ((po_line = '') = (po_currency = '')),
    CHECK (po_line = '' OR po_number <> '')
);

CREATE INDEX purchase_request_lines_purchase_request_id_idx ON purchase_request_lines (purchase_request_id);
CREATE INDEX purchase_request_lines_line_item_id_idx ON purchase_request_lines (line_item_id);
CREATE INDEX purchase_request_lines_po_number_idx ON purchase_request_lines (po_number);

CREATE TABLE purchase_request_routings (
    id                  text        PRIMARY KEY CHECK (id ~ '^PRR-[0-9]+$'),
    purchase_request_id text        NOT NULL REFERENCES purchase_requests (id) DEFERRABLE INITIALLY DEFERRED,
    type                text        NOT NULL CHECK (type IN ('original', 'modification', 'resend', 'close')),
    routed_at           timestamptz NOT NULL,
    routed_by_id        text        NOT NULL,
    routed_by_name      text        NOT NULL,
    recipients          json        NOT NULL CHECK (json_typeof(recipients) = 'array' AND json_array_length(recipients) >= 1),
    form                json        NOT NULL CHECK (json_typeof(form) = 'array' AND json_array_length(form) >= 1)
);

CREATE INDEX purchase_request_routings_purchase_request_idx ON purchase_request_routings (purchase_request_id, routed_at);

CREATE TRIGGER purchase_request_routings_write_once BEFORE UPDATE OR DELETE ON purchase_request_routings
    FOR EACH ROW EXECUTE FUNCTION refuse_change();

CREATE TABLE purchase_request_configuration (
    id                text        PRIMARY KEY CHECK (id = 'purchase-requests'),
    purchase_support  boolean     NOT NULL,
    lines_on_approved text        NOT NULL CHECK (lines_on_approved IN ('notAllowed', 'withModificationApproval', 'withoutApproval')),
    version           integer     NOT NULL CHECK (version >= 1),
    created_at        timestamptz NOT NULL,
    created_by_id     text        NOT NULL,
    created_by_name   text        NOT NULL,
    updated_at        timestamptz NOT NULL,
    updated_by_id     text        NOT NULL,
    updated_by_name   text        NOT NULL
);

ALTER TABLE shell_attributes
    DROP CONSTRAINT shell_attributes_level_check,
    ADD CONSTRAINT shell_attributes_level_check CHECK (level IN ('plan', 'campaign', 'initiative', 'activity', 'lineItem',
        'purchaseRequest', 'purchaseRequestLine')),
    DROP CONSTRAINT shell_attributes_role_check,
    ADD CONSTRAINT shell_attributes_role_check CHECK (role IN ('', 'businessUnit', 'vendor', 'vendorCode', 'vendorEmail',
        'shortText', 'prType', 'emailGroup', 'prRecipients', 'companyCode', 'costCentre', 'glCode'));

ALTER TABLE roles
    DROP CONSTRAINT roles_permissions_check,
    ADD CONSTRAINT roles_permissions_check CHECK (cardinality(permissions) >= 1 AND permissions <@ ARRAY[
        'plans.view', 'plans.create', 'plans.edit', 'plans.publish', 'plans.delete',
        'campaigns.edit', 'campaigns.delete', 'activities.edit', 'activities.delete',
        'activities.submit', 'data.export', 'finance.view', 'finance.link', 'finance.import',
        'purchaseRequests.edit', 'purchaseRequests.delete', 'purchaseRequests.submit', 'purchaseRequests.list',
        'approvals.decide', 'approvals.decideOnBehalf', 'globalBudgets.view',
        'globalBudgets.manage', 'settings.manage', 'exchangeRates.manage', 'workflows.manage',
        'people.manage', 'access.manage', 'activityLog.viewAll', 'recycleBin.purge']);

-- authz's view of a purchase request: its plan in the plans.view or purchaseRequests.list scope, its owner, its
-- MoA approver, or an approver named in a level of one of its rounds (032 data-model §1.6).
-- +goose StatementBegin
CREATE FUNCTION purchase_request_in_scope(
    target text, sees_everything boolean, viewer_id text, views_plans boolean, viewer_team_ids text[],
    lists boolean, list_team_ids text[]
) RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT sees_everything
        OR p.owner_id = viewer_id
        OR coalesce(p.moa_approver_id = viewer_id, false)
        OR plan_in_scope(p.plan_id, false, viewer_id, views_plans, viewer_team_ids)
        OR plan_in_permission_scope(p.plan_id, false, viewer_id, lists, list_team_ids)
        OR EXISTS (
            SELECT 1
            FROM approval_rounds r,
                json_array_elements(CASE json_typeof(r.levels) WHEN 'array' THEN r.levels ELSE '[]' END) AS l(level),
                json_array_elements(CASE json_typeof(l.level -> 'approvers') WHEN 'array' THEN l.level -> 'approvers'
                    ELSE '[]' END) AS ap(approver)
            WHERE r.subject_kind IN ('purchaseRequest', 'modifiedPurchaseRequest') AND r.subject_id = p.id
                AND ap.approver ->> 'id' = viewer_id)
    FROM purchase_requests p
    WHERE p.id = target
$$;
-- +goose StatementEnd
