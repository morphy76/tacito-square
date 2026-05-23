-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS mcp_servers (
    id UUID PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    transport VARCHAR(50) NOT NULL,
    command VARCHAR(500),
    args JSONB,
    env JSONB,
    url VARCHAR(500),
    auth_secret_ref VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_name ON mcp_servers(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS mcp_servers;
-- +goose StatementEnd
