-- +goose Up
-- +goose StatementBegin
CREATE TABLE community_assignments (
    community_id  UUID          NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    agent_id      UUID          NOT NULL REFERENCES agents(id)      ON DELETE CASCADE,
    tenant_id     VARCHAR(255)  NOT NULL,
    role          TEXT          NOT NULL,
    informed_at   TIMESTAMPTZ,
    assigned_at   TIMESTAMPTZ   NOT NULL,
    PRIMARY KEY (community_id, agent_id)
);
CREATE INDEX idx_community_assignments_agent_id  ON community_assignments(agent_id);
CREATE INDEX idx_community_assignments_tenant_id ON community_assignments(tenant_id);

-- Deprecated backward-compat: keep agents.role populated via trigger
CREATE OR REPLACE FUNCTION sync_agent_role_from_assignment()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    UPDATE agents SET role = NEW.role WHERE id = NEW.agent_id;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_sync_agent_role
AFTER INSERT OR UPDATE ON community_assignments
FOR EACH ROW EXECUTE FUNCTION sync_agent_role_from_assignment();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_sync_agent_role ON community_assignments;
DROP FUNCTION IF EXISTS sync_agent_role_from_assignment();
DROP TABLE IF EXISTS community_assignments;
-- +goose StatementEnd
