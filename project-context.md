# Amaze Search Engine

## 1. Tổng quan dự án

Amaze Search Engine là hệ thống tìm kiếm sản phẩm thông minh dành cho sàn thương mại điện tử đa ngôn ngữ.

Mục tiêu của hệ thống:

- Tìm kiếm sản phẩm tốc độ cao
- Autocomplete
- Spellcheck
- Synonym Expansion
- Multilingual Search
- Ranking
- Search Analytics
- AI Suggestion Engine

Dự án được xây dựng nhằm phục vụ bài đánh giá năng lực (PA) nên ngoài việc code chạy được, cần thể hiện:

- Business Analysis
- Solution Architecture
- Backend Engineering
- Frontend Engineering
- AI-assisted Development
- Documentation

---

# 2. Kiến trúc tổng thể

Seller
|
v
Product API
|
v
PostgreSQL
|
v
Search Ingestion Pipeline
|
+--> Google Translate
|
v
OpenSearch

-----------------------------------

Buyer
|
v
Search UI
|
v
Search API
|
+--> Redis
|
+--> OpenSearch
|
+--> PostgreSQL
|
v
Search Result

-----------------------------------

Analytics Job
|
v
OpenAI
|
v
AI Suggestions
|
v
Admin Approval

---

# 3. Công nghệ sử dụng

## Frontend

- ReactJS
- Vite

## Backend

- Golang
- Gin Framework

## Database

- PostgreSQL

## Search Engine

- OpenSearch

## Cache

- Redis

## Translation

- Google Translate API

## AI

- OpenAI API

## Infrastructure

- Docker Compose

---

# 4. Quyết định kiến trúc

## Product Service

Chịu trách nhiệm:

- Tạo sản phẩm
- Cập nhật sản phẩm
- Xóa sản phẩm
- Publish sản phẩm

Dữ liệu lưu tại:

- PostgreSQL

Product Database là Source Of Truth.

---

## Search Service

Chịu trách nhiệm:

- Search
- Autocomplete
- Spellcheck
- Synonym Expansion
- Translation Expansion
- Ranking

Dữ liệu lưu tại:

- OpenSearch

---

## Analytics Service

Chịu trách nhiệm:

- Search Logs
- Click Logs
- Search Statistics

Dữ liệu lưu tại:

- PostgreSQL

---

## AI Service

Chịu trách nhiệm:

- Đề xuất synonym
- Đề xuất typo correction
- Đề xuất translation mapping

Lưu ý:

OpenAI chỉ được sử dụng cho các batch job offline.

KHÔNG được gọi OpenAI trong luồng search realtime.

---

# 5. Luồng Search

1. User nhập từ khóa.
2. Search API nhận request.
3. Normalize query.
4. Spellcheck query.
5. Synonym expansion.
6. Translation expansion.
7. Kiểm tra Redis cache.
8. Query OpenSearch.
9. Ranking kết quả.
10. Trả kết quả.
11. Ghi analytics.

---

# 6. Luồng Product Ingestion

1. Seller tạo sản phẩm.
2. Lưu PostgreSQL.
3. Publish ProductCreated Event.
4. Dịch nội dung bằng Google Translate.
5. Sinh keyword.
6. Tokenization.
7. Tạo Search Document.
8. Index OpenSearch.

---

# 7. Cấu trúc repository

amaze-search-engine/

BA/
SA/
BE/
FE/
AI/
INFRA/

---

# 8. Cấu trúc Backend

BE/

cmd/

internal/

product/

search/

analytics/

ai/

shared/

pkg/

configs/

migrations/

tests/

---

# 9. Cấu trúc Frontend

FE/

src/

app/

components/

services/

hooks/

types/

---

# 10. Coding Convention

## Backend

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

## Frontend

Áp dụng:

- Feature-based structure
- Reusable Components
- API Service Layer

Không được:

- Gọi API trực tiếp trong page component

---

# 11. Thứ tự triển khai User Story

US-001 Product Ingestion

US-002 Product Search

US-003 Autocomplete

US-004 Spellcheck

US-005 Synonym Expansion

US-006 Multilingual Search

US-007 Ranking Engine

US-008 Click Tracking

US-009 Search Analytics

US-010 AI Suggestion Engine

US-011 Synonym Management

US-012 Approve AI Suggestion

---

# 12. Quy tắc dành cho AI Agent

Trước khi code bất kỳ User Story nào, phải đọc:

1. PROJECT_CONTEXT.md
2. SA/system-design.md
3. SA/erd.md
4. SA/opensearch-design.md
5. User Story tương ứng

Không được tự ý thay đổi kiến trúc nếu chưa cập nhật tài liệu SA.

Mọi implementation phải tuân thủ kiến trúc đã được mô tả trong tài liệu.