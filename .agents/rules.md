# Quy tắc lập trình cho AI Agent (Agent Rules & Guidelines)

Tài liệu này định nghĩa các quy tắc thiết kế, phát triển và lập trình bắt buộc đối với mọi AI Agent khi làm việc trên dự án **Amaze Search Engine**.

---

## 1. Kiến trúc tổng thể & Công nghệ (Stack Choices)

Hệ thống được thiết kế theo mô hình tách biệt Write-model (GORM/Postgres) và Read-model (OpenSearch) thông qua luồng sự kiện bất đồng bộ (Kafka).

*   **Database**: PostgreSQL 15+.
*   **Database ORM**: Sử dụng **GORM** làm ORM chính trong Go. Không viết SQL thuần trực tiếp trong handler.
*   **Database Migration**: Sử dụng thư viện **Goose**. Các file migration dạng SQL thuần nằm trong thư mục `backend/scripts/migrations/` hoặc `backend/migrations/`.
*   **Event Broker**: Apache Kafka (KRaft mode).
    *   Địa chỉ local: `localhost:29092`
    *   Các topic chính: `product-ingestion-events`, `analytics-events`
*   **Search Engine**: OpenSearch (Single-node).
    *   Địa chỉ local: `http://localhost:9200`
    *   Sử dụng native analyzers bản địa cho từng ngôn ngữ (`product_name_vi` dùng `vi_analyzer`, `product_name_en` dùng `english`, `product_name_th` dùng `thai`).
*   **Cache**: Redis 7+.
    *   Địa chỉ local: `localhost:6379`
*   **Multi-tenancy**: Tất cả các API bắt buộc phải nhận diện `tenant_id` từ HTTP Header **`X-Tenant-ID`**. Không tự ý bỏ qua kiểm tra header này.

---

## 2. Cấu trúc thư mục chuẩn (Repository Layout)

Mọi code Backend và Frontend phải được đặt đúng thư mục được phê duyệt:

### Go Backend (`backend/`)
```text
backend/
├── cmd/
│   ├── server/           # Khởi chạy HTTP API Server (Gin)
│   └── worker/           # Khởi chạy Kafka Consumer & Batch Jobs (Analytics/AI)
├── internal/
│   ├── product/          # Nghiệp vụ quản lý sản phẩm
│   ├── search/           # Nghiệp vụ tìm kiếm & autocomplete
│   ├── analytics/        # Nhật ký tìm kiếm, click & báo cáo phân tích
│   ├── ai/               # offline batch job đề xuất bằng AI
│   └── platform/         # Kết nối Postgres (GORM), Redis, Kafka, OpenSearch
├── pkg/
│   ├── config/           # Đọc cấu hình từ env/yaml
│   └── helper/           # Các hàm tiện ích dùng chung
├── api/                  # Định nghĩa OpenAPI spec
├── configs/              # File cấu hình (config.yaml, v.v.)
├── deployments/          # File docker-compose.yml và cấu hình container
├── scripts/              # Các file shell script, SQL migrations (goose)
├── go.mod
└── go.sum
```
*Lưu ý*: Áp dụng quy tắc thiết kế Clean Architecture bên trong từng gói nghiệp vụ của `internal/` (ví dụ: `internal/product/domain`, `internal/product/usecase`, `internal/product/repository`, `internal/product/delivery`) để tránh lỗi phụ thuộc vòng (circular dependency) trong Go.

### React Frontend (`frontend/`)
Sử dụng **ReactJS + Vite + TailwindCSS** được tổ chức theo Feature-based:
```text
frontend/
├── src/
│   ├── app/              # Cấu hình routes, global providers
│   ├── components/       # Các component dùng chung (Button, Input, Modal, v.v.)
│   ├── features/         # Các chức năng lớn (search, admin-synonyms, admin-suggestions)
│   │   ├── search/
│   │   │   ├── components/
│   │   │   ├── hooks/
│   │   │   └── services/
│   ├── services/         # Axios client cấu hình Header X-Tenant-ID mặc định
│   ├── types/            # TypeScript definitions
│   └── main.tsx
```

---

## 3. Các quy tắc lập trình bắt buộc (Coding Rules)

### A. Đối với Backend (Go)
1.  **Không gọi OpenAI trong luồng realtime**: Chỉ gọi OpenAI API trong các Offline Batch Job (worker chạy nền). Luồng search trực tiếp của Buyer tuyệt đối không được chứa bất kỳ lệnh gọi LLM trực tiếp nào.
2.  **Xử lý lỗi Google Translate**: Luồng đăng sản phẩm của Seller không được bị lỗi (crash/fail response) nếu API dịch thuật bên thứ ba gặp lỗi. Phải lưu Postgres thành công trước, sau đó xử lý dịch thuật bất đồng bộ ở Consumer. Nếu dịch lỗi, ghi nhận trạng thái và đưa vào hàng đợi thử lại (Kafka retry-topic).
3.  **Search-time Synonym Expansion**: Không lưu từ đồng nghĩa vào document của sản phẩm lúc index. Mở rộng từ đồng nghĩa phải được thực hiện bằng cách viết lại câu query OR từ phía Search API thời điểm chạy tìm kiếm.
4.  **Tối ưu hóa Spellcheck**: Spellcheck không được làm chậm luồng tìm kiếm. Sử dụng API `_msearch` của OpenSearch để gộp câu lệnh gợi ý chính tả và tìm kiếm sản phẩm vào chung một request mạng.

### B. Đối với Frontend (React)
1.  **Debounce Autocomplete**: Bắt buộc phải cấu hình delay (debounce) tối thiểu 150ms khi gọi API Autocomplete từ ô tìm kiếm để bảo vệ tài nguyên hệ thống.
2.  **Ghi Analytics bất đồng bộ**: Mọi sự kiện ghi Click Log phải được gửi ngầm bất đồng bộ bằng `fetch` with `keepalive: true` hoặc `navigator.sendBeacon` để không cản trở quá trình chuyển trang của người dùng.
3.  **Header mặc định**: Mọi HTTP request tới API của hệ thống phải tự động đính kèm `X-Tenant-ID` lấy từ cấu hình Tenant hiện tại của người dùng.

### C. Quy tắc viết chú thích code (Code Comments)
1.  **Ngôn ngữ**: Tất cả các đoạn chú thích (comment) trong code phải được viết hoàn toàn bằng **tiếng Anh** (English). Tuyệt đối không viết tiếng Việt hoặc kết hợp cả hai ngôn ngữ.
2.  **Định dạng**: Mỗi dòng chú thích chỉ được mô tả trực tiếp nội dung đoạn mã bên dưới thực hiện việc gì. Tuyệt đối **không đánh số thứ tự** hoặc phân chia thành các bước (ví dụ: viết `Normalize query` thay vì `1. Normalize query`).

---

## 4. Quy trình làm việc của Agent

Trước khi triển khai bất kỳ tính năng hoặc sửa lỗi nào:
1.  Đọc tài liệu kiến trúc tương ứng trong `sa/` và User Story trong `ba/user-stories/`.
2.  Đọc các quy tắc trong `.agents/rules.md`.
3.  Viết migration bằng Goose trước nếu có thay đổi cơ sở dữ liệu.
4.  Triển khai logic nghiệp vụ và viết Unit Test (sử dụng mocking cho các service ngoài như OpenSearch, Kafka, Translate, OpenAI).
