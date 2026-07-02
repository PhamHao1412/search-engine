# Kỹ năng vận hành cho AI Agent (Agent Skills & Commands)

Tài liệu này chứa các hướng dẫn chạy lệnh, kiểm tra hạ tầng, vận hành dữ liệu và kiểm thử dự án **SwiftSearch Engine** dành cho AI Agent.

---

## 1. Kỹ năng Database Migration (Goose)

Khi cần cập nhật cấu trúc cơ sở dữ liệu (PostgreSQL), Agent phải sử dụng công cụ **Goose**. 

*   Thư mục chứa migration: `backend/scripts/migrations/`
*   Chuỗi kết nối PostgreSQL local: `postgres://postgres:postgrespassword@localhost:5438/swiftsearch_search?sslmode=disable&search_path=public`

### Các lệnh phổ biến:

*   **Tạo mới một file migration**:
    ```bash
    goose -dir backend/scripts/migrations create <tên_migration> sql
    ```
*   **Chạy toàn bộ migrations lên (Up)**:
    ```bash
    goose -dir backend/scripts/migrations postgres "postgres://postgres:postgrespassword@localhost:5438/swiftsearch_search?sslmode=disable&search_path=public" up
    ```
*   **Hạ phiên bản migration xuống 1 cấp (Down)**:
    ```bash
    goose -dir backend/scripts/migrations postgres "postgres://postgres:postgrespassword@localhost:5438/swiftsearch_search?sslmode=disable&search_path=public" down
    ```
*   **Kiểm tra trạng thái các migrations**:
    ```bash
    goose -dir backend/scripts/migrations postgres "postgres://postgres:postgrespassword@localhost:5438/swiftsearch_search?sslmode=disable&search_path=public" status
    ```

---

## 2. Kỹ năng Kiểm tra hạ tầng (Health Check)

Trước khi chạy code hoặc chạy test tích hợp, Agent cần xác minh tính khả dụng của hạ tầng Docker:

*   **Kiểm tra các containers đang chạy**:
    ```bash
    docker compose ps
    ```
*   **Kiểm tra kết nối Redis**:
    ```bash
    docker exec -it swiftsearch-redis redis-cli ping
    # Kết quả mong muốn: PONG
    ```
*   **Kiểm tra kết nối OpenSearch**:
    ```bash
    curl -I http://localhost:9200
    # Kết quả mong muốn: HTTP/1.1 200 OK
    ```
*   **Kiểm tra Kafka Broker**:
    ```bash
    docker exec -it swiftsearch-kafka kafka-topics --bootstrap-server localhost:9092 --list
    ```

---

## 3. Kỹ năng Quản trị Index OpenSearch

Agent cần biết cách thiết lập cấu hình mapping và analyzers thủ công hoặc thông qua code chạy lúc khởi động:

*   **Tạo Index `products_v1` kèm Analyzers**:
    ```bash
    curl -X PUT "http://localhost:9200/products_v1" -H 'Content-Type: application/json' -d'
    {
      "settings": {
        "index": {
          "number_of_shards": 1,
          "number_of_replicas": 0
        }
      },
      "mappings": {
        "properties": {
          "id": { "type": "keyword" },
          "tenant_id": { "type": "keyword" },
          "product_name_vi": { "type": "text", "analyzer": "standard" },
          "product_name_en": { "type": "text", "analyzer": "english" },
          "product_name_th": { "type": "text", "analyzer": "thai" },
          "search_tags": { "type": "text", "analyzer": "standard" },
          "inventory": { "type": "integer" },
          "featured": { "type": "boolean" },
          "suggest": {
            "type": "completion",
            "contexts": [
              {
                "name": "tenant_context",
                "type": "category",
                "path": "tenant_id"
              }
            ]
          }
        }
      }
    }'
    ```
*   **Thiết lập Alias `products` trỏ tới `products_v1`**:
    ```bash
    curl -X POST "http://localhost:9200/_aliases" -H 'Content-Type: application/json' -d'
    {
      "actions": [
        { "add": { "index": "products_v1", "alias": "products" } }
      ]
    }'
    ```

---

## 4. Kỹ năng Kiểm thử và Gửi sự kiện Kafka

Khi cần kiểm tra luồng truyền nhận sự kiện (Event Stream) mà không chạy toàn bộ hệ thống API:

*   **Tạo một Topic mới**:
    ```bash
    docker exec -it swiftsearch-kafka kafka-topics --bootstrap-server localhost:9092 --create --topic product-ingestion-events --partitions 1 --replication-factor 1
    ```
*   **Gửi thử một message (Producer)**:
    ```bash
    docker exec -it swiftsearch-kafka kafka-console-producer --bootstrap-server localhost:9092 --topic product-ingestion-events
    # Gõ nội dung JSON rồi ấn Enter:
    # {"event_id":"123", "event_type":"ProductCreated", "product_id":"uuid-test"}
    ```
*   **Theo dõi và kiểm tra message đến (Consumer)**:
    ```bash
    docker exec -it swiftsearch-kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic product-ingestion-events --from-beginning
    ```

---

## 5. Kỹ năng Kiểm thử Backend (Go Tests)

Để đảm bảo code chạy ổn định và không làm hỏng các luồng nghiệp vụ hiện tại:

*   **Chạy toàn bộ unit tests**:
    ```bash
    cd backend && go test -v ./...
    ```
*   **Chạy test cho một package cụ thể**:
    ```bash
    cd backend && go test -v ./internal/product/...
    ```
