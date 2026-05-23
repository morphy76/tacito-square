-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS communities (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    topology VARCHAR(50) NOT NULL DEFAULT 'hub-spoke',
    configuration JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'created',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT uq_tenant_community_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_communities_tenant_id ON communities(tenant_id);

-- Alter agents table to add community_id column and RESTRICT foreign key
ALTER TABLE agents ADD COLUMN IF NOT EXISTS community_id UUID;
ALTER TABLE agents ADD CONSTRAINT fk_agents_community FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_agents_community ON agents(community_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE agents DROP CONSTRAINT IF EXISTS fk_agents_community;
ALTER TABLE agents DROP COLUMN IF EXISTS community_id;
DROP TABLE IF EXISTS communities;
-- +goose StatementEnd
