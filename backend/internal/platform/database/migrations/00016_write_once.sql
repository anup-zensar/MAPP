-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION refuse_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% rows are written once and never changed or deleted', TG_TABLE_NAME
        USING ERRCODE = 'restrict_violation';
END
$$;
-- +goose StatementEnd

CREATE TRIGGER activity_log_write_once BEFORE UPDATE OR DELETE ON activity_log
    FOR EACH ROW EXECUTE FUNCTION refuse_change();

CREATE TRIGGER approval_actions_write_once BEFORE UPDATE OR DELETE ON approval_actions
    FOR EACH ROW EXECUTE FUNCTION refuse_change();

CREATE TRIGGER rate_imports_write_once BEFORE UPDATE OR DELETE ON rate_imports
    FOR EACH ROW EXECUTE FUNCTION refuse_change();
