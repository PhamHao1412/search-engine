# Swift Search Engine - System Design

## 1. Mục tiêu

Thiết kế hệ thống tìm kiếm sản phẩm đa ngôn ngữ dành cho nền tảng thương mại điện tử.

Các tính năng chính:

* Product Search (Tìm kiếm đa ngôn ngữ, xếp hạng)
* Autocomplete (Gợi ý tự động hoàn thành dựa trên prefix/n-gram)
* Spellcheck (Sửa lỗi chính tả từ truy vấn thông qua từ điển và phrase suggester)
* Synonym Expansion (Mở rộng truy vấn qua từ đồng nghĩa tại search-time)
* Multilingual Search (Dịch nội dung tự động và hỗ trợ tìm kiếm vi, en, th)
* Ranking Engine (Xếp hạng sản phẩm theo độ liên quan và chỉ số kinh doanh)
* Search Analytics & Click Tracking (Theo dõi lịch sử tìm kiếm và lượt click bất đồng bộ)
* AI Suggestion Engine (Sinh từ đồng nghĩa & lỗi chính tả gợi ý tự động bằng AI)

---

## 2. Kiến trúc tổng thể

```text
                        Seller
                           |
                           v
               Product Service (serverd)
                           |
            +--------------+--------------+
            |                             |
            v                             v
       PostgreSQL                    Message Broker (Kafka)
  (product_svc.products)                  | (ProductCreated/Updated Event)
                                          v
                               Search Ingestion Worker (workerd)
                                          |
                        +-----------------+-----------------+
                        |                 |                 |
                        v                 v                 v
                Google Translate   Text Hash Check     OpenSearch
               (Dịch en, th)       (Tránh lặp dịch)   (products Index)
                        |                 |                 |
                        +-----------------+-----------------+
                                          |
                                          v
                                     OpenSearch


====================================================


                        Buyer
                           |
                           v
                       Search UI (React/Vite)
                           |
                           v
                Search Service (serverd)
                           |
            +--------------+--------------+
            |                             |
            v                             v
       Redis Cache                   OpenSearch
    (Search/Suggest)                      |
            |                             v
            +---------------------> Search Result
                                          |
                                          v
                                 Goroutines (Async) -> PostgreSQL (search_svc)


====================================================


                 Search Service (workerd offline job)
                           |
                           v
                         OpenAI (gpt-4o-mini)
                           |
                           v
                 AI Suggestions (Pending)
                           |
                           v
                 Admin UI (Duyệt/Bác bỏ)


====================================================


                 Admin UI (React/Vite) -> Trợ lý AI
                            |
                            v
                  Search Service (serverd) <---> PostgreSQL (assistant_conversations/messages)
                            |
                            v
                    OpenAI API (gpt-4o-mini)
```

---

## 3. Thành phần hệ thống

### Frontend

* React
* Vite
* TypeScript

#### Chức năng

* **Storefront UI**: Cho phép Buyer tìm kiếm sản phẩm, gợi ý autocomplete, chuyển đổi ngôn ngữ (vi, en, th).
* **Admin UI**: Quản trị viên quản lý danh mục từ đồng nghĩa (synonyms), sửa lỗi chính tả (spellcheck) và phê duyệt/bác bỏ đề xuất gợi ý của AI.

---

### Backend

* Golang (Gin Framework)
* GORM (Object Relational Mapping)

#### Services

* **Product Service (`serverd`)**: Tiếp nhận các nghiệp vụ CRUD sản phẩm từ Seller, đồng bộ Postgres và phát sự kiện sang Kafka.
* **Search Service (`serverd` & `workerd`)**:
  * `serverd`: API phục vụ tìm kiếm, gợi ý và quản trị Admin.
  * `workerd`: Background worker tiêu thụ sự kiện từ Kafka để ingest dữ liệu vào OpenSearch và chạy offline batch job để sinh gợi ý AI từ logs.

---

### Storage

#### PostgreSQL

Source of Truth (SoT) cho catalog và lưu trữ dữ liệu phân tích/logs:

* **Schema `product_svc`**:
  * `tenants`: Quản lý các tenants/marketplace độc lập.
  * `products`: Thông tin gốc của sản phẩm (ngôn ngữ gốc, giá, số lượng, v.v.).
  * `product_translations`: Bản dịch tên/mô tả sản phẩm phục vụ đa ngôn ngữ.
* **Schema `search_svc`**:
  * `search_logs`: Nhật ký tìm kiếm của người dùng.
  * `click_logs`: Nhật ký click sản phẩm (kết nối với `search_logs` qua `search_log_id`).
  * `search_synonyms`: Từ điển từ đồng nghĩa được quản trị viên thiết lập.
  * `search_translations`: Bản dịch các từ khóa tìm kiếm tĩnh nhằm hỗ trợ query expansion.
  * `spellcheck_dictionary`: Từ điển sửa lỗi chính tả (typo word -> correct word).
  * `ai_suggestions`: Các gợi ý do AI đề xuất đang chờ duyệt hoặc đã xử lý.
  * `search_sync_jobs`: Theo dõi trạng thái đồng bộ và mã băm text (`text_hash`) để tối ưu hóa việc ingest.
  * `assistant_conversations`: Quản lý các phiên hội thoại của trợ lý AI.
  * `assistant_messages`: Lưu lịch sử tin nhắn và trạng thái duyệt đề xuất của từng hội thoại.


---

#### Redis

Lớp lưu trữ đệm (Caching):

* **Search Cache**: Cache kết quả tìm kiếm đa ngôn ngữ theo từ khóa đã normalize để giảm tải OpenSearch.
* **Suggest Cache**: Cache gợi ý autocomplete.
* **Synonym / Translation Cache**: Tăng tốc độ truy xuất từ điển mở rộng câu truy vấn tại Search-time.

---

#### OpenSearch

Động cơ tìm kiếm (Search Engine):

* **Product Search Index (`products`)**: Lưu trữ tài liệu phẳng (flat document) chứa các trường tiếng Việt, Anh, Thái và các trường metadata khác.
* **Custom Analyzers**: 
  * `vi_ascii_analyzer`: Loại bỏ dấu tiếng Việt (asciifolding) và chuyển chữ thường để tìm kiếm không dấu/có dấu đồng nhất.
  * `english` & `thai`: Phân tích từ gốc tiếng Anh và tách câu tiếng Thái.
  * `autocomplete_analyzer`: Dùng n-gram filter (kích thước 2-10) để phân tách text phục vụ tìm kiếm gợi ý tức thời.

---

## 4. Luồng Product Ingestion

1. Seller tạo hoặc cập nhật sản phẩm qua Product API.
2. Product Service lưu thông tin gốc vào PostgreSQL (schema `product_svc`).
3. Product Service đẩy sự kiện `ProductCreated` / `ProductUpdated` vào Kafka topic `product-ingestion`.
4. Search Ingestion Worker (`workerd`) tiêu thụ sự kiện từ Kafka.
5. Kiểm tra mã băm văn bản (Text Hash Check) dựa trên tên và mô tả sản phẩm:
   * **Nếu không thay đổi** (và job trước đó thành công): Bỏ qua bước dịch thuật để tiết kiệm chi phí/tài nguyên. Chỉ thực hiện cập nhật một phần (`UpdateProduct`) các trường metadata (`brand`, `price`, `image_url`, `inventory`, `featured`, `status`) lên OpenSearch và cập nhật Redis Cache.
   * **Nếu thay đổi**:
     a. Gọi Google Translate dịch tiêu đề và mô tả sang các ngôn ngữ đích (English, Thai) từ ngôn ngữ gốc (`original_language`).
     b. Ghi nhận bản dịch mới vào bảng `product_svc.product_translations`.
6. Xây dựng tài liệu tìm kiếm phẳng (Search Document) bao gồm các thông tin gốc, bản dịch, và trường `suggest` (được ghép từ tên sản phẩm tiếng Việt, Anh, Thái để phục vụ autocomplete).
7. Index tài liệu đầy đủ vào OpenSearch.
8. Nếu gặp lỗi: Update trạng thái job thành `failed_opensearch` hoặc `failed_translation` trong `search_sync_jobs` để hỗ trợ retry.

---

## 5. Luồng Search

1. Người dùng nhập từ khóa trên Search UI.
2. Search API nhận request tìm kiếm kèm `X-Tenant-ID` và `X-Language-Key`.
3. Chuẩn hóa từ khóa (Lowercasing, loại bỏ khoảng trắng thừa, validate độ dài < 100 ký tự).
4. Kiểm tra Redis Cache theo khóa `search:{tenant_id}:{normalized_query}:{lang}`:
   * **Nếu Cache Hit**: Trả kết quả ngay cho người dùng. Đẩy logging bất đồng bộ qua Goroutine.
   * **Nếu Cache Miss**:
     a. **Spellcheck**: Kiểm tra gợi ý sửa lỗi chính tả từ từ điển cục bộ (`spellcheck_dictionary`). Nếu khớp, tự động sửa đổi từ khóa gốc.
     b. **Synonym & Translation Expansion**: Tải từ điển đồng nghĩa (`search_synonyms`) và từ điển dịch thuật (`search_translations`), gộp chung lại để tiến hành phân tách và mở rộng truy vấn (Search-time Expansion).
     c. **Query OpenSearch**: Thực hiện truy vấn Multi-Match trên các trường ngôn ngữ tương ứng (`product_name_{lang}`, `description_{lang}`, v.v.) kết hợp với các từ khóa đã mở rộng (Synonym được áp dụng boost `0.6`).
     d. **Phrase Match Boost**: Thêm truy vấn `match_phrase` trên trường tên sản phẩm của ngôn ngữ tương ứng để tăng điểm ưu tiên cho các kết quả khớp chính xác cụm từ.
     e. **Ranking Engine**: Sử dụng `function_score` để nhân thêm trọng số:
        * Boost sản phẩm nổi bật (`featured: true`).
        * Giảm điểm (decay) sản phẩm hết hàng (`inventory: 0`).
     f. **Spellcheck Fallback**: Nếu không có sửa lỗi chính tả từ dictionary, lấy đề xuất sửa lỗi từ phrase suggester đi kèm trong phản hồi của OpenSearch.
     g. **Save Cache**: Lưu kết quả tìm kiếm vào Redis.
5. Trả kết quả tìm kiếm cho người dùng.
6. Đẩy sự kiện ghi nhận Analytics (Search Log, Click Log) một cách bất đồng bộ qua Goroutines ghi vào PostgreSQL để không chặn luồng phản hồi UI.

---

## 6. Non Functional Requirements

### Performance

* Search latency: `< 50ms`
* Autocomplete latency: `< 10ms`

---

### Availability

* Nếu OpenSearch lỗi: Trả cache gần nhất từ Redis hoặc fallback thông báo lỗi hệ thống tạm thời không khả dụng nhưng không làm sập ứng dụng.

---

### Scalability

* Hỗ trợ tìm kiếm thời gian thực cho tối thiểu `500 concurrent users` nhờ cơ chế Redis caching và xử lý ghi log bất đồng bộ.

---

### Multi Tenancy

* Mọi dữ liệu (sản phẩm, logs, từ điển, cache) đều bắt buộc gắn kèm thuộc tính `tenant_id` để phân vùng dữ liệu an toàn giữa các đối tác/marketplace khác nhau.
