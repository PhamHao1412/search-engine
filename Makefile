# Decoupled Makefile managing migrations and run commands for product and search services
PRODUCT_MIGRATIONS_DIR=product-service/scripts/migrations
SEARCH_MIGRATIONS_DIR=search-service/scripts/migrations

.PHONY: help docker-up docker-down docker-clean \
	migrate-product-create migrate-product-up migrate-product-down migrate-product-status \
	migrate-search-create migrate-search-up migrate-search-down migrate-search-status \
	run-product run-search-api run-search-worker test-product test-search test clean

help:
	@echo "Các lệnh khả dụng trong dự án SwiftSearch Engine (Decoupled Architecture):"
	@echo "  make docker-up             - Khởi chạy các containers hạ tầng (Redis, OpenSearch, Kafka, Kafka UI)"
	@echo "  make docker-down           - Dừng các containers"
	@echo "  make docker-clean          - Dừng các containers và xóa dữ liệu volumes"
	@echo "  make migrate-product-up    - Áp dụng migrations lên schema 'product' (dùng product-service/.env)"
	@echo "  make migrate-search-up     - Áp dụng migrations lên schema 'search' (dùng search-service/.env)"
	@echo "  make run-product           - Khởi chạy API Ingestion Server"
	@echo "  make run-search-api        - Khởi chạy API Search Server"
	@echo "  make run-search-worker     - Khởi chạy Kafka Consumer Worker"
	@echo "  make test-product          - Chạy tests cho product-service"
	@echo "  make test-search           - Chạy tests cho search-service"
	@echo "  make test                  - Chạy toàn bộ tests"

# Docker Operations
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-clean:
	docker compose down -v

# Migrations for Product Service (schema: product)
migrate-product-create:
	@if [ -z "$(name)" ]; then echo "Lỗi: Vui lòng truyền tham số name (Ví dụ: make migrate-product-create name=init_schema)"; exit 1; fi
	goose -dir $(PRODUCT_MIGRATIONS_DIR) create $(name) sql

migrate-product-up:
	@export $$(cat product-service/.env | grep -v '^#' | xargs) && \
	psql "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE" -c "CREATE SCHEMA IF NOT EXISTS $$DB_SCHEMA;" && \
	goose -dir $(PRODUCT_MIGRATIONS_DIR) postgres "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE&search_path=$$DB_SCHEMA" up

migrate-product-down:
	@export $$(cat product-service/.env | grep -v '^#' | xargs) && \
	goose -dir $(PRODUCT_MIGRATIONS_DIR) postgres "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE&search_path=$$DB_SCHEMA" down

migrate-product-status:
	@export $$(cat product-service/.env | grep -v '^#' | xargs) && \
	goose -dir $(PRODUCT_MIGRATIONS_DIR) postgres "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE&search_path=$$DB_SCHEMA" status

# Migrations for Search Service (schema: search)
migrate-search-create:
	@if [ -z "$(name)" ]; then echo "Lỗi: Vui lòng truyền tham số name (Ví dụ: make migrate-search-create name=init_schema)"; exit 1; fi
	goose -dir $(SEARCH_MIGRATIONS_DIR) create $(name) sql

migrate-search-up:
	@export $$(cat search-service/.env | grep -v '^#' | xargs) && \
	psql "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE" -c "CREATE SCHEMA IF NOT EXISTS $$DB_SCHEMA;" && \
	goose -dir $(SEARCH_MIGRATIONS_DIR) postgres "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE&search_path=$$DB_SCHEMA" up

migrate-search-down:
	@export $$(cat search-service/.env | grep -v '^#' | xargs) && \
	goose -dir $(SEARCH_MIGRATIONS_DIR) postgres "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE&search_path=$$DB_SCHEMA" down

migrate-search-status:
	@export $$(cat search-service/.env | grep -v '^#' | xargs) && \
	goose -dir $(SEARCH_MIGRATIONS_DIR) postgres "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE&search_path=$$DB_SCHEMA" status

# Run Services (Go executables read their own local .env inside their working directory)
run-product:
	cd product-service && go run cmd/serverd/main.go

run-search-api:
	cd search-service && go run cmd/serverd/main.go

run-search-worker:
	cd search-service && go run cmd/workerd/main.go

# Tests
test-product:
	cd product-service && env GOCACHE=.cache/go-build GOPATH=.go go test -v ./...

test-search:
	cd search-service && env GOCACHE=.cache/go-build GOPATH=.go go test -v ./...

test: test-product test-search

clean:
	rm -rf product-service/.tmp product-service/.cache product-service/.go
	rm -rf search-service/.tmp search-service/.cache search-service/.go
