-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    brain JSONB NOT NULL,
    short_term_memory JSONB NOT NULL,
    long_term_memory JSONB NOT NULL,
    prompt_template UUID REFERENCES prompt_templates(id) ON DELETE SET NULL,
    mcp_clients JSONB NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_agents_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_agents_tenant_id ON agents(tenant_id);

-- Add foreign key constraint to agent_skills mapping table
ALTER TABLE agent_skills 
ADD CONSTRAINT fk_agent_skills_agent 
FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE agent_skills DROP CONSTRAINT IF EXISTS fk_agent_skills_agent;
DROP TABLE IF EXISTS agents;
-- +goose StatementEnd
