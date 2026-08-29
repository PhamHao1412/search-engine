-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS daily_query_analytics (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    query VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    search_count INTEGER NOT NULL DEFAULT 0,
    click_count INTEGER NOT NULL DEFAULT 0,
    zero_result_count INTEGER NOT NULL DEFAULT 0,
    sum_click_position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_tenant_query_date UNIQUE(tenant_id, query, date)
);

CREATE TABLE IF NOT EXISTS daily_category_analytics (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    category_id UUID NOT NULL,
    category_name VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    search_count INTEGER NOT NULL DEFAULT 0,
    click_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_tenant_category_date UNIQUE(tenant_id, category_id, date)
);

CREATE INDEX IF NOT EXISTS idx_daily_query_date ON daily_query_analytics(tenant_id, date);
CREATE INDEX IF NOT EXISTS idx_daily_category_date ON daily_category_analytics(tenant_id, date);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS daily_category_analytics;
DROP TABLE IF EXISTS daily_query_analytics;
-- +goose StatementEnd
