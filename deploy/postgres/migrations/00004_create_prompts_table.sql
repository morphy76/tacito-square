-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS prompt_templates (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    role VARCHAR(50) NOT NULL,
    version INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_name_version UNIQUE (name, version)
);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_name ON prompt_templates(name);

CREATE TABLE IF NOT EXISTS prompt_collections (
    id UUID PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_prompt_collections_name ON prompt_collections(name);

CREATE TABLE IF NOT EXISTS prompt_collection_templates (
    prompt_collection_id UUID REFERENCES prompt_collections(id) ON DELETE CASCADE,
    prompt_template_id UUID REFERENCES prompt_templates(id) ON DELETE CASCADE,
    PRIMARY KEY (prompt_collection_id, prompt_template_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS prompt_collection_templates;
DROP TABLE IF EXISTS prompt_collections;
DROP TABLE IF EXISTS prompt_templates;
-- +goose StatementEnd
