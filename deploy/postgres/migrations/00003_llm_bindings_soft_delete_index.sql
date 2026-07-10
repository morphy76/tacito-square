-- +goose Up
-- +goose StatementBegin
ALTER TABLE llm_bindings DROP CONSTRAINT IF EXISTS unique_llm_bindings_tenant_name;
CREATE UNIQUE INDEX unique_llm_bindings_tenant_name_active ON llm_bindings (tenant_id, name) WHERE status <> 'inactive';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS unique_llm_bindings_tenant_name_active;
ALTER TABLE llm_bindings ADD CONSTRAINT unique_llm_bindings_tenant_name UNIQUE (tenant_id, name);
-- +goose StatementEnd
