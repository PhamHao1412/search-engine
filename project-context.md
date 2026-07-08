# Swift Search Engine

## 1. Tổng quan dự án

Swift Search Engine là hệ thống tìm kiếm sản phẩm thông minh dành cho sàn thương mại điện tử đa ngôn ngữ.

Mục tiêu của hệ thống:

- Tìm kiếm sản phẩm tốc độ cao
- Autocomplete (Gợi ý tự động hoàn thành)
- Spellcheck (Sửa lỗi chính tả từ truy vấn)
- Synonym Expansion (Mở rộng từ đồng nghĩa)
- Multilingual Search (Tìm kiếm đa ngôn ngữ: vi, en, th)
- Ranking Engine (Xếp hạng sản phẩm thông minh)
- Search Analytics & Click Tracking
- AI Suggestion Engine (Sinh từ đồng nghĩa & lỗi chính tả gợi ý từ log)

Dự án được xây dựng nhằm phục vụ bài đánh giá năng lực (PA) nên ngoài việc code chạy được, cần thể hiện:

- Business Analysis
- Solution Architecture
- Backend Engineering
- Frontend Engineering
- AI-assisted Development
- Documentation

---

## 2. Kiến trúc tổng thể

### Luồng Product Ingestion (Đồng bộ dữ liệu sản phẩm)

```
Seller
  |
  v
Product Service (serverd)
  |
  +--> Ghi Database: PostgreSQL (schema: `product_svc`)
  |
  +--> Phát sự kiện `ProductCreated`
        |
        v
      Kafka (Topic: `product-ingestion`)
        |
        v (Consumer)
      Search Ingestion Worker (workerd)
        |
        +--> Gọi Google Translate API (Dịch đa ngôn ngữ sang: en, th)
        |
        v
      Index dữ liệu vào OpenSearch (Index: `products`)
      (đồng thời lưu các bản dịch phi bản xứ vào PostgreSQL schema `search_svc`)
```

-----------------------------------

### Luồng Buyer Search (Tìm kiếm phía người dùng)

```
Buyer
  |
  v
Search UI (Frontend React/Vite)
  |
  v
Search Service (serverd)
  |
  +--> Kiểm tra Cache: Redis (Lưu trữ kết quả tìm kiếm, synonyms)
  |
  +--> Tìm kiếm & Phân trang: OpenSearch (Sử dụng Multi-Match Query & Synonym/Spellcheck Expansion)
  |
  v (Trả kết quả về cho Buyer)
Search Result
  |
  +--> Click Tracking & Search Logs (Ghi nhận không đồng bộ qua Goroutines)
        |
        v
      Ghi vào Database: PostgreSQL (schema: `search_svc`)
```

-----------------------------------

### Luồng AI Suggestions (Offline Job & Admin Approval)

```
Search Service (workerd)
  | (Đọc Search Logs từ PostgreSQL)
  v
Gọi OpenAI API (gpt-4o-mini để sinh đề xuất Từ đồng nghĩa/Sửa lỗi chính tả)
  |
  v
Lưu ý kiến đề xuất dạng "pending" vào PostgreSQL (schema: `search_svc`)
  |
  v (Admin xem đề xuất trên Admin Panel)
Admin duyệt/từ chối đề xuất
  |
  +--> Nếu duyệt: Lưu vào PostgreSQL (active) & Đồng bộ cấu trúc Dictionary của OpenSearch
  |
  +--> Xóa Cache của Tenant tương ứng trên Redis
```

-----------------------------------

### Luồng AI Conversational Assistant (Trợ lý AI Đàm thoại)

```
Admin
  |
  v
Search UI (Frontend React/Vite - Tab Trợ lý AI)
  |
  v
Search Service (serverd)
  |
  +--> Quản lý phiên/Tin nhắn: PostgreSQL (bảng `assistant_conversations` & `assistant_messages`)
  |
  +--> Đàm thoại & Gọi hàm: OpenAI API (gpt-4o-mini & Tool Calling)
         |
         +--> Tra cứu sản phẩm (OpenSearch)
         +--> Lập đề xuất thêm/xóa từ đồng nghĩa/sửa lỗi chính tả
  |
  v
Giao diện hiển thị đề xuất (Action Cards)
  |
  v (Admin click phê duyệt)
Lưu DB chính thức & Invalidate Cache Redis
```

---


## 3. Công nghệ sử dụng

### Frontend

- ReactJS
- Vite
- TypeScript

### Backend

- Golang (Gin Framework)
- GORM (Object Relational Mapping)

### Database

- PostgreSQL (Source of Truth và Logs/Analytics)

### Search Engine

- OpenSearch (Document Store & Vector/Text Search)
- OpenSearch Dashboards (GUI Management)

### Cache

- Redis

### Message Broker

- Kafka (KRaft mode, topic: `product-ingestion`)
- Kafka UI (GUI Management)

### Translation

- Google Translate API (Free GTX API)

### AI

- OpenAI API (Model: `gpt-4o-mini` dùng cho job offline)

### Infrastructure

- Docker Compose

---

## 4. Quyết định kiến trúc

### Product Service

Chịu trách nhiệm:

- Tạo sản phẩm
- Cập nhật sản phẩm
- Xóa sản phẩm
- Publish sản phẩm qua Kafka event

Dữ liệu lưu tại:

- PostgreSQL (schema `product_svc`)

Product Database là Source Of Truth.

---

### Search Service

Chịu trách nhiệm:

- Search, Autocomplete & Hot Keywords
- Spellcheck & Synonym Expansion
- Event-driven Product Ingestion (Kafka Consumer)
- Click Tracking & Search Logs
- AI Suggestions Engine (Offline Job)
- Admin Management APIs (Synonyms, Spellcheck, AI Suggestions)

Dữ liệu lưu tại:

- OpenSearch (index `products`)
- PostgreSQL (schema `search_svc` cho logs, dictionaries, translations)
- Redis (cache tìm kiếm và dictionary)

---

### AI Service (Tích hợp trong Search Service)

Chịu trách nhiệm:

- Đề xuất synonym dựa trên Search Logs
- Đề xuất typo correction dựa trên Search Logs

Lưu ý:

- OpenAI chỉ được sử dụng cho các batch job offline.
- KHÔNG được gọi OpenAI trong luồng search realtime để đảm bảo hiệu năng.

---

## 5. Luồng Search

1. User nhập từ khóa trên Search UI.
2. Search API nhận request (X-Tenant-ID, X-Language-Key).
3. Chuẩn hóa (Normalize) query, kiểm tra độ dài (< 100 ký tự).
4. Kiểm tra Redis cache cho query. Nếu có cache, trả kết quả ngay.
5. Spellcheck query sử dụng OpenSearch Phrase Suggester hoặc dictionary cục bộ.
6. Mở rộng truy vấn (Synonym Expansion) bằng cách kết hợp từ đồng nghĩa trong PostgreSQL.
7. Query OpenSearch sử dụng Multi-Match Query trên các trường tương ứng với ngôn ngữ (vi, en, th).
8. Thực hiện chấm điểm (Ranking Engine) thông qua `function_score`:
   - Boost sản phẩm nổi bật (featured).
   - Giảm điểm (decay) sản phẩm hết hàng (inventory = 0).
   - Boost chính xác theo cụm từ (match_phrase).
9. Trả kết quả tìm kiếm cho người dùng.
10. Ghi nhận Search Logs không đồng bộ (qua Goroutine) vào PostgreSQL.

---

## 6. Luồng Product Ingestion

1. Seller tạo/cập nhật sản phẩm qua Product API.
2. Lưu thông tin gốc vào PostgreSQL (schema `product_svc`).
3. Phát sự kiện `ProductCreated` / `ProductUpdated` lên Kafka.
4. Search Ingestion Worker (`workerd`) nhận sự kiện từ Kafka.
5. Dịch các trường văn bản (name, description, category) sang tiếng Anh (en) và tiếng Thái (th) bằng Google Translate.
6. Lưu bản dịch phụ vào PostgreSQL (schema `search_svc`).
7. Tạo Search Document với đầy đủ các trường đa ngôn ngữ.
8. Index Search Document vào OpenSearch (index `products`).

---

## 7. Cấu trúc repository

```
swift-search-engine/
├── ba/                # Business Analysis (BRD, User Stories, Acceptance Criteria)
├── sa/                # Solution Architecture (System Design, ERD, OpenSearch Design)
├── backend/           # Mã nguồn backend (Golang)
│   ├── product-service/
│   └── search-service/
├── frontend/          # Mã nguồn frontend (React, Vite, TS)
├── ai/                # Thư mục trống (tương lai)
├── docs/              # Tài liệu dự án (tương lai)
├── docker-compose.yml # File cấu hình Docker Compose cho hạ tầng
└── Makefile           # Bộ lệnh Makefile tiện ích để chạy dịch vụ
```

---

## 8. Cấu trúc Backend

Mỗi service trong thư mục `backend/` được cấu trúc theo mô hình **Clean Architecture**:

```
backend/
├── product-service/
│   ├── cmd/
│   │   └── serverd/           # Điểm khởi chạy API Server
│   └── internal/
│       ├── app/               # Cấu hình, kết nối DB và khởi tạo router
│       ├── entity/            # Khai báo Struct / Model thực thể sản phẩm
│       ├── handler/           # HTTP Handlers (Gin) tiếp nhận request
│       ├── infrastructure/    # GORM Postgres, Kafka Writer setup
│       ├── repository/        # Lớp tương tác DB (GORM)
│       └── service/           # Lớp nghiệp vụ xử lý Product logic
│
└── search-service/
    ├── cmd/
    │   ├── serverd/           # Điểm khởi chạy API Server phục vụ Storefront & Admin
    │   └── workerd/           # Điểm khởi chạy background worker tiêu thụ sự kiện từ Kafka
    └── internal/
        ├── app/               # Khởi tạo App, kết nối DB/Kafka/OpenSearch/Redis
        ├── entity/            # Model dữ liệu (Search Logs, Synonyms, Spellcheck, v.v.)
        ├── handler/           # HTTP Handlers cho Search & Admin APIs
        ├── infrastructure/    # Khách hàng kết nối Redis, OpenSearch, Kafka Reader/Writer, GORM
        ├── repository/        # Lớp tương tác DB và OpenSearch Indexer
        └── service/           # Lớp nghiệp vụ xử lý tìm kiếm, đồng bộ, AI suggestions
```

---

## 9. Cấu trúc Frontend

Mã nguồn Frontend nằm tại thư mục `frontend/` sử dụng React, Vite và TS:

```
frontend/
└── src/
    ├── main.tsx              # Điểm khởi chạy ứng dụng React
    ├── App.tsx               # Cấu hình routes chính
    ├── index.css             # CSS toàn cục và Tailwind tokens
    ├── pages/                # Các trang chức năng của Storefront & Admin
    │   ├── Home.tsx          # Trang chủ lựa chọn Tenant / Demo
    │   ├── Storefront.tsx    # Trang giao diện tìm kiếm sản phẩm cho Buyer
    │   ├── ProductDetail.tsx # Trang xem chi tiết sản phẩm
    │   └── Admin.tsx         # Trang quản trị dành cho Admin (duyệt gợi ý AI, cấu hình từ điển)
    ├── components/           # Các component tái sử dụng (DebugPanel, v.v.)
    ├── context/              # Lưu trữ Context (TenantContext quản lý Tenant hiện tại)
    ├── services/             # Lớp giao tiếp gọi API Backend (api.ts)
    ├── types/                # Khai báo kiểu TypeScript
    └── vite-env.d.ts
```

---

## 10. Coding Convention

### Backend

Áp dụng:

- Clean Architecture
- Dependency Injection
- Repository Pattern
- Service Layer Pattern

Không được:

- Viết business logic trong handler
- Query database trực tiếp từ handler
- Query OpenSearch trực tiếp từ handler

---

### Frontend

Áp dụng:

- Feature-based structure
- Reusable Components
- API Service Layer

Không được:

- Gọi API trực tiếp trong page component

---

## 11. Thứ tự triển khai User Story

- [x] **US-001 Product Ingestion** [Completed] - Đồng bộ sản phẩm từ Postgres qua Kafka sang OpenSearch, có dịch đa ngôn ngữ.
- [x] **US-002 Product Search** [Completed] - Tìm kiếm sản phẩm đa ngôn ngữ trên OpenSearch.
- [x] **US-003 Autocomplete** [Completed] - Gợi ý tự động hoàn thành từ khóa khi nhập liệu.
- [x] **US-004 Spellcheck** [Completed] - Kiểm tra lỗi chính tả sử dụng suggester của OpenSearch.
- [x] **US-005 Synonym Expansion** [Completed] - Mở rộng truy vấn dựa trên từ điển đồng nghĩa của Tenant.
- [x] **US-006 Multilingual Search** [Completed] - Tìm kiếm và hiển thị đa ngôn ngữ dựa trên Language Key (`vi`, `en`, `th`).
- [x] **US-007 Ranking Engine** [Completed] - Chấm điểm xếp hạng sản phẩm (featured boost, inventory decay, match_phrase boost).
- [x] **US-008 Click Tracking** [Completed] - Ghi nhận không đồng bộ hành vi click vào sản phẩm của Buyer.
- [x] **US-009 Search Analytics** [Completed] - Ghi nhận lịch sử tìm kiếm, hiển thị các từ khóa hot (Hot Keywords).
- [x] **US-010 AI Suggestion Engine** [Completed] - Sử dụng OpenAI offline job để quét log tìm gợi ý Synonym & Spellcheck mới.
- [x] **US-011 Synonym Management** [Completed] - API và UI quản trị thêm/xóa từ đồng nghĩa & sửa lỗi chính tả thủ công.
- [x] **US-012 Approve AI Suggestion** [Completed] - API và UI quản trị phê duyệt/bác bỏ đề xuất gợi ý của AI.
- [x] **US-013 Admin AI Assistant** [Completed] - Trợ lý AI đàm thoại quản trị, lưu trữ phiên chat vào Postgres, hỗ trợ đề xuất synonym/spellcheck bằng ngôn ngữ tự nhiên.

---

## 12. Quy tắc dành cho AI Agent

Trước khi code bất kỳ User Story nào, phải đọc:

1. [project-context.md](file:///Users/haopham/go-playground/search-engine/project-context.md)
2. [sa/01-architecture.md](file:///Users/haopham/go-playground/search-engine/sa/01-architecture.md)
3. [sa/02-erd.md](file:///Users/haopham/go-playground/search-engine/sa/02-erd.md)
4. [sa/03-opensearch-design.md](file:///Users/haopham/go-playground/search-engine/sa/03-opensearch-design.md)
5. User Story tương ứng nằm trong `ba/user-stories/`

Không được tự ý thay đổi kiến trúc nếu chưa cập nhật tài liệu SA.

Mọi implementation phải tuân thủ kiến trúc đã được mô tả trong tài liệu.