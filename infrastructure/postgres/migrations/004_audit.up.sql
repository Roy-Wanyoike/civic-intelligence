-- 004_audit.up.sql
CREATE TABLE audit.log_entries (
    id              BIGSERIAL PRIMARY KEY,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_user_id   UUID REFERENCES identity.users(id),
    action          TEXT NOT NULL,
    entity_type     TEXT NOT NULL,
    entity_id       UUID,
    before          JSONB,
    after           JSONB,
    ip              INET,
    user_agent      TEXT,
    correlation_id  UUID
);
CREATE INDEX idx_audit_entity ON audit.log_entries(entity_type, entity_id);
CREATE INDEX idx_audit_actor   ON audit.log_entries(actor_user_id, occurred_at DESC);
CREATE INDEX idx_audit_correlation ON audit.log_entries(correlation_id);

CREATE OR REPLACE FUNCTION audit.tg_record() RETURNS TRIGGER AS $$
DECLARE
    v_actor UUID;
BEGIN
    SELECT COALESCE(current_setting('app.actor_user_id', true), '')::uuid INTO v_actor;
    INSERT INTO audit.log_entries (actor_user_id, action, entity_type, entity_id, before, after)
    VALUES (
        v_actor,
        TG_ARGV[0],
        TG_TABLE_NAME,
        COALESCE(NEW.id, OLD.id),
        CASE WHEN TG_OP = 'UPDATE' THEN to_jsonb(OLD) ELSE NULL END,
        CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE to_jsonb(NEW) END
    );
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
