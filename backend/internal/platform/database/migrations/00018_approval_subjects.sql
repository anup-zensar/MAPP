-- +goose Up
-- Feature 031: rounds, history entries and notifications belong to a subject (record type + record id), and
-- workflows have an object type. Every row stored before is about an activity.
ALTER TABLE approval_workflows
    ADD COLUMN object_type text NOT NULL DEFAULT 'activity'
        CHECK (object_type IN ('activity', 'purchaseRequest', 'modifiedPurchaseRequest'));
ALTER TABLE approval_workflows ALTER COLUMN object_type DROP DEFAULT;

ALTER TABLE approval_workflow_levels DROP CONSTRAINT approval_workflow_levels_approver_type_check;
ALTER TABLE approval_workflow_levels
    ADD CONSTRAINT approval_workflow_levels_approver_type_check
        CHECK (approver_type IN ('people', 'manager', 'role', 'owner', 'recordField')),
    ADD COLUMN record_field text NOT NULL DEFAULT '' CHECK (record_field IN ('', 'moaApprover'));
ALTER TABLE approval_workflow_levels ALTER COLUMN record_field DROP DEFAULT;
ALTER TABLE approval_workflow_levels
    ADD CONSTRAINT approval_workflow_levels_record_field_source_check
        CHECK ((approver_type = 'recordField') = (record_field <> ''));

ALTER TABLE approval_rounds RENAME COLUMN activity_id TO subject_id;
ALTER TABLE approval_rounds
    ADD COLUMN subject_kind text NOT NULL DEFAULT 'activity'
        CHECK (subject_kind IN ('activity', 'purchaseRequest', 'modifiedPurchaseRequest')),
    DROP CONSTRAINT approval_rounds_outcome_check,
    ADD CONSTRAINT approval_rounds_outcome_check
        CHECK (outcome IN ('open', 'approved', 'declined', 'recalled', 'restarted', 'cancelled', 'superseded'));
ALTER TABLE approval_rounds ALTER COLUMN subject_kind DROP DEFAULT;

ALTER TABLE approval_actions RENAME COLUMN activity_id TO subject_id;
ALTER INDEX approval_actions_activity_id_idx RENAME TO approval_actions_subject_id_idx;
ALTER TABLE approval_actions
    ADD COLUMN subject_kind text NOT NULL DEFAULT 'activity'
        CHECK (subject_kind IN ('activity', 'purchaseRequest', 'modifiedPurchaseRequest')),
    ADD COLUMN self_approval boolean NOT NULL DEFAULT false,
    DROP CONSTRAINT approval_actions_action_check,
    ADD CONSTRAINT approval_actions_action_check
        CHECK (action IN ('submitted', 'resubmitted', 'approved', 'declined', 'recalled', 'restarted', 'cancelled',
            'autoApproved', 'skipped', 'reminded', 'escalated', 'superseded'));
ALTER TABLE approval_actions ALTER COLUMN subject_kind DROP DEFAULT, ALTER COLUMN self_approval DROP DEFAULT;

ALTER TABLE notifications RENAME COLUMN activity_id TO subject_id;
ALTER TABLE notifications RENAME COLUMN activity_name TO subject_name;
ALTER TABLE notifications
    ADD COLUMN subject_kind text NOT NULL DEFAULT 'activity'
        CHECK (subject_kind IN ('activity', 'purchaseRequest', 'modifiedPurchaseRequest'));
ALTER TABLE notifications ALTER COLUMN subject_kind DROP DEFAULT;

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
            WHERE r.subject_kind = 'activity' AND r.subject_id = a.id AND ap.approver ->> 'id' = viewer_id)
    FROM activities a
    JOIN initiatives i ON i.id = a.initiative_id
    JOIN campaigns c ON c.id = i.campaign_id
    WHERE a.id = target_activity
$$;
-- +goose StatementEnd
