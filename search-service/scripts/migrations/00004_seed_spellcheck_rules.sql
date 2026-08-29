-- +goose Up
-- +goose StatementBegin
INSERT INTO search_svc.spellcheck_dictionary (id, tenant_id, typo_word, correct_word, status, created_at, updated_at) VALUES
('4e2f8902-3b2d-45db-9ee4-c9c2c62c2f01', 'd3b07384-d113-4956-a5db-251d50c18d01', 'ako', 'akko', 'active', NOW(), NOW()),
('4e2f8902-3b2d-45db-9ee4-c9c2c62c2f02', 'd3b07384-d113-4956-a5db-251d50c18d01', 'logitek', 'logitech', 'active', NOW(), NOW()),
('4e2f8902-3b2d-45db-9ee4-c9c2c62c2f03', 'd3b07384-d113-4956-a5db-251d50c18d01', 'chuot', 'chuột', 'active', NOW(), NOW()),
('4e2f8902-3b2d-45db-9ee4-c9c2c62c2f04', 'd3b07384-d113-4956-a5db-251d50c18d01', 'ban phim', 'bàn phím', 'active', NOW(), NOW()),

('4e2f8902-3b2d-45db-9ee4-c9c2c62c2f05', 'a1a2a3a4-b1b2-c1c2-d1d2-e1e2e3e4e5e6', 'son duong', 'son dưỡng', 'active', NOW(), NOW()),
('4e2f8902-3b2d-45db-9ee4-c9c2c62c2f06', 'a1a2a3a4-b1b2-c1c2-d1d2-e1e2e3e4e5e6', 'my pham', 'mỹ phẩm', 'active', NOW(), NOW()),
('4e2f8902-3b2d-45db-9ee4-c9c2c62c2f07', 'a1a2a3a4-b1b2-c1c2-d1d2-e1e2e3e4e5e6', 'lip balm', 'son dưỡng', 'active', NOW(), NOW())
ON CONFLICT (tenant_id, typo_word, correct_word) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM search_svc.spellcheck_dictionary WHERE id IN (
  '4e2f8902-3b2d-45db-9ee4-c9c2c62c2f01',
  '4e2f8902-3b2d-45db-9ee4-c9c2c62c2f02',
  '4e2f8902-3b2d-45db-9ee4-c9c2c62c2f03',
  '4e2f8902-3b2d-45db-9ee4-c9c2c62c2f04',
  '4e2f8902-3b2d-45db-9ee4-c9c2c62c2f05',
  '4e2f8902-3b2d-45db-9ee4-c9c2c62c2f06',
  '4e2f8902-3b2d-45db-9ee4-c9c2c62c2f07'
);
-- +goose StatementEnd
