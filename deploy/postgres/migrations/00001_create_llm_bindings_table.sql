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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS llm_bindings;
-- +goose StatementEnd
