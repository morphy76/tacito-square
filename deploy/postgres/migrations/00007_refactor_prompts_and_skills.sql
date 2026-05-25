-- +goose Up
-- +goose StatementBegin
-- 1. Drop role and version from prompt_templates
ALTER TABLE prompt_templates DROP CONSTRAINT IF EXISTS unique_tenant_name_version;
ALTER TABLE prompt_templates DROP COLUMN IF EXISTS role;
ALTER TABLE prompt_templates DROP COLUMN IF EXISTS version;
ALTER TABLE prompt_templates ADD CONSTRAINT unique_prompt_templates_tenant_name UNIQUE (tenant_id, name);

-- 2. Drop allowed_tools and denied_tools from skills, and drop skill_mcp_servers table
ALTER TABLE skills DROP COLUMN IF EXISTS allowed_tools;
ALTER TABLE skills DROP COLUMN IF EXISTS denied_tools;
DROP TABLE IF EXISTS skill_mcp_servers;

-- 3. Create skill_collections table
CREATE TABLE IF NOT EXISTS skill_collections (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_skill_collections_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_skill_collections_tenant_id ON skill_collections(tenant_id);

-- 4. Create skill_collection_skills table
CREATE TABLE IF NOT EXISTS skill_collection_skills (
    skill_collection_id UUID REFERENCES skill_collections(id) ON DELETE CASCADE,
    skill_id UUID REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (skill_collection_id, skill_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS skill_collection_skills;
DROP TABLE IF EXISTS skill_collections;

CREATE TABLE IF NOT EXISTS skill_mcp_servers (
    skill_id UUID REFERENCES skills(id) ON DELETE CASCADE,
    mcp_server_id UUID REFERENCES mcp_servers(id) ON DELETE CASCADE,
    PRIMARY KEY (skill_id, mcp_server_id)
);

ALTER TABLE skills ADD COLUMN allowed_tools JSONB;
ALTER TABLE skills ADD COLUMN denied_tools JSONB;

ALTER TABLE prompt_templates ADD COLUMN role VARCHAR(50) NOT NULL DEFAULT 'system';
ALTER TABLE prompt_templates ADD COLUMN version INT NOT NULL DEFAULT 1;
ALTER TABLE prompt_templates DROP CONSTRAINT IF EXISTS unique_prompt_templates_tenant_name;
ALTER TABLE prompt_templates ADD CONSTRAINT unique_tenant_name_version UNIQUE (tenant_id, name, version);
-- +goose StatementEnd
