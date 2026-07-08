-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS search_svc.assistant_conversations (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS search_svc.assistant_messages (
    id VARCHAR(36) PRIMARY KEY,
    conversation_id VARCHAR(36) NOT NULL REFERENCES search_svc.assistant_conversations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    proposed_actions TEXT,
    action_states TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_conversations_tenant ON search_svc.assistant_conversations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON search_svc.assistant_messages(conversation_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS search_svc.assistant_messages;
DROP TABLE IF EXISTS search_svc.assistant_conversations;
-- +goose StatementEnd
