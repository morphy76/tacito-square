-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS mcp_servers (
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
    CONSTRAINT unique_mcp_servers_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_tenant_id ON mcp_servers(tenant_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS mcp_servers;
-- +goose StatementEnd
