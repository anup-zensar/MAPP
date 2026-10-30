-- +goose Up
-- Feature 038: the built-in Global planner and Planner no longer link financial transactions (D-359).
UPDATE roles
SET permissions = array_remove(permissions, 'finance.link'), version = version + 1, updated_at = now(),
    updated_by_id = 'system', updated_by_name = 'SP&A One'
WHERE built_in IN ('globalPlanner', 'planner') AND 'finance.link' = ANY (permissions);
