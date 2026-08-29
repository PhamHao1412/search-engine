-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS search_sync_jobs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    product_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'success', 'failed_translation', 'failed_ai', 'failed_opensearch'
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_product_sync_job UNIQUE(product_id)
);

CREATE INDEX IF NOT EXISTS idx_search_sync_jobs_status ON search_sync_jobs(status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS search_sync_jobs;
-- +goose StatementEnd
