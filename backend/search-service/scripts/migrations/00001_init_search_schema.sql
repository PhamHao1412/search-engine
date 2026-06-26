-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS search_synonyms (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    keyword VARCHAR(255) NOT NULL,
    synonym VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_tenant_synonym UNIQUE(tenant_id, keyword, synonym)
);

CREATE TABLE IF NOT EXISTS search_logs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    query VARCHAR(255) NOT NULL,
    normalized_query VARCHAR(255) NOT NULL,
    result_count INTEGER NOT NULL DEFAULT 0,
    searched_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS click_logs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    search_log_id UUID NOT NULL REFERENCES search_logs(id) ON DELETE CASCADE,
    query VARCHAR(255) NOT NULL,
    product_id UUID NOT NULL,
    clicked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS spellcheck_dictionary (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    typo_word VARCHAR(255) NOT NULL,
    correct_word VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_tenant_spellcheck UNIQUE(tenant_id, typo_word, correct_word)
);

CREATE TABLE IF NOT EXISTS ai_suggestions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    suggestion_type VARCHAR(100) NOT NULL, -- 'synonym', 'spellcheck'
    source_value VARCHAR(255) NOT NULL,
    suggested_value VARCHAR(255) NOT NULL,
    confidence_score DECIMAL(5, 4) NOT NULL DEFAULT 0.0000,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected'
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_search_synonyms_keyword ON search_synonyms(tenant_id, keyword);
CREATE INDEX IF NOT EXISTS idx_search_logs_query ON search_logs(tenant_id, normalized_query);
CREATE INDEX IF NOT EXISTS idx_click_logs_search_id ON click_logs(search_log_id);
CREATE INDEX IF NOT EXISTS idx_spellcheck_typo ON spellcheck_dictionary(tenant_id, typo_word);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS ai_suggestions;
DROP TABLE IF EXISTS spellcheck_dictionary;
DROP TABLE IF EXISTS click_logs;
DROP TABLE IF EXISTS search_logs;
DROP TABLE IF EXISTS search_synonyms;
-- +goose StatementEnd
