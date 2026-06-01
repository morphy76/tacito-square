-- +goose Up
-- +goose StatementBegin
ALTER TABLE skills ADD COLUMN IF NOT EXISTS content TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE skills DROP COLUMN IF EXISTS content;
-- +goose StatementEnd
