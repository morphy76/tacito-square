-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS llm_bindings (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    provider VARCHAR(50) NOT NULL,
    api_base_url VARCHAR(500) NOT NULL,
    api_key_secret_ref VARCHAR(255) NOT NULL,
    default_model VARCHAR(255) NOT NULL,
    default_temperature DOUBLE PRECISION NOT NULL,
    default_max_tokens INTEGER NOT NULL,
    timeout_seconds INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_llm_bindings_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_llm_bindings_tenant_id ON llm_bindings(tenant_id);

CREATE TABLE IF NOT EXISTS mcp_clients (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    transport VARCHAR(50) NOT NULL,
    command VARCHAR(500),
    args JSONB,
    env JSONB,
    url VARCHAR(500),
    auth_secret_ref VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_mcp_clients_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_mcp_clients_tenant_id ON mcp_clients(tenant_id);

CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL,
    content TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_skills_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_skills_tenant_id ON skills(tenant_id);

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

CREATE TABLE IF NOT EXISTS skill_collection_skills (
    skill_collection_id UUID REFERENCES skill_collections(id) ON DELETE CASCADE,
    skill_id UUID REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (skill_collection_id, skill_id)
);

CREATE TABLE IF NOT EXISTS prompt_templates (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_prompt_templates_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_tenant_id ON prompt_templates(tenant_id);

CREATE TABLE IF NOT EXISTS prompt_collections (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_prompt_collections_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_prompt_collections_tenant_id ON prompt_collections(tenant_id);

CREATE TABLE IF NOT EXISTS prompt_collection_templates (
    prompt_collection_id UUID REFERENCES prompt_collections(id) ON DELETE CASCADE,
    prompt_template_id UUID REFERENCES prompt_templates(id) ON DELETE CASCADE,
    PRIMARY KEY (prompt_collection_id, prompt_template_id)
);

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
    community_id UUID REFERENCES communities(id) ON DELETE RESTRICT,
    CONSTRAINT unique_agents_tenant_name UNIQUE (tenant_id, name),
    CONSTRAINT check_agent_brain CHECK (
        (brain->>'model') IS NOT NULL AND (brain->>'model') <> '' AND
        (brain->>'endpoint') IS NOT NULL AND (brain->>'endpoint') <> '' AND
        (brain->>'credentials_secret') IS NOT NULL AND (brain->>'credentials_secret') <> ''
    )
);
CREATE INDEX IF NOT EXISTS idx_agents_tenant_id ON agents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_agents_community ON agents(community_id);

CREATE TABLE IF NOT EXISTS agent_skills (
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    skill_id UUID REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (agent_id, skill_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS agent_skills;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS communities;
DROP TABLE IF EXISTS prompt_collection_templates;
DROP TABLE IF EXISTS prompt_collections;
DROP TABLE IF EXISTS prompt_templates;
DROP TABLE IF EXISTS skill_collection_skills;
DROP TABLE IF EXISTS skill_collections;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS mcp_clients;
DROP TABLE IF EXISTS llm_bindings;
-- +goose StatementEnd
