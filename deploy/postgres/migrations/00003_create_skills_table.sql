-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    allowed_tools JSONB,
    denied_tools JSONB,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name);

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
