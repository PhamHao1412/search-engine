-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS search_svc.search_translations (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    keyword VARCHAR(255) NOT NULL,
    lang_code VARCHAR(10) NOT NULL,
    translation VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_tenant_keyword_lang UNIQUE(tenant_id, keyword, lang_code)
);

CREATE INDEX IF NOT EXISTS idx_search_translations_kw ON search_svc.search_translations(tenant_id, keyword);

-- Seed data for testing multilingual search
-- Demo Tenant ID: d3b07384-d113-4956-a5db-251d50c18d01
INSERT INTO search_svc.search_translations (id, tenant_id, keyword, lang_code, translation, status)
VALUES 
    -- coffee
    ('a1111111-1111-1111-1111-111111111111', 'd3b07384-d113-4956-a5db-251d50c18d01', 'coffee', 'vi', 'cà phê', 'active'),
    ('a1111111-1111-1111-1111-111111111112', 'd3b07384-d113-4956-a5db-251d50c18d01', 'coffee', 'th', 'กาแฟ', 'active'),
    ('a1111111-1111-1111-1111-111111111113', 'd3b07384-d113-4956-a5db-251d50c18d01', 'cà phê', 'en', 'coffee', 'active'),
    ('a1111111-1111-1111-1111-111111111114', 'd3b07384-d113-4956-a5db-251d50c18d01', 'cà phê', 'th', 'กาแฟ', 'active'),
    ('a1111111-1111-1111-1111-111111111115', 'd3b07384-d113-4956-a5db-251d50c18d01', 'กาแฟ', 'vi', 'cà phê', 'active'),
    ('a1111111-1111-1111-1111-111111111116', 'd3b07384-d113-4956-a5db-251d50c18d01', 'กาแฟ', 'en', 'coffee', 'active'),
    
    -- keyboard
    ('a2222222-2222-2222-2222-222222222221', 'd3b07384-d113-4956-a5db-251d50c18d01', 'keyboard', 'vi', 'bàn phím', 'active'),
    ('a2222222-2222-2222-2222-222222222222', 'd3b07384-d113-4956-a5db-251d50c18d01', 'keyboard', 'th', 'คีย์บอร์ด', 'active'),
    ('a2222222-2222-2222-2222-222222222223', 'd3b07384-d113-4956-a5db-251d50c18d01', 'bàn phím', 'en', 'keyboard', 'active'),
    ('a2222222-2222-2222-2222-222222222224', 'd3b07384-d113-4956-a5db-251d50c18d01', 'bàn phím', 'th', 'คีย์บอร์ด', 'active'),
    
    -- mouse
    ('a3333333-3333-3333-3333-333333333331', 'd3b07384-d113-4956-a5db-251d50c18d01', 'mouse', 'vi', 'chuột', 'active'),
    ('a3333333-3333-3333-3333-333333333332', 'd3b07384-d113-4956-a5db-251d50c18d01', 'mouse', 'th', 'เมาส์', 'active'),
    ('a3333333-3333-3333-3333-333333333333', 'd3b07384-d113-4956-a5db-251d50c18d01', 'chuột', 'en', 'mouse', 'active'),
    ('a3333333-3333-3333-3333-333333333334', 'd3b07384-d113-4956-a5db-251d50c18d01', 'chuột', 'th', 'เมาส์', 'active')
ON CONFLICT (tenant_id, keyword, lang_code) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS search_svc.search_translations;
-- +goose StatementEnd
