-- +goose Up
-- Feature 038: approving while an activity's line items exceed its planned cost warns or is refused; organisations keep warning (D-354).
ALTER TABLE overspend_configuration
    ADD COLUMN approval_over_planned_cost text NOT NULL DEFAULT 'warn'
        CHECK (approval_over_planned_cost IN ('warn', 'refuse'));
