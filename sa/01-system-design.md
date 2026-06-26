# SA/system-design.md

# Amaze Search Engine - System Design

## 1. Mục tiêu

Thiết kế hệ thống tìm kiếm sản phẩm đa ngôn ngữ dành cho nền tảng thương mại điện tử.

Các tính năng chính:

* Product Search
* Autocomplete
* Spellcheck
* Synonym Expansion
* Multilingual Search
* Ranking Engine
* Search Analytics
* AI Suggestion Engine

---

# 2. Kiến trúc tổng thể

```text
                        Seller
                           |
                           v
                    Product API
                           |
            +--------------+--------------+
            |                             |
            v                             v
       PostgreSQL                    Event Broker (RabbitMQ/Kafka)
      (Source of Truth)                   | (ProductCreated/Updated Event)
                                          v
                              Search Ingestion Pipeline
                                          |
                        +-----------------+-----------------+
                        |                 |                 |
                        v                 v                 v
                Google Translate     Text Hash Check   OpenSearch (Native Analyzers)
                        |           (Avoid duplicate)       |
                        +-----------------+-----------------+
                                          |
                                          v
                                     OpenSearch


====================================================


                        Buyer
                           |
                           v
                       Search UI
                           |
                           v
                       Search API
                           |
            +--------------+--------------+
            |                             |
            v                             v
       Redis Cache                   OpenSearch
     (Query/Autocomplete)                 |
                                          v
                                   Search Result
                                          |
                                          v
                                    Event Broker -> Analytics Consumer -> PostgreSQL


====================================================


                    Analytics Job
                           |
                           v
                        OpenAI
                           |
                           v
                    AI Suggestions
                           |
                           v
                       Admin UI
```

---

# 3. Thành phần hệ thống

## Frontend

* React
* Vite
* TypeScript
* TailwindCSS

### Chức năng

* Search UI
* Admin UI
* Analytics UI

---

## Backend

* Golang
* Gin Framework

### Modules

* Product Module
* Search Module
* Analytics Module
* AI Module

---

## Storage

### PostgreSQL

Source of Truth (SoT).

Lưu:

* Products (Bảng sản phẩm: thêm thông tin giá, ảnh, trạng thái đặc biệt, ngôn ngữ gốc)
* Categories (Bảng danh mục sản phẩm phục vụ hiển thị/tìm kiếm)
* Search Logs (Nhật ký tìm kiếm thô)
* Click Logs (Nhật ký click liên kết với search log)
* Synonym Dictionary (Từ điển từ đồng nghĩa phục vụ search-time expansion)
* Translation Dictionary (Từ điển dịch thuật tĩnh)
* Spellcheck Dictionary (Từ điển sửa lỗi chính tả đã phê duyệt)
* AI Suggestions (Đề xuất gợi ý từ AI)

---

### Redis

Lưu:

* Search Cache (Lưu kết quả tìm kiếm theo từ khóa đã normalize để giảm tải OpenSearch)
* Autocomplete Cache / Synonym Dictionary Cache

---

### OpenSearch

Lưu:

* Product Search Documents (Dữ liệu thô phân vùng theo tenant và ngôn ngữ)
* Search Index & Native Language Analyzers (Bộ phân tích tiếng Việt, tiếng Anh, tiếng Thái)
* Completion Suggester (Gợi ý tự động có lọc theo tenant context)

---

# 4. Luồng Product Ingestion

1. Seller tạo hoặc cập nhật sản phẩm.
2. Product API lưu thông tin vào PostgreSQL.
3. Product API đẩy event `ProductCreated` hoặc `ProductUpdated` vào Event Broker.
4. Search Ingestion Pipeline tiêu thụ event từ queue.
5. Kiểm tra mã băm văn bản (Text Hash Check) của tên/mô tả sản phẩm:
   * Nếu không thay đổi: Bỏ qua bước dịch thuật và sinh keyword AI để tối ưu hóa hiệu năng/chi phí.
   * Nếu thay đổi:
     a. Gọi Google Translate dịch tiêu đề/mô tả sang các ngôn ngữ đích (English, Thai) từ ngôn ngữ gốc (`original_language`).
     b. Gọi AI Service (OpenAI API chạy bất đồng bộ trong pipeline) để phân tích tiêu đề/mô tả và tự động sinh danh sách từ khóa tìm kiếm mở rộng (AI Search Tags) như từ viết tắt, từ lóng hoặc từ đồng nghĩa thương mại.
6. Tạo tài liệu tìm kiếm (Search Document) bao gồm các trường văn bản thô đa ngôn ngữ và danh sách nhãn tìm kiếm mở rộng `search_tags` do AI sinh ra.
7. Đẩy tài liệu tìm kiếm vào OpenSearch để OpenSearch tự động phân tích và tạo chỉ mục (Index-time Tokenization).
8. Nếu gặp lỗi kết nối dịch thuật hoặc OpenSearch: Chuyển tin nhắn vào Retry Queue với cơ chế Exponential Backoff, tối đa 5 lần trước khi chuyển sang Dead Letter Queue (DLQ).

---

# 5. Luồng Search

1. Người dùng nhập từ khóa tìm kiếm trên Search UI.
2. Search API nhận request tìm kiếm.
3. Chuẩn hóa từ khóa (Normalize Query: lowercase, trim, remove special characters).
4. Kiểm tra Redis Cache theo khóa `search:{tenant_id}:{normalized_query}`:
   * **Nếu Cache Hit**: Lấy kết quả trả về ngay cho người dùng. Đẩy sự kiện ghi nhận Analytics bất đồng bộ vào Event Broker.
   * **Nếu Cache Miss**:
     a. **Spellcheck**: Kiểm tra gợi ý sửa lỗi chính tả từ từ điển đã lưu hoặc OpenSearch.
     b. **Synonym Expansion**: Tra cứu các từ đồng nghĩa của từ khóa (từ Redis Cache / PostgreSQL) để mở rộng query tại Search-time.
     c. **Translation Expansion**: Tra cứu từ điển dịch thuật để mở rộng từ khóa sang các ngôn ngữ khác.
     d. **Query OpenSearch**: Thực hiện truy vấn trên các trường ngôn ngữ tương ứng (`product_name_vi`, `product_name_en`, etc.).
     e. **Ranking**: Sắp xếp kết quả dựa trên các tiêu chí (Exact Match, Field Weight, Featured Boost, Inventory).
     f. **Save Cache**: Lưu kết quả tìm kiếm vào Redis Cache.
5. Trả kết quả tìm kiếm cho người dùng.
6. Đẩy sự kiện ghi nhận Analytics (Search Log, Click Log) một cách bất đồng bộ qua Event Broker để đảm bảo độ trễ thấp cho người dùng.

---

# 6. Non Functional Requirements

## Performance

Search latency:

< 50ms

Autocomplete latency:

< 5ms

---

## Availability

Nếu OpenSearch lỗi:

* Trả cache gần nhất từ Redis

---

## Scalability

Hỗ trợ:

* 500 concurrent users

---

## Multi Tenancy

Mọi dữ liệu đều gắn:

tenant_id

để hỗ trợ nhiều marketplace trong tương lai.
