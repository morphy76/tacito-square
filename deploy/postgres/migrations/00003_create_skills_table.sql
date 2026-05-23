-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    allowed_tools JSONB,
    denied_tools JSONB,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_skills_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_skills_tenant_id ON skills(tenant_id);

CREATE TABLE IF NOT EXISTS skill_mcp_servers (
    skill_id UUID REFERENCES skills(id) ON DELETE CASCADE,
    mcp_server_id UUID REFERENCES mcp_servers(id) ON DELETE CASCADE,
    PRIMARY KEY (skill_id, mcp_server_id)
);

CREATE TABLE IF NOT EXISTS agent_skills (
    agent_id UUID NOT NULL,
    skill_id UUID REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (agent_id, skill_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS agent_skills;
DROP TABLE IF EXISTS skill_mcp_servers;
DROP TABLE IF EXISTS skills;
-- +goose StatementEnd
