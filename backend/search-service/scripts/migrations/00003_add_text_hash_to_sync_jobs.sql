-- +goose Up
-- +goose StatementBegin
ALTER TABLE search_svc.search_sync_jobs ADD COLUMN IF NOT EXISTS text_hash VARCHAR(64);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE search_svc.search_sync_jobs DROP COLUMN IF EXISTS text_hash;
-- +goose StatementEnd
